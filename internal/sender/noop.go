package sender

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// Noop records every Send call. Useful for tests and for local dev when no
// real provider is configured.
type Noop struct {
	mu   sync.Mutex
	sent []Message
}

func NewNoop() *Noop { return &Noop{} }

func (n *Noop) Name() string { return "noop" }

func (n *Noop) Send(_ context.Context, m Message) (SendResult, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, m)
	return SendResult{ProviderMessageID: "noop:" + uuid.NewString()}, nil
}

// Sent returns a snapshot of every message sent so far.
func (n *Noop) Sent() []Message {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make([]Message, len(n.sent))
	copy(out, n.sent)
	return out
}

// Reset clears the recorded message history.
func (n *Noop) Reset() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = nil
}
