package operatorsvc_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc/operatorsvctest"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// testService is a Service over in-memory doubles, with the doubles kept for assertions.
type testService struct {
	*operatorsvc.Service

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
		append([]operatorsvc.Opt{
			operatorsvc.WithOperatorStore(operators),
			operatorsvc.WithWorkerStore(workers),
			operatorsvc.WithDispatcherBackend(d),
			operatorsvc.WithDispatcherId(dispatcherId),
			operatorsvc.WithLogger(&l),
		}, opts...)...,
	)
	require.NoError(t, err)

	t.Cleanup(func() { _ = svc.Cleanup() })

	return &testService{Service: svc, operators: operators, workers: workers, dispatcher: d, dispatcherId: dispatcherId}
}

// registeredOperator seeds an operator with one worker of its own.
func registeredOperator(t *testing.T, svc *testService, tenant *sqlcv1.Tenant) (*sqlcv1.V1Operator, *sqlcv1.Worker) {
	t.Helper()

	op, err := svc.operators.UpsertGRPCOperator(t.Context(), tenant.ID, "op")
	require.NoError(t, err)

	worker := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &op.ID})

	return op, worker
}

// nopStream discards every message; newNopSender wraps it the way a Listen handler would. The dispatcher
// owns the stream, and the dispatcher here is a double, so nothing is ever called on it.
type nopStream struct{}

func (nopStream) Send(*v1contracts.OperatorListenResponse) error { return nil }

func newNopSender(t *testing.T) *rpcstream.Sender[v1contracts.OperatorListenResponse] {
	t.Helper()

	sender := rpcstream.NewSender[v1contracts.OperatorListenResponse](t.Context(), nopStream{})
	t.Cleanup(sender.Close)

	return sender
}

// nopHandler stands in for an in-process operator's action handler.
type nopHandler struct{}

func (nopHandler) HandleAction(_ context.Context, _ *contracts.AssignedAction) error { return nil }

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
