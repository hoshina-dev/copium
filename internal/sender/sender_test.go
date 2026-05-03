package sender_test

import (
	"context"
	"testing"

	"github.com/hoshina-dev/copium/internal/sender"
)

func TestNoop_RecordsAndReturnsID(t *testing.T) {
	n := sender.NewNoop()
	if n.Name() != "noop" {
		t.Errorf("name=%q", n.Name())
	}
	res, err := n.Send(context.Background(), sender.Message{
		To:       "a@b.com",
		From:     "x@y.com",
		Subject:  "hi",
		BodyHTML: "<p>hi</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ProviderMessageID == "" {
		t.Fatal("expected non-empty provider id")
	}
	sent := n.Sent()
	if len(sent) != 1 {
		t.Fatalf("len(Sent)=%d", len(sent))
	}
	if sent[0].To != "a@b.com" {
		t.Errorf("recorded To=%q", sent[0].To)
	}
}

func TestNoop_MultipleSends(t *testing.T) {
	n := sender.NewNoop()
	for i := 0; i < 3; i++ {
		_, err := n.Send(context.Background(), sender.Message{To: "a@b.com"})
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := len(n.Sent()); got != 3 {
		t.Errorf("len=%d want 3", got)
	}
}

func TestNewFromConfig_Noop(t *testing.T) {
	s, err := sender.NewFromConfig(sender.FromConfig{Provider: "noop"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Name() != "noop" {
		t.Errorf("got %q", s.Name())
	}
}

func TestNewFromConfig_UnknownProvider(t *testing.T) {
	_, err := sender.NewFromConfig(sender.FromConfig{Provider: "carrier-pigeon"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewFromConfig_SMTPRequiresHost(t *testing.T) {
	_, err := sender.NewFromConfig(sender.FromConfig{Provider: "smtp"})
	if err == nil {
		t.Fatal("expected error: smtp host required")
	}
}

func TestNewFromConfig_SMTP(t *testing.T) {
	s, err := sender.NewFromConfig(sender.FromConfig{
		Provider: "smtp",
		SMTP:     sender.SMTP{Host: "localhost", Port: 1025},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Name() != "smtp" {
		t.Errorf("got %q", s.Name())
	}
}

func TestNewFromConfig_StubsForSESAndSendGrid(t *testing.T) {
	for _, p := range []string{"ses", "sendgrid"} {
		p := p
		t.Run(p, func(t *testing.T) {
			s, err := sender.NewFromConfig(sender.FromConfig{Provider: p})
			if err != nil {
				t.Fatal(err)
			}
			if s.Name() != p {
				t.Errorf("got %q", s.Name())
			}
			// Stub must return a clear "not implemented" error so the operator
			// knows their config landed on a placeholder.
			_, sendErr := s.Send(context.Background(), sender.Message{To: "a@b"})
			if sendErr == nil {
				t.Errorf("stub sender %s must error until implemented", p)
			}
		})
	}
}
