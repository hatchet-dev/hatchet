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
	events              int
	retriedEvents       int
	orchestratorUpdates int
}

func (f *fakeMonitoringOLAP) CreateTaskEvents(_ context.Context, _ uuid.UUID, events []sqlcv1.CreateTaskEventsOLAPParams, _ map[uuid.UUID]uuid.UUID, retriedEventIds map[uuid.UUID]struct{}, orchestratorUpdates []v1.OrchestratorDAGStatusUpdateOpt, _ map[uuid.UUID]struct{}) (*v1.StatusUpdateResult, map[uuid.UUID]struct{}, error) {
	f.events += len(events)
	f.retriedEvents += len(retriedEventIds)
	f.orchestratorUpdates += len(orchestratorUpdates)
	return nil, nil, nil
}

func TestHandleCreateMonitoringEvent_OrchestratorStatusOnly(t *testing.T) {
	orchestrator := &sqlcv1.ListTaskMetasRow{
		ID:                1,
		InsertedAt:        sqlchelpers.TimestamptzFromTime(time.Now()),
		WorkflowRunID:     uuid.New(),
		IsDagOrchestrator: true,
	}

	finished := func(statusOnly bool) tasktypes.CreateMonitoringEventPayload {
		return tasktypes.CreateMonitoringEventPayload{
			TaskId:         orchestrator.ID,
			EventType:      sqlcv1.V1EventTypeOlapFINISHED,
			EventTimestamp: time.Now(),
			StatusOnly:     statusOnly,
		}
	}

	failed := func(willRetry bool) tasktypes.CreateMonitoringEventPayload {
		return tasktypes.CreateMonitoringEventPayload{
			TaskId:         orchestrator.ID,
			EventType:      sqlcv1.V1EventTypeOlapFAILED,
			EventTimestamp: time.Now(),
			WillRetry:      willRetry,
		}
	}

	tests := []struct {
		name                                 string
		msgs                                 []tasktypes.CreateMonitoringEventPayload
		wantEvents, wantRetried, wantUpdates int
	}{
		{
			name:        "regular and status-only terminal events write one task event",
			msgs:        []tasktypes.CreateMonitoringEventPayload{finished(false), finished(true)},
			wantEvents:  1,
			wantUpdates: 2,
		},
		{
			name:        "status-only batch still updates the DAG",
			msgs:        []tasktypes.CreateMonitoringEventPayload{finished(true)},
			wantEvents:  0,
			wantUpdates: 1,
		},
		{
			name:        "retried failure writes its event without a terminal update",
			msgs:        []tasktypes.CreateMonitoringEventPayload{failed(true)},
			wantEvents:  1,
			wantRetried: 1,
			wantUpdates: 0,
		},
		{
			name:        "final failure updates the DAG",
			msgs:        []tasktypes.CreateMonitoringEventPayload{failed(false)},
			wantEvents:  1,
			wantRetried: 0,
			wantUpdates: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := zerolog.Nop()
			olap := &fakeMonitoringOLAP{}
			tc := &OLAPControllerImpl{
				l:    &l,
				repo: &fakeMonitoringRepo{tasks: &fakeMonitoringTasks{metas: []*sqlcv1.ListTaskMetasRow{orchestrator}}, olap: olap},
			}

			payloads := make([][]byte, 0, len(tt.msgs))
			for _, msg := range tt.msgs {
				b, err := json.Marshal(msg)
				require.NoError(t, err)
				payloads = append(payloads, b)
			}

			require.NoError(t, tc.handleCreateMonitoringEvent(context.Background(), uuid.New(), payloads))
			assert.Equal(t, tt.wantEvents, olap.events)
			assert.Equal(t, tt.wantRetried, olap.retriedEvents)
			assert.Equal(t, tt.wantUpdates, olap.orchestratorUpdates)
		})
	}
}
