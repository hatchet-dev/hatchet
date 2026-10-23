package middleware

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

const traceparent = "00-00000000000000000000000000000001-0000000000000002-01"

func (ts *testServer) serverAddrAttrs(t *testing.T) []attribute.KeyValue {
	t.Helper()

	u, err := url.Parse(ts.url)
	require.NoError(t, err)

	host, portStr, err := net.SplitHostPort(u.Host)
	require.NoError(t, err)

	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	return []attribute.KeyValue{attribute.String("server.address", host), attribute.Int("server.port", port)}
}

func (ts *testServer) durationPoints(t *testing.T) []metricdata.HistogramDataPoint[float64] {
	t.Helper()

	var collected metricdata.ResourceMetrics
	require.NoError(t, ts.metrics.Collect(context.Background(), &collected))

	require.Len(t, collected.ScopeMetrics, 1)
	assert.Equal(t, "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc", collected.ScopeMetrics[0].Scope.Name)
	assert.Equal(t, "0.71.0", collected.ScopeMetrics[0].Scope.Version)
	assert.Equal(t, "https://opentelemetry.io/schemas/1.43.0", collected.ScopeMetrics[0].Scope.SchemaURL)

	// one instrument, as with otelgrpc's defaults
	require.Len(t, collected.ScopeMetrics[0].Metrics, 1)

	m := collected.ScopeMetrics[0].Metrics[0]
	assert.Equal(t, "rpc.server.call.duration", m.Name)
	assert.Equal(t, "s", m.Unit)

	histogram, ok := m.Data.(metricdata.Histogram[float64])
	require.True(t, ok, "data was %T", m.Data)

	return histogram.DataPoints
}

// The golden values are what otelgrpc v0.71.0's server stats handler records for the same call
// with its defaults (the stable RPC conventions).
func TestTelemetryMatchesOtelGRPCForAnOKCall(t *testing.T) {
	ts := startTestServer(t, nil, 1000, 1000)

	_, err := ts.client().Register(authContext(testToken, "traceparent", traceparent), &contracts.WorkerRegisterRequest{WorkerName: "w"})
	require.NoError(t, err)

	spans := ts.spans.Ended()
	require.Len(t, spans, 1)

	span := spans[0]

	assert.Equal(t, "Dispatcher/Register", span.Name())
	assert.Equal(t, trace.SpanKindServer, span.SpanKind())
	assert.Equal(t, "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc", span.InstrumentationScope().Name)
	assert.Equal(t, "0.71.0", span.InstrumentationScope().Version)

	// the caller's span is the parent, not a link
	assert.Equal(t, "00000000000000000000000000000001", span.SpanContext().TraceID().String())
	assert.Equal(t, "0000000000000002", span.Parent().SpanID().String())
	assert.True(t, span.Parent().IsRemote())
	assert.Empty(t, span.Links())

	assert.Empty(t, span.Events())
	assert.Equal(t, sdktrace.Status{Code: codes.Unset}, span.Status())

	assert.ElementsMatch(t, append([]attribute.KeyValue{
		attribute.String("rpc.method", "Dispatcher/Register"),
		attribute.String("rpc.system.name", "grpc"),
		attribute.String("rpc.response.status_code", "OK"),
		// the engine's own attribute, on top of otelgrpc's
		attribute.String("hatchet.run/tenant.id", testTenantID.String()),
	}, ts.serverAddrAttrs(t)...), span.Attributes())

	points := ts.durationPoints(t)
	require.Len(t, points, 1)
	assert.Equal(t, uint64(1), points[0].Count)
	assert.Greater(t, points[0].Sum, 0.0)
	assert.Less(t, points[0].Sum, 5.0, "the unit is seconds")
	assert.Equal(t, []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10}, points[0].Bounds)
	assert.ElementsMatch(t, []attribute.KeyValue{
		attribute.String("rpc.method", "Dispatcher/Register"),
		attribute.String("rpc.system.name", "grpc"),
		attribute.String("rpc.response.status_code", "OK"),
	}, points[0].Attributes.ToSlice())
}

// Only codes that mean the server failed mark the span as an error and carry error.type.
func TestTelemetryStatusFollowsTheServerRule(t *testing.T) {
	ts := startTestServer(t, nil, 1000, 1000)

	_, err := ts.client().Register(authContext(testToken), &contracts.WorkerRegisterRequest{WorkerName: "invalid"})
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = ts.client().Register(authContext(testToken), &contracts.WorkerRegisterRequest{WorkerName: "internal"})
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))

	spans := ts.spans.Ended()
	require.Len(t, spans, 2)

	assert.Equal(t, sdktrace.Status{Code: codes.Unset}, spans[0].Status())
	assert.Contains(t, spans[0].Attributes(), attribute.String("rpc.response.status_code", "INVALID_ARGUMENT"))

	assert.Equal(t, sdktrace.Status{Code: codes.Error, Description: "An internal error occurred."}, spans[1].Status())
	assert.Contains(t, spans[1].Attributes(), attribute.String("rpc.response.status_code", "INTERNAL"))

	for _, point := range ts.durationPoints(t) {
		status, _ := point.Attributes.Value("rpc.response.status_code")
		errorType, hasErrorType := point.Attributes.Value("error.type")

		switch status.AsString() {
		case "INVALID_ARGUMENT":
			assert.False(t, hasErrorType)
		case "INTERNAL":
			assert.Equal(t, "INTERNAL", errorType.AsString())
		default:
			t.Fatalf("unexpected status %s", status.AsString())
		}
	}
}

func TestTelemetryCoversStreams(t *testing.T) {
	ts := startTestServer(t, nil, 1000, 1000)

	stream, err := ts.client().ListenV2(authContext(testToken, "traceparent", traceparent), &contracts.WorkerListenRequest{WorkerId: "w"})
	require.NoError(t, err)
	require.True(t, stream.Receive())
	require.False(t, stream.Receive())
	require.NoError(t, stream.Err())
	require.NoError(t, stream.Close())

	require.Eventually(t, func() bool { return len(ts.spans.Ended()) == 1 }, 5*time.Second, 10*time.Millisecond)

	span := ts.spans.Ended()[0]

	assert.Equal(t, "Dispatcher/ListenV2", span.Name())
	assert.Equal(t, "0000000000000002", span.Parent().SpanID().String())
	assert.Empty(t, span.Events())
	assert.Contains(t, span.Attributes(), attribute.String("rpc.response.status_code", "OK"))
}

func TestCanonicalCodeNames(t *testing.T) {
	assert.Equal(t, "OK", canonicalCodeName(0))
	assert.Equal(t, "CANCELLED", canonicalCodeName(connect.CodeCanceled))
	assert.Equal(t, "UNAUTHENTICATED", canonicalCodeName(connect.CodeUnauthenticated))
	assert.Equal(t, "CODE(42)", canonicalCodeName(connect.Code(42)))
}
