package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/models"
)

type EmailDeps struct {
	Templates        TemplateRepository
	TemplateVersions TemplateVersionRepository
	Outbox           OutboxRepository
	Users            UserResolver
	Renderer         Renderer
	Sender           Sender
	Clock            Clock
	IDs              IDGen

	DefaultFrom string
	MaxAttempts int // optional; defaults to 5
}

type EmailService struct{ deps EmailDeps }

func NewEmailService(d EmailDeps) *EmailService {
	if d.MaxAttempts <= 0 {
		d.MaxAttempts = 5
	}
	return &EmailService{deps: d}
}

// SendEmail validates inputs, resolves recipient, renders, and snapshots
// everything into email_outbox. The actual transmission is the worker's job.
func (s *EmailService) SendEmail(ctx context.Context, req models.SendEmailRequest) (*models.SendEmailResponse, error) {
	tpl, err := s.deps.Templates.GetByID(ctx, req.TemplateID)
	if err != nil {
		return nil, err
	}
	if tpl.ActiveVersionID == nil {
		return nil, apperrors.InvalidParams(
			fmt.Sprintf("template %s has no active version", tpl.ID), nil)
	}
	ver, err := s.deps.TemplateVersions.GetByID(ctx, *tpl.ActiveVersionID)
	if err != nil {
		return nil, fmt.Errorf("load active version: %w", err)
	}

	rendered, err := s.deps.Renderer.Render(ver, req.Params)
	if err != nil {
		return nil, err
	}

	email, err := s.deps.Users.ResolveEmail(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	from := s.deps.DefaultFrom
	if ver.FromAddress != "" {
		from = ver.FromAddress
	}

	now := s.deps.Clock.Now()
	row := &models.EmailOutbox{
		ID:                s.deps.IDs.New(),
		TemplateVersionID: ver.ID,
		UserID:            req.UserID,
		ToAddress:         email,
		FromAddress:       from,
		Subject:           rendered.Subject,
		BodyHTML:          rendered.BodyHTML,
		BodyText:          rendered.BodyText,
		Params:            cloneParams(req.Params),
		Status:            models.OutboxStatusQueued,
		Attempts:          0,
		MaxAttempts:       s.deps.MaxAttempts,
		ScheduledAt:       now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.deps.Outbox.Create(ctx, row); err != nil {
		// Use ErrInternal so handlers map to 500 unless caller wraps something
		// more specific.
		if errors.Is(err, apperrors.ErrConflict) || errors.Is(err, apperrors.ErrNotFound) || errors.Is(err, apperrors.ErrInvalidParams) {
			return nil, err
		}
		return nil, apperrors.Internal("enqueue outbox", err)
	}
	return &models.SendEmailResponse{OutboxID: row.ID, Status: string(row.Status)}, nil
}

func (s *EmailService) GetOutbox(ctx context.Context, id uuid.UUID) (*models.EmailOutbox, error) {
	return s.deps.Outbox.GetByID(ctx, id)
}

func cloneParams(p models.JSONMap) models.JSONMap {
	if p == nil {
		return models.JSONMap{}
	}
	out := make(models.JSONMap, len(p))
	for k, v := range p {
		out[k] = v
	}
	return out
}
