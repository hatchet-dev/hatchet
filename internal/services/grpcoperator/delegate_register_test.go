package grpcoperator

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc/operatorsvctest"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// registerTwiceDispatcher reads two messages from the durable stream and records the worker
// ids of the register messages it was handed.
type registerTwiceDispatcher struct {
	*operatorsvctest.Dispatcher

	seen []string
	err  error
}

func (d *registerTwiceDispatcher) DurableTask(stream v1contracts.V1Dispatcher_DurableTaskServer) error {
	for i := 0; i < 2; i++ {
		req, err := stream.Recv()

		if err != nil {
			d.err = err
			return err
		}

		d.seen = append(d.seen, req.GetRegisterWorker().GetWorkerId())
	}

	return nil
}

// The SDK's durable listener registers exactly once per stream, so a second register_worker
// is a protocol error. It is refused with InvalidArgument before the dispatcher sees it,
// whichever worker it names, so the ownership check on the first registration cannot be
// bypassed by re-registering another worker of the same tenant.
func TestDurableTaskRejectsSecondRegisterWorker(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}

	cases := []struct {
		name   string
		second func(own, other string) string
	}{
		{name: "another operator's worker", second: func(_, other string) string { return other }},
		{name: "the same worker", second: func(own, _ string) string { return own }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t, nil)
			ctx, _, own := registeredOperator(t, svc, tenant)

			otherOp := uuid.New()
			other := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &otherOp})

			d := &registerTwiceDispatcher{Dispatcher: svc.dispatcher}
			svc.durable = d

			stream := &fakeDurableStream{ctx: ctx, recv: make(chan *v1contracts.DurableTaskRequest, 2)}
			stream.recv <- registerWorkerMsg(own.ID.String())
			stream.recv <- registerWorkerMsg(tc.second(own.ID.String(), other.ID.String()))

			err := svc.DurableTask(stream)
			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err), err)
			assert.Equal(t, []string{own.ID.String()}, d.seen, "only the first registration reaches the dispatcher")
			assert.Equal(t, codes.InvalidArgument, status.Code(d.err), "the dispatcher observes the refusal as the stream error")
		})
	}
}

// Messages after the first registration keep flowing to the dispatcher unchanged.
func TestDurableTaskPassesLaterMessagesThrough(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	ctx, _, own := registeredOperator(t, svc, tenant)

	d := &registerTwiceDispatcher{Dispatcher: svc.dispatcher}
	svc.durable = d

	stream := &fakeDurableStream{ctx: context.WithoutCancel(ctx), recv: make(chan *v1contracts.DurableTaskRequest, 2)}
	stream.recv <- registerWorkerMsg(own.ID.String())
	stream.recv <- &v1contracts.DurableTaskRequest{Message: &v1contracts.DurableTaskRequest_Memo{Memo: &v1contracts.DurableTaskMemoRequest{}}}

	require.NoError(t, svc.DurableTask(stream))
	assert.Equal(t, []string{own.ID.String(), ""}, d.seen)
}
