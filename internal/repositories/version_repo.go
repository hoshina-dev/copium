package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/models"
)

type TemplateVersionRepo struct{ db *gorm.DB }

func NewTemplateVersionRepo(db *gorm.DB) *TemplateVersionRepo { return &TemplateVersionRepo{db: db} }

func (r *TemplateVersionRepo) Create(ctx context.Context, v *models.EmailTemplateVersion) error {
	if err := r.db.WithContext(ctx).Create(v).Error; err != nil {
		if isUniqueViolation(err) {
			return apperrors.Conflict("template_id+version", err)
		}
		return err
	}
	return nil
}

func (r *TemplateVersionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.EmailTemplateVersion, error) {
	var v models.EmailTemplateVersion
	if err := r.db.WithContext(ctx).First(&v, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NotFound("template_version "+id.String(), err)
		}
		return nil, err
	}
	return &v, nil
}

func (r *TemplateVersionRepo) GetByTemplateAndVersion(ctx context.Context, templateID uuid.UUID, version int) (*models.EmailTemplateVersion, error) {
	var v models.EmailTemplateVersion
	if err := r.db.WithContext(ctx).
		First(&v, "template_id = ? AND version = ?", templateID, version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NotFound("template_version", err)
		}
		return nil, err
	}
	return &v, nil
}

func (r *TemplateVersionRepo) NextVersionNumber(ctx context.Context, templateID uuid.UUID) (int, error) {
	var max int
	row := r.db.WithContext(ctx).Raw(
		`SELECT COALESCE(MAX(version), 0) FROM email_template_versions WHERE template_id = ?`,
		templateID).Row()
	if err := row.Scan(&max); err != nil {
		return 0, err
	}
	return max + 1, nil
}

func (r *TemplateVersionRepo) ListByTemplate(ctx context.Context, templateID uuid.UUID) ([]models.EmailTemplateVersion, error) {
	var vs []models.EmailTemplateVersion
	if err := r.db.WithContext(ctx).
		Where("template_id = ?", templateID).
		Order("version DESC").
		Find(&vs).Error; err != nil {
		return nil, err
	}
	return vs, nil
}
