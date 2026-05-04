package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/models"
)

type OutboxRepo struct{ db *gorm.DB }

func NewOutboxRepo(db *gorm.DB) *OutboxRepo { return &OutboxRepo{db: db} }

func (r *OutboxRepo) Create(ctx context.Context, o *models.EmailOutbox) error {
	return r.db.WithContext(ctx).Create(o).Error
}

// List returns outbox rows matching the filter, newest first.
func (r *OutboxRepo) List(ctx context.Context, f models.OutboxListFilter) ([]*models.EmailOutbox, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	q := r.db.WithContext(ctx).Model(&models.EmailOutbox{})
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at < ?", *f.To)
	}
	var rows []*models.EmailOutbox
	if err := q.Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *OutboxRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.EmailOutbox, error) {
	var o models.EmailOutbox
	if err := r.db.WithContext(ctx).First(&o, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NotFound("outbox "+id.String(), err)
		}
		return nil, err
	}
	return &o, nil
}

// ClaimDue atomically picks up to limit ready 'queued' rows whose scheduled_at
// has passed, flips them to 'sending', and returns them. Uses
// FOR UPDATE SKIP LOCKED so concurrent workers never pick the same row.
func (r *OutboxRepo) ClaimDue(ctx context.Context, now time.Time, limit int) ([]*models.EmailOutbox, error) {
	var picked []*models.EmailOutbox
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ids []uuid.UUID
		// Step 1: lock ready rows.
		rows, err := tx.Raw(`
			SELECT id FROM email_outbox
			WHERE status = 'queued' AND scheduled_at <= ?
			ORDER BY scheduled_at
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		`, now, limit).Rows()
		if err != nil {
			return err
		}
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		// Step 2: flip status to 'sending' so other workers ignore.
		if err := tx.Model(&models.EmailOutbox{}).
			Where("id IN ?", ids).
			Updates(map[string]any{"status": models.OutboxStatusSending, "updated_at": now}).
			Error; err != nil {
			return err
		}
		// Step 3: hydrate full rows.
		return tx.Where("id IN ?", ids).Find(&picked).Error
	})
	return picked, err
}

func (r *OutboxRepo) MarkSent(ctx context.Context, id uuid.UUID, provider, providerID string, sentAt time.Time) error {
	res := r.db.WithContext(ctx).Model(&models.EmailOutbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":              models.OutboxStatusSent,
			"provider":            provider,
			"provider_message_id": providerID,
			"sent_at":             sentAt,
			"updated_at":          sentAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperrors.NotFound("outbox "+id.String(), nil)
	}
	return nil
}

func (r *OutboxRepo) MarkFailureAndReschedule(ctx context.Context, id uuid.UUID, attemptErr string, nextAttemptAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row models.EmailOutbox
		if err := tx.First(&row, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.NotFound("outbox "+id.String(), err)
			}
			return err
		}
		row.Attempts++
		row.LastError = attemptErr
		row.UpdatedAt = nextAttemptAt
		if row.Attempts >= row.MaxAttempts {
			row.Status = models.OutboxStatusDead
		} else {
			row.Status = models.OutboxStatusQueued
			row.ScheduledAt = nextAttemptAt
		}
		return tx.Save(&row).Error
	})
}
