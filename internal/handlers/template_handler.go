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

// Create registers a new template (no version yet).
//
//	@Summary  Create a template
//	@Tags     templates
//	@Accept   json
//	@Produce  json
//	@Param    request body     models.CreateTemplateRequest true "Template metadata"
//	@Success  201     {object} models.TemplateResponse
//	@Failure  400     {object} models.ErrorResponse
//	@Failure  409     {object} models.ErrorResponse "code already exists"
//	@Router   /templates [post]
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

// List returns all templates.
//
//	@Summary  List templates
//	@Tags     templates
//	@Produce  json
//	@Success  200 {array} models.TemplateResponse
//	@Router   /templates [get]
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

// Get returns one template by id.
//
//	@Summary  Get a template
//	@Tags     templates
//	@Produce  json
//	@Param    id  path     string true "template UUID"
//	@Success  200 {object} models.TemplateResponse
//	@Failure  400 {object} models.ErrorResponse
//	@Failure  404 {object} models.ErrorResponse
//	@Router   /templates/{id} [get]
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

// CreateVersion appends an immutable version to a template. The first version
// is auto-activated.
//
//	@Summary     Create a template version
//	@Description Validates that params_schema is itself a compilable JSON Schema.
//	@Tags        templates
//	@Accept      json
//	@Produce     json
//	@Param       id      path     string                              true "template UUID"
//	@Param       request body     models.CreateTemplateVersionRequest true "Version content"
//	@Success     201     {object} models.TemplateVersionResponse
//	@Failure     400     {object} models.ErrorResponse "invalid body or schema"
//	@Failure     404     {object} models.ErrorResponse "template not found"
//	@Router      /templates/{id}/versions [post]
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

// ListVersions returns every version of a template, newest first.
//
//	@Summary  List template versions
//	@Tags     templates
//	@Produce  json
//	@Param    id  path  string true "template UUID"
//	@Success  200 {array} models.TemplateVersionResponse
//	@Failure  404 {object} models.ErrorResponse
//	@Router   /templates/{id}/versions [get]
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

// GetVersion returns one specific version of a template.
//
//	@Summary  Get a template version
//	@Tags     templates
//	@Produce  json
//	@Param    id      path     string true "template UUID"
//	@Param    version path     int    true "version number"
//	@Success  200     {object} models.TemplateVersionResponse
//	@Failure  400     {object} models.ErrorResponse
//	@Failure  404     {object} models.ErrorResponse
//	@Router   /templates/{id}/versions/{version} [get]
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

// SetActiveVersion repoints email_templates.active_version_id at an existing version.
//
//	@Summary  Set the active version
//	@Tags     templates
//	@Accept   json
//	@Param    id      path string                          true "template UUID"
//	@Param    request body models.SetActiveVersionRequest  true "Target version id"
//	@Success  200
//	@Failure  400 {object} models.ErrorResponse "version belongs to another template"
//	@Failure  404 {object} models.ErrorResponse
//	@Router   /templates/{id}/active-version [patch]
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
