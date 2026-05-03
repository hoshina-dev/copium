// Package routes wires Fiber routes to handlers.
package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/hoshina-dev/copium/internal/handlers"
	cmw "github.com/hoshina-dev/copium/internal/middleware"
)

type Handlers struct {
	Email    *handlers.EmailHandler
	Template *handlers.TemplateHandler
}

// NewApp builds a Fiber app with all routes mounted under /api/v1 and the
// global error handler installed. Use this from cmd/server/main.go.
func NewApp(h Handlers) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: cmw.ErrorHandler,
	})
	app.Use(recover.New())

	app.Get("/healthz", handlers.Healthz)
	app.Get("/readyz", handlers.Readyz)

	v1 := app.Group("/api/v1")

	tpl := v1.Group("/templates")
	tpl.Get("/", h.Template.List)
	tpl.Post("/", h.Template.Create)
	tpl.Get("/:id", h.Template.Get)
	tpl.Get("/:id/versions", h.Template.ListVersions)
	tpl.Post("/:id/versions", h.Template.CreateVersion)
	tpl.Get("/:id/versions/:version", h.Template.GetVersion)
	tpl.Patch("/:id/active-version", h.Template.SetActiveVersion)

	em := v1.Group("/emails")
	em.Post("/send", h.Email.Send)
	em.Get("/:id", h.Email.Get)

	return app
}
