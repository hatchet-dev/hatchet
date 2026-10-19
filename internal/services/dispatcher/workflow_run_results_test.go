package dispatcher

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// fakeOrchestratorChildren serves the child output events of one DAG orchestrator, the way
// ListDurableOrchestratorChildOutputEvents does for the dispatcher's result assembly.
type fakeOrchestratorChildren struct {
	v1.TaskRepository
	orchestratorExternalId uuid.UUID
	children               []*v1.TaskOutputEvent
	calls                  int
}

func (f *fakeOrchestratorChildren) ListDurableOrchestratorChildOutputEvents(_ context.Context, _ uuid.UUID, orchestratorExternalId uuid.UUID) ([]*v1.TaskOutputEvent, error) {
	f.calls++

	if orchestratorExternalId != f.orchestratorExternalId {
		return nil, nil
	}

	return f.children, nil
}

type fakeResultsRepo struct {
	v1.Repository
	tasks *fakeOrchestratorChildren
}

func (f fakeResultsRepo) Tasks() v1.TaskRepository { return f.tasks }

func completedEvent(externalId uuid.UUID, readableId string, output string) *v1.TaskOutputEvent {
	return &v1.TaskOutputEvent{
		EventType:      sqlcv1.V1TaskEventTypeCOMPLETED,
		TaskExternalId: externalId,
		StepReadableID: readableId,
		Output:         []byte(output),
	}
}

func failedEvent(externalId uuid.UUID, readableId string, errorMessage string) *v1.TaskOutputEvent {
	return &v1.TaskOutputEvent{
		EventType:      sqlcv1.V1TaskEventTypeFAILED,
		IsFailure:      true,
		TaskExternalId: externalId,
		StepReadableID: readableId,
		ErrorMessage:   errorMessage,
	}
}

func cancelledEvent(externalId uuid.UUID, readableId string) *v1.TaskOutputEvent {
	return &v1.TaskOutputEvent{
		EventType:      sqlcv1.V1TaskEventTypeCANCELLED,
		TaskExternalId: externalId,
		StepReadableID: readableId,
	}
}

type resultSummary struct {
	taskName string
	output   string
	err      string
	hasErr   bool
}

func summarize(results []*contracts.StepRunResult) []resultSummary {
	out := make([]resultSummary, 0, len(results))

	for _, r := range results {
		s := resultSummary{taskName: r.TaskName}

		if r.Output != nil {
			s.output = *r.Output
		}

		if r.Error != nil {
			s.hasErr = true
			s.err = *r.Error
		}

		out = append(out, s)
	}

	return out
}

// TestTaskEventsToWorkflowRunEvent_OperatorDag covers the FINISHED workflow-run event the
// dispatcher sends for a DAG run executed by the DAG operator. The run is a single
// orchestrator task whose finalizing event carries is_dag_orchestrator; its children are
// separate runs looked up through ListDurableOrchestratorChildOutputEvents.
func TestTaskEventsToWorkflowRunEvent_OperatorDag(t *testing.T) {
	const orchestratorError = `dag failed: non-determinism error in task "TaskConditionWorkflow": expected kind WAIT_FOR at node 8:1`

	orchestratorId := uuid.New()
	startId := uuid.New()
	sumId := uuid.New()
	workflowRunId := uuid.New()

	tests := []struct {
		name         string
		orchestrator *v1.TaskOutputEvent
		children     []*v1.TaskOutputEvent
		want         []resultSummary
		wantLookups  int
	}{
		{
			// The CI symptom: the orchestrator FAILED (non-determinism on replay, should not
			// retry) after every child COMPLETED. The SDKs decide success by the presence of an
			// error in the results, so the orchestrator's error has to be one of them.
			name:         "failed orchestrator with completed children carries the orchestrator error",
			orchestrator: failedEvent(orchestratorId, "TaskConditionWorkflow", orchestratorError),
			children: []*v1.TaskOutputEvent{
				completedEvent(startId, "start", `{"random_number":4}`),
				completedEvent(sumId, "sum", `{"sum":9}`),
			},
			want: []resultSummary{
				{taskName: "start", output: `{"random_number":4}`},
				{taskName: "sum", output: `{"sum":9}`},
				{taskName: "TaskConditionWorkflow", err: orchestratorError, hasErr: true},
			},
			wantLookups: 1,
		},
		{
			// An ordinary task failure: the child's result carries the error and the
			// orchestrator's "dag failed: child task ... failed" only restates it, so the
			// result set matches what the non-operator path sends for a failed DAG task.
			name:         "failed child carries the error and the orchestrator stays hidden",
			orchestrator: failedEvent(orchestratorId, "TaskConditionWorkflow", `dag failed: child task "default:sum" failed: boom`),
			children: []*v1.TaskOutputEvent{
				completedEvent(startId, "start", `{"random_number":4}`),
				failedEvent(sumId, "sum", "boom"),
			},
			want: []resultSummary{
				{taskName: "start", output: `{"random_number":4}`},
				{taskName: "sum", err: "boom", hasErr: true},
			},
			wantLookups: 1,
		},
		{
			// Cancellation: the children's CANCELLED results already represent it and carry no
			// message, and neither does the orchestrator, so the result set is unchanged.
			name:         "cancelled orchestrator with cancelled children stays hidden",
			orchestrator: cancelledEvent(orchestratorId, "TaskConditionWorkflow"),
			children: []*v1.TaskOutputEvent{
				completedEvent(startId, "start", `{"random_number":4}`),
				cancelledEvent(sumId, "sum"),
			},
			want: []resultSummary{
				{taskName: "start", output: `{"random_number":4}`},
				{taskName: "sum", hasErr: true},
			},
			wantLookups: 1,
		},
		{
			name:         "completed orchestrator expands its output map without a child lookup",
			orchestrator: completedEvent(orchestratorId, "TaskConditionWorkflow", `{"start":{"random_number":4}}`),
			want: []resultSummary{
				{taskName: "start", output: `{"random_number":4}`},
			},
			wantLookups: 0,
		},
		{
			name:         "failed orchestrator without children carries its own error",
			orchestrator: failedEvent(orchestratorId, "TaskConditionWorkflow", orchestratorError),
			want: []resultSummary{
				{taskName: "TaskConditionWorkflow", err: orchestratorError, hasErr: true},
			},
			wantLookups: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.orchestrator.IsDagOrchestrator = true

			tasks := &fakeOrchestratorChildren{
				orchestratorExternalId: orchestratorId,
				children:               tc.children,
			}

			s := &DispatcherImpl{repov1: fakeResultsRepo{tasks: tasks}}

			events, err := s.taskEventsToWorkflowRunEvent(context.Background(), uuid.New(), []*v1.ListFinalizedWorkflowRunsResponse{
				{
					WorkflowRunId: workflowRunId,
					OutputEvents:  []*v1.TaskOutputEvent{tc.orchestrator},
				},
			})
			require.NoError(t, err)
			require.Len(t, events, 1)

			event := events[0]
			require.Equal(t, workflowRunId.String(), event.WorkflowRunId)
			require.Equal(t, contracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED, event.EventType)
			require.Equal(t, tc.wantLookups, tasks.calls)
			require.Equal(t, tc.want, summarize(event.Results))

			for _, r := range event.Results {
				require.Equal(t, r.TaskRunExternalId, r.JobRunId)
			}
		})
	}
}
