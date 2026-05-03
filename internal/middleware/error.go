// Package middleware hosts cross-cutting Fiber middleware: error mapping,
// request logging, etc.
package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/models"
)

// ErrorHandler is the global Fiber ErrorHandler. It maps:
//   - apperrors sentinels -> their HTTP status with the wrapped message
//   - *fiber.Error        -> the embedded code and message
//   - everything else     -> 500 with a generic message (cause is logged)
func ErrorHandler(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}
	var fe *fiber.Error
	switch {
	case errors.As(err, &fe):
		return c.Status(fe.Code).JSON(models.ErrorResponse{Error: fe.Message})
	case isApperror(err):
		return c.Status(apperrors.StatusCode(err)).JSON(models.ErrorResponse{Error: err.Error()})
	default:
		log.Printf("internal error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(models.ErrorResponse{Error: "internal error"})
	}
}

func isApperror(err error) bool {
	for _, sentinel := range []error{
		apperrors.ErrNotFound,
		apperrors.ErrInvalidParams,
		apperrors.ErrConflict,
		apperrors.ErrUpstream,
		apperrors.ErrInternal,
	} {
		if errors.Is(err, sentinel) {
			return true
		}
	}
	return false
}
