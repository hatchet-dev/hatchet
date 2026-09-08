package grpcoperator

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// An operator may hold at most maxListenStreamsPerOperator streams on one replica; a stream
// over the cap is refused with ResourceExhausted before it is activated, and the slot is
// returned when a stream ends.
func TestListenRejectsStreamsOverOperatorCap(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	svc.maxListenStreamsPerOperator = 1
	ctx, op, worker := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	second := svc.workers.add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &op.ID})

	first := newFakeListenStream(ctx, startMsg(worker.ID.String()))
	firstDone := runListen(svc, first)

	eventually(t, func() bool { return svc.dispatcher.sessionCount() == 1 }, "first session was not registered")
	assert.Equal(t, 1, svc.listenStreamCount(op.ID))

	err := waitListen(t, runListen(svc, newFakeListenStream(ctx, startMsg(second.ID.String()))))
	assert.Equal(t, codes.ResourceExhausted, status.Code(err), err)
	assert.Equal(t, 1, svc.dispatcher.sessionCount(), "a refused stream registers no session")
	assert.Len(t, svc.workers.sessionLog(), 1, "a refused stream never activates its worker")

	close(first.recv)
	require.NoError(t, waitListen(t, firstDone))
	assert.Zero(t, svc.listenStreamCount(op.ID), "the slot is released when the stream ends")

	third := newFakeListenStream(ctx, startMsg(second.ID.String()))
	thirdDone := runListen(svc, third)

	eventually(t, func() bool { return svc.dispatcher.sessionCount() == 2 }, "a stream after the release was not admitted")

	close(third.recv)
	require.NoError(t, waitListen(t, thirdDone))
}

// Another operator's streams do not count against the cap.
func TestListenStreamCapIsPerOperator(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	svc.maxListenStreamsPerOperator = 1
	ctx, _, worker := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	otherOp, err := svc.operators.UpsertGRPCOperator(tenantContext(tenant), tenant.ID, "other")
	require.NoError(t, err)
	otherWorker := svc.workers.add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &otherOp.ID})
	otherCtx, cancelOther := context.WithCancel(operatorContext(tenant, otherOp.ID.String()))
	defer cancelOther()

	first := newFakeListenStream(ctx, startMsg(worker.ID.String()))
	firstDone := runListen(svc, first)
	other := newFakeListenStream(otherCtx, startMsg(otherWorker.ID.String()))
	otherDone := runListen(svc, other)

	eventually(t, func() bool { return svc.dispatcher.sessionCount() == 2 }, "both operators' streams were not admitted")

	close(first.recv)
	close(other.recv)
	require.NoError(t, waitListen(t, firstDone))
	require.NoError(t, waitListen(t, otherDone))
}

// The action links across all workers of an operator are capped: a delta whose adds would
// exceed the cap is refused with ResourceExhausted and ends the stream, and the budget follows
// the links the stream applies and removes.
func TestListenRejectsActionsOverOperatorCap(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	svc.maxActionsPerOperator = 3
	ctx, op, worker := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// links held by another worker of the same operator count against the budget
	sibling := svc.workers.add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &op.ID})
	_, err := svc.workers.AddWorkerActions(ctx, tenant.ID, sibling.ID, []string{"svc:held"})
	require.NoError(t, err)

	stream := newFakeListenStream(ctx, startMsg(worker.ID.String()))
	done := runListen(svc, stream)

	eventually(t, func() bool { return svc.dispatcher.sessionCount() == 1 }, "session was not registered")

	stream.push(sequencedDeltaMsg(1, []string{"svc:a", "svc:b"}, nil))
	eventually(t, func() bool { return len(svc.dispatcher.ackedSequences()) == 1 }, "delta within the budget was not applied")

	// a repeated add does not consume budget, a removal gives it back
	stream.push(sequencedDeltaMsg(2, []string{"svc:a"}, []string{"svc:b"}))
	eventually(t, func() bool { return len(svc.dispatcher.ackedSequences()) == 2 }, "second delta was not applied")
	stream.push(sequencedDeltaMsg(3, []string{"svc:c"}, nil))
	eventually(t, func() bool { return len(svc.dispatcher.ackedSequences()) == 3 }, "delta after the removal was not applied")
	assert.ElementsMatch(t, []string{"svc:a", "svc:c"}, svc.workers.actionSet(worker.ID))

	stream.push(sequencedDeltaMsg(4, []string{"svc:d"}, nil))

	err = waitListen(t, done)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err), err)
	assert.ElementsMatch(t, []string{"svc:a", "svc:c"}, svc.workers.actionSet(worker.ID), "a refused delta is not applied")
	assert.Equal(t, []uint64{1, 2, 3}, svc.dispatcher.ackedSequences(), "a refused delta is not acknowledged")
}

// A stream that starts with the operator already at the cap can still remove actions.
func TestListenAtActionCapStillRemoves(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	svc.maxActionsPerOperator = 1
	ctx, _, worker := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := svc.workers.AddWorkerActions(ctx, tenant.ID, worker.ID, []string{"svc:a"})
	require.NoError(t, err)

	stream := newFakeListenStream(ctx, startMsg(worker.ID.String()))
	done := runListen(svc, stream)

	eventually(t, func() bool { return svc.dispatcher.sessionCount() == 1 }, "session was not registered")

	stream.push(sequencedDeltaMsg(1, nil, []string{"svc:a"}))
	eventually(t, func() bool { return len(svc.dispatcher.ackedSequences()) == 1 }, "removal was not applied")

	stream.push(sequencedDeltaMsg(2, []string{"svc:b"}, nil))
	eventually(t, func() bool { return len(svc.dispatcher.ackedSequences()) == 2 }, "add after the removal was not applied")
	assert.Equal(t, []string{"svc:b"}, svc.workers.actionSet(worker.ID))

	close(stream.recv)
	require.NoError(t, waitListen(t, done))
}
