package custapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/clients/custapi"
)

func TestGetUserByID_OK(t *testing.T) {
	id := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/id/"+id.String() {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"` + id.String() + `","email":"alice@example.com","name":"Alice"}`))
	}))
	t.Cleanup(srv.Close)

	c := custapi.New(srv.URL, custapi.WithTimeout(2*time.Second))
	u, err := c.GetUserByID(context.Background(), id)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if u.Email != "alice@example.com" {
		t.Errorf("got %q", u.Email)
	}
	if u.ID != id {
		t.Errorf("id mismatch")
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"user not found"}`))
	}))
	t.Cleanup(srv.Close)

	c := custapi.New(srv.URL)
	_, err := c.GetUserByID(context.Background(), uuid.New())
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestGetUserByID_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	c := custapi.New(srv.URL)
	_, err := c.GetUserByID(context.Background(), uuid.New())
	if !errors.Is(err, apperrors.ErrUpstream) {
		t.Fatalf("want ErrUpstream, got %v", err)
	}
}

func TestGetUserByID_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<not json>`))
	}))
	t.Cleanup(srv.Close)
	c := custapi.New(srv.URL)
	_, err := c.GetUserByID(context.Background(), uuid.New())
	if !errors.Is(err, apperrors.ErrUpstream) {
		t.Fatalf("want ErrUpstream, got %v", err)
	}
}

func TestResolveEmail_DelegatesToGetUser(t *testing.T) {
	id := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"` + id.String() + `","email":"bob@example.com"}`))
	}))
	t.Cleanup(srv.Close)

	c := custapi.New(srv.URL)
	email, err := c.ResolveEmail(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if email != "bob@example.com" {
		t.Errorf("got %q", email)
	}
}

func TestGetUserByID_ContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	c := custapi.New(srv.URL, custapi.WithTimeout(50*time.Millisecond))
	_, err := c.GetUserByID(context.Background(), uuid.New())
	if !errors.Is(err, apperrors.ErrUpstream) {
		t.Fatalf("expected ErrUpstream from timeout, got %v", err)
	}
}

func TestNew_TrimsTrailingSlash(t *testing.T) {
	id := uuid.New()
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"` + id.String() + `","email":"x@x"}`))
	}))
	t.Cleanup(srv.Close)

	c := custapi.New(srv.URL + "/")
	_, err := c.GetUserByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	want := "/api/v1/users/id/" + id.String()
	if seen != want {
		t.Errorf("seen=%q want=%q (no double //)", seen, want)
	}
}
