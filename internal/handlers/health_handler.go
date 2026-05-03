package handlers

import "github.com/gofiber/fiber/v2"

// Healthz reports the process is alive.
//
//	@Summary  Liveness probe
//	@Tags     health
//	@Produce  json
//	@Success  200 {object} map[string]string
//	@Router   /healthz [get]
func Healthz(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) }

// Readyz reports the process is ready to serve.
//
//	@Summary  Readiness probe
//	@Tags     health
//	@Produce  json
//	@Success  200 {object} map[string]string
//	@Router   /readyz [get]
func Readyz(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ready"}) }
