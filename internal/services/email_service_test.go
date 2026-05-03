package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/clock"
	"github.com/hoshina-dev/copium/internal/idgen"
	"github.com/hoshina-dev/copium/internal/models"
	"github.com/hoshina-dev/copium/internal/renderer"
	"github.com/hoshina-dev/copium/internal/services"
	"github.com/hoshina-dev/copium/internal/services/servicestest"
)

func newEmailSvc(t *testing.T, fixed time.Time, ids ...uuid.UUID) (*services.EmailService, *servicestest.Fakes) {
	t.Helper()
	f := servicestest.New()
	r, err := renderer.New()
	if err != nil {
		t.Fatalf("renderer: %v", err)
	}
	clk := clock.NewFake(fixed)
	gen := idgen.NewStatic(ids...)
	svc := services.NewEmailService(services.EmailDeps{
		Templates:        f.TemplateRepo,
		TemplateVersions: f.VersionRepo,
		Outbox:           f.OutboxRepo,
		Users:            f.UserResolver,
		Renderer:         r,
		Sender:           f.Sender,
		Clock:            clk,
		IDs:              gen,
		DefaultFrom:      "Copium <noreply@example.com>",
	})
	return svc, f
}

func makeTemplateAndVersion(t *testing.T, f *servicestest.Fakes, schema models.JSONMap) (*models.EmailTemplate, *models.EmailTemplateVersion) {
	t.Helper()
	tplID := uuid.New()
	verID := uuid.New()
	tpl := &models.EmailTemplate{ID: tplID, Code: "welcome", Name: "Welcome", ActiveVersionID: &verID}
	ver := &models.EmailTemplateVersion{
		ID: verID, TemplateID: tplID, Version: 1,
		Subject: "Hi {{.name}}", BodyHTML: "<p>{{.name}}</p>",
		ParamsSchema: schema,
	}
	f.TemplateRepo.Templates[tplID] = tpl
	f.VersionRepo.Versions[verID] = ver
	return tpl, ver
}

