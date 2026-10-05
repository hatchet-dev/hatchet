package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/ratelimit"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestGateSetsTheAuthContextForTheHandler(t *testing.T) {
	ts := startTestServer(t, nil, 1000, 1000)

	_, err := ts.client().Register(authContext(testToken, analytics.SourceMetadataKey, "cli"), &contracts.WorkerRegisterRequest{WorkerName: "w"})
	require.NoError(t, err)

	ctx := *ts.dispatcher.lastCtx.Load().(*context.Context)

	assert.Equal(t, testTenantID, ctx.Value(analytics.TenantIDKey))
	assert.Equal(t, testTokenID, ctx.Value(analytics.APITokenIDKey))
	assert.Equal(t, analytics.Source("cli"), ctx.Value(analytics.SourceKey))
	assert.Equal(t, testTenantID, ctx.Value("tenant").(*sqlcv1.Tenant).ID) // nolint:staticcheck
}

// An unauthenticated caller must be turned away before its body is read: a body that cannot be
// decoded is reported as unauthenticated, not as a decoding failure.
func TestGateRejectsBeforeTheBodyIsDecoded(t *testing.T) {
	ts := startTestServer(t, nil, 1000, 1000)

	for name, contentType := range map[string]string{
		"connect json":  "application/json",
		"grpc-web json": "application/grpc-web+json",
	} {
		req, err := http.NewRequest(http.MethodPost, ts.url+"/Dispatcher/Register", strings.NewReader(`{"workerName":`))
		require.NoError(t, err, name)

		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer nope")

		res, err := ts.httpClient.Do(req)
		require.NoError(t, err, name)

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err, name)
		require.NoError(t, res.Body.Close(), name)

		if contentType == "application/json" {
			assert.Equal(t, http.StatusUnauthorized, res.StatusCode, name)
			assert.Contains(t, string(body), `"unauthenticated"`, name)
		} else {
			assert.Equal(t, "16", res.Header.Get("Grpc-Status")+res.Trailer.Get("Grpc-Status"), name)
		}

		assert.NotContains(t, string(body), "WorkerRegisterRequest", name)
	}

	assert.Equal(t, int64(2), ts.jwt.calls.Load(), "the token must be checked before decoding")
	assert.Equal(t, int64(0), ts.dispatcher.calls.Load())
}

func TestGateAuthenticatesStreams(t *testing.T) {
	ts := startTestServer(t, nil, 1000, 1000)

	stream, err := ts.client().ListenV2(authContext("nope"), &contracts.WorkerListenRequest{WorkerId: "w"})
	require.NoError(t, err)
	require.False(t, stream.Receive())

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(stream.Err()))
	assert.Equal(t, int64(0), ts.dispatcher.calls.Load())

	var connectErr *connect.Error
	require.ErrorAs(t, stream.Err(), &connectErr)
	assert.Equal(t, "invalid auth token", connectErr.Message())
}

type exhaustedLimiter struct{}

func (exhaustedLimiter) Limit(context.Context) error {
	return status.Errorf(codes.ResourceExhausted, "dispatcher rate limit exceeded")
}

// The message is the one go-grpc-middleware's ratelimit interceptor produces around the
// limiter's status error, byte for byte. Callers may match on it.
func TestGateRateLimitMessageMatchesTheGRPCMiddleware(t *testing.T) {
	_, baseline := ratelimit.UnaryServerInterceptor(exhaustedLimiter{})(
		context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/Dispatcher/Register"}, nil,
	)
	require.Equal(t, codes.ResourceExhausted, status.Code(baseline))

	// the dispatcher bucket holds ten times the burst
	ts := startTestServer(t, nil, 1, 1)

	var limited error

	for range 50 {
		if _, err := ts.client().Register(authContext(testToken), &contracts.WorkerRegisterRequest{WorkerName: "w"}); err != nil {
			limited = err
			break
		}
	}

	var connectErr *connect.Error
	require.ErrorAs(t, limited, &connectErr)

	assert.Equal(t, connect.CodeResourceExhausted, connectErr.Code())
	assert.Equal(t, status.Convert(baseline).Message(), connectErr.Message())
	assert.Equal(t,
		"/Dispatcher/Register is rejected by grpc_ratelimit middleware, please retry later. rpc error: code = ResourceExhausted desc = dispatcher rate limit exceeded",
		connectErr.Message(),
	)
}

// Every limiter outcome is reported as resource exhausted, as the wrapper always did.
func TestRateLimitWrapsEveryLimiterError(t *testing.T) {
	l := zerolog.Nop()
	limiter := NewHatchetRateLimiter(1, 1, &l)

	noToken := rateLimit(context.Background(), limiter, "/Dispatcher/Register")
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(noToken))
	assert.Equal(t,
		"/Dispatcher/Register is rejected by grpc_ratelimit middleware, please retry later. rpc error: code = Unauthenticated desc = no rate limit token found",
		errorMessage(noToken),
	)

	ctx := context.WithValue(context.Background(), analytics.APITokenIDKey, testTokenID)

	unknown := rateLimit(ctx, limiter, "/Nope/Method")
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(unknown))
	assert.Equal(t,
		"/Nope/Method is rejected by grpc_ratelimit middleware, please retry later. rpc error: code = Internal desc = service /Nope/Method not recognized",
		errorMessage(unknown),
	)
}

// Rejected calls never reach the logging or telemetry interceptors, so the gate has to leave the
// same trail they would have.
func TestGateLogsAndTracesRejectedCalls(t *testing.T) {
	var logs bytes.Buffer

	l := zerolog.New(&logs)
	ts := startTestServer(t, &l, 1000, 1000)

	_, err := ts.client().Register(authContext("nope"), &contracts.WorkerRegisterRequest{WorkerName: "w"})
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	var finished map[string]any

	for _, line := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))

		if entry["message"] == "finished call" {
			finished = entry
		}
	}

	require.NotNil(t, finished, "logs were: %s", logs.String())
	assert.Equal(t, "info", finished["level"])
	assert.Equal(t, "Unauthenticated", finished["grpc.code"])
	assert.Equal(t, "rpc error: code = Unauthenticated desc = invalid auth token", finished["grpc.error"])
	assert.Equal(t, "Dispatcher", finished["grpc.service"])
	assert.Equal(t, "Register", finished["grpc.method"])
	assert.Equal(t, "unary", finished["grpc.method_type"])
	assert.Contains(t, logs.String(), "started call")

	spans := ts.spans.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, "Dispatcher/Register", spans[0].Name())
	assert.Contains(t, spans[0].Attributes(), attribute.String("rpc.response.status_code", "UNAUTHENTICATED"))
}

func TestGateTracesRateLimitedCallsWithTheirTenant(t *testing.T) {
	ts := startTestServer(t, nil, 1, 1)

	var limited error

	for range 50 {
		if _, limited = ts.client().Register(authContext(testToken), &contracts.WorkerRegisterRequest{WorkerName: "w"}); limited != nil {
			break
		}
	}

	require.Error(t, limited)
	require.False(t, errors.Is(limited, context.Canceled))

	spans := ts.spans.Ended()
	last := spans[len(spans)-1]

	assert.Contains(t, last.Attributes(), attribute.String("rpc.response.status_code", "RESOURCE_EXHAUSTED"))
	assert.Contains(t, last.Attributes(), attribute.String("hatchet.run/tenant.id", testTenantID.String()))
}
