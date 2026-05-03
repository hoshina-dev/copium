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

func TestTemplateRepo_CreateAndGet(t *testing.T) {
	db := repotest.DB(t)
	repo := repositories.NewTemplateRepo(db)

	tpl := &models.EmailTemplate{ID: uuid.New(), Code: "welcome", Name: "Welcome"}
	if err := repo.Create(context.Background(), tpl); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.GetByID(context.Background(), tpl.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Code != "welcome" {
		t.Errorf("code=%q", got.Code)
	}

	got2, err := repo.GetByCode(context.Background(), "welcome")
	if err != nil {
		t.Fatal(err)
	}
	if got2.ID != tpl.ID {
		t.Errorf("get by code returned wrong row")
	}
}

func TestTemplateRepo_Get_NotFound(t *testing.T) {
	db := repotest.DB(t)
	repo := repositories.NewTemplateRepo(db)
	_, err := repo.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestTemplateRepo_DuplicateCode(t *testing.T) {
	db := repotest.DB(t)
	repo := repositories.NewTemplateRepo(db)
	if err := repo.Create(context.Background(), &models.EmailTemplate{ID: uuid.New(), Code: "dup", Name: "x"}); err != nil {
		t.Fatal(err)
	}
	err := repo.Create(context.Background(), &models.EmailTemplate{ID: uuid.New(), Code: "dup", Name: "y"})
	if !errors.Is(err, apperrors.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestTemplateRepo_SetActiveVersion(t *testing.T) {
	db := repotest.DB(t)
	tplRepo := repositories.NewTemplateRepo(db)
	verRepo := repositories.NewTemplateVersionRepo(db)

	tpl := &models.EmailTemplate{ID: uuid.New(), Code: "t1", Name: "x"}
	if err := tplRepo.Create(context.Background(), tpl); err != nil {
		t.Fatal(err)
	}
	v := &models.EmailTemplateVersion{
		ID: uuid.New(), TemplateID: tpl.ID, Version: 1,
		Subject: "s", BodyHTML: "x",
		ParamsSchema: models.JSONMap{"type": "object"},
	}
	if err := verRepo.Create(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	if err := tplRepo.SetActiveVersion(context.Background(), tpl.ID, v.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := tplRepo.GetByID(context.Background(), tpl.ID)
	if got.ActiveVersionID == nil || *got.ActiveVersionID != v.ID {
		t.Errorf("active not set: %v", got.ActiveVersionID)
	}
}
