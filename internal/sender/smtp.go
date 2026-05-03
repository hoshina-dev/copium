package sender

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	mail "github.com/wneessen/go-mail"
)

// SMTPSender uses go-mail to talk to any SMTP server (Mailhog locally, real
// SMTP in production). TLS is opt-in.
type SMTPSender struct {
	cfg SMTP
}

func NewSMTP(cfg SMTP) *SMTPSender { return &SMTPSender{cfg: cfg} }

func (s *SMTPSender) Name() string { return "smtp" }

func (s *SMTPSender) Send(ctx context.Context, m Message) (SendResult, error) {
	msg := mail.NewMsg()
	if err := msg.From(m.From); err != nil {
		return SendResult{}, fmt.Errorf("smtp from: %w", err)
	}
	if err := msg.To(m.To); err != nil {
		return SendResult{}, fmt.Errorf("smtp to: %w", err)
	}
	msg.Subject(m.Subject)
	if m.BodyText != "" {
		msg.SetBodyString(mail.TypeTextPlain, m.BodyText)
		if m.BodyHTML != "" {
			msg.AddAlternativeString(mail.TypeTextHTML, m.BodyHTML)
		}
	} else {
		msg.SetBodyString(mail.TypeTextHTML, m.BodyHTML)
	}
	for k, v := range m.Headers {
		msg.SetGenHeader(mail.Header(k), v)
	}
	msgID := "smtp:" + uuid.NewString()
	msg.SetMessageIDWithValue(msgID)

	opts := []mail.Option{mail.WithPort(s.cfg.Port)}
	if s.cfg.User != "" {
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthLogin),
			mail.WithUsername(s.cfg.User),
			mail.WithPassword(s.cfg.Password),
		)
	}
	if !s.cfg.UseTLS {
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	} else {
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	}

	client, err := mail.NewClient(s.cfg.Host, opts...)
	if err != nil {
		return SendResult{}, fmt.Errorf("smtp client: %w", err)
	}
	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return SendResult{}, fmt.Errorf("smtp send: %w", err)
	}
	return SendResult{ProviderMessageID: msgID}, nil
}
