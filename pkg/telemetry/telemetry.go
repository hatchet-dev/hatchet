package telemetry

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/credentials"
)

type TracerOpts struct {
	ServiceName   string
	CollectorURL  string
	Insecure      bool
	TraceIdRatio  string
	CollectorAuth string
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
			otlptracegrpc.WithHeaders(map[string]string{
				"Authorization": opts.CollectorAuth,
			}),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	resourceAttrs := []attribute.KeyValue{
		attribute.String("service.name", opts.ServiceName),
		attribute.String("library.language", "go"),
	}

	// Add Kubernetes pod information if available
	if podName := os.Getenv("K8S_POD_NAME"); podName != "" {
		resourceAttrs = append(resourceAttrs, attribute.String("k8s.pod.name", podName))
	}
	if podNamespace := os.Getenv("K8S_POD_NAMESPACE"); podNamespace != "" {
		resourceAttrs = append(resourceAttrs, attribute.String("k8s.namespace.name", podNamespace))
	}

	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(resourceAttrs...),
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

func InitMeter(opts *TracerOpts) (func(context.Context) error, error) {
	if opts.CollectorURL == "" {
		// no-op
		return func(context.Context) error {
			return nil
		}, nil
	}

	var secureOption otlpmetricgrpc.Option

	if !opts.Insecure {
		secureOption = otlpmetricgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
	} else {
		otlpmetricgrpc.WithInsecure()
		secureOption = otlpmetricgrpc.WithInsecure()
	}

	exporter, err := otlpmetricgrpc.New(
		context.Background(),
		secureOption,
		otlpmetricgrpc.WithEndpoint(opts.CollectorURL),
		otlpmetricgrpc.WithHeaders(map[string]string{
			"Authorization": opts.CollectorAuth,
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	resourceAttrs := []attribute.KeyValue{
		attribute.String("service.name", opts.ServiceName),
		attribute.String("library.language", "go"),
	}

	// Add Kubernetes pod information if available
	if podName := os.Getenv("K8S_POD_NAME"); podName != "" {
		resourceAttrs = append(resourceAttrs, attribute.String("k8s.pod.name", podName))
	}
	if podNamespace := os.Getenv("K8S_POD_NAMESPACE"); podNamespace != "" {
		resourceAttrs = append(resourceAttrs, attribute.String("k8s.namespace.name", podNamespace))
	}

	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(resourceAttrs...),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to set resources: %w", err)
	}

	otel.SetMeterProvider(
		metric.NewMeterProvider(
			metric.WithResource(resources),
			metric.WithReader(
				metric.NewPeriodicReader(
					exporter,
					metric.WithInterval(3*time.Second),
				),
			),
			metric.WithResource(resources),
		),
	)

	return nil, nil
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

// CollectUniqueTenantIDs extracts unique tenant IDs from a slice of items.
// The extractFn should return the tenant ID string for each item.
func CollectUniqueTenantIDs[T any](items []T, extractFn func(T) string) []string {
	if len(items) == 0 {
		return nil
	}

	tenantIds := make(map[string]bool)
	for _, item := range items {
		tenantIds[extractFn(item)] = true
	}

	uniqueTenantIds := make([]string, 0, len(tenantIds))
	for tid := range tenantIds {
		uniqueTenantIds = append(uniqueTenantIds, tid)
	}

	return uniqueTenantIds
}
