package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OutboxStatus is the lifecycle of a queued message.
type OutboxStatus string

const (
	OutboxStatusQueued  OutboxStatus = "queued"
	OutboxStatusSending OutboxStatus = "sending"
	OutboxStatusSent    OutboxStatus = "sent"
	OutboxStatusFailed  OutboxStatus = "failed"
	OutboxStatusDead    OutboxStatus = "dead"
)

// EmailTemplate is the logical template; ActiveVersionID points at the
// version used when callers reference this template by id/code.
type EmailTemplate struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code            string     `gorm:"type:text;uniqueIndex;not null"`
	Name            string     `gorm:"type:text;not null"`
	Description     string     `gorm:"type:text"`
	ActiveVersionID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (EmailTemplate) TableName() string { return "email_templates" }

// EmailTemplateVersion is an immutable snapshot. (template_id, version) is unique.
type EmailTemplateVersion struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TemplateID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Version      int       `gorm:"not null"`
	Subject      string    `gorm:"type:text;not null"`
	BodyHTML     string    `gorm:"type:text;not null"`
	BodyText     string    `gorm:"type:text"`
	ParamsSchema JSONMap   `gorm:"type:jsonb;not null"`
	FromAddress  string    `gorm:"type:text"`
	CreatedBy    string    `gorm:"type:text"`
	CreatedAt    time.Time
}

func (EmailTemplateVersion) TableName() string { return "email_template_versions" }

// EmailOutbox is one queued/historical email send. All fields needed to
// transmit are snapshotted here so a later template edit doesn't change what
// gets sent.
//
// UserID is nullable: it's set when the recipient was resolved from custapi,
// and left nil when the caller dispatched directly to ToAddress (eg. an
// external address that isn't in our system).
type EmailOutbox struct {
	ID                uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TemplateVersionID uuid.UUID    `gorm:"type:uuid;not null;index"`
	UserID            *uuid.UUID   `gorm:"type:uuid;index"`
	ToAddress         string       `gorm:"type:text;not null"`
	FromAddress       string       `gorm:"type:text;not null"`
	Subject           string       `gorm:"type:text;not null"`
	BodyHTML          string       `gorm:"type:text;not null"`
	BodyText          string       `gorm:"type:text"`
	Params            JSONMap      `gorm:"type:jsonb;not null"`
	Status            OutboxStatus `gorm:"type:text;not null;index:idx_outbox_dispatch,priority:1"`
	Attempts          int          `gorm:"not null;default:0"`
	MaxAttempts       int          `gorm:"not null;default:5"`
	ScheduledAt       time.Time    `gorm:"not null;default:now();index:idx_outbox_dispatch,priority:2"`
	LastError         string       `gorm:"type:text"`
	Provider          string       `gorm:"type:text"`
	ProviderMessageID string       `gorm:"type:text"`
	SentAt            *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (EmailOutbox) TableName() string { return "email_outbox" }

// OutboxListFilter narrows queries against email_outbox. Zero-valued fields
// are ignored. Lives in models so both repositories and services can share
// it without an import cycle.
type OutboxListFilter struct {
	Status string
	From   *time.Time // inclusive, filters on created_at
	To     *time.Time // exclusive, filters on created_at
	Limit  int        // defaults to 200, capped at 1000 in the repo
}
