package grpcoperator

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc/operatorsvctest"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// testService is the gRPC handler set over an operator service backed by in-memory doubles, so
// the tests drive the real protocol loop and see what reached the stores.
type testService struct {
	*OperatorServiceImpl

	operators  *operatorsvctest.OperatorStore
	workers    *operatorsvctest.WorkerStore
	dispatcher *operatorsvctest.Dispatcher

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
	dispatcherId := uuid.New()

	svc, err := operatorsvc.New(
		operatorsvc.Deps{Operators: operators, Workers: workers, Dispatcher: d, DispatcherId: dispatcherId},
		append([]operatorsvc.Opt{operatorsvc.WithLogger(&l)}, opts...)...,
	)
	require.NoError(t, err)

	impl := &OperatorServiceImpl{svc: svc, durable: d, l: &l}

	t.Cleanup(func() { _ = impl.Cleanup() })

	return &testService{
		OperatorServiceImpl: impl,
		operators:           operators,
		workers:             workers,
		dispatcher:          d,
		dispatcherId:        dispatcherId,
	}
}

func tenantContext(tenant *sqlcv1.Tenant) context.Context {
	return context.WithValue(context.Background(), "tenant", tenant) //nolint:staticcheck // key must match the gRPC auth middleware's
}

func operatorContext(tenant *sqlcv1.Tenant, operatorId string) context.Context {
	return metadata.NewIncomingContext(tenantContext(tenant), metadata.Pairs(OperatorIdMetadataKey, operatorId))
}

// registeredOperator seeds an operator with one worker and returns the operator-scoped context.
func registeredOperator(t *testing.T, svc *testService, tenant *sqlcv1.Tenant) (context.Context, *sqlcv1.V1Operator, *sqlcv1.Worker) {
	t.Helper()

	op, err := svc.operators.UpsertGRPCOperator(t.Context(), tenant.ID, "op")
	require.NoError(t, err)

	worker := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &op.ID})

	return operatorContext(tenant, op.ID.String()), op, worker
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
