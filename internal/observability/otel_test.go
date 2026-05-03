package observability_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/hoshina-dev/copium/internal/config"
	"github.com/hoshina-dev/copium/internal/observability"
)

func TestSetup_DisabledReturnsNoopProvider(t *testing.T) {
	p, err := observability.Setup(context.Background(), config.OtelConfig{Enabled: false})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if p == nil {
		t.Fatal("nil provider")
	}
	if _, ok := p.TracerProvider().(noop.TracerProvider); !ok {
		t.Errorf("disabled config must return noop tracer; got %T", p.TracerProvider())
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("shutdown noop err: %v", err)
	}
}

func TestSetup_EnabledWithoutEndpoint_NoopFallback(t *testing.T) {
	p, err := observability.Setup(context.Background(), config.OtelConfig{
		Enabled:      true,
		ServiceName:  "test",
		OTLPEndpoint: "",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, ok := p.TracerProvider().(noop.TracerProvider); !ok {
		t.Errorf("enabled but no endpoint -> noop; got %T", p.TracerProvider())
	}
}
