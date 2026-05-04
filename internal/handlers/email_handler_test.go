package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/hoshina-dev/copium/internal/clock"
	"github.com/hoshina-dev/copium/internal/handlers"
	"github.com/hoshina-dev/copium/internal/idgen"
	"github.com/hoshina-dev/copium/internal/middleware"
	"github.com/hoshina-dev/copium/internal/models"
	"github.com/hoshina-dev/copium/internal/renderer"
	"github.com/hoshina-dev/copium/internal/services"
	"github.com/hoshina-dev/copium/internal/services/servicestest"
)

func newEmailHandler(t *testing.T, fixedID uuid.UUID) (*handlers.EmailHandler, *servicestest.Fakes) {
	t.Helper()
	f := servicestest.New()
	r, _ := renderer.New()
	svc := services.NewEmailService(services.EmailDeps{
		Templates:        f.TemplateRepo,
		TemplateVersions: f.VersionRepo,
		Outbox:           f.OutboxRepo,
		Users:            f.UserResolver,
		Renderer:         r,
		Sender:           f.Sender,
		Clock:            clock.NewFake(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		IDs:              idgen.NewStatic(fixedID),
		DefaultFrom:      "Copium <noreply@example.com>",
	})
	return handlers.NewEmailHandler(svc), f
}

func newApp(register func(app *fiber.App)) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	register(app)
	return app
}

func doJSON(t *testing.T, app *fiber.App, method, path string, body any) (*http.Response, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	out, _ := io.ReadAll(resp.Body)
	return resp, out
}

func setupTemplate(t *testing.T, f *servicestest.Fakes) (uuid.UUID, uuid.UUID) {
	t.Helper()
	tplID := uuid.New()
	verID := uuid.New()
	f.TemplateRepo.Templates[tplID] = &models.EmailTemplate{
		ID: tplID, Code: "welcome", Name: "Welcome", ActiveVersionID: &verID,
	}
	f.VersionRepo.Versions[verID] = &models.EmailTemplateVersion{
		ID: verID, TemplateID: tplID, Version: 1,
		Subject:  "Hi {{.name}}",
		BodyHTML: "<p>{{.name}}</p>",
		ParamsSchema: models.JSONMap{
			"type": "object", "required": []any{"name"},
			"properties": map[string]any{"name": map[string]any{"type": "string"}},
		},
	}
	return tplID, verID
}

func TestEmailHandler_Send_202(t *testing.T) {
	outID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	h, f := newEmailHandler(t, outID)
	tplID, _ := setupTemplate(t, f)
	uid := uuid.New()
	f.UserResolver.Emails[uid] = "a@b.com"

	app := newApp(func(app *fiber.App) {
		app.Post("/emails/send", h.Send)
	})
	resp, body := doJSON(t, app, "POST", "/emails/send", models.SendEmailRequest{
		UserID: &uid, TemplateID: tplID, Params: models.JSONMap{"name": "Alice"},
	})
	if resp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	var got models.SendEmailResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.OutboxID != outID || got.Status != "queued" {
		t.Errorf("got %+v", got)
	}
}

