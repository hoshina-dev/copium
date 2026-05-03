// Package renderer compiles a template version (subject + html + text +
// params schema) and produces a fully-rendered Output for one set of params.
//
// It validates params against the version's JSON Schema before rendering so
// invalid input fails with apperrors.ErrInvalidParams (HTTP 400) instead of
// producing a malformed email.
package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	htmltpl "html/template"
	"strings"
	texttpl "text/template"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/models"
)

type Output struct {
	Subject  string
	BodyHTML string
	BodyText string
}

type Renderer struct{}

func New() (*Renderer, error) { return &Renderer{}, nil }

// Render validates params against version.ParamsSchema, then renders all
// three text fields. Subject + BodyText use text/template (raw); BodyHTML
// uses html/template (auto-escapes user input).
func (r *Renderer) Render(v *models.EmailTemplateVersion, params models.JSONMap) (*Output, error) {
	if err := validateParams(v.ParamsSchema, params); err != nil {
		return nil, err
	}
	subject, err := renderText("subject", v.Subject, params)
	if err != nil {
		return nil, fmt.Errorf("render subject: %w", err)
	}
	bodyHTML, err := renderHTML("body_html", v.BodyHTML, params)
	if err != nil {
		return nil, fmt.Errorf("render body_html: %w", err)
	}
	var bodyText string
	if strings.TrimSpace(v.BodyText) != "" {
		bodyText, err = renderText("body_text", v.BodyText, params)
		if err != nil {
			return nil, fmt.Errorf("render body_text: %w", err)
		}
	}
	return &Output{Subject: subject, BodyHTML: bodyHTML, BodyText: bodyText}, nil
}

func validateParams(schema models.JSONMap, params models.JSONMap) error {
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	const url = "schema://params"
	if err := compiler.AddResource(url, bytes.NewReader(schemaJSON)); err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}
	sch, err := compiler.Compile(url)
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	if params == nil {
		params = models.JSONMap{}
	}
	paramsJSON, err := json.Marshal(map[string]any(params))
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}
	var v any
	if err := json.Unmarshal(paramsJSON, &v); err != nil {
		return fmt.Errorf("unmarshal params: %w", err)
	}
	if err := sch.Validate(v); err != nil {
		return apperrors.InvalidParams(err.Error(), nil)
	}
	return nil
}

func renderText(name, src string, data any) (string, error) {
	t, err := texttpl.New(name).Parse(src)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderHTML(name, src string, data any) (string, error) {
	t, err := htmltpl.New(name).Parse(src)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
