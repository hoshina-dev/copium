package handlers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/hoshina-dev/copium/internal/clock"
	"github.com/hoshina-dev/copium/internal/handlers"
	"github.com/hoshina-dev/copium/internal/idgen"
	"github.com/hoshina-dev/copium/internal/models"
	"github.com/hoshina-dev/copium/internal/services"
	"github.com/hoshina-dev/copium/internal/services/servicestest"
)

func newTemplateHandler(t *testing.T, ids ...uuid.UUID) (*handlers.TemplateHandler, *servicestest.Fakes) {
	t.Helper()
	f := servicestest.New()
	svc := services.NewTemplateService(services.TemplateDeps{
		Templates:        f.TemplateRepo,
		TemplateVersions: f.VersionRepo,
		Clock:            clock.NewFake(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		IDs:              idgen.NewStatic(ids...),
	})
	return handlers.NewTemplateHandler(svc), f
}

func TestTemplateHandler_Create_201(t *testing.T) {
	id := uuid.New()
	h, _ := newTemplateHandler(t, id)
	app := newApp(func(app *fiber.App) { app.Post("/templates", h.Create) })

	resp, body := doJSON(t, app, "POST", "/templates", models.CreateTemplateRequest{
		Code: "welcome", Name: "Welcome",
	})
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("code=%d body=%s", resp.StatusCode, body)
	}
	var r models.TemplateResponse
	_ = json.Unmarshal(body, &r)
	if r.Code != "welcome" {
		t.Errorf("code=%q", r.Code)
	}
}

func TestTemplateHandler_Create_DuplicateCode_409(t *testing.T) {
	id := uuid.New()
	h, f := newTemplateHandler(t, id)
	f.TemplateRepo.Existing["welcome"] = true
	app := newApp(func(app *fiber.App) { app.Post("/templates", h.Create) })
	resp, _ := doJSON(t, app, "POST", "/templates", models.CreateTemplateRequest{Code: "welcome", Name: "x"})
	if resp.StatusCode != 409 {
		t.Errorf("code=%d", resp.StatusCode)
	}
}

func TestTemplateHandler_Create_ValidatorMissingFields_400(t *testing.T) {
	h, _ := newTemplateHandler(t, uuid.New())
	app := newApp(func(app *fiber.App) { app.Post("/templates", h.Create) })
	resp, _ := doJSON(t, app, "POST", "/templates", models.CreateTemplateRequest{})
	if resp.StatusCode != 400 {
		t.Errorf("code=%d", resp.StatusCode)
	}
}

func TestTemplateHandler_Get_OK(t *testing.T) {
	h, f := newTemplateHandler(t, uuid.New())
	id := uuid.New()
	f.TemplateRepo.Templates[id] = &models.EmailTemplate{ID: id, Code: "x", Name: "x"}
	app := newApp(func(app *fiber.App) { app.Get("/templates/:id", h.Get) })
	resp, body := doJSON(t, app, "GET", "/templates/"+id.String(), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("code=%d body=%s", resp.StatusCode, body)
	}
}

func TestTemplateHandler_CreateVersion_201(t *testing.T) {
	verID := uuid.New()
	h, f := newTemplateHandler(t, verID)
	tplID := uuid.New()
	f.TemplateRepo.Templates[tplID] = &models.EmailTemplate{ID: tplID, Code: "x", Name: "x"}
	f.VersionRepo.NextNumbers[tplID] = 1
	app := newApp(func(app *fiber.App) { app.Post("/templates/:id/versions", h.CreateVersion) })
	resp, body := doJSON(t, app, "POST", "/templates/"+tplID.String()+"/versions",
		models.CreateTemplateVersionRequest{
			Subject: "hi", BodyHTML: "<p>x</p>",
			ParamsSchema: models.JSONMap{"type": "object"},
		})
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("code=%d body=%s", resp.StatusCode, body)
	}
}

func TestTemplateHandler_SetActive_200(t *testing.T) {
	h, f := newTemplateHandler(t, uuid.New())
	tplID := uuid.New()
	verID := uuid.New()
	f.TemplateRepo.Templates[tplID] = &models.EmailTemplate{ID: tplID, Code: "x", Name: "x"}
	f.VersionRepo.Versions[verID] = &models.EmailTemplateVersion{ID: verID, TemplateID: tplID, Version: 1}
	app := newApp(func(app *fiber.App) { app.Patch("/templates/:id/active-version", h.SetActiveVersion) })
	resp, body := doJSON(t, app, "PATCH", "/templates/"+tplID.String()+"/active-version",
		models.SetActiveVersionRequest{VersionID: verID})
	if resp.StatusCode != 200 {
		t.Fatalf("code=%d body=%s", resp.StatusCode, body)
	}
}
