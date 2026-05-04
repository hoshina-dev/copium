// Package routes wires Fiber routes to handlers.
package routes

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberSwagger "github.com/gofiber/swagger"

	// Side-effect import: registers the generated SwaggerInfo so
	// /swagger/doc.json serves the spec built by `make swagger`.
	_ "github.com/hoshina-dev/copium/docs"
	"github.com/hoshina-dev/copium/internal/handlers"
	cmw "github.com/hoshina-dev/copium/internal/middleware"
	"github.com/hoshina-dev/copium/webui"
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
	tpl.Post("/preview", h.Template.Preview)
	tpl.Get("/:id", h.Template.Get)
	tpl.Delete("/:id", h.Template.Delete)
	tpl.Get("/:id/versions", h.Template.ListVersions)
	tpl.Post("/:id/versions", h.Template.CreateVersion)
	tpl.Get("/:id/versions/:version", h.Template.GetVersion)
	tpl.Patch("/:id/active-version", h.Template.SetActiveVersion)

	em := v1.Group("/emails")
	em.Post("/send", h.Email.Send)
	em.Get("/", h.Email.List)
	em.Get("/:id", h.Email.Get)

	// Mount the embedded Vite build LAST so /api/v1, /healthz, /readyz,
	// /swagger and /scalar always win. The filesystem middleware serves
	// real files (eg. /assets/index-*.js); anything else (including deep
	// links like /templates/abc-123) falls back to index.html so React
	// Router can take over on the client.
	mountWebUI(app)

	return app
}

// mountWebUI registers the SPA assets + index.html fallback. If no
// production build is embedded (fresh checkout, dev workflow), we serve a
// tiny placeholder page on `/` instead so the user gets a helpful nudge
// instead of a confusing 404.
func mountWebUI(app *fiber.App) {
	if !webui.HasIndex() {
		app.Get("/", func(c *fiber.Ctx) error {
			c.Type("html")
			return c.SendString(devPlaceholderHTML)
		})
		return
	}

	uiFS := http.FS(webui.FS())

	app.Use("/assets", filesystem.New(filesystem.Config{
		Root:       uiFS,
		PathPrefix: "assets",
		Browse:     false,
	}))

	indexHandler := filesystem.New(filesystem.Config{
		Root:         uiFS,
		Browse:       false,
		Index:        "index.html",
		NotFoundFile: "index.html",
	})

	for _, asset := range []string{"/favicon.ico", "/robots.txt", "/vite.svg"} {
		app.Get(asset, indexHandler)
	}

	// SPA fallback: any unmatched GET serves index.html so client-side
	// routing works on hard reloads. We deliberately skip API, health
	// and docs paths so unknown routes there return 404 (proper API
	// behaviour) instead of an HTML page.
	app.Get("/", indexHandler)
	app.Get("/*", func(c *fiber.Ctx) error {
		p := c.Path()
		if isReservedPath(p) {
			return fiber.ErrNotFound
		}
		return indexHandler(c)
	})
}

// isReservedPath returns true for paths that belong to the backend (API,
// health probes, docs) and must never be served the SPA index.
func isReservedPath(p string) bool {
	switch {
	case p == "/healthz", p == "/readyz", p == "/scalar":
		return true
	case len(p) >= 5 && p[:5] == "/api/":
		return true
	case len(p) >= 9 && p[:9] == "/swagger/":
		return true
	default:
		return false
	}
}

const devPlaceholderHTML = `<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>Copium - dev</title>
    <style>
      body { font-family: -apple-system, system-ui, sans-serif; max-width: 640px; margin: 4rem auto; padding: 0 1rem; line-height: 1.5; }
      code { background: #f4f4f5; padding: 2px 6px; border-radius: 4px; }
    </style>
  </head>
  <body>
    <h1>Copium backend is running</h1>
    <p>The web UI hasn't been built yet. Run one of:</p>
    <ul>
      <li><code>make webui-dev</code> - hot-reloading Vite on http://localhost:5173</li>
      <li><code>make webui-build</code> - one-shot production build, then restart this server</li>
    </ul>
    <p>API docs: <a href="/swagger/index.html">Swagger UI</a> &middot; <a href="/scalar">Scalar UI</a></p>
  </body>
</html>`

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
