package handlers

import "github.com/gofiber/fiber/v2"

func Healthz(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) }

func Readyz(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ready"}) }
