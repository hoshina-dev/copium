// Package worker is the outbox dispatcher. It polls email_outbox for due
// 'queued' rows, sends them via the injected Sender, and updates state.
//
// Collaborators (OutboxStore, Sender, Clock) are interfaces declared here, so
// tests inject in-memory fakes and the composition root injects the real
// adapters.
package worker

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/hoshina-dev/copium/internal/models"
	"github.com/hoshina-dev/copium/internal/sender"
)

// OutboxStore is the persistence boundary the worker needs. It deliberately
// does NOT expose generic CRUD - only the verbs the dispatcher uses.
type OutboxStore interface {
	// ClaimDue atomically moves up to limit ready 'queued' rows (where
	// scheduled_at <= now) into 'sending' status and returns them. The
	// real implementation uses FOR UPDATE SKIP LOCKED so concurrent
	// workers never pick the same row.
	ClaimDue(ctx context.Context, now time.Time, limit int) ([]*models.EmailOutbox, error)
	MarkSent(ctx context.Context, id uuid.UUID, provider, providerID string, sentAt time.Time) error
	MarkFailureAndReschedule(ctx context.Context, id uuid.UUID, attemptErr string, nextAttemptAt time.Time) error
}

type Sender interface {
	Name() string
	Send(ctx context.Context, m sender.Message) (sender.SendResult, error)
}

type Clock interface {
	Now() time.Time
}

type Deps struct {
	Store        OutboxStore
	Sender       Sender
	Clock        Clock
	BatchSize    int
	PollInterval time.Duration
	BaseBackoff  time.Duration // first retry delay; doubles each attempt
}

type Worker struct{ deps Deps }

func New(d Deps) *Worker {
	if d.BatchSize <= 0 {
		d.BatchSize = 10
	}
	if d.PollInterval <= 0 {
		d.PollInterval = 2 * time.Second
	}
	if d.BaseBackoff <= 0 {
		d.BaseBackoff = time.Second
	}
	return &Worker{deps: d}
}

// Run blocks, polling on PollInterval until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.deps.PollInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := w.ProcessOnce(ctx); err != nil {
				log.Printf("worker: process: %v", err)
			}
		}
	}
}

// ProcessOnce claims one batch and dispatches each row sequentially.
// Returns the number of rows attempted (not necessarily succeeded).
func (w *Worker) ProcessOnce(ctx context.Context) (int, error) {
	now := w.deps.Clock.Now()
	rows, err := w.deps.Store.ClaimDue(ctx, now, w.deps.BatchSize)
	if err != nil {
		return 0, err
	}
	for _, r := range rows {
		w.dispatch(ctx, r)
	}
	return len(rows), nil
}

func (w *Worker) dispatch(ctx context.Context, r *models.EmailOutbox) {
	now := w.deps.Clock.Now()
	res, err := w.deps.Sender.Send(ctx, sender.Message{
		To:       r.ToAddress,
		From:     r.FromAddress,
		Subject:  r.Subject,
		BodyHTML: r.BodyHTML,
		BodyText: r.BodyText,
	})
	if err != nil {
		nextAttempt := r.Attempts + 1
		next := now.Add(w.Backoff(nextAttempt))
		if err2 := w.deps.Store.MarkFailureAndReschedule(ctx, r.ID, err.Error(), next); err2 != nil {
			log.Printf("worker: mark fail %s: %v", r.ID, err2)
		}
		return
	}
	if err := w.deps.Store.MarkSent(ctx, r.ID, w.deps.Sender.Name(), res.ProviderMessageID, now); err != nil {
		log.Printf("worker: mark sent %s: %v", r.ID, err)
	}
}

// Backoff is exponential: BaseBackoff * 2^(attempts-1).
func (w *Worker) Backoff(attempts int) time.Duration {
	if attempts <= 0 {
		return 0
	}
	d := w.deps.BaseBackoff
	for i := 1; i < attempts; i++ {
		d *= 2
	}
	return d
}
