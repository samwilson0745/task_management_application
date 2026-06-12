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
	SizeBytes   int64
	StoragePath string
}

func (r *AttachmentRepository) Create(ctx context.Context, p CreateAttachmentParams) (*models.TaskAttachment, error) {
	var a models.TaskAttachment
	err := r.pool.QueryRow(ctx, `
		INSERT INTO task_attachments (task_id, user_id, filename, content_type, size_bytes, storage_path)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, task_id, user_id, filename, content_type, size_bytes, storage_path, created_at
	`, p.TaskID, p.UserID, p.Filename, p.ContentType, p.SizeBytes, p.StoragePath).
		Scan(&a.ID, &a.TaskID, &a.UserID, &a.Filename, &a.ContentType, &a.SizeBytes, &a.StoragePath, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating attachment: %w", err)
	}
	return &a, nil
}

func (r *AttachmentRepository) ListByTask(ctx context.Context, taskID string) ([]models.TaskAttachment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, task_id, user_id, filename, content_type, size_bytes, storage_path, created_at
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
		if err := rows.Scan(&a.ID, &a.TaskID, &a.UserID, &a.Filename, &a.ContentType, &a.SizeBytes, &a.StoragePath, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning attachment: %w", err)
		}
		attachments = append(attachments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *AttachmentRepository) GetByID(ctx context.Context, id string) (*models.TaskAttachment, error) {
	var a models.TaskAttachment
	err := r.pool.QueryRow(ctx, `
		SELECT id, task_id, user_id, filename, content_type, size_bytes, storage_path, created_at
		FROM task_attachments WHERE id = $1
	`, id).Scan(&a.ID, &a.TaskID, &a.UserID, &a.Filename, &a.ContentType, &a.SizeBytes, &a.StoragePath, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
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
