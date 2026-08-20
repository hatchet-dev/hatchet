//go:build !e2e && !load && !rampup && !integration

package dagoperator

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// fakeTaskEventWriter captures every reported step action event.
type fakeTaskEventWriter struct {
	events []*contracts.StepActionEvent
}

func (f *fakeTaskEventWriter) SendStepActionEvent(_ context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	f.events = append(f.events, request)
	return &contracts.ActionEventResponse{}, nil
}

func (f *fakeTaskEventWriter) RegisterDurableTask(_ context.Context, _ uuid.UUID) (chan<- *v1contracts.DurableTaskRequest, <-chan *v1contracts.DurableTaskResponse, error) {
	return nil, nil, nil
}

func (f *fakeTaskEventWriter) CancelTaskEvent(_ context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	f.events = append(f.events, request)
	return &contracts.ActionEventResponse{}, nil
}

func (f *fakeTaskEventWriter) TriggerDAGStep(_ context.Context, _ uuid.UUID, _ *operator.DAGStepTriggerRequest) (*operator.DAGStepTriggerResult, error) {
	return nil, nil
}

func (f *fakeTaskEventWriter) CancelDAGChildren(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

// newTestDAGOperator builds a DAGOperator whose shared state is wired to a fake event writer,
// without going through NewDAGOperator (which would start a real workflow-polling goroutine
// and require a real repository). repo is intentionally left nil: HandleAction's cancel path
// must not touch it, and a nil-repo panic would be a clear signal that it did.
func newTestDAGOperator(t *testing.T, workerId uuid.UUID, writer operator.TaskEventWriter) *DAGOperator {
	t.Helper()

	l := zerolog.Nop()

	shared, err := operator.NewSharedOperator(&sqlcv1.V1Operator{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Config:   []byte(`{}`),
	}, &l, nil, writer, workerId, DAGOperatorConfig{})
	require.NoError(t, err)

	return &DAGOperator{
		SharedOperator: shared,
	}
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
	workerId := uuid.New()

	d := newTestDAGOperator(t, workerId, writer)

	action := testAction()
	action.ActionType = contracts.ActionType_CANCEL_STEP_RUN

	// d.repo is nil; if the cancel branch fell through to d.run (which lists DAG workflows
	// via the repo), this call would panic instead of returning cleanly.
	err := d.HandleAction(context.Background(), action)
	require.NoError(t, err)

	require.Len(t, writer.events, 1, "cancelling a task must report exactly one step action event")
	got := writer.events[0]
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED, got.EventType)
	assert.Equal(t, action.TaskRunExternalId, got.TaskRunExternalId)
	assert.Equal(t, action.TaskId, got.TaskId)
	assert.Equal(t, workerId.String(), got.WorkerId)
}

func TestHandleAction_UnsupportedActionType_AcknowledgesWithoutReporting(t *testing.T) {
	writer := &fakeTaskEventWriter{}
	workerId := uuid.New()

	d := newTestDAGOperator(t, workerId, writer)

	action := testAction()
	action.ActionType = contracts.ActionType_START_GET_GROUP_KEY

	err := d.HandleAction(context.Background(), action)
	require.NoError(t, err)
	assert.Empty(t, writer.events, "an unsupported action type must not report any step action event")
}
