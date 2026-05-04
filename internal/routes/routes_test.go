package routes_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
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

func TestRoutes_SwaggerSpecServed(t *testing.T) {
	app := buildApp(t)
	resp, err := app.Test(httptest.NewRequest("GET", "/swagger/doc.json", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("doc.json -> %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var spec map[string]any
	if err := json.Unmarshal(body, &spec); err != nil {
		t.Fatalf("doc.json must be valid JSON: %v", err)
	}
	if _, ok := spec["paths"]; !ok {
		t.Errorf("doc.json missing 'paths' key; got: %s", body[:min(200, len(body))])
	}
}

func TestRoutes_ScalarUIServed(t *testing.T) {
	app := buildApp(t)
	resp, err := app.Test(httptest.NewRequest("GET", "/scalar", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("/scalar -> %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected HTML, got Content-Type=%q", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "/swagger/doc.json") {
		t.Errorf("scalar HTML must reference doc.json; got: %s", body)
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

// TestRoutes_RootServesPlaceholderOrSPA covers both states: when the Vite
// build is embedded `/` returns the SPA index; when it isn't, we serve a
// helpful dev placeholder. Either way it must be a 200 OK HTML response.
func TestRoutes_RootServesPlaceholderOrSPA(t *testing.T) {
	app := buildApp(t)
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("/ -> %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected HTML, got Content-Type=%q", ct)
	}
}

// TestRoutes_UnknownAPIStillReturns404 guards against the SPA fallback
// accidentally swallowing /api/v1/* misses.
func TestRoutes_UnknownAPIStillReturns404(t *testing.T) {
	app := buildApp(t)
	for _, p := range []string{
		"/api/v1/does-not-exist",
		"/api/v1/templates/not-a-uuid/wat",
	} {
		resp, err := app.Test(httptest.NewRequest("GET", p, nil))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 404 {
			t.Errorf("%s -> %d (want 404)", p, resp.StatusCode)
		}
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
