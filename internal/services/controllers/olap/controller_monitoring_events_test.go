package olap

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type createTaskEventsCall struct {
	events              []sqlcv1.CreateTaskEventsOLAPParams
	orchestratorUpdates []v1.OrchestratorDAGStatusUpdateOpt
}

type fakeMonitoringRepo struct {
	v1.Repository
	tasks *fakeMonitoringTasks
	olap  *fakeMonitoringOLAP
}

func (r *fakeMonitoringRepo) Tasks() v1.TaskRepository { return r.tasks }
func (r *fakeMonitoringRepo) OLAP() v1.OLAPRepository  { return r.olap }

type fakeMonitoringTasks struct {
	v1.TaskRepository
	metas []*sqlcv1.ListTaskMetasRow
}

func (f *fakeMonitoringTasks) ListTaskMetas(_ context.Context, _ uuid.UUID, _ []int64) ([]*sqlcv1.ListTaskMetasRow, error) {
	return f.metas, nil
}

type fakeMonitoringOLAP struct {
	v1.OLAPRepository
	calls []createTaskEventsCall
}

func (f *fakeMonitoringOLAP) CreateTaskEvents(_ context.Context, _ uuid.UUID, events []sqlcv1.CreateTaskEventsOLAPParams, _ map[uuid.UUID]uuid.UUID, orchestratorUpdates []v1.OrchestratorDAGStatusUpdateOpt, _ map[uuid.UUID]struct{}) (*v1.StatusUpdateResult, map[uuid.UUID]struct{}, error) {
	f.calls = append(f.calls, createTaskEventsCall{events: events, orchestratorUpdates: orchestratorUpdates})
	return nil, nil, nil
}

func newMonitoringTestController(metas ...*sqlcv1.ListTaskMetasRow) (*OLAPControllerImpl, *fakeMonitoringOLAP) {
	l := zerolog.Nop()
	olap := &fakeMonitoringOLAP{}

	return &OLAPControllerImpl{
		l: &l,
		repo: &fakeMonitoringRepo{
			tasks: &fakeMonitoringTasks{metas: metas},
			olap:  olap,
		},
	}, olap
}

func monitoringPayloads(t *testing.T, msgs ...tasktypes.CreateMonitoringEventPayload) [][]byte {
	t.Helper()

	payloads := make([][]byte, 0, len(msgs))

	for _, msg := range msgs {
		b, err := json.Marshal(msg)
		require.NoError(t, err)
		payloads = append(payloads, b)
	}

	return payloads
}

func TestHandleCreateMonitoringEvent_OrchestratorStatusOnly(t *testing.T) {
	tenantId := uuid.New()
	runId := uuid.New()
	workerId := uuid.New()
	insertedAt := sqlchelpers.TimestamptzFromTime(time.Now().UTC())

	orchestrator := &sqlcv1.ListTaskMetasRow{
		ID:                1,
		InsertedAt:        insertedAt,
		WorkflowRunID:     runId,
		IsDagOrchestrator: true,
	}

	child := &sqlcv1.ListTaskMetasRow{
		ID:                            2,
		InsertedAt:                    insertedAt,
		WorkflowRunID:                 runId,
		WasTriggeredByDagOrchestrator: true,
	}

	regularFinished := tasktypes.CreateMonitoringEventPayload{
		TaskId:         orchestrator.ID,
		WorkerId:       &workerId,
		EventType:      sqlcv1.V1EventTypeOlapFINISHED,
		EventTimestamp: time.Now().UTC(),
		EventPayload:   `{"ok":true}`,
	}

	statusOnlyFinished := tasktypes.CreateMonitoringEventPayload{
		TaskId:         orchestrator.ID,
		EventType:      sqlcv1.V1EventTypeOlapFINISHED,
		EventTimestamp: time.Now().UTC(),
		EventPayload:   `{"ok":true}`,
		StatusOnly:     true,
	}

	childStarted := tasktypes.CreateMonitoringEventPayload{
		TaskId:         child.ID,
		WorkerId:       &workerId,
		EventType:      sqlcv1.V1EventTypeOlapSTARTED,
		EventTimestamp: time.Now().UTC(),
	}

	t.Run("status-only orchestrator event updates the DAG without a task event", func(t *testing.T) {
		tc, olap := newMonitoringTestController(orchestrator, child)

		err := tc.handleCreateMonitoringEvent(context.Background(), tenantId, monitoringPayloads(t, regularFinished, statusOnlyFinished, childStarted))
		require.NoError(t, err)

		require.Len(t, olap.calls, 1)

		events := olap.calls[0].events
		require.Len(t, events, 2)
		assert.Equal(t, orchestrator.ID, events[0].TaskID)
		assert.Equal(t, sqlcv1.V1EventTypeOlapFINISHED, events[0].EventType)
		assert.Equal(t, child.ID, events[1].TaskID)

		updates := olap.calls[0].orchestratorUpdates
		require.Len(t, updates, 2)

		for _, update := range updates {
			assert.Equal(t, orchestrator.ID, update.DagId)
			assert.Equal(t, sqlcv1.V1ReadableStatusOlapCOMPLETED, update.ReadableStatus)
		}
	})

	t.Run("batch of only status-only events still applies the DAG update", func(t *testing.T) {
		tc, olap := newMonitoringTestController(orchestrator)

		err := tc.handleCreateMonitoringEvent(context.Background(), tenantId, monitoringPayloads(t, statusOnlyFinished))
		require.NoError(t, err)

		require.Len(t, olap.calls, 1)
		assert.Empty(t, olap.calls[0].events)
		require.Len(t, olap.calls[0].orchestratorUpdates, 1)
		assert.Equal(t, sqlcv1.V1ReadableStatusOlapCOMPLETED, olap.calls[0].orchestratorUpdates[0].ReadableStatus)
	})

	t.Run("status-only is ignored for non-orchestrator tasks", func(t *testing.T) {
		tc, olap := newMonitoringTestController(child)

		childStatusOnly := childStarted
		childStatusOnly.StatusOnly = true

		err := tc.handleCreateMonitoringEvent(context.Background(), tenantId, monitoringPayloads(t, childStatusOnly))
		require.NoError(t, err)

		require.Len(t, olap.calls, 1)
		require.Len(t, olap.calls[0].events, 1)
		assert.Equal(t, child.ID, olap.calls[0].events[0].TaskID)
		assert.Empty(t, olap.calls[0].orchestratorUpdates)
	})
}
