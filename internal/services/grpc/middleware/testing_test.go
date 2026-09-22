package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"golang.org/x/time/rate"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts/contractsconnect"
	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const testToken = "valid-token"

var (
	testTenantID = uuid.MustParse("707d0855-80ab-4e1f-a156-f1c4546cbf52")
	testTokenID  = uuid.MustParse("11111111-1111-4111-8111-111111111111")
)

type fakeJWTManager struct {
	calls atomic.Int64
}

func (*fakeJWTManager) GenerateTenantToken(context.Context, uuid.UUID, string, bool, *time.Time) (*token.Token, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeJWTManager) ValidateTenantToken(_ context.Context, tok string) (uuid.UUID, uuid.UUID, error) {
	f.calls.Add(1)

	if tok != testToken {
		return uuid.Nil, uuid.Nil, errors.New("invalid token")
	}

	return testTenantID, testTokenID, nil
}

type fakeTenantRepo struct {
	v1.TenantRepository
}

func (fakeTenantRepo) GetTenantByID(_ context.Context, id uuid.UUID) (*sqlcv1.Tenant, error) {
	if id != testTenantID {
		return nil, pgx.ErrNoRows
	}

	return &sqlcv1.Tenant{ID: id}, nil
}

type fakeRepository struct {
	v1.Repository
}

func (fakeRepository) Tenant() v1.TenantRepository {
	return fakeTenantRepo{}
}

// fakeDispatcher implements Register and ListenV2; everything else is unimplemented.
type fakeDispatcher struct {
	contractsconnect.UnimplementedDispatcherHandler

	calls   atomic.Int64
	lastCtx atomic.Value
}

func (f *fakeDispatcher) Register(ctx context.Context, req *contracts.WorkerRegisterRequest) (*contracts.WorkerRegisterResponse, error) {
	f.calls.Add(1)
	f.lastCtx.Store(&ctx)

	switch req.WorkerName {
	case "invalid":
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid worker name"))
	case "internal":
		return nil, connect.NewError(connect.CodeInternal, errors.New("An internal error occurred."))
	}

	return &contracts.WorkerRegisterResponse{WorkerName: req.WorkerName}, nil
}

func (f *fakeDispatcher) ListenV2(_ context.Context, _ *contracts.WorkerListenRequest, stream *connect.ServerStream[contracts.AssignedAction]) error {
	f.calls.Add(1)

	return stream.Send(&contracts.AssignedAction{ActionId: "action"})
}

type testServer struct {
	url        string
	httpClient *http.Client
	dispatcher *fakeDispatcher
	jwt        *fakeJWTManager
	spans      *tracetest.SpanRecorder
	metrics    *sdkmetric.ManualReader
}

// startTestServer serves the fake dispatcher over HTTP/2 behind the request gate and the
// telemetry interceptor, recording spans and metrics locally. l may be nil.
func startTestServer(t *testing.T, l *zerolog.Logger, limit rate.Limit, burst int) *testServer {
	t.Helper()

	if l == nil {
		nop := zerolog.Nop()
		l = &nop
	}

	ts := &testServer{
		dispatcher: &fakeDispatcher{},
		jwt:        &fakeJWTManager{},
		spans:      tracetest.NewSpanRecorder(),
		metrics:    sdkmetric.NewManualReader(),
	}

	recorder := newTelemetryRecorder(
		sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(ts.spans)),
		sdkmetric.NewMeterProvider(sdkmetric.WithReader(ts.metrics)),
		propagation.TraceContext{},
	)

	sc := &server.ServerConfig{Layer: &database.Layer{V1: fakeRepository{}}}
	sc.Logger = l
	sc.Auth.JWTManager = ts.jwt

	nop := zerolog.Nop()

	mux := http.NewServeMux()
	mux.Handle(contractsconnect.NewDispatcherHandler(
		ts.dispatcher,
		connect.WithRequestGate(newRequestGate(NewAuthN(sc), NewHatchetRateLimiter(limit, burst, &nop), l, recorder)),
		connect.WithInterceptors(newTelemetryInterceptor(recorder)),
	))

	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	t.Cleanup(srv.Close)

	ts.url = srv.URL
	ts.httpClient = srv.Client()

	return ts
}

func (ts *testServer) client() contractsconnect.DispatcherClient {
	return contractsconnect.NewDispatcherClient(ts.httpClient, ts.url, connect.WithGRPC())
}

func authContext(tok string, headers ...string) context.Context {
	ctx, callInfo := connect.NewClientContext(context.Background())
	callInfo.RequestHeader().Set("Authorization", "Bearer "+tok)

	for i := 0; i+1 < len(headers); i += 2 {
		callInfo.RequestHeader().Set(headers[i], headers[i+1])
	}

	return ctx
}
