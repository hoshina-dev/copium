// Package repositories holds GORM-backed implementations of the consumer-side
// interfaces declared in package services. Each repo translates database
// errors to apperrors so service layer branches on errors.Is.
package repositories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/models"
)

type TemplateRepo struct{ db *gorm.DB }

func NewTemplateRepo(db *gorm.DB) *TemplateRepo { return &TemplateRepo{db: db} }

func (r *TemplateRepo) Create(ctx context.Context, t *models.EmailTemplate) error {
	if err := r.db.WithContext(ctx).Create(t).Error; err != nil {
		if isUniqueViolation(err) {
			return apperrors.Conflict("template code "+t.Code, err)
		}
		return err
	}
	return nil
}

func (r *TemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.EmailTemplate, error) {
	var t models.EmailTemplate
	if err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NotFound("template "+id.String(), err)
		}
		return nil, err
	}
	return &t, nil
}

func (r *TemplateRepo) GetByCode(ctx context.Context, code string) (*models.EmailTemplate, error) {
	var t models.EmailTemplate
	if err := r.db.WithContext(ctx).First(&t, "code = ?", code).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NotFound("template code "+code, err)
		}
		return nil, err
	}
	return &t, nil
}

func (r *TemplateRepo) List(ctx context.Context) ([]models.EmailTemplate, error) {
	var ts []models.EmailTemplate
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&ts).Error; err != nil {
		return nil, err
	}
	return ts, nil
}

func (r *TemplateRepo) SetActiveVersion(ctx context.Context, templateID, versionID uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&models.EmailTemplate{}).
		Where("id = ?", templateID).
		Update("active_version_id", versionID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperrors.NotFound("template "+templateID.String(), nil)
	}
	return nil
}

// Delete soft-deletes the template (GORM populates deleted_at). Versions
// and outbox rows are intentionally left alone so audit/history links keep
// working; the deleted template simply stops appearing in List/Get.
func (r *TemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&models.EmailTemplate{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperrors.NotFound("template "+id.String(), nil)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// pgx/lib/pq style: "duplicate key value violates unique constraint"
	return strings.Contains(err.Error(), "duplicate key value")
}
