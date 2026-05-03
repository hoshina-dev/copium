package apperrors_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hoshina-dev/copium/internal/apperrors"
)

func TestSentinelsAreDistinct(t *testing.T) {
	all := []error{
		apperrors.ErrNotFound,
		apperrors.ErrInvalidParams,
		apperrors.ErrConflict,
		apperrors.ErrUpstream,
		apperrors.ErrInternal,
	}
	for i, a := range all {
		for j, b := range all {
			if i == j {
				continue
			}
			if errors.Is(a, b) {
				t.Fatalf("expected %v and %v to be distinct sentinels", a, b)
			}
		}
	}
}

func TestWrapPreservesIs(t *testing.T) {
	cases := map[string]struct {
		sentinel error
		wrap     func(error) error
	}{
		"NotFound":      {apperrors.ErrNotFound, func(e error) error { return apperrors.NotFound("user", e) }},
		"InvalidParams": {apperrors.ErrInvalidParams, func(e error) error { return apperrors.InvalidParams("bad", e) }},
		"Conflict":      {apperrors.ErrConflict, func(e error) error { return apperrors.Conflict("dup", e) }},
		"Upstream":      {apperrors.ErrUpstream, func(e error) error { return apperrors.Upstream("custapi", e) }},
		"Internal":      {apperrors.ErrInternal, func(e error) error { return apperrors.Internal("boom", e) }},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cause := errors.New("root cause")
			err := tc.wrap(cause)
			if err == nil {
				t.Fatal("expected non-nil error")
			}
			if !errors.Is(err, tc.sentinel) {
				t.Fatalf("errors.Is(err, %v) = false; want true", tc.sentinel)
			}
			if !errors.Is(err, cause) {
				t.Fatalf("errors.Is(err, cause) = false; want true (must preserve cause)")
			}
		})
	}
}

func TestWrapWithoutCauseStillMatchesSentinel(t *testing.T) {
	err := apperrors.NotFound("template", nil)
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("wrap with nil cause must still match sentinel")
	}
}

func TestErrorMessageIncludesContext(t *testing.T) {
	err := apperrors.NotFound("template foo", errors.New("zero rows"))
	got := err.Error()
	for _, want := range []string{"not found", "template foo", "zero rows"} {
		if !contains(got, want) {
			t.Errorf("Error() = %q; missing %q", got, want)
		}
	}
}

func TestStatusCodeMapping(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{apperrors.ErrNotFound, 404},
		{apperrors.NotFound("x", nil), 404},
		{apperrors.ErrInvalidParams, 400},
		{apperrors.InvalidParams("x", nil), 400},
		{apperrors.ErrConflict, 409},
		{apperrors.Conflict("x", nil), 409},
		{apperrors.ErrUpstream, 502},
		{apperrors.Upstream("x", nil), 502},
		{apperrors.ErrInternal, 500},
		{apperrors.Internal("x", nil), 500},
		{errors.New("anything else"), 500},
		{nil, 200},
		{fmt.Errorf("wrapped: %w", apperrors.ErrNotFound), 404},
	}
	for _, tc := range cases {
		got := apperrors.StatusCode(tc.err)
		if got != tc.code {
			t.Errorf("StatusCode(%v) = %d; want %d", tc.err, got, tc.code)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
