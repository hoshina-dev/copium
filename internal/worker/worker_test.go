package worker_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hoshina-dev/copium/internal/clock"
	"github.com/hoshina-dev/copium/internal/models"
	"github.com/hoshina-dev/copium/internal/sender"
	"github.com/hoshina-dev/copium/internal/worker"
)

// --- fakes ---

type fakeStore struct {
	mu          sync.Mutex
	rows        map[uuid.UUID]*models.EmailOutbox
	claimErr    error
	updateErr   error
	claimedHist [][]uuid.UUID
}

func newFakeStore() *fakeStore { return &fakeStore{rows: map[uuid.UUID]*models.EmailOutbox{}} }

func (f *fakeStore) put(o *models.EmailOutbox) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[o.ID] = o
}

func (f *fakeStore) get(id uuid.UUID) *models.EmailOutbox {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rows[id]
}

// ClaimDue moves up to limit ready 'queued' rows to 'sending' and returns them.
// 'now' is supplied by the worker (deterministic clock).
func (f *fakeStore) ClaimDue(_ context.Context, now time.Time, limit int) ([]*models.EmailOutbox, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	var picked []*models.EmailOutbox
	var ids []uuid.UUID
	for _, r := range f.rows {
		if r.Status == models.OutboxStatusQueued && !r.ScheduledAt.After(now) {
			r.Status = models.OutboxStatusSending
			picked = append(picked, r)
			ids = append(ids, r.ID)
			if len(picked) >= limit {
				break
			}
		}
	}
	f.claimedHist = append(f.claimedHist, ids)
	return picked, nil
}

func (f *fakeStore) MarkSent(_ context.Context, id uuid.UUID, provider, providerID string, sentAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	r := f.rows[id]
	r.Status = models.OutboxStatusSent
	r.Provider = provider
	r.ProviderMessageID = providerID
	t := sentAt
	r.SentAt = &t
	r.UpdatedAt = sentAt
	return nil
}

func (f *fakeStore) MarkFailureAndReschedule(_ context.Context, id uuid.UUID, attemptErr string, nextAttemptAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.rows[id]
	r.Attempts++
	r.LastError = attemptErr
	r.UpdatedAt = nextAttemptAt
	if r.Attempts >= r.MaxAttempts {
		r.Status = models.OutboxStatusDead
	} else {
		r.Status = models.OutboxStatusQueued
		r.ScheduledAt = nextAttemptAt
	}
	return nil
}

type fakeSender struct {
	mu      sync.Mutex
	sent    []sender.Message
	failN   int // first N calls fail
	failErr error
	calls   int
}

func (f *fakeSender) Name() string { return "fake" }
func (f *fakeSender) Send(_ context.Context, m sender.Message) (sender.SendResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls <= f.failN {
		return sender.SendResult{}, f.failErr
	}
	f.sent = append(f.sent, m)
	return sender.SendResult{ProviderMessageID: "fake:" + m.To}, nil
}

// --- helpers ---

func newRow(t *testing.T, when time.Time, maxAttempts int) *models.EmailOutbox {
	t.Helper()
	uid := uuid.New()
	return &models.EmailOutbox{
		ID:                uuid.New(),
		TemplateVersionID: uuid.New(),
		UserID:            &uid,
		ToAddress:         "rcpt@example.com",
		FromAddress:       "from@example.com",
		Subject:           "hi",
		BodyHTML:          "<p>hi</p>",
		Status:            models.OutboxStatusQueued,
		MaxAttempts:       maxAttempts,
		ScheduledAt:       when,
	}
}

// --- tests ---

func TestProcessOnce_HappyPath_QueuedToSent(t *testing.T) {
	now := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	store := newFakeStore()
	row := newRow(t, now, 5)
	store.put(row)

	snd := &fakeSender{}
	clk := clock.NewFake(now)
	w := worker.New(worker.Deps{Store: store, Sender: snd, Clock: clk, BatchSize: 10})

	n, err := w.ProcessOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("processed=%d want 1", n)
	}
	got := store.get(row.ID)
	if got.Status != models.OutboxStatusSent {
		t.Errorf("status=%v want sent", got.Status)
	}
	if got.SentAt == nil || !got.SentAt.Equal(now) {
		t.Errorf("SentAt=%v want %v", got.SentAt, now)
	}
	if got.Provider != "fake" {
		t.Errorf("Provider=%q", got.Provider)
	}
	if got.ProviderMessageID == "" {
		t.Errorf("ProviderMessageID empty")
	}
	if len(snd.sent) != 1 {
		t.Errorf("sender called %d times", len(snd.sent))
	}
}

func TestProcessOnce_NotYetDue_Skipped(t *testing.T) {
	now := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	store := newFakeStore()
	store.put(newRow(t, now.Add(time.Hour), 5))

	w := worker.New(worker.Deps{Store: store, Sender: &fakeSender{}, Clock: clock.NewFake(now), BatchSize: 10})
	n, err := w.ProcessOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("future row processed=%d", n)
	}
}

func TestProcessOnce_TransientFailure_Retries(t *testing.T) {
	now := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	store := newFakeStore()
	row := newRow(t, now, 3)
	store.put(row)

	snd := &fakeSender{failN: 1, failErr: errors.New("smtp 421 retry")}
	clk := clock.NewFake(now)
	w := worker.New(worker.Deps{Store: store, Sender: snd, Clock: clk, BatchSize: 10, BaseBackoff: time.Second})

	if _, err := w.ProcessOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := store.get(row.ID)
	if got.Status != models.OutboxStatusQueued {
		t.Errorf("after 1 fail status=%v want queued", got.Status)
	}
	if got.Attempts != 1 {
		t.Errorf("attempts=%d want 1", got.Attempts)
	}
	if got.LastError == "" {
		t.Errorf("LastError empty")
	}
	if !got.ScheduledAt.After(now) {
		t.Errorf("ScheduledAt=%v must be in the future", got.ScheduledAt)
	}
}

func TestProcessOnce_MaxAttempts_DeadLetter(t *testing.T) {
	now := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	store := newFakeStore()
	row := newRow(t, now, 1)
	store.put(row)
	snd := &fakeSender{failN: 100, failErr: errors.New("permafail")}
	w := worker.New(worker.Deps{Store: store, Sender: snd, Clock: clock.NewFake(now), BatchSize: 10})

	if _, err := w.ProcessOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := store.get(row.ID)
	if got.Status != models.OutboxStatusDead {
		t.Errorf("status=%v want dead (max attempts hit)", got.Status)
	}
}

func TestProcessOnce_Backoff_IsExponential(t *testing.T) {
	now := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	w := worker.New(worker.Deps{Clock: clock.NewFake(now), BaseBackoff: time.Second})
	cases := []struct {
		attempts int
		want     time.Duration
	}{
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
	}
	for _, tc := range cases {
		if got := w.Backoff(tc.attempts); got != tc.want {
			t.Errorf("Backoff(%d)=%v want %v", tc.attempts, got, tc.want)
		}
	}
}

func TestRun_LoopUntilContextCancelled(t *testing.T) {
	now := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	store := newFakeStore()
	store.put(newRow(t, now, 3))
	store.put(newRow(t, now, 3))
	snd := &fakeSender{}
	w := worker.New(worker.Deps{
		Store: store, Sender: snd, Clock: clock.NewFake(now),
		BatchSize: 10, PollInterval: 5 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	w.Run(ctx)
	if got := snd.calls; got < 2 {
		t.Errorf("sender calls=%d, want >= 2 (both rows processed)", got)
	}
}
