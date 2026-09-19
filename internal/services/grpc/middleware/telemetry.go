package middleware

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/semconv/v1.43.0/rpcconv"
	"go.opentelemetry.io/otel/trace"

	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

// The engine's gRPC server was instrumented by otelgrpc's stats handler. Spans and metrics keep
// its instrumentation scope, names, attributes and units so that existing dashboards, alerts
// and trace queries keep matching.
const (
	telemetryScopeName    = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	telemetryScopeVersion = "0.71.0"
)

type telemetryCallKey struct{}

type telemetryCall struct {
	start       time.Time
	metricAttrs []attribute.KeyValue
}

// NewTelemetryInterceptor traces and times every unary and streaming call. It continues the
// caller's trace, and it must be the outermost interceptor so that the span covers the rest of
// the chain and records the error the client receives.
func NewTelemetryInterceptor() connect.Interceptor {
	return newTelemetryInterceptor(newGlobalTelemetryRecorder())
}

func newTelemetryInterceptor(recorder *telemetryRecorder) connect.Interceptor {
	return &handlerInterceptor{
		before: func(ctx context.Context, spec connect.Spec, header http.Header, _ connect.Peer) (context.Context, error) {
			ctx = recorder.start(ctx, spec, header, time.Now())

			setTenantAttribute(ctx, ctx)

			return ctx, nil
		},
		after: func(ctx context.Context, _ connect.Spec, err error) error {
			recorder.end(ctx, err)

			return err
		},
	}
}

type telemetryRecorder struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
	duration   metric.Float64Histogram
}

func newGlobalTelemetryRecorder() *telemetryRecorder {
	return newTelemetryRecorder(otel.GetTracerProvider(), otel.GetMeterProvider(), otel.GetTextMapPropagator())
}

func newTelemetryRecorder(tp trace.TracerProvider, mp metric.MeterProvider, propagator propagation.TextMapPropagator) *telemetryRecorder {
	meter := mp.Meter(
		telemetryScopeName,
		metric.WithInstrumentationVersion(telemetryScopeVersion),
		metric.WithSchemaURL(semconv.SchemaURL),
	)

	duration, err := rpcconv.NewServerCallDuration(
		meter,
		metric.WithExplicitBucketBoundaries(
			0.005, 0.01, 0.025, 0.05, 0.075, 0.1,
			0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10,
		),
	)

	if err != nil {
		otel.Handle(err)
	}

	return &telemetryRecorder{
		tracer:     tp.Tracer(telemetryScopeName, trace.WithInstrumentationVersion(telemetryScopeVersion)),
		propagator: propagator,
		duration:   duration.Inst(),
	}
}

// start begins the server span for a call, with the span propagated in header as its parent.
func (t *telemetryRecorder) start(ctx context.Context, spec connect.Spec, header http.Header, at time.Time) context.Context {
	ctx = t.propagator.Extract(ctx, propagation.HeaderCarrier(header))

	// the span is named after the full method without its leading slash, as is rpc.method
	name := spec.Procedure

	attrs := make([]attribute.KeyValue, 0, 2)

	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
		attrs = append(attrs, semconv.RPCMethod(name))
	}

	attrs = append(attrs, semconv.RPCSystemNameGRPC)

	spanAttrs := append(make([]attribute.KeyValue, 0, len(attrs)+2), attrs...)

	if addr, ok := ctx.Value(http.LocalAddrContextKey).(net.Addr); ok && addr != nil {
		spanAttrs = append(spanAttrs, serverAddrAttrs(addr.String())...)
	}

	ctx, _ = t.tracer.Start(
		trace.ContextWithRemoteSpanContext(ctx, trace.SpanContextFromContext(ctx)),
		name,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(spanAttrs...),
		trace.WithTimestamp(at),
	)

	return context.WithValue(ctx, telemetryCallKey{}, &telemetryCall{start: at, metricAttrs: attrs})
}

