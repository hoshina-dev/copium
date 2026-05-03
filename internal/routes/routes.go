// Package routes wires Fiber routes to handlers.
package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberSwagger "github.com/gofiber/swagger"

	// Side-effect import: registers the generated SwaggerInfo so
	// /swagger/doc.json serves the spec built by `make swagger`.
	_ "github.com/hoshina-dev/copium/docs"
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

	// Docs UIs:
	//   /swagger/         - classic Swagger UI (loads doc.json automatically)
	//   /swagger/doc.json - the OpenAPI 2.0 spec
	//   /scalar           - modern Scalar UI (loads /swagger/doc.json)
	app.Get("/swagger/*", fiberSwagger.HandlerDefault)
	app.Get("/scalar", scalarHandler)

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

// scalarHandler serves the Scalar UI. It's a tiny static HTML page that loads
// the same OpenAPI spec as /swagger/doc.json.
func scalarHandler(c *fiber.Ctx) error {
	c.Type("html")
	return c.SendString(scalarHTML)
}

const scalarHTML = `<!DOCTYPE html>
<html>
  <head>
    <title>Copium API - Scalar</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <script id="api-reference" data-url="/swagger/doc.json"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`
