// Package worker is the outbox dispatcher. It polls email_outbox for due
// 'queued' rows, sends them via the injected Sender, and updates state.
//
// Collaborators (OutboxStore, Sender, Clock) are interfaces declared here, so
// tests inject in-memory fakes and the composition root injects the real
// adapters.
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

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

type Worker struct {
	deps    Deps
	metrics workerMetrics
}

type workerMetrics struct {
	claimed  metric.Int64Counter
	sent     metric.Int64Counter
	failed   metric.Int64Counter
	duration metric.Float64Histogram
}

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
	// Pull from the globally-installed MeterProvider. When otel is disabled
	// this is a noop meter and the counters are free.
	m := otel.Meter("copium/worker")
	claimed, _ := m.Int64Counter("copium.worker.claimed",
		metric.WithDescription("Outbox rows claimed by this worker"))
	sent, _ := m.Int64Counter("copium.worker.sent",
		metric.WithDescription("Outbox rows successfully delivered"))
	failed, _ := m.Int64Counter("copium.worker.failed",
		metric.WithDescription("Outbox rows where sender returned an error"))
	dur, _ := m.Float64Histogram("copium.worker.dispatch.duration",
		metric.WithDescription("Time spent in Sender.Send per outbox row"),
		metric.WithUnit("s"))
	return &Worker{
		deps:    d,
		metrics: workerMetrics{claimed: claimed, sent: sent, failed: failed, duration: dur},
	}
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
				slog.Error("worker.process_batch", "error", err)
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
	if len(rows) > 0 {
		slog.Info("worker.claimed", "count", len(rows))
		w.metrics.claimed.Add(ctx, int64(len(rows)))
	}
	for _, r := range rows {
		w.dispatch(ctx, r)
	}
	return len(rows), nil
}

func (w *Worker) dispatch(ctx context.Context, r *models.EmailOutbox) {
	start := w.deps.Clock.Now()
	res, err := w.deps.Sender.Send(ctx, sender.Message{
		To:       r.ToAddress,
		From:     r.FromAddress,
		Subject:  r.Subject,
		BodyHTML: r.BodyHTML,
		BodyText: r.BodyText,
	})
	dur := w.deps.Clock.Now().Sub(start).Seconds()
	provAttr := attribute.String("provider", w.deps.Sender.Name())
	w.metrics.duration.Record(ctx, dur, metric.WithAttributes(provAttr))

	if err != nil {
		nextAttempt := r.Attempts + 1
		next := start.Add(w.Backoff(nextAttempt))
		slog.Error("worker.send_failed",
			"outbox_id", r.ID,
			"to", r.ToAddress,
			"attempt", nextAttempt,
			"error", err,
			"retry_at", next.Format(time.RFC3339))
		w.metrics.failed.Add(ctx, 1, metric.WithAttributes(provAttr))
		if err2 := w.deps.Store.MarkFailureAndReschedule(ctx, r.ID, err.Error(), next); err2 != nil {
			slog.Error("worker.mark_fail_error", "outbox_id", r.ID, "error", err2)
		}
		return
	}
	slog.Info("worker.send_ok",
		"outbox_id", r.ID,
		"to", r.ToAddress,
		"provider", w.deps.Sender.Name(),
		"provider_msg_id", res.ProviderMessageID)
	w.metrics.sent.Add(ctx, 1, metric.WithAttributes(provAttr))
	if err := w.deps.Store.MarkSent(ctx, r.ID, w.deps.Sender.Name(), res.ProviderMessageID, start); err != nil {
		slog.Error("worker.mark_sent_error", "outbox_id", r.ID, "error", err)
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
