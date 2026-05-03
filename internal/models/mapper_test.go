package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hoshina-dev/copium/internal/models"
)

func TestTemplateToResponse(t *testing.T) {
	id := uuid.New()
	verID := uuid.New()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tpl := &models.EmailTemplate{
		ID:              id,
		Code:            "welcome",
		Name:            "Welcome",
		Description:     "Greet new user",
		ActiveVersionID: &verID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	got := models.TemplateToResponse(tpl)
	if got.ID != id || got.Code != "welcome" || got.Name != "Welcome" {
		t.Fatalf("unexpected response: %+v", got)
	}
	if got.ActiveVersionID == nil || *got.ActiveVersionID != verID {
		t.Errorf("ActiveVersionID lost")
	}
}

func TestTemplateVersionToResponse(t *testing.T) {
	id := uuid.New()
	tplID := uuid.New()
	v := &models.EmailTemplateVersion{
		ID:           id,
		TemplateID:   tplID,
		Version:      3,
		Subject:      "Hi {{.name}}",
		BodyHTML:     "<p>Hello {{.name}}</p>",
		BodyText:     "Hello {{.name}}",
		ParamsSchema: models.JSONMap{"type": "object"},
		FromAddress:  "x@y.com",
	}
	got := models.TemplateVersionToResponse(v)
	if got.ID != id || got.TemplateID != tplID || got.Version != 3 {
		t.Fatalf("unexpected response: %+v", got)
	}
	if got.ParamsSchema["type"] != "object" {
		t.Errorf("schema lost")
	}
}

func TestOutboxToResponse(t *testing.T) {
	id := uuid.New()
	uid := uuid.New()
	verID := uuid.New()
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	out := &models.EmailOutbox{
		ID:                id,
		TemplateVersionID: verID,
		UserID:            uid,
		ToAddress:         "a@b.com",
		FromAddress:       "x@y.com",
		Subject:           "Hi",
		BodyHTML:          "<p>hi</p>",
		Status:            models.OutboxStatusQueued,
		Attempts:          0,
		MaxAttempts:       5,
		ScheduledAt:       now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	got := models.OutboxToResponse(out)
	if got.ID != id || got.UserID != uid || got.Status != "queued" {
		t.Fatalf("unexpected: %+v", got)
	}
}