func TestEmailHandler_Send_BadJSON(t *testing.T) {
	h, _ := newEmailHandler(t, uuid.New())
	app := newApp(func(app *fiber.App) { app.Post("/emails/send", h.Send) })
	req := httptest.NewRequest("POST", "/emails/send", bytes.NewReader([]byte(`{not json`)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 400 {
		t.Errorf("code=%d", resp.StatusCode)
	}
}

func TestEmailHandler_Send_MissingFields(t *testing.T) {
	h, _ := newEmailHandler(t, uuid.New())
	app := newApp(func(app *fiber.App) { app.Post("/emails/send", h.Send) })
	resp, _ := doJSON(t, app, "POST", "/emails/send", map[string]any{})
	if resp.StatusCode != 400 {
		t.Errorf("code=%d", resp.StatusCode)
	}
}

func TestEmailHandler_Send_DirectEmail_202(t *testing.T) {
	outID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	h, f := newEmailHandler(t, outID)
	tplID, _ := setupTemplate(t, f)
	app := newApp(func(app *fiber.App) { app.Post("/emails/send", h.Send) })

	resp, body := doJSON(t, app, "POST", "/emails/send", models.SendEmailRequest{
		TemplateID: tplID,
		ToAddress:  "external@example.com",
		Params:     models.JSONMap{"name": "Pat"},
	})
	if resp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	row := f.OutboxRepo.Rows[outID]
	if row == nil {
		t.Fatal("expected outbox row")
	}
	if row.UserID != nil {
		t.Errorf("UserID must be nil for direct sends, got %v", row.UserID)
	}
	if row.ToAddress != "external@example.com" {
		t.Errorf("To=%q", row.ToAddress)
	}
}

func TestEmailHandler_Send_MissingRecipient_400(t *testing.T) {
	h, f := newEmailHandler(t, uuid.New())
	tplID, _ := setupTemplate(t, f)
	app := newApp(func(app *fiber.App) { app.Post("/emails/send", h.Send) })

	resp, _ := doJSON(t, app, "POST", "/emails/send", models.SendEmailRequest{
		TemplateID: tplID,
		Params:     models.JSONMap{"name": "x"},
	})
	if resp.StatusCode != 400 {
		t.Errorf("code=%d (want 400 for missing recipient)", resp.StatusCode)
	}
}

func TestEmailHandler_Send_BothRecipients_400(t *testing.T) {
	h, f := newEmailHandler(t, uuid.New())
	tplID, _ := setupTemplate(t, f)
	uid := uuid.New()
	f.UserResolver.Emails[uid] = "u@x"
	app := newApp(func(app *fiber.App) { app.Post("/emails/send", h.Send) })

	resp, _ := doJSON(t, app, "POST", "/emails/send", models.SendEmailRequest{
		TemplateID: tplID,
		UserID:     &uid,
		ToAddress:  "external@example.com",
		Params:     models.JSONMap{"name": "x"},
	})
	if resp.StatusCode != 400 {
		t.Errorf("code=%d (want 400 when both recipients given)", resp.StatusCode)
	}
}

func TestEmailHandler_Send_BadEmail_400(t *testing.T) {
	h, f := newEmailHandler(t, uuid.New())
	tplID, _ := setupTemplate(t, f)
	app := newApp(func(app *fiber.App) { app.Post("/emails/send", h.Send) })

	resp, _ := doJSON(t, app, "POST", "/emails/send", models.SendEmailRequest{
		TemplateID: tplID,
		ToAddress:  "not-an-email",
		Params:     models.JSONMap{"name": "x"},
	})
	if resp.StatusCode != 400 {
		t.Errorf("code=%d (want 400 for malformed to_address)", resp.StatusCode)
	}
}

func TestEmailHandler_Send_TemplateNotFound(t *testing.T) {
	h, f := newEmailHandler(t, uuid.New())
	uid := uuid.New()
	f.UserResolver.Emails[uid] = "x@x"
	app := newApp(func(app *fiber.App) { app.Post("/emails/send", h.Send) })
	resp, _ := doJSON(t, app, "POST", "/emails/send", models.SendEmailRequest{
		UserID: &uid, TemplateID: uuid.New(), Params: models.JSONMap{"name": "x"},
	})
	if resp.StatusCode != 404 {
		t.Errorf("code=%d", resp.StatusCode)
	}
}

func TestEmailHandler_Send_InvalidParams(t *testing.T) {
	h, f := newEmailHandler(t, uuid.New())
	tplID, _ := setupTemplate(t, f)
	uid := uuid.New()
	f.UserResolver.Emails[uid] = "x@x"
	app := newApp(func(app *fiber.App) { app.Post("/emails/send", h.Send) })
	resp, _ := doJSON(t, app, "POST", "/emails/send", models.SendEmailRequest{
		UserID: &uid, TemplateID: tplID, Params: models.JSONMap{},
	})
	if resp.StatusCode != 400 {
		t.Errorf("code=%d", resp.StatusCode)
	}
}

func TestEmailHandler_GetOutbox_OK(t *testing.T) {
	h, f := newEmailHandler(t, uuid.New())
	id := uuid.New()
	uid := uuid.New()
	f.OutboxRepo.Rows[id] = &models.EmailOutbox{
		ID: id, TemplateVersionID: uuid.New(), UserID: &uid,
		ToAddress: "a@b", FromAddress: "x@y", Subject: "hi", BodyHTML: "x",
		Status: models.OutboxStatusSent, MaxAttempts: 5,
	}
	app := newApp(func(app *fiber.App) { app.Get("/emails/:id", h.Get) })
	resp, body := doJSON(t, app, "GET", "/emails/"+id.String(), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("code=%d body=%s", resp.StatusCode, body)
	}
	var r models.OutboxResponse
	_ = json.Unmarshal(body, &r)
	if r.Status != "sent" {
		t.Errorf("status=%q", r.Status)
	}
}

func TestEmailHandler_GetOutbox_BadUUID(t *testing.T) {
	h, _ := newEmailHandler(t, uuid.New())
	app := newApp(func(app *fiber.App) { app.Get("/emails/:id", h.Get) })
	resp, _ := doJSON(t, app, "GET", "/emails/not-a-uuid", nil)
	if resp.StatusCode != 400 {
		t.Errorf("code=%d", resp.StatusCode)
	}
}

func TestEmailHandler_GetOutbox_NotFound(t *testing.T) {
	h, _ := newEmailHandler(t, uuid.New())
	app := newApp(func(app *fiber.App) { app.Get("/emails/:id", h.Get) })
	resp, _ := doJSON(t, app, "GET", "/emails/"+uuid.New().String(), nil)
	if resp.StatusCode != 404 {
		t.Errorf("code=%d", resp.StatusCode)
	}
}

// Quiet unused import warning for context (used by future tests).
var _ = context.Background
