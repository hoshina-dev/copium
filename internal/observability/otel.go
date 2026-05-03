// Package observability bootstraps OpenTelemetry. When OTEL_ENABLED=false (or
// no endpoint is set), Setup returns a noop provider so the rest of the code
// never has to branch.
package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/hoshina-dev/copium/internal/config"
)

type Provider struct {
	tracerProvider trace.TracerProvider
	shutdown       func(context.Context) error
}

func (p *Provider) TracerProvider() trace.TracerProvider { return p.tracerProvider }

func (p *Provider) Shutdown(ctx context.Context) error {
	if p.shutdown == nil {
		return nil
	}
	return p.shutdown(ctx)
}

// Setup builds a Provider. Returns a noop provider when disabled or when no
// endpoint is configured, so callers don't need to branch on cfg.Enabled.
func Setup(ctx context.Context, cfg config.OtelConfig) (*Provider, error) {
	if !cfg.Enabled || cfg.OTLPEndpoint == "" {
		return &Provider{tracerProvider: noop.NewTracerProvider()}, nil
	}

	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint)}
	if cfg.OTLPInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(opts...))
	if err != nil {
		return nil, fmt.Errorf("otlp exporter: %w", err)
	}
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(cfg.ServiceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return &Provider{
		tracerProvider: tp,
		shutdown: func(c context.Context) error {
			ctx, cancel := context.WithTimeout(c, 5*time.Second)
			defer cancel()
			return tp.Shutdown(ctx)
		},
	}, nil
}
