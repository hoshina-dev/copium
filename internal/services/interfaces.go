// Package services orchestrates use cases. Every collaborator is declared as
// an interface OWNED by services (the consumer-side interface idiom), so the
// concrete adapter packages don't leak into business logic and tests can
// inject hand-rolled fakes or mockery-generated mocks.
package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hoshina-dev/copium/internal/models"
	"github.com/hoshina-dev/copium/internal/renderer"
	"github.com/hoshina-dev/copium/internal/sender"
)

// --- repositories ---

type TemplateRepository interface {
	Create(ctx context.Context, t *models.EmailTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.EmailTemplate, error)
	GetByCode(ctx context.Context, code string) (*models.EmailTemplate, error)
	List(ctx context.Context) ([]models.EmailTemplate, error)
	SetActiveVersion(ctx context.Context, templateID, versionID uuid.UUID) error
}

type TemplateVersionRepository interface {
	Create(ctx context.Context, v *models.EmailTemplateVersion) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.EmailTemplateVersion, error)
	GetByTemplateAndVersion(ctx context.Context, templateID uuid.UUID, version int) (*models.EmailTemplateVersion, error)
	NextVersionNumber(ctx context.Context, templateID uuid.UUID) (int, error)
	ListByTemplate(ctx context.Context, templateID uuid.UUID) ([]models.EmailTemplateVersion, error)
}

type OutboxRepository interface {
	Create(ctx context.Context, o *models.EmailOutbox) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.EmailOutbox, error)
	List(ctx context.Context, f models.OutboxListFilter) ([]*models.EmailOutbox, error)
}

// --- collaborators (signatures match the adapter packages so they satisfy
//     these interfaces structurally without any shim) ---

type UserResolver interface {
	ResolveEmail(ctx context.Context, userID uuid.UUID) (string, error)
}

type Renderer interface {
	Render(v *models.EmailTemplateVersion, params models.JSONMap) (*renderer.Output, error)
}

// Sender is intentionally narrower than sender.Sender: services only need to
// know the From-defaulted "send" verb. Adapters in internal/sender satisfy
// this via structural typing.
type Sender interface {
	Name() string
	Send(ctx context.Context, m sender.Message) (sender.SendResult, error)
}

// --- platform ---

type Clock interface {
	Now() time.Time
}

type IDGen interface {
	New() uuid.UUID
}
