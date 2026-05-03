package renderer_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/models"
	"github.com/hoshina-dev/copium/internal/renderer"
)

func newRenderer(t *testing.T) *renderer.Renderer {
	t.Helper()
	r, err := renderer.New()
	if err != nil {
		t.Fatalf("renderer.New: %v", err)
	}
	return r
}

func TestRender_Happy(t *testing.T) {
	r := newRenderer(t)
	v := &models.EmailTemplateVersion{
		Subject:  "Hi {{.name}}",
		BodyHTML: "<p>Hello {{.name}}, you owe {{.amount}}</p>",
		BodyText: "Hello {{.name}}, you owe {{.amount}}",
		ParamsSchema: models.JSONMap{
			"type":     "object",
			"required": []any{"name", "amount"},
			"properties": map[string]any{
				"name":   map[string]any{"type": "string"},
				"amount": map[string]any{"type": "number"},
			},
		},
	}
	out, err := r.Render(v, models.JSONMap{"name": "Alice", "amount": float64(42)})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out.Subject != "Hi Alice" {
		t.Errorf("subject=%q", out.Subject)
	}
	if out.BodyHTML != "<p>Hello Alice, you owe 42</p>" {
		t.Errorf("html=%q", out.BodyHTML)
	}
	if out.BodyText != "Hello Alice, you owe 42" {
		t.Errorf("text=%q", out.BodyText)
	}
}

func TestRender_MissingRequired(t *testing.T) {
	r := newRenderer(t)
	v := &models.EmailTemplateVersion{
		Subject:  "Hi {{.name}}",
		BodyHTML: "<p>Hi {{.name}}</p>",
		ParamsSchema: models.JSONMap{
			"type": "object", "required": []any{"name"},
			"properties": map[string]any{"name": map[string]any{"type": "string"}},
		},
	}
	_, err := r.Render(v, models.JSONMap{})
	if !errors.Is(err, apperrors.ErrInvalidParams) {
		t.Fatalf("want ErrInvalidParams, got %v", err)
	}
}

func TestRender_WrongType(t *testing.T) {
	r := newRenderer(t)
	v := &models.EmailTemplateVersion{
		Subject:  "x",
		BodyHTML: "x",
		ParamsSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]any{
				"amount": map[string]any{"type": "number"},
			},
		},
	}
	_, err := r.Render(v, models.JSONMap{"amount": "not-a-number"})
	if !errors.Is(err, apperrors.ErrInvalidParams) {
		t.Fatalf("want ErrInvalidParams, got %v", err)
	}
}

func TestRender_InvalidSchema(t *testing.T) {
	r := newRenderer(t)
	v := &models.EmailTemplateVersion{
		Subject:      "x",
		BodyHTML:     "x",
		ParamsSchema: models.JSONMap{"type": "no-such-type"},
	}
	_, err := r.Render(v, models.JSONMap{})
	if err == nil {
		t.Fatal("expected error for invalid schema")
	}
}

func TestRender_TemplateSyntaxError(t *testing.T) {
	r := newRenderer(t)
	v := &models.EmailTemplateVersion{
		Subject:      "{{ broken",
		BodyHTML:     "ok",
		ParamsSchema: models.JSONMap{"type": "object"},
	}
	_, err := r.Render(v, models.JSONMap{})
	if err == nil {
		t.Fatal("expected error for invalid subject template")
	}
}

func TestRender_HTMLEscaping(t *testing.T) {
	r := newRenderer(t)
	v := &models.EmailTemplateVersion{
		Subject:      "Hi {{.name}}",
		BodyHTML:     "<p>{{.name}}</p>",
		ParamsSchema: models.JSONMap{"type": "object"},
	}
	out, err := r.Render(v, models.JSONMap{"name": "<script>alert(1)</script>"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.BodyHTML, "<script>") {
		t.Errorf("html body must escape: %q", out.BodyHTML)
	}
	// Subject is text/template - no HTML escaping (raw passthrough is correct).
	if out.Subject != "Hi <script>alert(1)</script>" {
		t.Errorf("subject text-template: %q", out.Subject)
	}
}

func TestRender_NilBodyText(t *testing.T) {
	r := newRenderer(t)
	v := &models.EmailTemplateVersion{
		Subject:      "x",
		BodyHTML:     "x",
		BodyText:     "",
		ParamsSchema: models.JSONMap{"type": "object"},
	}
	out, err := r.Render(v, models.JSONMap{})
	if err != nil {
		t.Fatal(err)
	}
	if out.BodyText != "" {
		t.Errorf("expected empty body_text, got %q", out.BodyText)
	}
}
