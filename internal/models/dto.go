package models

import (
	"time"

	"github.com/google/uuid"
)

// --- requests ---

type CreateTemplateRequest struct {
	Code        string `json:"code" validate:"required,min=1,max=128"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`
}

type CreateTemplateVersionRequest struct {
	Subject      string  `json:"subject" validate:"required"`
	BodyHTML     string  `json:"body_html" validate:"required"`
	BodyText     string  `json:"body_text"`
	ParamsSchema JSONMap `json:"params_schema" validate:"required"`
	FromAddress  string  `json:"from_address"`
}

type SetActiveVersionRequest struct {
	VersionID uuid.UUID `json:"version_id" validate:"required"`
}

type SendEmailRequest struct {
	UserID     uuid.UUID `json:"user_id" validate:"required"`
	TemplateID uuid.UUID `json:"template_id" validate:"required"`
	Params     JSONMap   `json:"params"`
}

// --- responses ---

type TemplateResponse struct {
	ID              uuid.UUID  `json:"id"`
	Code            string     `json:"code"`
	Name            string     `json:"name"`
	Description     string     `json:"description,omitempty"`
	ActiveVersionID *uuid.UUID `json:"active_version_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type TemplateVersionResponse struct {
	ID           uuid.UUID `json:"id"`
	TemplateID   uuid.UUID `json:"template_id"`
	Version      int       `json:"version"`
	Subject      string    `json:"subject"`
	BodyHTML     string    `json:"body_html"`
	BodyText     string    `json:"body_text,omitempty"`
	ParamsSchema JSONMap   `json:"params_schema"`
	FromAddress  string    `json:"from_address,omitempty"`
	CreatedBy    string    `json:"created_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type SendEmailResponse struct {
	OutboxID uuid.UUID `json:"outbox_id"`
	Status   string    `json:"status"`
}

type OutboxResponse struct {
	ID                uuid.UUID  `json:"id"`
	TemplateVersionID uuid.UUID  `json:"template_version_id"`
	UserID            uuid.UUID  `json:"user_id"`
	ToAddress         string     `json:"to_address"`
	Subject           string     `json:"subject"`
	Status            string     `json:"status"`
	Attempts          int        `json:"attempts"`
	MaxAttempts       int        `json:"max_attempts"`
	ScheduledAt       time.Time  `json:"scheduled_at"`
	LastError         string     `json:"last_error,omitempty"`
	Provider          string     `json:"provider,omitempty"`
	ProviderMessageID string     `json:"provider_message_id,omitempty"`
	SentAt            *time.Time `json:"sent_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
