// Package opentelemetry provides OpenTelemetry instrumentation for the Hatchet Go SDK.
//
// It automatically creates spans for each step run, propagates hatchet.* attributes
// to all child spans, supports W3C traceparent propagation, and optionally sends
// traces to the Hatchet engine's OTLP collector.
//
// Basic usage (sends traces to Hatchet by default):
//
//	instrumentor, err := opentelemetry.NewInstrumentor()
//	worker.Use(instrumentor.Middleware())
package opentelemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/hatchet-dev/hatchet/pkg/client/loader"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// Instrumentor sets up OpenTelemetry tracing for Hatchet workers.
type Instrumentor struct {
	tracerProvider trace.TracerProvider
	tracer         trace.Tracer
	opts           instrumentorOptions
}

type instrumentorOptions struct {
	tracerProvider *sdktrace.TracerProvider

	enableCollector bool
}

// InstrumentorOption configures the Instrumentor.
type InstrumentorOption func(*instrumentorOptions)

// WithTracerProvider sets a custom TracerProvider. If not set, a new one is created.
// The provider must be an SDK TracerProvider to support adding span processors.
func WithTracerProvider(tp *sdktrace.TracerProvider) InstrumentorOption {
	return func(o *instrumentorOptions) {
		o.tracerProvider = tp
	}
}

// DisableHatchetCollector disables sending traces to the Hatchet engine's OTLP collector.
// By default, the collector is enabled and connection settings (endpoint, token, TLS)
// are automatically loaded from the same environment variables used by the Hatchet client
// (HATCHET_CLIENT_HOST_PORT, HATCHET_CLIENT_TOKEN, HATCHET_CLIENT_TLS_STRATEGY).
func DisableHatchetCollector() InstrumentorOption {
	return func(o *instrumentorOptions) {
		o.enableCollector = false
	}
}

// EnableHatchetCollector enables sending traces to the Hatchet engine's OTLP collector.
// This is the default behavior; this option exists for explicitness.
//
// Deprecated: The collector is enabled by default. Use DisableHatchetCollector() to opt out.
func EnableHatchetCollector() InstrumentorOption {
	return func(o *instrumentorOptions) {
		o.enableCollector = true
	}
}

// NewInstrumentor creates a new HatchetInstrumentor.
func NewInstrumentor(opts ...InstrumentorOption) (*Instrumentor, error) {
	o := &instrumentorOptions{
		enableCollector: true,
	}
	for _, opt := range opts {
		opt(o)
	}

	// Set up TracerProvider
	tp := o.tracerProvider
	if tp == nil {
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("hatchet-worker"),
			)),
		)
	}

	// Add Hatchet collector exporter if enabled
	if o.enableCollector {
		cfgFile, err := loader.LoadClientConfigFile()
		if err != nil {
			return nil, fmt.Errorf("failed to load client config for OTel collector: %w", err)
		}

		clientCfg, err := loader.GetClientConfigFromConfigFile(nil, cfgFile)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve client config for OTel collector: %w", err)
		}

		insecure := clientCfg.TLSConfig == nil

		exporter, err := newHatchetExporter(clientCfg.GRPCBroadcastAddress, clientCfg.Token, insecure)
		if err != nil {
			return nil, fmt.Errorf("failed to create Hatchet OTLP exporter: %w", err)
		}
		bsp := sdktrace.NewBatchSpanProcessor(exporter)
		tp.RegisterSpanProcessor(NewHatchetAttributeSpanProcessor(bsp))
	}

	// Set as global provider
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("github.com/hatchet-dev/hatchet/sdks/go/opentelemetry")

	return &Instrumentor{
		tracerProvider: tp,
		tracer:         tracer,
		opts:           *o,
	}, nil
}

// Middleware returns the OTel middleware that should be registered on the worker.
//
//nolint:staticcheck // SA1019: worker.MiddlewareFunc is deprecated but still used internally
func (i *Instrumentor) Middleware() worker.MiddlewareFunc {
	return NewMiddleware(i.tracer)
}

// TracerProvider returns the TracerProvider used by the instrumentor.
func (i *Instrumentor) TracerProvider() trace.TracerProvider {
	return i.tracerProvider
}

// Shutdown flushes any remaining spans and shuts down the TracerProvider.
// Call this before your application exits to ensure all spans are exported.
func (i *Instrumentor) Shutdown(ctx context.Context) error {
	if tp, ok := i.tracerProvider.(*sdktrace.TracerProvider); ok {
		return tp.Shutdown(ctx)
	}
	return nil
}
