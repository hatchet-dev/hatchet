package telemetry

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/credentials"
)

type TracerOpts struct {
	ServiceName  string
	CollectorURL string
	Insecure     bool
	TraceIdRatio string
}

func InitTracer(opts *TracerOpts) (func(context.Context) error, error) {
	if opts.CollectorURL == "" {
		// no-op
		return func(context.Context) error {
			return nil
		}, nil
	}

	var secureOption otlptracegrpc.Option

	if !opts.Insecure {
		secureOption = otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
	} else {
		secureOption = otlptracegrpc.WithInsecure()
	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			secureOption,
			otlptracegrpc.WithEndpoint(opts.CollectorURL),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", opts.ServiceName),
			attribute.String("library.language", "go"),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to set resources: %w", err)
	}

	var traceIdRatio float64 = 1

	if opts.TraceIdRatio != "" {
		traceIdRatio, err = strconv.ParseFloat(opts.TraceIdRatio, 64)

		if err != nil {
			return nil, fmt.Errorf("failed to parse traceIdRatio: %w", err)
		}
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(traceIdRatio)),
			sdktrace.WithBatcher(
				exporter,
				sdktrace.WithMaxQueueSize(sdktrace.DefaultMaxQueueSize*10),
				sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize*10),
			),
			sdktrace.WithResource(resources),
		),
	)

	return exporter.Shutdown, nil
}

func NewSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	ctx, span := otel.Tracer("").Start(ctx, prefixSpanKey(name))
	return ctx, span
}

func NewSpanWithCarrier(ctx context.Context, name string, carrier map[string]string) (context.Context, trace.Span) {
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	otelCarrier := propagation.MapCarrier(carrier)
	parentCtx := propagator.Extract(ctx, otelCarrier)

	ctx, span := otel.Tracer("").Start(parentCtx, prefixSpanKey(name))
	return ctx, span
}

func GetCarrier(ctx context.Context) map[string]string {
	propgator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	// Serialize the context into carrier
	carrier := propagation.MapCarrier{}
	propgator.Inject(ctx, carrier)

	return carrier
}

type AttributeKey string

// AttributeKV is a wrapper for otel attributes KV
type AttributeKV struct {
	Key   AttributeKey
	Value any
}

func WithAttributes(span trace.Span, attrs ...AttributeKV) {
	for _, attr := range attrs {
		if attr.Key != "" {
			switch val := attr.Value.(type) {
			case uuid.UUID:
				span.SetAttributes(attribute.String(prefixSpanKey(string(attr.Key)), val.String()))
			case string:
				span.SetAttributes(attribute.String(prefixSpanKey(string(attr.Key)), val))
			case []string:
				span.SetAttributes(attribute.String(prefixSpanKey(string(attr.Key)), strings.Join(val, ", ")))
			case int:
				span.SetAttributes(attribute.Int(prefixSpanKey(string(attr.Key)), val))
			case int64:
				span.SetAttributes(attribute.Int64(prefixSpanKey(string(attr.Key)), val))
			case int32:
				span.SetAttributes(attribute.Int64(prefixSpanKey(string(attr.Key)), int64(val)))
			case uint:
				span.SetAttributes(attribute.Int(prefixSpanKey(string(attr.Key)), int(val))) // nolint: gosec
			case float64:
				span.SetAttributes(attribute.Float64(prefixSpanKey(string(attr.Key)), val))
			case bool:
				span.SetAttributes(attribute.Bool(prefixSpanKey(string(attr.Key)), val))
			case time.Time:
				span.SetAttributes(attribute.String(prefixSpanKey(string(attr.Key)), val.String()))
				zone, offset := val.Zone()
				span.SetAttributes(attribute.String(prefixSpanKey(fmt.Sprintf("%s-timezone", string(attr.Key))), zone))
				span.SetAttributes(attribute.Int(prefixSpanKey(fmt.Sprintf("%s-offset", string(attr.Key))), offset))
			}
		}
	}
}

func prefixSpanKey(name string) string {
	return fmt.Sprintf("hatchet.run/%s", name)
}
