package opentelemetry

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const retryAfter = 5 * time.Minute

type hatchetExporter struct {
	retryAt    time.Time
	inner      sdktrace.SpanExporter
	retryAfter time.Duration
	mu         sync.Mutex
}

func (e *hatchetExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	if !e.retryAt.IsZero() && time.Now().Before(e.retryAt) {
		e.mu.Unlock()
		return nil
	}
	e.mu.Unlock()

	err := e.inner.ExportSpans(ctx, spans)
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.Unimplemented {
			e.mu.Lock()
			e.retryAt = time.Now().Add(e.retryAfter)
			e.mu.Unlock()
			return nil
		}
	}

	e.mu.Lock()
	e.retryAt = time.Time{}
	e.mu.Unlock()

	return err
}

func (e *hatchetExporter) Shutdown(ctx context.Context) error {
	return e.inner.Shutdown(ctx)
}

func newHatchetExporter(endpoint, token string, tlsCfg *tls.Config) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithHeaders(map[string]string{
			"authorization": "Bearer " + token,
		}),
	}

	if tlsCfg == nil {
		opts = append(opts, otlptracegrpc.WithInsecure())
	} else {
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsCfg)))
	}

	inner, err := otlptracegrpc.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	return &hatchetExporter{inner: inner, retryAfter: retryAfter}, nil
}
