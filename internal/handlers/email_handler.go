// Package handlers binds Fiber routes to service methods. Handlers depend on
// concrete service structs by design - service interfaces only get introduced
// if a handler test ever needs to fake one (YAGNI).
package handlers

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/models"
	"github.com/hoshina-dev/copium/internal/services"
)

type EmailHandler struct {
	svc      *services.EmailService
	validate *validator.Validate
}

func NewEmailHandler(svc *services.EmailService) *EmailHandler {
	return &EmailHandler{svc: svc, validate: validator.New()}
}

func (h *EmailHandler) Send(c *fiber.Ctx) error {
	var req models.SendEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.InvalidParams("invalid JSON body", err)
	}
	if err := h.validate.Struct(req); err != nil {
		return apperrors.InvalidParams(err.Error(), nil)
	}
	res, err := h.svc.SendEmail(c.Context(), req)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusAccepted).JSON(res)
}

func (h *EmailHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.InvalidParams("id must be a uuid", err)
	}
	row, err := h.svc.GetOutbox(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(models.OutboxToResponse(row))
}
