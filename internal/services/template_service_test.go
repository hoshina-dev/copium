package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/clock"
	"github.com/hoshina-dev/copium/internal/idgen"
	"github.com/hoshina-dev/copium/internal/models"
	"github.com/hoshina-dev/copium/internal/services"
	"github.com/hoshina-dev/copium/internal/services/servicestest"
)

func newTplSvc(t *testing.T, ids ...uuid.UUID) (*services.TemplateService, *servicestest.Fakes) {
	t.Helper()
	f := servicestest.New()
	svc := services.NewTemplateService(services.TemplateDeps{
		Templates:        f.TemplateRepo,
		TemplateVersions: f.VersionRepo,
		Clock:            clock.NewFake(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		IDs:              idgen.NewStatic(ids...),
	})
	return svc, f
}

func TestCreateTemplate(t *testing.T) {
	id := uuid.New()
	svc, f := newTplSvc(t, id)
	got, err := svc.Create(context.Background(), models.CreateTemplateRequest{
		Code: "welcome", Name: "Welcome",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.ID != id {
		t.Errorf("id mismatch")
	}
	if _, ok := f.TemplateRepo.Templates[id]; !ok {
		t.Errorf("not persisted")
	}
}

func TestCreateTemplate_DuplicateCode(t *testing.T) {
	svc, f := newTplSvc(t, uuid.New(), uuid.New())
	f.TemplateRepo.Existing["welcome"] = true
	_, err := svc.Create(context.Background(), models.CreateTemplateRequest{Code: "welcome", Name: "X"})
	if !errors.Is(err, apperrors.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestCreateVersion_AssignsMonotonicNumberAndDoesNotActivate(t *testing.T) {
	verID := uuid.New()
	svc, f := newTplSvc(t, verID)
	tplID := uuid.New()
	f.TemplateRepo.Templates[tplID] = &models.EmailTemplate{ID: tplID, Code: "x", Name: "x"}
	f.VersionRepo.NextNumbers[tplID] = 7

	v, err := svc.CreateVersion(context.Background(), tplID, models.CreateTemplateVersionRequest{
		Subject: "s", BodyHTML: "<p>x</p>",
		ParamsSchema: models.JSONMap{"type": "object"},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if v.Version != 7 {
		t.Errorf("version=%d want 7", v.Version)
	}
	stored := f.VersionRepo.Versions[verID]
	if stored == nil {
		t.Fatal("version not persisted")
	}
	tpl := f.TemplateRepo.Templates[tplID]
	if tpl.ActiveVersionID != nil {
		t.Errorf("must NOT auto-activate; got %v", tpl.ActiveVersionID)
	}
}

func TestCreateVersion_FirstAutoActivates(t *testing.T) {
	verID := uuid.New()
	svc, f := newTplSvc(t, verID)
	tplID := uuid.New()
	f.TemplateRepo.Templates[tplID] = &models.EmailTemplate{ID: tplID, Code: "x", Name: "x"}
	f.VersionRepo.NextNumbers[tplID] = 1

	_, err := svc.CreateVersion(context.Background(), tplID, models.CreateTemplateVersionRequest{
		Subject: "s", BodyHTML: "<p>x</p>",
		ParamsSchema: models.JSONMap{"type": "object"},
	})
	if err != nil {
		t.Fatal(err)
	}
	tpl := f.TemplateRepo.Templates[tplID]
	if tpl.ActiveVersionID == nil || *tpl.ActiveVersionID != verID {
		t.Errorf("first version must auto-activate, got %v", tpl.ActiveVersionID)
	}
}

func TestCreateVersion_TemplateMissing(t *testing.T) {
	svc, _ := newTplSvc(t, uuid.New())
	_, err := svc.CreateVersion(context.Background(), uuid.New(), models.CreateTemplateVersionRequest{
		Subject: "s", BodyHTML: "x", ParamsSchema: models.JSONMap{"type": "object"},
	})
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCreateVersion_InvalidJSONSchemaRejected(t *testing.T) {
	svc, f := newTplSvc(t, uuid.New())
	tplID := uuid.New()
	f.TemplateRepo.Templates[tplID] = &models.EmailTemplate{ID: tplID, Code: "x", Name: "x"}
	f.VersionRepo.NextNumbers[tplID] = 1
	_, err := svc.CreateVersion(context.Background(), tplID, models.CreateTemplateVersionRequest{
		Subject: "s", BodyHTML: "x",
		ParamsSchema: models.JSONMap{"type": "no-such-type"},
	})
	if !errors.Is(err, apperrors.ErrInvalidParams) {
		t.Fatalf("want ErrInvalidParams (bad schema), got %v", err)
	}
}

func TestSetActive_VersionMustBelongToTemplate(t *testing.T) {
	svc, f := newTplSvc(t)
	tplID := uuid.New()
	verID := uuid.New()
	otherTplID := uuid.New()
	f.TemplateRepo.Templates[tplID] = &models.EmailTemplate{ID: tplID, Code: "x", Name: "x"}
	f.VersionRepo.Versions[verID] = &models.EmailTemplateVersion{ID: verID, TemplateID: otherTplID, Version: 1}
	err := svc.SetActiveVersion(context.Background(), tplID, verID)
	if !errors.Is(err, apperrors.ErrInvalidParams) {
		t.Fatalf("want ErrInvalidParams (cross-template version), got %v", err)
	}
}

func TestSetActive_OK(t *testing.T) {
	svc, f := newTplSvc(t)
	tplID := uuid.New()
	verID := uuid.New()
	f.TemplateRepo.Templates[tplID] = &models.EmailTemplate{ID: tplID, Code: "x", Name: "x"}
	f.VersionRepo.Versions[verID] = &models.EmailTemplateVersion{ID: verID, TemplateID: tplID, Version: 2}
	if err := svc.SetActiveVersion(context.Background(), tplID, verID); err != nil {
		t.Fatal(err)
	}
	if got := f.TemplateRepo.Templates[tplID].ActiveVersionID; got == nil || *got != verID {
		t.Errorf("activation didn't take")
	}
}
