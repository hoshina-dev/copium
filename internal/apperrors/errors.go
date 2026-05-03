// Package apperrors defines typed sentinel errors used across copium.
//
// Layers (repository, service, client) wrap their failures with one of the
// constructors here. Handlers and middleware translate to HTTP via StatusCode.
//
// Sentinels are matched with errors.Is so callers can branch without comparing
// strings; wrapping always preserves the original cause via fmt.Errorf %w.
package apperrors

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidParams = errors.New("invalid params")
	ErrConflict      = errors.New("conflict")
	ErrUpstream      = errors.New("upstream failure")
	ErrInternal      = errors.New("internal error")
)

// chainErr carries both a sentinel (matched with errors.Is) and an optional
// underlying cause. We can't use a single fmt.Errorf because we want
// errors.Is(err, sentinel) AND errors.Is(err, cause) to both be true even when
// cause exists.
type chainErr struct {
	sentinel error
	context  string
	cause    error
}

func (e *chainErr) Error() string {
	switch {
	case e.context != "" && e.cause != nil:
		return fmt.Sprintf("%s: %s: %s", e.sentinel.Error(), e.context, e.cause.Error())
	case e.context != "":
		return fmt.Sprintf("%s: %s", e.sentinel.Error(), e.context)
	case e.cause != nil:
		return fmt.Sprintf("%s: %s", e.sentinel.Error(), e.cause.Error())
	default:
		return e.sentinel.Error()
	}
}

// Is reports a match for either the sentinel OR the wrapped cause.
func (e *chainErr) Is(target error) bool {
	if target == nil {
		return false
	}
	if errors.Is(e.sentinel, target) {
		return true
	}
	if e.cause != nil && errors.Is(e.cause, target) {
		return true
	}
	return false
}

// Unwrap returns the cause so errors.Is/As can keep walking.
func (e *chainErr) Unwrap() error { return e.cause }

func wrap(sentinel error, context string, cause error) error {
	return &chainErr{sentinel: sentinel, context: context, cause: cause}
}

func NotFound(context string, cause error) error      { return wrap(ErrNotFound, context, cause) }
func InvalidParams(context string, cause error) error { return wrap(ErrInvalidParams, context, cause) }
func Conflict(context string, cause error) error      { return wrap(ErrConflict, context, cause) }
func Upstream(context string, cause error) error      { return wrap(ErrUpstream, context, cause) }
func Internal(context string, cause error) error      { return wrap(ErrInternal, context, cause) }

// StatusCode maps an error (possibly wrapped) to an HTTP status code.
// Returns 200 for nil and 500 for any unknown error.
func StatusCode(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrInvalidParams):
		return http.StatusBadRequest
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrUpstream):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}
