package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/internal/models"
)

type ActivityRepository struct {
	pool *pgxpool.Pool
}

func NewActivityRepository(pool *pgxpool.Pool) *ActivityRepository {
	return &ActivityRepository{pool: pool}
}

func (r *ActivityRepository) Log(ctx context.Context, taskID, userID, action, details string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO task_activity (task_id, user_id, action, details)
		VALUES ($1, $2, $3, $4)
	`, taskID, userID, action, details)
	if err != nil {
		return fmt.Errorf("logging activity: %w", err)
	}
	return nil
}

func (r *ActivityRepository) ListByTask(ctx context.Context, taskID string) ([]models.TaskActivity, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.task_id, a.user_id, u.email, a.action, a.details, a.created_at
		FROM task_activity a
		LEFT JOIN users u ON u.id = a.user_id
		WHERE a.task_id = $1
		ORDER BY a.created_at DESC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("listing activity: %w", err)
	}
	defer rows.Close()

	activity := []models.TaskActivity{}
	for rows.Next() {
		var a models.TaskActivity
		if err := rows.Scan(&a.ID, &a.TaskID, &a.UserID, &a.UserEmail, &a.Action, &a.Details, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning activity: %w", err)
		}
		activity = append(activity, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return activity, nil
}
