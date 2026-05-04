package services

import (
	"context"
	"errors"
	"fmt"
	"net/mail"

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
//
// The caller must supply EXACTLY ONE of req.UserID (recipient resolved via
// custapi) or req.ToAddress (direct dispatch for external addresses that
// aren't in our system). Supplying both - or neither - is rejected as
// invalid input before we touch the DB or the renderer.
func (s *EmailService) SendEmail(ctx context.Context, req models.SendEmailRequest) (*models.SendEmailResponse, error) {
	recipient, err := resolveRecipient(ctx, s.deps.Users, req)
	if err != nil {
		return nil, err
	}

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

	from := s.deps.DefaultFrom
	if ver.FromAddress != "" {
		from = ver.FromAddress
	}

	now := s.deps.Clock.Now()
	row := &models.EmailOutbox{
		ID:                s.deps.IDs.New(),
		TemplateVersionID: ver.ID,
		UserID:            recipient.UserID, // nil for direct sends
		ToAddress:         recipient.Email,
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

// recipient is the result of normalising req.UserID/req.ToAddress into one
// canonical (UserID, Email) pair. UserID is nil when the caller dispatched
// directly to ToAddress.
type recipient struct {
	UserID *uuid.UUID
	Email  string
}

func resolveRecipient(ctx context.Context, users UserResolver, req models.SendEmailRequest) (recipient, error) {
	hasUser := req.UserID != nil && *req.UserID != uuid.Nil
	hasAddr := req.ToAddress != ""

	switch {
	case hasUser && hasAddr:
		return recipient{}, apperrors.InvalidParams(
			"provide either user_id OR to_address, not both", nil)
	case !hasUser && !hasAddr:
		return recipient{}, apperrors.InvalidParams(
			"recipient required: provide user_id (resolved via custapi) or to_address", nil)
	case hasUser:
		email, err := users.ResolveEmail(ctx, *req.UserID)
		if err != nil {
			return recipient{}, err
		}
		uid := *req.UserID
		return recipient{UserID: &uid, Email: email}, nil
	default:
		// Direct send: validate the address up-front so a malformed
		// to_address doesn't silently land in the outbox just to fail at
		// the SMTP layer minutes later.
		if _, err := mail.ParseAddress(req.ToAddress); err != nil {
			return recipient{}, apperrors.InvalidParams(
				"to_address is not a valid email: "+err.Error(), err)
		}
		return recipient{UserID: nil, Email: req.ToAddress}, nil
	}
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
