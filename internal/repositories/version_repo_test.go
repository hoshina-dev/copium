//go:build integration

package repositories_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/hoshina-dev/copium/internal/apperrors"
	"github.com/hoshina-dev/copium/internal/models"
	"github.com/hoshina-dev/copium/internal/repositories"
	"github.com/hoshina-dev/copium/internal/repositories/repotest"
)

func TestVersionRepo_NextVersionNumber_StartsAt1AndIncrements(t *testing.T) {
	db := repotest.DB(t)
	tplRepo := repositories.NewTemplateRepo(db)
	verRepo := repositories.NewTemplateVersionRepo(db)
	ctx := context.Background()

	tpl := &models.EmailTemplate{ID: uuid.New(), Code: "v1", Name: "x"}
	if err := tplRepo.Create(ctx, tpl); err != nil {
		t.Fatal(err)
	}
	n, err := verRepo.NextVersionNumber(ctx, tpl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("next on empty=%d want 1", n)
	}
	v := &models.EmailTemplateVersion{
		ID: uuid.New(), TemplateID: tpl.ID, Version: 1,
		Subject: "s", BodyHTML: "x", ParamsSchema: models.JSONMap{"type": "object"},
	}
	if err := verRepo.Create(ctx, v); err != nil {
		t.Fatal(err)
	}
	n2, err := verRepo.NextVersionNumber(ctx, tpl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 2 {
		t.Errorf("next after 1=%d want 2", n2)
	}
}

func TestVersionRepo_DuplicateVersionNumberRejected(t *testing.T) {
	db := repotest.DB(t)
	tplRepo := repositories.NewTemplateRepo(db)
	verRepo := repositories.NewTemplateVersionRepo(db)
	ctx := context.Background()

	tpl := &models.EmailTemplate{ID: uuid.New(), Code: "v2", Name: "x"}
	if err := tplRepo.Create(ctx, tpl); err != nil {
		t.Fatal(err)
	}
	v := &models.EmailTemplateVersion{
		ID: uuid.New(), TemplateID: tpl.ID, Version: 1,
		Subject: "s", BodyHTML: "x", ParamsSchema: models.JSONMap{"type": "object"},
	}
	if err := verRepo.Create(ctx, v); err != nil {
		t.Fatal(err)
	}
	dup := &models.EmailTemplateVersion{
		ID: uuid.New(), TemplateID: tpl.ID, Version: 1,
		Subject: "s", BodyHTML: "x", ParamsSchema: models.JSONMap{"type": "object"},
	}
	err := verRepo.Create(ctx, dup)
	if !errors.Is(err, apperrors.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
}

func TestVersionRepo_GetByTemplateAndVersion(t *testing.T) {
	db := repotest.DB(t)
	tplRepo := repositories.NewTemplateRepo(db)
	verRepo := repositories.NewTemplateVersionRepo(db)
	ctx := context.Background()

	tpl := &models.EmailTemplate{ID: uuid.New(), Code: "v3", Name: "x"}
	_ = tplRepo.Create(ctx, tpl)
	v := &models.EmailTemplateVersion{
		ID: uuid.New(), TemplateID: tpl.ID, Version: 7,
		Subject: "s", BodyHTML: "x", ParamsSchema: models.JSONMap{"type": "object"},
	}
	_ = verRepo.Create(ctx, v)

	got, err := verRepo.GetByTemplateAndVersion(ctx, tpl.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != v.ID {
		t.Errorf("wrong row")
	}
}
