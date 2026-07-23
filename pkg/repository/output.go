package repository

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type TaskOutputEvent struct {
	IsFailure bool `json:"is_failure"`

	EventType sqlcv1.V1TaskEventType `json:"event_type"`

	TaskExternalId uuid.UUID `json:"task_external_id"`

	TaskId int64 `json:"task_id"`

	RetryCount int32 `json:"retry_count"`

	WorkerId *uuid.UUID `json:"worker_id"`

	Output []byte `json:"output"`

	ErrorMessage string `json:"error_message"`

	StepReadableID string `json:"step_readable_id"`

	IsDagOrchestrator bool `json:"is_dag_orchestrator"`
}

func (e *TaskOutputEvent) IsCompleted() bool {
	return e.EventType == sqlcv1.V1TaskEventTypeCOMPLETED
}

func (e *TaskOutputEvent) IsFailed() bool {
	return e.EventType == sqlcv1.V1TaskEventTypeFAILED
}

func (e *TaskOutputEvent) IsCancelled() bool {
	return e.EventType == sqlcv1.V1TaskEventTypeCANCELLED
}

func NewSkippedTaskOutputEventFromTask(task *V1TaskWithPayload) *TaskOutputEvent {
	outputMap := map[string]bool{
		"skipped": true,
	}

	outputMapBytes, _ := json.Marshal(outputMap) // nolint: errcheck

	e := baseFromTasksRow(task)
	e.Output = outputMapBytes
	e.EventType = sqlcv1.V1TaskEventTypeCOMPLETED

	if task.DesiredWorkerID != nil {
		e.WorkerId = task.DesiredWorkerID
	}

	return e
}

func NewFailedTaskOutputEventFromTask(task *V1TaskWithPayload) *TaskOutputEvent {
	e := baseFromTasksRow(task)
	e.IsFailure = true
	e.ErrorMessage = task.InitialStateReason.String
	e.EventType = sqlcv1.V1TaskEventTypeFAILED

	if task.DesiredWorkerID != nil {
		e.WorkerId = task.DesiredWorkerID
	}

	return e
}

func NewCancelledTaskOutputEventFromTask(task *V1TaskWithPayload) *TaskOutputEvent {
	e := baseFromTasksRow(task)
	e.EventType = sqlcv1.V1TaskEventTypeCANCELLED
	return e
}

func baseFromTasksRow(task *V1TaskWithPayload) *TaskOutputEvent {
	return &TaskOutputEvent{
		TaskExternalId: task.ExternalID,
		TaskId:         task.ID,
		RetryCount:     task.RetryCount,
		StepReadableID: task.StepReadableID,
	}
}

func NewCompletedTaskOutputEvent(row *sqlcv1.ReleaseTasksRow, output []byte) *TaskOutputEvent {
	e := baseFromReleaseTasksRow(row)
	e.Output = output
	e.EventType = sqlcv1.V1TaskEventTypeCOMPLETED

	if row.WorkerID != uuid.Nil {
		e.WorkerId = &row.WorkerID
	}

	return e
}

func NewFailedTaskOutputEvent(row *sqlcv1.ReleaseTasksRow, errorMsg string) *TaskOutputEvent {
	e := baseFromReleaseTasksRow(row)
	e.IsFailure = true
	e.ErrorMessage = errorMsg
	e.EventType = sqlcv1.V1TaskEventTypeFAILED
	return e
}

func NewCancelledTaskOutputEvent(row *sqlcv1.ReleaseTasksRow) *TaskOutputEvent {
	e := baseFromReleaseTasksRow(row)
	e.EventType = sqlcv1.V1TaskEventTypeCANCELLED
	return e
}

func baseFromReleaseTasksRow(row *sqlcv1.ReleaseTasksRow) *TaskOutputEvent {
	return &TaskOutputEvent{
		TaskExternalId:    row.ExternalID,
		TaskId:            row.ID,
		RetryCount:        row.RetryCount,
		StepReadableID:    row.StepReadableID,
		IsDagOrchestrator: row.IsDagOrchestrator,
	}
}

func (e *TaskOutputEvent) Bytes() []byte {
	resBytes, err := json.Marshal(e)

	if err != nil {
		return []byte("{}")
	}

	return resBytes
}

func newTaskEventFromBytes(b []byte) (*TaskOutputEvent, error) {
	var e TaskOutputEvent

	err := json.Unmarshal(b, &e)

	return &e, err
}

func ExtractOutputFromMatchData(data []byte) ([]byte, error) {
	// note: this is kind of an unfortunate method we need for durable execution in order to return
	// the result payload of the child that was spawned from a durable task to the parent.
	// it's confusing because in other places we use the task events on the SDK itself to aggregate
	// the outputs of each child into a map, but here we do it on the engine. it'd be a good fixme for the future
	// to consolidate the different ways we handle this kind of thing in different places to be more consistent.
	// I (Matt) opted to do it this way in https://github.com/hatchet-dev/hatchet/pull/4008 to maintain
	// backwards compatibility, and because it was the simplest bug fix for the issue we saw at the time.
	var outer map[string]map[string][]json.RawMessage
	if err := json.Unmarshal(data, &outer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal match data: %w", err)
	}

	for _, keyMap := range outer {
		if len(keyMap) == 0 {
			continue
		}

		if len(keyMap) == 1 {
			// this is a special case: if the child is a DAG, we want to return the output
			// of each task in the DAG keyed by the task name (below), but if it's a standalone task,
			// we want to return the task's output directly without the extra nesting, so we have to handle this
			// situation separately
			if entries, ok := keyMap["output"]; ok && len(entries) > 0 {
				var event TaskOutputEvent
				if err := json.Unmarshal(entries[0], &event); err != nil {
					return nil, fmt.Errorf("failed to unmarshal task output event from match data: %w", err)
				}

				return event.Output, nil
			}
		}

		aggregated := make(map[string]json.RawMessage, len(keyMap))
		for key, entries := range keyMap {
			if len(entries) == 0 {
				continue
			}

			var event TaskOutputEvent
			if err := json.Unmarshal(entries[0], &event); err != nil {
				return nil, fmt.Errorf("failed to unmarshal task output event from match data: %w", err)
			}

			if len(event.Output) > 0 {
				aggregated[key] = json.RawMessage(event.Output)
			}
		}

		result, err := json.Marshal(aggregated)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal aggregated DAG output: %w", err)
		}

		return result, nil
	}

	return nil, fmt.Errorf("no entries found in match data")
}

const TaskCancelledErrorMessage = "task was cancelled"

func ExtractFailureFromMatchData(data []byte) (bool, *string, error) {
	var outer map[string]map[string][]json.RawMessage
	if err := json.Unmarshal(data, &outer); err != nil {
		return false, nil, err
	}

	for _, keyMap := range outer {
		for _, entries := range keyMap {
			if len(entries) == 0 {
				continue
			}
			var event TaskOutputEvent
			if err := json.Unmarshal(entries[0], &event); err != nil {
				continue
			}
			if event.IsFailure {
				return true, &event.ErrorMessage, nil
			}
			if event.EventType == sqlcv1.V1TaskEventTypeCANCELLED {
				msg := TaskCancelledErrorMessage
				return true, &msg, nil
			}
		}
	}
	return false, nil, nil
}
