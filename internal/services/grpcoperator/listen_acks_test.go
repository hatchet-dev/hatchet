package grpcoperator

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func sequencedDeltaMsg(seq uint64, add, remove []string) *v1contracts.OperatorListenRequest {
	return &v1contracts.OperatorListenRequest{Message: &v1contracts.OperatorListenRequest_Actions{
		Actions: &v1contracts.OperatorActionsDelta{Add: add, Remove: remove, Sequence: seq},
	}}
}

// Every sequenced delta is acknowledged once it is applied, in stream order, whether or not
// it changed the set; a delta without a sequence asks for no ack.
func TestListenAcknowledgesCommittedDeltas(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	ctx, _, worker := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream := newFakeListenStream(ctx, startMsg(worker.ID.String()))
	done := runListen(svc, stream)

	eventually(t, func() bool { return svc.dispatcher.SessionCount() == 1 }, "session was not registered")

	stream.push(sequencedDeltaMsg(7, []string{"svc:a"}, nil))
	stream.push(deltaMsg([]string{"svc:b"}, nil))
	stream.push(sequencedDeltaMsg(9, []string{"svc:a"}, []string{"svc:never"}))

	eventually(t, func() bool { return len(svc.dispatcher.AckedSequences()) == 2 }, "deltas were not acknowledged")
	assert.Equal(t, []uint64{7, 9}, svc.dispatcher.AckedSequences(), "acks carry the delta's sequence and skip unsequenced deltas")
	assert.ElementsMatch(t, []string{"svc:a", "svc:b"}, svc.workers.ActionSet(worker.ID))

	close(stream.recv)
	assert.NoError(t, waitListen(t, done))
}

// A delta the store rejects is never acknowledged: the stream ends so the client resends it
// after reconnecting.
func TestListenDoesNotAcknowledgeRejectedDelta(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	ctx, _, worker := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream := newFakeListenStream(ctx, startMsg(worker.ID.String()), sequencedDeltaMsg(3, []string{"not an action"}, nil))

	err := waitListen(t, runListen(svc, stream))
	assert.Equal(t, codes.InvalidArgument, status.Code(err), err)
	assert.Empty(t, svc.dispatcher.AckedSequences())
}

// An ack that cannot be written ends the stream instead of leaving the delta unconfirmed.
func TestListenEndsWhenAckCannotBeSent(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	ctx, _, worker := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	svc.dispatcher.SetSendErr(errors.New("flow control"))

	stream := newFakeListenStream(ctx, startMsg(worker.ID.String()), sequencedDeltaMsg(1, []string{"svc:a"}, nil))

	err := waitListen(t, runListen(svc, stream))
	assert.Equal(t, codes.Unavailable, status.Code(err), err)
	assert.Equal(t, []string{"svc:a"}, svc.workers.ActionSet(worker.ID), "the delta itself was committed before the ack failed")
	assert.Len(t, svc.workers.SessionLog(), 2, "the worker is deactivated on exit")
}

// The dispatcher fan-out wraps every assigned action in the Listen response envelope.
func TestWrapAssignedAction(t *testing.T) {
	action := &contracts.AssignedAction{ActionId: "svc:a", TaskRunExternalId: uuid.NewString()}

	resp, ok := wrapAssignedAction(action).(*v1contracts.OperatorListenResponse)
	require.True(t, ok)
	assert.True(t, proto.Equal(resp.GetAction(), action))
	assert.Nil(t, resp.GetAck())
}
