package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/models"
)

type TemplateDeps struct {
	Templates        TemplateRepository
	TemplateVersions TemplateVersionRepository
	Clock            Clock
	IDs              IDGen
}

type TemplateService struct{ deps TemplateDeps }

func NewTemplateService(d TemplateDeps) *TemplateService { return &TemplateService{deps: d} }

func (s *TemplateService) Create(ctx context.Context, req models.CreateTemplateRequest) (*models.EmailTemplate, error) {
	t := &models.EmailTemplate{
		ID:          s.deps.IDs.New(),
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   s.deps.Clock.Now(),
		UpdatedAt:   s.deps.Clock.Now(),
	}
	if err := s.deps.Templates.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TemplateService) Get(ctx context.Context, id uuid.UUID) (*models.EmailTemplate, error) {
	return s.deps.Templates.GetByID(ctx, id)
}

func (s *TemplateService) List(ctx context.Context) ([]models.EmailTemplate, error) {
	return s.deps.Templates.List(ctx)
}

func (s *TemplateService) CreateVersion(ctx context.Context, templateID uuid.UUID, req models.CreateTemplateVersionRequest) (*models.EmailTemplateVersion, error) {
	if _, err := s.deps.Templates.GetByID(ctx, templateID); err != nil {
		return nil, err
	}
	if err := compileSchema(req.ParamsSchema); err != nil {
		return nil, apperrors.InvalidParams("params_schema is not a valid JSON Schema", err)
	}
	next, err := s.deps.TemplateVersions.NextVersionNumber(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("compute next version: %w", err)
	}
	v := &models.EmailTemplateVersion{
		ID:           s.deps.IDs.New(),
		TemplateID:   templateID,
		Version:      next,
		Subject:      req.Subject,
		BodyHTML:     req.BodyHTML,
		BodyText:     req.BodyText,
		ParamsSchema: req.ParamsSchema,
		FromAddress:  req.FromAddress,
		CreatedAt:    s.deps.Clock.Now(),
	}
	if err := s.deps.TemplateVersions.Create(ctx, v); err != nil {
		return nil, err
	}
	if next == 1 {
		if err := s.deps.Templates.SetActiveVersion(ctx, templateID, v.ID); err != nil {
			return nil, fmt.Errorf("auto-activate first version: %w", err)
		}
	}
	return v, nil
}

func (s *TemplateService) ListVersions(ctx context.Context, templateID uuid.UUID) ([]models.EmailTemplateVersion, error) {
	if _, err := s.deps.Templates.GetByID(ctx, templateID); err != nil {
		return nil, err
	}
	return s.deps.TemplateVersions.ListByTemplate(ctx, templateID)
}

func (s *TemplateService) GetVersion(ctx context.Context, templateID uuid.UUID, version int) (*models.EmailTemplateVersion, error) {
	if _, err := s.deps.Templates.GetByID(ctx, templateID); err != nil {
		return nil, err
	}
	return s.deps.TemplateVersions.GetByTemplateAndVersion(ctx, templateID, version)
}

func (s *TemplateService) SetActiveVersion(ctx context.Context, templateID, versionID uuid.UUID) error {
	if _, err := s.deps.Templates.GetByID(ctx, templateID); err != nil {
		return err
	}
	v, err := s.deps.TemplateVersions.GetByID(ctx, versionID)
	if err != nil {
		return err
	}
	if v.TemplateID != templateID {
		return apperrors.InvalidParams(
			fmt.Sprintf("version %s belongs to template %s, not %s", versionID, v.TemplateID, templateID),
			nil)
	}
	return s.deps.Templates.SetActiveVersion(ctx, templateID, versionID)
}

// compileSchema verifies that params_schema is itself valid JSON Schema
// (draft 2020-12 by default). We only check compilation; we don't validate
// any document against it here.
func compileSchema(schema models.JSONMap) error {
	if schema == nil {
		return errors.New("params_schema is required")
	}
	b, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	c := jsonschema.NewCompiler()
	const url = "schema://check"
	if err := c.AddResource(url, bytes.NewReader(b)); err != nil {
		return err
	}
	if _, err := c.Compile(url); err != nil {
		return err
	}
	return nil
}
