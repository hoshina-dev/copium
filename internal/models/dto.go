package models

import (
	"time"

	"github.com/google/uuid"
)

// --- requests ---

// CreateTemplateRequest is the body of POST /templates.
type CreateTemplateRequest struct {
	Code        string `json:"code" validate:"required,min=1,max=128"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`
} // @name CreateTemplateRequest

// CreateTemplateVersionRequest is the body of POST /templates/{id}/versions.
type CreateTemplateVersionRequest struct {
	Subject      string  `json:"subject" validate:"required"`
	BodyHTML     string  `json:"body_html" validate:"required"`
	BodyText     string  `json:"body_text"`
	ParamsSchema JSONMap `json:"params_schema" swaggertype:"object" validate:"required"`
	FromAddress  string  `json:"from_address"`
} // @name CreateTemplateVersionRequest

// SetActiveVersionRequest is the body of PATCH /templates/{id}/active-version.
type SetActiveVersionRequest struct {
	VersionID uuid.UUID `json:"version_id" validate:"required" swaggertype:"string" format:"uuid"`
} // @name SetActiveVersionRequest

// SendEmailRequest is the body of POST /emails/send.
type SendEmailRequest struct {
	UserID     uuid.UUID `json:"user_id" validate:"required" swaggertype:"string" format:"uuid"`
	TemplateID uuid.UUID `json:"template_id" validate:"required" swaggertype:"string" format:"uuid"`
	Params     JSONMap   `json:"params" swaggertype:"object"`
} // @name SendEmailRequest

// --- responses ---

// TemplateResponse is one logical template.
type TemplateResponse struct {
	ID              uuid.UUID  `json:"id" swaggertype:"string" format:"uuid"`
	Code            string     `json:"code"`
	Name            string     `json:"name"`
	Description     string     `json:"description,omitempty"`
	ActiveVersionID *uuid.UUID `json:"active_version_id,omitempty" swaggertype:"string" format:"uuid"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
} // @name TemplateResponse

// TemplateVersionResponse is one immutable template version.
type TemplateVersionResponse struct {
	ID           uuid.UUID `json:"id" swaggertype:"string" format:"uuid"`
	TemplateID   uuid.UUID `json:"template_id" swaggertype:"string" format:"uuid"`
	Version      int       `json:"version"`
	Subject      string    `json:"subject"`
	BodyHTML     string    `json:"body_html"`
	BodyText     string    `json:"body_text,omitempty"`
	ParamsSchema JSONMap   `json:"params_schema" swaggertype:"object"`
	FromAddress  string    `json:"from_address,omitempty"`
	CreatedBy    string    `json:"created_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
} // @name TemplateVersionResponse

// SendEmailResponse is the 202 body returned from POST /emails/send.
type SendEmailResponse struct {
	OutboxID uuid.UUID `json:"outbox_id" swaggertype:"string" format:"uuid"`
	Status   string    `json:"status" example:"queued"`
} // @name SendEmailResponse

// OutboxResponse is the public view of one queued/sent email.
type OutboxResponse struct {
	ID                uuid.UUID  `json:"id" swaggertype:"string" format:"uuid"`
	TemplateVersionID uuid.UUID  `json:"template_version_id" swaggertype:"string" format:"uuid"`
	UserID            uuid.UUID  `json:"user_id" swaggertype:"string" format:"uuid"`
	ToAddress         string     `json:"to_address"`
	Subject           string     `json:"subject"`
	Status            string     `json:"status" example:"sent"`
	Attempts          int        `json:"attempts"`
	MaxAttempts       int        `json:"max_attempts"`
	ScheduledAt       time.Time  `json:"scheduled_at"`
	LastError         string     `json:"last_error,omitempty"`
	Provider          string     `json:"provider,omitempty"`
	ProviderMessageID string     `json:"provider_message_id,omitempty"`
	SentAt            *time.Time `json:"sent_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
} // @name OutboxResponse

// ErrorResponse is the body returned on any non-2xx status.
type ErrorResponse struct {
	Error string `json:"error" example:"not found: template <uuid>"`
} // @name ErrorResponse