func TestSendEmail_Happy_EnqueuesSnapshot(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	outID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	svc, f := newEmailSvc(t, fixed, outID)
	tpl, ver := makeTemplateAndVersion(t, f, models.JSONMap{
		"type": "object", "required": []any{"name"},
		"properties": map[string]any{"name": map[string]any{"type": "string"}},
	})
	uid := uuid.New()
	f.UserResolver.Emails[uid] = "alice@example.com"

	res, err := svc.SendEmail(context.Background(), models.SendEmailRequest{
		UserID:     uid,
		TemplateID: tpl.ID,
		Params:     models.JSONMap{"name": "Alice"},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.OutboxID != outID {
		t.Errorf("outbox id mismatch got %v want %v", res.OutboxID, outID)
	}
	if res.Status != "queued" {
		t.Errorf("status=%q", res.Status)
	}

	if len(f.OutboxRepo.Rows) != 1 {
		t.Fatalf("expected 1 outbox row, got %d", len(f.OutboxRepo.Rows))
	}
	row := f.OutboxRepo.Rows[outID]
	if row.ToAddress != "alice@example.com" {
		t.Errorf("To=%q", row.ToAddress)
	}
	if row.FromAddress != "Copium <noreply@example.com>" {
		t.Errorf("From=%q", row.FromAddress)
	}
	if row.Subject != "Hi Alice" {
		t.Errorf("Subject=%q (must be rendered snapshot)", row.Subject)
	}
	if row.BodyHTML != "<p>Alice</p>" {
		t.Errorf("BodyHTML=%q", row.BodyHTML)
	}
	if row.Status != models.OutboxStatusQueued {
		t.Errorf("Status=%v", row.Status)
	}
	if row.MaxAttempts == 0 {
		t.Errorf("MaxAttempts must be set")
	}
	if !row.ScheduledAt.Equal(fixed) {
		t.Errorf("ScheduledAt=%v want %v (clock-injected)", row.ScheduledAt, fixed)
	}
	if row.TemplateVersionID != ver.ID {
		t.Errorf("TemplateVersionID=%v want %v", row.TemplateVersionID, ver.ID)
	}
	if row.UserID != uid {
		t.Errorf("UserID=%v", row.UserID)
	}
	if got := row.Params["name"]; got != "Alice" {
		t.Errorf("Params not snapshotted: %v", row.Params)
	}

	if got := f.UserResolver.Calls; got != 1 {
		t.Errorf("UserResolver called %d times", got)
	}
	if got := len(f.Sender.Sent); got != 0 {
		t.Errorf("Sender.Send must NOT be called by enqueue (worker's job); got %d", got)
	}
}

func TestSendEmail_TemplateFromAddressOverridesDefault(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	outID := uuid.New()
	svc, f := newEmailSvc(t, fixed, outID)
	tpl, ver := makeTemplateAndVersion(t, f, models.JSONMap{"type": "object"})
	ver.FromAddress = "ops@example.com"
	uid := uuid.New()
	f.UserResolver.Emails[uid] = "x@x"

	_, err := svc.SendEmail(context.Background(), models.SendEmailRequest{
		UserID: uid, TemplateID: tpl.ID, Params: models.JSONMap{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := f.OutboxRepo.Rows[outID].FromAddress; got != "ops@example.com" {
		t.Errorf("From=%q", got)
	}
}

func TestSendEmail_TemplateNotFound(t *testing.T) {
	svc, f := newEmailSvc(t, time.Now(), uuid.New())
	_ = f
	_, err := svc.SendEmail(context.Background(), models.SendEmailRequest{
		UserID: uuid.New(), TemplateID: uuid.New(),
	})
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSendEmail_NoActiveVersion(t *testing.T) {
	svc, f := newEmailSvc(t, time.Now(), uuid.New())
	tpl := &models.EmailTemplate{ID: uuid.New(), Code: "x", Name: "x"}
	f.TemplateRepo.Templates[tpl.ID] = tpl
	uid := uuid.New()
	f.UserResolver.Emails[uid] = "x@x"
	_, err := svc.SendEmail(context.Background(), models.SendEmailRequest{
		UserID: uid, TemplateID: tpl.ID,
	})
	if !errors.Is(err, apperrors.ErrInvalidParams) {
		t.Fatalf("want ErrInvalidParams (no active version), got %v", err)
	}
}

func TestSendEmail_InvalidParams(t *testing.T) {
	svc, f := newEmailSvc(t, time.Now(), uuid.New())
	tpl, _ := makeTemplateAndVersion(t, f, models.JSONMap{
		"type": "object", "required": []any{"name"},
		"properties": map[string]any{"name": map[string]any{"type": "string"}},
	})
	uid := uuid.New()
	f.UserResolver.Emails[uid] = "x@x"
	_, err := svc.SendEmail(context.Background(), models.SendEmailRequest{
		UserID: uid, TemplateID: tpl.ID, Params: models.JSONMap{},
	})
	if !errors.Is(err, apperrors.ErrInvalidParams) {
		t.Fatalf("want ErrInvalidParams, got %v", err)
	}
	if len(f.OutboxRepo.Rows) != 0 {
		t.Errorf("must NOT enqueue on validation failure")
	}
}

func TestSendEmail_UserNotFound(t *testing.T) {
	svc, f := newEmailSvc(t, time.Now(), uuid.New())
	tpl, _ := makeTemplateAndVersion(t, f, models.JSONMap{"type": "object"})
	f.UserResolver.MissingErr = apperrors.NotFound("user", nil)
	_, err := svc.SendEmail(context.Background(), models.SendEmailRequest{
		UserID: uuid.New(), TemplateID: tpl.ID,
	})
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSendEmail_OutboxFailureSurfaced(t *testing.T) {
	svc, f := newEmailSvc(t, time.Now(), uuid.New())
	tpl, _ := makeTemplateAndVersion(t, f, models.JSONMap{"type": "object"})
	uid := uuid.New()
	f.UserResolver.Emails[uid] = "x@x"
	f.OutboxRepo.CreateErr = errors.New("db down")
	_, err := svc.SendEmail(context.Background(), models.SendEmailRequest{
		UserID: uid, TemplateID: tpl.ID,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrInternal) {
		t.Fatalf("want ErrInternal, got %v", err)
	}
}
