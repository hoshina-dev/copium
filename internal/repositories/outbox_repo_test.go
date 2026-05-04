//go:build integration

package repositories_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hoshina-dev/copium/internal/models"
	"github.com/hoshina-dev/copium/internal/repositories"
	"github.com/hoshina-dev/copium/internal/repositories/repotest"
)

func uuidPtr(u uuid.UUID) *uuid.UUID { return &u }

func seedTemplate(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	tplRepo := repositories.NewTemplateRepo(db)
	verRepo := repositories.NewTemplateVersionRepo(db)
	ctx := context.Background()
	tpl := &models.EmailTemplate{ID: uuid.New(), Code: uuid.NewString(), Name: "x"}
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
	return tpl.ID, v.ID
}

func TestOutboxRepo_CreateAndGet(t *testing.T) {
	db := repotest.DB(t)
	repo := repositories.NewOutboxRepo(db)
	ctx := context.Background()
	_, verID := seedTemplate(t, db)

	row := &models.EmailOutbox{
		ID: uuid.New(), TemplateVersionID: verID, UserID: uuidPtr(uuid.New()),
		ToAddress: "a@b", FromAddress: "x@y", Subject: "hi",
		BodyHTML: "<p>hi</p>", Status: models.OutboxStatusQueued,
		MaxAttempts: 5, ScheduledAt: time.Now().UTC(),
	}
	if err := repo.Create(ctx, row); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ToAddress != "a@b" {
		t.Errorf("To=%q", got.ToAddress)
	}
	if got.Status != models.OutboxStatusQueued {
		t.Errorf("status=%v", got.Status)
	}
}

func TestOutboxRepo_ClaimDue_Skips_FutureRows(t *testing.T) {
	db := repotest.DB(t)
	repo := repositories.NewOutboxRepo(db)
	ctx := context.Background()
	_, verID := seedTemplate(t, db)

	now := time.Now().UTC()
	due := &models.EmailOutbox{
		ID: uuid.New(), TemplateVersionID: verID, UserID: uuidPtr(uuid.New()),
		ToAddress: "a@b", FromAddress: "x@y", Subject: "hi", BodyHTML: "x",
		Status: models.OutboxStatusQueued, MaxAttempts: 5,
		ScheduledAt: now.Add(-time.Minute),
	}
	notDue := &models.EmailOutbox{
		ID: uuid.New(), TemplateVersionID: verID, UserID: uuidPtr(uuid.New()),
		ToAddress: "a@b", FromAddress: "x@y", Subject: "hi", BodyHTML: "x",
		Status: models.OutboxStatusQueued, MaxAttempts: 5,
		ScheduledAt: now.Add(time.Hour),
	}
	if err := repo.Create(ctx, due); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, notDue); err != nil {
		t.Fatal(err)
	}

	rows, err := repo.ClaimDue(ctx, now, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != due.ID {
		t.Fatalf("wrong rows claimed: %+v", rows)
	}
	got, _ := repo.GetByID(ctx, due.ID)
	if got.Status != models.OutboxStatusSending {
		t.Errorf("must flip to sending: %v", got.Status)
	}
}

func TestOutboxRepo_ClaimDue_SkipLocked_Concurrent(t *testing.T) {
	db := repotest.DB(t)
	repo := repositories.NewOutboxRepo(db)
	ctx := context.Background()
	_, verID := seedTemplate(t, db)

	now := time.Now().UTC()
	for i := 0; i < 6; i++ {
		row := &models.EmailOutbox{
			ID: uuid.New(), TemplateVersionID: verID, UserID: uuidPtr(uuid.New()),
			ToAddress: "a@b", FromAddress: "x@y", Subject: "hi", BodyHTML: "x",
			Status: models.OutboxStatusQueued, MaxAttempts: 5,
			ScheduledAt: now.Add(-time.Second),
		}
		if err := repo.Create(ctx, row); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	results := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := repo.ClaimDue(ctx, now, 3)
			if err != nil {
				t.Errorf("claim err: %v", err)
				return
			}
			results <- len(rows)
		}()
	}
	wg.Wait()
	close(results)
	total := 0
	for n := range results {
		total += n
	}
	if total != 6 {
		t.Errorf("two concurrent claimers got %d total; want 6 (no overlap, no missing)", total)
	}
}

func TestOutboxRepo_MarkSentAndMarkFailureAndReschedule(t *testing.T) {
	db := repotest.DB(t)
	repo := repositories.NewOutboxRepo(db)
	ctx := context.Background()
	_, verID := seedTemplate(t, db)
	now := time.Now().UTC()

	r := &models.EmailOutbox{
		ID: uuid.New(), TemplateVersionID: verID, UserID: uuidPtr(uuid.New()),
		ToAddress: "a@b", FromAddress: "x@y", Subject: "hi", BodyHTML: "x",
		Status: models.OutboxStatusSending, MaxAttempts: 3,
		ScheduledAt: now.Add(-time.Hour),
	}
	if err := repo.Create(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkSent(ctx, r.ID, "smtp", "msg-1", now); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetByID(ctx, r.ID)
	if got.Status != models.OutboxStatusSent {
		t.Errorf("status=%v", got.Status)
	}
	if got.Provider != "smtp" || got.ProviderMessageID != "msg-1" {
		t.Errorf("provider info missing")
	}
	if got.SentAt == nil {
		t.Errorf("SentAt nil")
	}

	r2 := &models.EmailOutbox{
		ID: uuid.New(), TemplateVersionID: verID, UserID: uuidPtr(uuid.New()),
		ToAddress: "a@b", FromAddress: "x@y", Subject: "hi", BodyHTML: "x",
		Status: models.OutboxStatusSending, Attempts: 0, MaxAttempts: 2,
	}
	if err := repo.Create(ctx, r2); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkFailureAndReschedule(ctx, r2.ID, "smtp 421", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got2, _ := repo.GetByID(ctx, r2.ID)
	if got2.Attempts != 1 {
		t.Errorf("attempts=%d", got2.Attempts)
	}
	if got2.Status != models.OutboxStatusQueued {
		t.Errorf("status=%v want queued", got2.Status)
	}

	if err := repo.MarkFailureAndReschedule(ctx, r2.ID, "smtp 421 again", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got3, _ := repo.GetByID(ctx, r2.ID)
	if got3.Status != models.OutboxStatusDead {
		t.Errorf("status=%v want dead at max attempts", got3.Status)
	}
}
