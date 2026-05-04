//go:build integration

package sender_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hoshina-dev/copium/internal/sender"
)

// TestSMTPSender_AgainstMailhog spins up Mailhog, sends a real message, and
// asserts via Mailhog's HTTP API that it arrived.
func TestSMTPSender_AgainstMailhog(t *testing.T) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "mailhog/mailhog:latest",
		ExposedPorts: []string{"1025/tcp", "8025/tcp"},
		WaitingFor:   wait.ForListeningPort("1025/tcp").WithStartupTimeout(30 * time.Second),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Skipf("docker unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.Terminate(ctx) })

	host, _ := c.Host(ctx)
	smtpPort, _ := c.MappedPort(ctx, "1025/tcp")
	httpPort, _ := c.MappedPort(ctx, "8025/tcp")

	s := sender.NewSMTP(sender.SMTP{Host: host, Port: smtpPort.Int()})

	res, err := s.Send(ctx, sender.Message{
		To:       "rcpt@example.com",
		From:     "from@example.com",
		Subject:  "hi from copium",
		BodyHTML: "<p>hello</p>",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if res.ProviderMessageID == "" {
		t.Fatal("missing provider id")
	}

	mhURL := fmt.Sprintf("http://%s:%d/api/v2/messages", host, httpPort.Int())
	resp, err := http.Get(mhURL)
	if err != nil {
		t.Fatalf("mailhog GET: %v", err)
	}
	defer resp.Body.Close()
	var body struct {
		Total int `json:"total"`
		Items []struct {
			Content struct {
				Headers map[string][]string `json:"Headers"`
			} `json:"Content"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Total != 1 {
		t.Fatalf("mailhog total=%d want 1", body.Total)
	}
	if subj := body.Items[0].Content.Headers["Subject"]; len(subj) == 0 || !strings.Contains(subj[0], "hi from copium") {
		t.Errorf("subject not delivered: %v", subj)
	}
}
