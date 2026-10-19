package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/internal/models"
)

type AttachmentRepository struct {
	pool *pgxpool.Pool
}

func NewAttachmentRepository(pool *pgxpool.Pool) *AttachmentRepository {
	return &AttachmentRepository{pool: pool}
}

type CreateAttachmentParams struct {
	TaskID      string
	UserID      string
	Filename    string
	ContentType string
	Data        []byte
}

const attachmentCols = "id, task_id, user_id, filename, content_type, size_bytes, created_at"

func scanAttachment(row pgx.Row, a *models.TaskAttachment) error {
	return row.Scan(&a.ID, &a.TaskID, &a.UserID, &a.Filename, &a.ContentType, &a.SizeBytes, &a.CreatedAt)
}

func (r *AttachmentRepository) Create(ctx context.Context, p CreateAttachmentParams) (*models.TaskAttachment, error) {
	var a models.TaskAttachment
	err := scanAttachment(r.pool.QueryRow(ctx, `
		INSERT INTO task_attachments (task_id, user_id, filename, content_type, size_bytes, data)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+attachmentCols,
		p.TaskID, p.UserID, p.Filename, p.ContentType, int64(len(p.Data)), p.Data), &a)
	if err != nil {
		return nil, fmt.Errorf("creating attachment: %w", err)
	}
	return &a, nil
}

func (r *AttachmentRepository) ListByTask(ctx context.Context, taskID string) ([]models.TaskAttachment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+attachmentCols+`
		FROM task_attachments
		WHERE task_id = $1
		ORDER BY created_at DESC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("listing attachments: %w", err)
	}
	defer rows.Close()

	attachments := []models.TaskAttachment{}
	for rows.Next() {
		var a models.TaskAttachment
		if err := scanAttachment(rows, &a); err != nil {
			return nil, fmt.Errorf("scanning attachment: %w", err)
		}
		attachments = append(attachments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return attachments, nil
}

// GetByID returns attachment metadata without its contents.
func (r *AttachmentRepository) GetByID(ctx context.Context, id string) (*models.TaskAttachment, error) {
	var a models.TaskAttachment
	err := scanAttachment(r.pool.QueryRow(ctx, `SELECT `+attachmentCols+` FROM task_attachments WHERE id = $1`, id), &a)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

// GetData returns the stored contents of an attachment. ErrNotFound is returned
// for legacy rows that were stored on disk and have no contents.
func (r *AttachmentRepository) GetData(ctx context.Context, id string) ([]byte, error) {
	var data []byte
	err := r.pool.QueryRow(ctx, `SELECT data FROM task_attachments WHERE id = $1`, id).Scan(&data)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if data == nil {
		return nil, ErrNotFound
	}
	return data, nil
}

func (r *AttachmentRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM task_attachments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting attachment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
