package middleware_test

import (
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/middleware"
)

func newApp(handler fiber.Handler) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Get("/x", handler)
	return app
}

func do(t *testing.T, app *fiber.App) (int, string) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", "/x", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestErrorHandler_NotFound(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error { return apperrors.NotFound("template", nil) })
	code, body := do(t, app)
	if code != 404 {
		t.Errorf("code=%d", code)
	}
	if !strings.Contains(body, "not found") {
		t.Errorf("body=%q", body)
	}
}

func TestErrorHandler_InvalidParams(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error { return apperrors.InvalidParams("bad", nil) })
	code, _ := do(t, app)
	if code != 400 {
		t.Errorf("code=%d", code)
	}
}

func TestErrorHandler_Conflict(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error { return apperrors.Conflict("dup", nil) })
	code, _ := do(t, app)
	if code != 409 {
		t.Errorf("code=%d", code)
	}
}

func TestErrorHandler_Upstream(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error { return apperrors.Upstream("custapi", nil) })
	code, _ := do(t, app)
	if code != 502 {
		t.Errorf("code=%d", code)
	}
}

func TestErrorHandler_Internal_HidesCause(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error { return errors.New("db connection refused") })
	code, body := do(t, app)
	if code != 500 {
		t.Errorf("code=%d", code)
	}
	if strings.Contains(body, "db connection refused") {
		t.Errorf("must not leak internal cause: %q", body)
	}
}

func TestErrorHandler_FiberError_RespectsCode(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error { return fiber.NewError(418, "teapot") })
	code, _ := do(t, app)
	if code != 418 {
		t.Errorf("code=%d", code)
	}
}
