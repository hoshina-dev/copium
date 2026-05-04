// Package handlers binds Fiber routes to service methods. Handlers depend on
// concrete service structs by design - service interfaces only get introduced
// if a handler test ever needs to fake one (YAGNI).
package handlers

import (
	"time"

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

// Send enqueues an email for asynchronous delivery.
//
//	@Summary     Enqueue an email
//	@Description Validates params against the active template version's JSON Schema,
//	@Description resolves the recipient via custapi, renders the message, and writes
//	@Description a snapshot to email_outbox. The worker performs the actual send.
//	@Tags        emails
//	@Accept      json
//	@Produce     json
//	@Param       request body     models.SendEmailRequest true "Send request"
//	@Success     202     {object} models.SendEmailResponse
//	@Failure     400     {object} models.ErrorResponse "invalid JSON or params failed schema"
//	@Failure     404     {object} models.ErrorResponse "template or user not found"
//	@Failure     502     {object} models.ErrorResponse "custapi unreachable"
//	@Router      /emails/send [post]
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

// Get returns the current state of one outbox row.
//
//	@Summary     Inspect an outbox row
//	@Description Returns the snapshot, status, attempts, last_error and provider
//	@Description message id for one queued/sent/dead email.
//	@Tags        emails
//	@Produce     json
//	@Param       id   path     string true "outbox UUID"
//	@Success     200  {object} models.OutboxResponse
//	@Failure     400  {object} models.ErrorResponse "id is not a UUID"
//	@Failure     404  {object} models.ErrorResponse
//	@Router      /emails/{id} [get]
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

// List returns recent outbox rows, optionally narrowed by status and a
// created_at time window. Designed for the web UI "queue" view.
//
//	@Summary     List outbox rows
//	@Description Returns recent outbox rows newest-first, optionally narrowed by
//	@Description status and a created_at window. `from`/`to` accept RFC3339 time.
//	@Tags        emails
//	@Produce     json
//	@Param       status query string false "queued|sending|sent|failed|dead"
//	@Param       from   query string false "RFC3339 start (inclusive)"
//	@Param       to     query string false "RFC3339 end (exclusive)"
//	@Param       limit  query int    false "max rows (default 200, max 1000)"
//	@Success     200    {array}  models.OutboxResponse
//	@Failure     400    {object} models.ErrorResponse "bad query params"
//	@Router      /emails [get]
func (h *EmailHandler) List(c *fiber.Ctx) error {
	f := models.OutboxListFilter{Status: c.Query("status")}
	if s := c.Query("from"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return apperrors.InvalidParams("from must be RFC3339", err)
		}
		f.From = &t
	}
	if s := c.Query("to"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return apperrors.InvalidParams("to must be RFC3339", err)
		}
		f.To = &t
	}
	if c.Query("limit") != "" {
		n := c.QueryInt("limit")
		if n <= 0 {
			return apperrors.InvalidParams("limit must be positive", nil)
		}
		f.Limit = n
	}
	rows, err := h.svc.ListOutbox(c.Context(), f)
	if err != nil {
		return err
	}
	out := make([]models.OutboxResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.OutboxToResponse(r))
	}
	return c.JSON(out)
}
