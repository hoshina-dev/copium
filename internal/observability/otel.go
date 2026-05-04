// Package observability bootstraps OpenTelemetry for traces, metrics, and logs.
//
// When OTEL_ENABLED=false (or no endpoint is set), Setup returns a noop-backed
// Provider so the rest of the code never has to branch. When enabled, it wires
// up OTLP/gRPC exporters for all three signals and installs them as the global
// providers so the convenience helpers (otelfiber, otelslog, otel.Meter) work
// without further plumbing.
package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otellog "go.opentelemetry.io/otel/log"
	logglobal "go.opentelemetry.io/otel/log/global"
	lognoop "go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"go.opentelemetry.io/contrib/bridges/otelslog"

	"github.com/hoshina-dev/copium/internal/config"
)

// Provider holds every signal's shutdown hook. The tracer provider is exposed
// mainly for tests; production code reaches for otel.GetTracerProvider(),
// otel.Meter(), and slog.Default() which are installed globally below.
type Provider struct {
	tracerProvider trace.TracerProvider
	loggerProvider otellog.LoggerProvider
	shutdowns      []func(context.Context) error
}

func (p *Provider) TracerProvider() trace.TracerProvider { return p.tracerProvider }
func (p *Provider) LoggerProvider() otellog.LoggerProvider {
	return p.loggerProvider
}

// Shutdown flushes and closes every registered exporter. Errors are joined so
// one bad exporter doesn't hide the others.
func (p *Provider) Shutdown(ctx context.Context) error {
	var firstErr error
	for _, fn := range p.shutdowns {
		if fn == nil {
			continue
		}
		if err := fn(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Setup builds a Provider. When disabled (or no endpoint) we still install a
// stdout slog handler so operators keep getting console logs, and we return
// noop tracer/logger providers so call sites never need to branch.
func Setup(ctx context.Context, cfg config.OtelConfig) (*Provider, error) {
	if !cfg.Enabled || cfg.OTLPEndpoint == "" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
		return &Provider{
			tracerProvider: noop.NewTracerProvider(),
			loggerProvider: lognoop.NewLoggerProvider(),
		}, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(cfg.ServiceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	p := &Provider{}

	// --- traces ---
	traceExp, err := otlptrace.New(ctx, otlptracegrpc.NewClient(traceOpts(cfg)...))
	if err != nil {
		return nil, fmt.Errorf("otlp trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	p.tracerProvider = tp
	p.shutdowns = append(p.shutdowns, withTimeout(tp.Shutdown))

	// --- metrics ---
	metricExp, err := otlpmetricgrpc.New(ctx, metricOpts(cfg)...)
	if err != nil {
		return nil, fmt.Errorf("otlp metric exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	p.shutdowns = append(p.shutdowns, withTimeout(mp.Shutdown))

	// --- logs ---
	// Install the OTLP log exporter AND keep a stdout handler so the console
	// stays useful during prod incident triage (kubectl logs etc.). The slog
	// default routes to both handlers via tee.
	logExp, err := otlploggrpc.New(ctx, logOpts(cfg)...)
	if err != nil {
		return nil, fmt.Errorf("otlp log exporter: %w", err)
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
		sdklog.WithResource(res),
	)
	logglobal.SetLoggerProvider(lp)
	p.loggerProvider = lp
	p.shutdowns = append(p.shutdowns, withTimeout(lp.Shutdown))

	otelHandler := otelslog.NewHandler(cfg.ServiceName, otelslog.WithLoggerProvider(lp))
	stdoutHandler := slog.NewTextHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(teeHandler{handlers: []slog.Handler{stdoutHandler, otelHandler}}))

	return p, nil
}

func traceOpts(cfg config.OtelConfig) []otlptracegrpc.Option {
	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint)}
	if cfg.OTLPInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	return opts
}

func metricOpts(cfg config.OtelConfig) []otlpmetricgrpc.Option {
	opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint)}
	if cfg.OTLPInsecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}
	return opts
}

func logOpts(cfg config.OtelConfig) []otlploggrpc.Option {
	opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(cfg.OTLPEndpoint)}
	if cfg.OTLPInsecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}
	return opts
}

func withTimeout(fn func(context.Context) error) func(context.Context) error {
	return func(c context.Context) error {
		ctx, cancel := context.WithTimeout(c, 5*time.Second)
		defer cancel()
		return fn(ctx)
	}
}

// teeHandler fans slog records out to multiple handlers. We only need this
// because we want simultaneous stdout + OTLP log shipping without writing a
// bespoke handler per signal.
type teeHandler struct {
	handlers []slog.Handler
}

func (t teeHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	for _, h := range t.handlers {
		if h.Enabled(ctx, lvl) {
			return true
		}
	}
	return false
}

func (t teeHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range t.handlers {
		if err := h.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (t teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, len(t.handlers))
	for i, h := range t.handlers {
		next[i] = h.WithAttrs(attrs)
	}
	return teeHandler{handlers: next}
}

func (t teeHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, len(t.handlers))
	for i, h := range t.handlers {
		next[i] = h.WithGroup(name)
	}
	return teeHandler{handlers: next}
}
