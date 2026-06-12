package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/internal/models"
)

type TaskRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

type TaskFilter struct {
	UserID           string // empty for admin "all users" view
	Status           string
	Search           string
	SortBy           string // due_date, priority, created_at
	SortDir          string // asc, desc
	Page             int
	PageSize         int
	IncludeUserEmail bool // when true (admin "all users" view), join in the owner's email
}

var allowedSortColumns = map[string]string{
	"due_date":   "due_date",
	"priority":   "priority_rank",
	"created_at": "created_at",
}

func (r *TaskRepository) List(ctx context.Context, f TaskFilter) ([]models.Task, int, error) {
	var conditions []string
	var args []interface{}
	argN := 1

	if f.UserID != "" {
		conditions = append(conditions, fmt.Sprintf("tasks.user_id = $%d", argN))
		args = append(args, f.UserID)
		argN++
	}

	if f.Status != "" {
		conditions = append(conditions, fmt.Sprintf("tasks.status = $%d", argN))
		args = append(args, f.Status)
		argN++
	}

	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf("tasks.title ILIKE $%d", argN))
		args = append(args, "%"+f.Search+"%")
		argN++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Total count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks %s", where)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting tasks: %w", err)
	}

	sortCol := "created_at"
	if col, ok := allowedSortColumns[f.SortBy]; ok {
		sortCol = col
	}
	sortDir := "DESC"
	if strings.EqualFold(f.SortDir, "asc") {
		sortDir = "ASC"
	}

	limit := f.PageSize
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	page := f.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	selectCols := "tasks.id, tasks.user_id, tasks.title, tasks.description, tasks.status, tasks.priority, tasks.due_date, tasks.created_at, tasks.updated_at"
	join := ""
	if f.IncludeUserEmail {
		selectCols += ", users.email"
		join = "LEFT JOIN users ON users.id = tasks.user_id"
	}

	query := fmt.Sprintf(`
		SELECT %s,
			CASE tasks.priority WHEN 'high' THEN 3 WHEN 'medium' THEN 2 ELSE 1 END AS priority_rank
		FROM tasks
		%s
		%s
		ORDER BY %s %s NULLS LAST, tasks.created_at DESC
		LIMIT $%d OFFSET $%d
	`, selectCols, join, where, sortCol, sortDir, argN, argN+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		var priorityRank int
		if f.IncludeUserEmail {
			if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Priority,
				&t.DueDate, &t.CreatedAt, &t.UpdatedAt, &t.UserEmail, &priorityRank); err != nil {
				return nil, 0, fmt.Errorf("scanning task: %w", err)
			}
		} else {
			if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Priority,
				&t.DueDate, &t.CreatedAt, &t.UpdatedAt, &priorityRank); err != nil {
				return nil, 0, fmt.Errorf("scanning task: %w", err)
			}
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if tasks == nil {
		tasks = []models.Task{}
	}

	return tasks, total, nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id string) (*models.Task, error) {
	var t models.Task
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, title, description, status, priority, due_date, created_at, updated_at
		FROM tasks WHERE id = $1
	`, id).Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

type CreateTaskParams struct {
	UserID      string
	Title       string
	Description string
	Status      models.TaskStatus
	Priority    models.TaskPriority
	DueDate     *string // RFC3339, nullable
}

func (r *TaskRepository) Create(ctx context.Context, p CreateTaskParams) (*models.Task, error) {
	var t models.Task
	err := r.pool.QueryRow(ctx, `
		INSERT INTO tasks (user_id, title, description, status, priority, due_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, title, description, status, priority, due_date, created_at, updated_at
	`, p.UserID, p.Title, p.Description, p.Status, p.Priority, p.DueDate).
		Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("creating task: %w", err)
	}
	return &t, nil
}

type UpdateTaskParams struct {
	Title       *string
	Description *string
	Status      *models.TaskStatus
	Priority    *models.TaskPriority
	DueDate     **string // pointer-to-pointer: nil = not provided, non-nil pointing to nil = clear due date
}

func (r *TaskRepository) Update(ctx context.Context, id string, p UpdateTaskParams) (*models.Task, error) {
	var setClauses []string
	var args []interface{}
	argN := 1

	if p.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argN))
		args = append(args, *p.Title)
		argN++
	}
	if p.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argN))
		args = append(args, *p.Description)
		argN++
	}
	if p.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argN))
		args = append(args, *p.Status)
		argN++
	}
	if p.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argN))
		args = append(args, *p.Priority)
		argN++
	}
	if p.DueDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("due_date = $%d", argN))
		args = append(args, *p.DueDate)
		argN++
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	setClauses = append(setClauses, "updated_at = now()")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE tasks SET %s
		WHERE id = $%d
		RETURNING id, user_id, title, description, status, priority, due_date, created_at, updated_at
	`, strings.Join(setClauses, ", "), argN)

	var t models.Task
	err := r.pool.QueryRow(ctx, query, args...).
		Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("updating task: %w", err)
	}
	return &t, nil
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
