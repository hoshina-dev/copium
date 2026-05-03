package handlers

import (
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/models"
	"github.com/hoshina-dev/copium/internal/services"
)

type TemplateHandler struct {
	svc      *services.TemplateService
	validate *validator.Validate
}

func NewTemplateHandler(svc *services.TemplateService) *TemplateHandler {
	return &TemplateHandler{svc: svc, validate: validator.New()}
}

func (h *TemplateHandler) Create(c *fiber.Ctx) error {
	var req models.CreateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.InvalidParams("invalid JSON body", err)
	}
	if err := h.validate.Struct(req); err != nil {
		return apperrors.InvalidParams(err.Error(), nil)
	}
	t, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(models.TemplateToResponse(t))
}

func (h *TemplateHandler) List(c *fiber.Ctx) error {
	ts, err := h.svc.List(c.Context())
	if err != nil {
		return err
	}
	out := make([]models.TemplateResponse, len(ts))
	for i := range ts {
		out[i] = models.TemplateToResponse(&ts[i])
	}
	return c.JSON(out)
}

func (h *TemplateHandler) Get(c *fiber.Ctx) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	t, err := h.svc.Get(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(models.TemplateToResponse(t))
}

func (h *TemplateHandler) CreateVersion(c *fiber.Ctx) error {
	tplID, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	var req models.CreateTemplateVersionRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.InvalidParams("invalid JSON body", err)
	}
	if err := h.validate.Struct(req); err != nil {
		return apperrors.InvalidParams(err.Error(), nil)
	}
	v, err := h.svc.CreateVersion(c.Context(), tplID, req)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(models.TemplateVersionToResponse(v))
}

func (h *TemplateHandler) ListVersions(c *fiber.Ctx) error {
	tplID, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	vs, err := h.svc.ListVersions(c.Context(), tplID)
	if err != nil {
		return err
	}
	out := make([]models.TemplateVersionResponse, len(vs))
	for i := range vs {
		out[i] = models.TemplateVersionToResponse(&vs[i])
	}
	return c.JSON(out)
}

func (h *TemplateHandler) GetVersion(c *fiber.Ctx) error {
	tplID, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	versionStr := c.Params("version")
	version, perr := strconv.Atoi(versionStr)
	if perr != nil {
		return apperrors.InvalidParams("version must be int", perr)
	}
	v, err := h.svc.GetVersion(c.Context(), tplID, version)
	if err != nil {
		return err
	}
	return c.JSON(models.TemplateVersionToResponse(v))
}

func (h *TemplateHandler) SetActiveVersion(c *fiber.Ctx) error {
	tplID, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	var req models.SetActiveVersionRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.InvalidParams("invalid JSON body", err)
	}
	if err := h.validate.Struct(req); err != nil {
		return apperrors.InvalidParams(err.Error(), nil)
	}
	if err := h.svc.SetActiveVersion(c.Context(), tplID, req.VersionID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusOK)
}

func parseUUID(c *fiber.Ctx, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(name))
	if err != nil {
		return uuid.Nil, apperrors.InvalidParams(name+" must be a uuid", err)
	}
	return id, nil
}
