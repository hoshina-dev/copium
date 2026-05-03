package routes_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/hoshina-dev/copium/internal/clock"
	"github.com/hoshina-dev/copium/internal/handlers"
	"github.com/hoshina-dev/copium/internal/idgen"
	"github.com/hoshina-dev/copium/internal/renderer"
	"github.com/hoshina-dev/copium/internal/routes"
	"github.com/hoshina-dev/copium/internal/services"
	"github.com/hoshina-dev/copium/internal/services/servicestest"
)

func TestRoutes_HealthzAndReadyz(t *testing.T) {
	app := buildApp(t)
	for _, path := range []string{"/healthz", "/readyz"} {
		resp, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("%s -> %d", path, resp.StatusCode)
		}
	}
}

func TestRoutes_NotFound404(t *testing.T) {
	app := buildApp(t)
	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/nope", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("got %d", resp.StatusCode)
	}
}

func TestRoutes_TemplatesEndpointReachable(t *testing.T) {
	app := buildApp(t)
	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/templates/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("templates list -> %d", resp.StatusCode)
	}
}

func buildApp(t *testing.T) *fiber.App {
	t.Helper()
	f := servicestest.New()
	r, _ := renderer.New()
	tplSvc := services.NewTemplateService(services.TemplateDeps{
		Templates:        f.TemplateRepo,
		TemplateVersions: f.VersionRepo,
		Clock:            clock.NewFake(time.Now()),
		IDs:              idgen.NewStatic(uuid.New()),
	})
	emSvc := services.NewEmailService(services.EmailDeps{
		Templates:        f.TemplateRepo,
		TemplateVersions: f.VersionRepo,
		Outbox:           f.OutboxRepo,
		Users:            f.UserResolver,
		Renderer:         r,
		Sender:           f.Sender,
		Clock:            clock.NewFake(time.Now()),
		IDs:              idgen.NewStatic(uuid.New()),
		DefaultFrom:      "x@y",
	})
	return routes.NewApp(routes.Handlers{
		Email:    handlers.NewEmailHandler(emSvc),
		Template: handlers.NewTemplateHandler(tplSvc),
	})
}