// end finishes the span started by start and records the call's duration.
func (t *telemetryRecorder) end(ctx context.Context, err error) {
	call, ok := ctx.Value(telemetryCallKey{}).(*telemetryCall)

	if !ok {
		return
	}

	statusAttr := semconv.RPCResponseStatusCode(canonicalCodeName(0))
	isServerError := false

	if err != nil {
		code := connect.CodeOf(err)
		statusAttr = semconv.RPCResponseStatusCode(canonicalCodeName(code))
		isServerError = isServerErrorCode(code)
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		if isServerError {
			span.SetStatus(codes.Error, errorMessage(err))
		}

		span.SetAttributes(statusAttr)
		span.End()
	}

	if !t.duration.Enabled(ctx) {
		return
	}

	metricAttrs := append(make([]attribute.KeyValue, 0, len(call.metricAttrs)+2), call.metricAttrs...)
	metricAttrs = append(metricAttrs, statusAttr)

	if isServerError {
		metricAttrs = append(metricAttrs, semconv.ErrorTypeKey.String(statusAttr.Value.AsString()))
	}

	elapsed := float64(time.Since(call.start)) / float64(time.Second)

	t.duration.Record(ctx, elapsed, metric.WithAttributeSet(attribute.NewSet(metricAttrs...)))
}

// recordRejected traces and times a call that the request gate turned away. authenticated is the
// context auth produced, or nil when auth itself failed.
func (t *telemetryRecorder) recordRejected(authenticated, ctx context.Context, spec connect.Spec, header http.Header, at time.Time, err error) {
	ctx = t.start(ctx, spec, header, at)

	if authenticated != nil {
		setTenantAttribute(ctx, authenticated)
	}

	t.end(ctx, err)
}

// setTenantAttribute puts the tenant that auth stored in authenticated on the span in ctx.
func setTenantAttribute(ctx, authenticated context.Context) {
	if tenantId, ok := authenticated.Value(analytics.TenantIDKey).(uuid.UUID); ok {
		telemetry.WithAttributes(trace.SpanFromContext(ctx),
			telemetry.AttributeKV{Key: "tenant.id", Value: tenantId},
		)
	}
}

// isServerErrorCode reports whether a code counts as a server-side failure. Codes caused by the
// caller, such as a cancellation or an invalid argument, leave the span status unset.
func isServerErrorCode(code connect.Code) bool {
	switch code {
	case connect.CodeUnknown, connect.CodeDeadlineExceeded, connect.CodeUnimplemented,
		connect.CodeInternal, connect.CodeUnavailable, connect.CodeDataLoss:
		return true
	default:
		return false
	}
}

func serverAddrAttrs(hostport string) []attribute.KeyValue {
	host, portStr, err := net.SplitHostPort(hostport)

	if err != nil {
		return []attribute.KeyValue{semconv.ServerAddress(hostport)}
	}

	port, err := strconv.Atoi(portStr)

	if err != nil {
		return []attribute.KeyValue{semconv.ServerAddress(host)}
	}

	return []attribute.KeyValue{semconv.ServerAddress(host), semconv.ServerPort(port)}
}

// canonicalCodeName is the value of rpc.response.status_code: the code's name in the gRPC
// specification (NOT_FOUND, not NotFound). Zero is OK.
func canonicalCodeName(code connect.Code) string {
	switch code {
	case 0:
		return "OK"
	case connect.CodeCanceled:
		return "CANCELLED"
	case connect.CodeUnknown:
		return "UNKNOWN"
	case connect.CodeInvalidArgument:
		return "INVALID_ARGUMENT"
	case connect.CodeDeadlineExceeded:
		return "DEADLINE_EXCEEDED"
	case connect.CodeNotFound:
		return "NOT_FOUND"
	case connect.CodeAlreadyExists:
		return "ALREADY_EXISTS"
	case connect.CodePermissionDenied:
		return "PERMISSION_DENIED"
	case connect.CodeResourceExhausted:
		return "RESOURCE_EXHAUSTED"
	case connect.CodeFailedPrecondition:
		return "FAILED_PRECONDITION"
	case connect.CodeAborted:
		return "ABORTED"
	case connect.CodeOutOfRange:
		return "OUT_OF_RANGE"
	case connect.CodeUnimplemented:
		return "UNIMPLEMENTED"
	case connect.CodeInternal:
		return "INTERNAL"
	case connect.CodeUnavailable:
		return "UNAVAILABLE"
	case connect.CodeDataLoss:
		return "DATA_LOSS"
	case connect.CodeUnauthenticated:
		return "UNAUTHENTICATED"
	default:
		return "CODE(" + strconv.FormatInt(int64(code), 10) + ")"
	}
}
