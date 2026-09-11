//go:build !e2e && !load && !rampup && !integration

package dagoperator

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc/operatorsvctest"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// These tests run the interpreter over the engine's own channel (internal/services/operatorsvc
// over the dispatcher double), which applies the two rules the raw test channel in dag_test.go
// does not: an entry completion is held until the ack naming its ref, and one ack-bearing
// request is in flight at a time.

type nopDagHandler struct{}

func (nopDagHandler) HandleAction(context.Context, *contracts.AssignedAction) error { return nil }

// engineChannel opens one invocation through operatorsvc with the handshake done, and returns
// the invocation's engine side.
func engineChannel(t *testing.T) (operator.DurableChannel, *operatorsvctest.DurableInvocation, uuid.UUID, uuid.UUID) {
	t.Helper()

	l := zerolog.Nop()
	operators := operatorsvctest.NewOperatorStore()
	workers := operatorsvctest.NewWorkerStore()
	dispatcher := operatorsvctest.NewDispatcher()

	svc, err := operatorsvc.New(
		operatorsvc.WithOperatorStore(operators),
		operatorsvc.WithWorkerStore(workers),
		operatorsvc.WithDispatcherBackend(dispatcher),
		operatorsvc.WithDispatcherId(uuid.New()),
		operatorsvc.WithLogger(&l),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Cleanup() })

	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	op, err := operators.UpsertGRPCOperator(t.Context(), tenant.ID, "dag")
	require.NoError(t, err)

	worker := workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &op.ID})

	session, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Handler: nopDagHandler{}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close(t.Context()) })

	handshake := make(chan *operatorsvctest.DurableInvocation, 1)

	go func() {
		deadline := time.Now().Add(5 * time.Second)

		for len(dispatcher.Durables()) == 0 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}

		if len(dispatcher.Durables()) == 0 {
			return
		}

		inv := dispatcher.Durables()[0]

		select {
		case <-inv.Requests:
		case <-t.Context().Done():
			return
		}

		inv.Responses <- &v1contracts.DurableTaskResponse{Message: &v1contracts.DurableTaskResponse_RegisterWorker{
			RegisterWorker: &v1contracts.DurableTaskResponseRegisterWorker{WorkerId: worker.ID.String()},
		}}
		handshake <- inv
	}()

	externalId := uuid.New()

	ch, err := session.OpenDurable(t.Context(), externalId, 1)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ch.Close() })

	return ch, <-handshake, externalId, worker.ID
}

// A child the interpreter creates through the direct trigger callback has no trigger_runs ack
// on the channel. Its completion must still reach the interpreter, even when it arrives before
// the callback has returned the child's ref.
func TestDagOverEngineChannelCompletesDirectlyTriggeredChild(t *testing.T) {
	ch, inv, externalId, workerId := engineChannel(t)

	a := newTestTask("a", "action-a", 0)

	trigger := func(context.Context, string, string, int32, []uuid.UUID, bool, bool, bool) (*operator.DAGStepTriggerResult, error) {
		// the child finishes before the interpreter has its ref
		inv.Responses <- &v1contracts.DurableTaskResponse{Message: &v1contracts.DurableTaskResponse_EntryCompleted{
			EntryCompleted: &v1contracts.DurableTaskEventLogEntryCompletedResponse{
				Ref:     &v1contracts.DurableEventLogEntryRef{DurableTaskExternalId: externalId.String(), InvocationCount: 1, BranchId: 1, NodeId: 1},
				Payload: []byte(`{"ok":true}`),
			},
		}}

		return &operator.DAGStepTriggerResult{NodeId: 1, BranchId: 1, WorkflowRunExternalId: uuid.New()}, nil
	}

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	require.NoError(t, dagDurableTask(ctx, []*task{a}, nil, externalId, workerId, 1, `{}`, ch, nil, trigger))
	assert.True(t, a.isTriggered)
	assert.True(t, a.isCompleted, "the child's completion was held for an ack that never comes")
}

// A task with both a skip and a cancel condition needs two registrations. The channel admits
// one ack-bearing request at a time, so the interpreter sends the second only once the first
// is acknowledged, and an engine that takes its time acknowledging is not an error.
func TestDagOverEngineChannelRegistersConditionsOneAtATime(t *testing.T) {
	ch, inv, externalId, workerId := engineChannel(t)

	a := newTestTask("a", "action-a", 0)

	for _, action := range []sqlcv1.V1MatchConditionAction{sqlcv1.V1MatchConditionActionSKIP, sqlcv1.V1MatchConditionActionCANCEL} {
		a.stepConditions = append(a.stepConditions, &sqlcv1.V1StepMatchCondition{
			Kind:            sqlcv1.V1StepMatchConditionKindSLEEP,
			Action:          action,
			OrGroupID:       uuid.New(),
			ReadableDataKey: string(action),
			SleepDuration:   sqlchelpers.TextFromStr("5s"),
		})
	}

	var registrations atomic.Int32
	var overlapping atomic.Bool

	// the engine side: every wait_for is acknowledged after a delay, and a second one arriving
	// before the first was acknowledged is recorded as an overlap
	go func() {
		var next int64 = 10

		for {
			var req *v1contracts.DurableTaskRequest

			select {
			case req = <-inv.Requests:
			case <-t.Context().Done():
				return
			}

			if req.GetWaitFor() == nil {
				continue
			}

			registrations.Add(1)

			if len(inv.Requests) > 0 {
				overlapping.Store(true)
			}

			time.Sleep(50 * time.Millisecond)

			ref := &v1contracts.DurableEventLogEntryRef{DurableTaskExternalId: externalId.String(), InvocationCount: 1, BranchId: next, NodeId: next}
			next++

			inv.Responses <- &v1contracts.DurableTaskResponse{Message: &v1contracts.DurableTaskResponse_WaitForAck{
				WaitForAck: &v1contracts.DurableTaskEventWaitForAckResponse{Ref: ref},
			}}
		}
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	require.NoError(t, dagDurableTask(ctx, []*task{a}, nil, externalId, workerId, 1, `{}`, ch, nil, stubTriggerStep(t, nil)))
	assert.EqualValues(t, 2, registrations.Load(), "both conditions are registered")
	assert.False(t, overlapping.Load(), "the second registration waits for the first ack")
	assert.True(t, a.isTriggered)
	assert.True(t, a.isCompleted)
	assert.True(t, a.skipWatchRegistered)
	assert.True(t, a.cancelWatchRegistered)
}
