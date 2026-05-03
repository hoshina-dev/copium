// Package servicestest hosts hand-rolled in-memory fakes for the consumer-side
// interfaces declared in package services. They keep service tests fast and
// hermetic while still satisfying the same interfaces the real adapters do.
//
// Mockery-generated mocks (under /mocks) remain available for test cases that
// need EXPECT-style call assertions; these fakes are for behaviour tests.
package servicestest

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/models"
	"github.com/hoshina-dev/copium/internal/sender"
)

type Fakes struct {
	TemplateRepo *FakeTemplateRepo
	VersionRepo  *FakeVersionRepo
	OutboxRepo   *FakeOutboxRepo
	UserResolver *FakeUserResolver
	Sender       *FakeSender
}

func New() *Fakes {
	return &Fakes{
		TemplateRepo: NewFakeTemplateRepo(),
		VersionRepo:  NewFakeVersionRepo(),
		OutboxRepo:   NewFakeOutboxRepo(),
		UserResolver: NewFakeUserResolver(),
		Sender:       NewFakeSender(),
	}
}

// --- FakeTemplateRepo ---

type FakeTemplateRepo struct {
	Templates map[uuid.UUID]*models.EmailTemplate
	Existing  map[string]bool // codes that should trigger conflict
	CreateErr error
}

func NewFakeTemplateRepo() *FakeTemplateRepo {
	return &FakeTemplateRepo{
		Templates: map[uuid.UUID]*models.EmailTemplate{},
		Existing:  map[string]bool{},
	}
}

func (f *FakeTemplateRepo) Create(_ context.Context, t *models.EmailTemplate) error {
	if f.CreateErr != nil {
		return f.CreateErr
	}
	if f.Existing[t.Code] {
		return apperrors.Conflict("template code "+t.Code, nil)
	}
	for _, existing := range f.Templates {
		if existing.Code == t.Code {
			return apperrors.Conflict("template code "+t.Code, nil)
		}
	}
	f.Templates[t.ID] = t
	return nil
}

func (f *FakeTemplateRepo) GetByID(_ context.Context, id uuid.UUID) (*models.EmailTemplate, error) {
	t, ok := f.Templates[id]
	if !ok {
		return nil, apperrors.NotFound("template "+id.String(), nil)
	}
	return t, nil
}

func (f *FakeTemplateRepo) GetByCode(_ context.Context, code string) (*models.EmailTemplate, error) {
	for _, t := range f.Templates {
		if t.Code == code {
			return t, nil
		}
	}
	return nil, apperrors.NotFound("template code "+code, nil)
}

func (f *FakeTemplateRepo) List(_ context.Context) ([]models.EmailTemplate, error) {
	out := make([]models.EmailTemplate, 0, len(f.Templates))
	for _, t := range f.Templates {
		out = append(out, *t)
	}
	return out, nil
}

func (f *FakeTemplateRepo) SetActiveVersion(_ context.Context, templateID, versionID uuid.UUID) error {
	t, ok := f.Templates[templateID]
	if !ok {
		return apperrors.NotFound("template "+templateID.String(), nil)
	}
	v := versionID
	t.ActiveVersionID = &v
	return nil
}

// --- FakeVersionRepo ---

type FakeVersionRepo struct {
	Versions    map[uuid.UUID]*models.EmailTemplateVersion
	NextNumbers map[uuid.UUID]int
	CreateErr   error
}

func NewFakeVersionRepo() *FakeVersionRepo {
	return &FakeVersionRepo{
		Versions:    map[uuid.UUID]*models.EmailTemplateVersion{},
		NextNumbers: map[uuid.UUID]int{},
	}
}

func (f *FakeVersionRepo) Create(_ context.Context, v *models.EmailTemplateVersion) error {
	if f.CreateErr != nil {
		return f.CreateErr
	}
	f.Versions[v.ID] = v
	return nil
}

func (f *FakeVersionRepo) GetByID(_ context.Context, id uuid.UUID) (*models.EmailTemplateVersion, error) {
	v, ok := f.Versions[id]
	if !ok {
		return nil, apperrors.NotFound("template version "+id.String(), nil)
	}
	return v, nil
}

func (f *FakeVersionRepo) GetByTemplateAndVersion(_ context.Context, templateID uuid.UUID, version int) (*models.EmailTemplateVersion, error) {
	for _, v := range f.Versions {
		if v.TemplateID == templateID && v.Version == version {
			return v, nil
		}
	}
	return nil, apperrors.NotFound("version", nil)
}

func (f *FakeVersionRepo) NextVersionNumber(_ context.Context, templateID uuid.UUID) (int, error) {
	if n, ok := f.NextNumbers[templateID]; ok {
		return n, nil
	}
	return 1, nil
}

func (f *FakeVersionRepo) ListByTemplate(_ context.Context, templateID uuid.UUID) ([]models.EmailTemplateVersion, error) {
	var out []models.EmailTemplateVersion
	for _, v := range f.Versions {
		if v.TemplateID == templateID {
			out = append(out, *v)
		}
	}
	return out, nil
}

// --- FakeOutboxRepo ---

type FakeOutboxRepo struct {
	Rows      map[uuid.UUID]*models.EmailOutbox
	CreateErr error
}

func NewFakeOutboxRepo() *FakeOutboxRepo {
	return &FakeOutboxRepo{Rows: map[uuid.UUID]*models.EmailOutbox{}}
}

func (f *FakeOutboxRepo) Create(_ context.Context, o *models.EmailOutbox) error {
	if f.CreateErr != nil {
		return f.CreateErr
	}
	f.Rows[o.ID] = o
	return nil
}

func (f *FakeOutboxRepo) GetByID(_ context.Context, id uuid.UUID) (*models.EmailOutbox, error) {
	o, ok := f.Rows[id]
	if !ok {
		return nil, apperrors.NotFound("outbox "+id.String(), nil)
	}
	return o, nil
}

// --- FakeUserResolver ---

type FakeUserResolver struct {
	Emails     map[uuid.UUID]string
	MissingErr error
	Calls      int
}

func NewFakeUserResolver() *FakeUserResolver {
	return &FakeUserResolver{Emails: map[uuid.UUID]string{}}
}

func (f *FakeUserResolver) ResolveEmail(_ context.Context, id uuid.UUID) (string, error) {
	f.Calls++
	if f.MissingErr != nil {
		return "", f.MissingErr
	}
	if email, ok := f.Emails[id]; ok {
		return email, nil
	}
	return "", apperrors.NotFound("user "+id.String(), errors.New("not in fake"))
}

// --- FakeSender ---

type FakeSender struct {
	Sent     []sender.Message
	SendErr  error
	NameStr  string
	NextID   string
}

func NewFakeSender() *FakeSender { return &FakeSender{NameStr: "fake"} }

func (f *FakeSender) Name() string { return f.NameStr }

func (f *FakeSender) Send(_ context.Context, m sender.Message) (sender.SendResult, error) {
	if f.SendErr != nil {
		return sender.SendResult{}, f.SendErr
	}
	f.Sent = append(f.Sent, m)
	id := f.NextID
	if id == "" {
		id = "fake:" + m.To
	}
	return sender.SendResult{ProviderMessageID: id}, nil
}
