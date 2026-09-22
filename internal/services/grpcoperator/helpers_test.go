package grpcoperator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc/operatorsvctest"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1/v1connect"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// testService is the connect handler set over an operator service backed by in-memory doubles,
// so the tests drive the real protocol loop and see what reached the stores.
type testService struct {
	*OperatorServiceImpl

	operators  *operatorsvctest.OperatorStore
	workers    *operatorsvctest.WorkerStore
	dispatcher *operatorsvctest.Dispatcher
	recorder   *durableRecorder

	dispatcherId uuid.UUID
}

func newTestService(t *testing.T, operators *operatorsvctest.OperatorStore, opts ...operatorsvc.Opt) *testService {
	t.Helper()

	if operators == nil {
		operators = operatorsvctest.NewOperatorStore()
	}

	l := zerolog.Nop()
	workers := operatorsvctest.NewWorkerStore()
	d := operatorsvctest.NewDispatcher()
	recorder := &durableRecorder{}
	dispatcherId := uuid.New()

	svc, err := operatorsvc.New(
		append([]operatorsvc.Opt{
			operatorsvc.WithOperatorStore(operators),
			operatorsvc.WithWorkerStore(workers),
			operatorsvc.WithDispatcherBackend(d),
			operatorsvc.WithDispatcherId(dispatcherId),
			operatorsvc.WithLogger(&l),
		}, opts...)...,
	)
	require.NoError(t, err)

	impl := &OperatorServiceImpl{svc: svc, durable: recorder, l: &l}

	t.Cleanup(func() { _ = impl.Cleanup() })

	return &testService{
		OperatorServiceImpl: impl,
		operators:           operators,
		workers:             workers,
		dispatcher:          d,
		recorder:            recorder,
		dispatcherId:        dispatcherId,
	}
}

// durableRecorder stands in for the dispatcher's durable task session: it reads the first
// message the operator service hands over and records it.
type durableRecorder struct {
	mu       sync.Mutex
	register *v1contracts.DurableTaskRequest
}

func (r *durableRecorder) DurableTaskWithReceive(_ context.Context, receive func() (*v1contracts.DurableTaskRequest, error), _ *rpcstream.Sender[v1contracts.DurableTaskResponse]) error {
	req, err := receive()

	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.register = req

	return nil
}

// Register returns the first message delegated to the dispatcher, nil if none was.
func (r *durableRecorder) Register() *v1contracts.DurableTaskRequest {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.register
}

func tenantContext(tenant *sqlcv1.Tenant) context.Context {
	return context.WithValue(context.Background(), "tenant", tenant) //nolint:staticcheck // key must match the gRPC auth middleware's
}

// registeredOperator seeds an operator with one worker and returns the tenant-scoped context
// the run halves take alongside the operator and its worker.
func registeredOperator(t *testing.T, svc *testService, tenant *sqlcv1.Tenant) (context.Context, *sqlcv1.V1Operator, *sqlcv1.Worker) {
	t.Helper()

	op, err := svc.operators.UpsertGRPCOperator(t.Context(), tenant.ID, "op")
	require.NoError(t, err)

	worker := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &op.ID})

	return tenantContext(tenant), op, worker
}

// tenantInterceptor attaches the tenant to every request context the way the auth middleware
// does; a nil tenant leaves it off, as for a request that carried no valid token.
type tenantInterceptor struct {
	tenant *sqlcv1.Tenant
}

func (i tenantInterceptor) withTenant(ctx context.Context) context.Context {
	if i.tenant == nil {
		return ctx
	}

	return context.WithValue(ctx, "tenant", i.tenant) //nolint:staticcheck // key must match the gRPC auth middleware's
}

func (i tenantInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return next(i.withTenant(ctx), req)
	}
}

func (i tenantInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i tenantInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(i.withTenant(ctx), conn)
	}
}

// operatorClient serves svc over connect and returns a client for it, so tests that exercise
// authorizeOperator see the request header the way connect's handler machinery delivers it.
// Only authorization itself goes through here; everything else drives the run halves directly.
func operatorClient(t *testing.T, svc *testService, tenant *sqlcv1.Tenant) v1connect.OperatorServiceClient {
	t.Helper()

	path, handler := v1connect.NewOperatorServiceHandler(svc.OperatorServiceImpl, connect.WithInterceptors(tenantInterceptor{tenant: tenant}))

	mux := http.NewServeMux()
	mux.Handle(path, handler)

	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	return v1connect.NewOperatorServiceClient(server.Client(), server.URL, connect.WithGRPC())
}

// operatorCallContext returns a client context whose requests carry the operator id header.
func operatorCallContext(ctx context.Context, operatorId string) context.Context {
	ctx, info := connect.NewClientContext(ctx)
	info.RequestHeader().Set(OperatorIdMetadataKey, operatorId)

	return ctx
}

func eventually(t *testing.T, cond func() bool, msg string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		if cond() {
			return
		}

		time.Sleep(2 * time.Millisecond)
	}

	t.Fatal(msg)
}
