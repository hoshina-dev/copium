// Package sender hosts the Sender interface and its concrete implementations
// (noop, smtp, ses, sendgrid). The composition root picks one via NewFromConfig
// and injects it into services/worker behind their consumer-owned interface.
package sender

import (
	"context"
	"errors"
	"fmt"
)

type Message struct {
	To       string
	From     string
	Subject  string
	BodyHTML string
	BodyText string
	Headers  map[string]string
}

type SendResult struct {
	ProviderMessageID string
}

type Sender interface {
	Name() string
	Send(ctx context.Context, m Message) (SendResult, error)
}

// FromConfig is the subset of config a sender adapter needs. We avoid
// importing internal/config so that tests don't need a full Config struct.
type FromConfig struct {
	Provider string
	SMTP     SMTP
}

type SMTP struct {
	Host     string
	Port     int
	User     string
	Password string
	UseTLS   bool
}

func NewFromConfig(c FromConfig) (Sender, error) {
	switch c.Provider {
	case "noop", "":
		return NewNoop(), nil
	case "smtp":
		if c.SMTP.Host == "" {
			return nil, errors.New("sender: smtp host required")
		}
		return NewSMTP(c.SMTP), nil
	case "ses":
		return &stubSender{name: "ses"}, nil
	case "sendgrid":
		return &stubSender{name: "sendgrid"}, nil
	default:
		return nil, fmt.Errorf("sender: unknown provider %q", c.Provider)
	}
}

// stubSender is the placeholder for providers we declare in config but
// haven't implemented yet. It refuses to send so misconfiguration surfaces
// loudly instead of silently dropping mail.
type stubSender struct{ name string }

func (s *stubSender) Name() string { return s.name }
func (s *stubSender) Send(_ context.Context, _ Message) (SendResult, error) {
	return SendResult{}, fmt.Errorf("sender %q is a stub; implement before enabling", s.name)
}
