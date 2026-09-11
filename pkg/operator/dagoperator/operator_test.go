//go:build !e2e && !load && !rampup && !integration

package dagoperator

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/operatortest"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// fakeTaskEventWriter is the engine-internal writer; nothing in these tests reaches it.
type fakeTaskEventWriter struct {
	events []*contracts.StepActionEvent
}

func (f *fakeTaskEventWriter) CancelTaskEventCustom(_ context.Context, _ uuid.UUID, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	f.events = append(f.events, request)
	return &contracts.ActionEventResponse{}, nil
}

func (f *fakeTaskEventWriter) TriggerDAGStep(_ context.Context, _ uuid.UUID, _ *operator.DAGStepTriggerRequest) (*operator.DAGStepTriggerResult, error) {
	return nil, nil
}

func (f *fakeTaskEventWriter) CancelDAGChildren(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

// newTestDAGOperator builds a DAGOperator whose shared state is started on a recording session,
// without going through Start (which would refresh actions from a real repository). repo is
// intentionally left nil: HandleAction's cancel path must not touch it, and a nil-repo panic
// would be a clear signal that it did.
func newTestDAGOperator(t *testing.T, writer operator.TaskEventWriter) (*DAGOperator, *operatortest.Session) {
	t.Helper()

	l := zerolog.Nop()
	op := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: uuid.New(), Config: []byte(`{}`)}

	shared, err := operator.NewSharedOperator(op, &l, writer, DAGOperatorConfig{})
	require.NoError(t, err)

	session := operatortest.NewSession(op.TenantID, op.ID)
	require.NoError(t, shared.Start(context.Background(), session))

	return &DAGOperator{SharedOperator: shared}, session
}

func testAction() *contracts.AssignedAction {
	return &contracts.AssignedAction{
		ActionType:        contracts.ActionType_START_STEP_RUN,
		TenantId:          "tenant-1",
		TaskId:            "task-1",
		TaskRunExternalId: "run-1",
		TaskName:          "my-task",
		ActionId:          "action-1",
		RetryCount:        2,
	}
}

func TestHandleAction_CancelStepRun_ReportsCancelledWithoutRunning(t *testing.T) {
	writer := &fakeTaskEventWriter{}

	d, session := newTestDAGOperator(t, writer)

	action := testAction()
	action.ActionType = contracts.ActionType_CANCEL_STEP_RUN

	// d.repo is nil; if the cancel branch fell through to d.run (which lists DAG workflows
	// via the repo), this call would panic instead of returning cleanly.
	err := d.HandleAction(context.Background(), action)
	require.NoError(t, err)

	events := session.Events()
	require.Len(t, events, 1, "cancelling a task must report exactly one step action event, through the session")
	got := events[0]
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED, got.EventType)
	assert.Equal(t, action.TaskRunExternalId, got.TaskRunExternalId)
	assert.Equal(t, action.TaskId, got.TaskId)
	assert.Equal(t, session.Registration().WorkerId.String(), got.WorkerId)
	assert.Empty(t, writer.events, "the plain cancel does not use the engine-internal writer")
}

func TestHandleAction_UnsupportedActionType_AcknowledgesWithoutReporting(t *testing.T) {
	writer := &fakeTaskEventWriter{}

	d, session := newTestDAGOperator(t, writer)

	action := testAction()
	action.ActionType = contracts.ActionType_START_GET_GROUP_KEY

	err := d.HandleAction(context.Background(), action)
	require.NoError(t, err)
	assert.Empty(t, session.Events(), "an unsupported action type must not report any step action event")
}

// SlotConfig derives the worker's durable slots from the row's config, or the server default.
func TestSlotConfig(t *testing.T) {
	cfg, err := SlotConfig(&sqlcv1.V1Operator{Config: []byte(`{"slots": 7}`)}, 100)
	require.NoError(t, err)
	assert.Equal(t, map[string]int32{"durable": 7}, cfg)

	cfg, err = SlotConfig(&sqlcv1.V1Operator{Config: []byte(`{}`)}, 100)
	require.NoError(t, err)
	assert.Equal(t, map[string]int32{"durable": 100}, cfg)

	_, err = SlotConfig(&sqlcv1.V1Operator{Config: []byte(`nope`)}, 100)
	require.Error(t, err)
}
