package repository

import (
	"encoding/json"

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
		TaskExternalId: row.ExternalID,
		TaskId:         row.ID,
		RetryCount:     row.RetryCount,
		StepReadableID: row.StepReadableID,
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
