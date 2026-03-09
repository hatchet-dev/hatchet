package opentelemetry

import (
	"context"
	"crypto/tls"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// newHatchetExporter creates a gRPC OTLP trace exporter that sends spans
// to the Hatchet engine's collector endpoint.
func newHatchetExporter(endpoint, token string, insecureConn bool) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithHeaders(map[string]string{
			"authorization": "Bearer " + token,
		}),
	}

	if insecureConn {
		opts = append(opts, otlptracegrpc.WithInsecure())
	} else {
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12}))) //nolint:gosec // G402
	}

	return otlptracegrpc.New(context.Background(), opts...)
}
