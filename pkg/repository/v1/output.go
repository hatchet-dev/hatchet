package v1

import (
	"encoding/json"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type TaskOutputEvent struct {
	IsFailure bool `json:"is_failure"`

	EventType sqlcv1.V1TaskEventType `json:"event_type"`

	TaskExternalId string `json:"task_external_id"`

	TaskId int64 `json:"task_id"`

	RetryCount int32 `json:"retry_count"`

	WorkerId *string `json:"worker_id"`

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

func NewSkippedTaskOutputEventFromTask(task *sqlcv1.V1Task) *TaskOutputEvent {
	outputMap := map[string]bool{
		"skipped": true,
	}

	outputMapBytes, _ := json.Marshal(outputMap) // nolint: errcheck

	e := baseFromTasksRow(task)
	e.Output = outputMapBytes
	e.EventType = sqlcv1.V1TaskEventTypeCOMPLETED

	if task.DesiredWorkerID.Valid {
		workerId := sqlchelpers.UUIDToStr(task.DesiredWorkerID)
		e.WorkerId = &workerId
	}

	return e
}

func NewFailedTaskOutputEventFromTask(task *sqlcv1.V1Task) *TaskOutputEvent {
	e := baseFromTasksRow(task)
	e.IsFailure = true
	e.ErrorMessage = task.InitialStateReason.String
	e.EventType = sqlcv1.V1TaskEventTypeFAILED

	if task.DesiredWorkerID.Valid {
		workerId := sqlchelpers.UUIDToStr(task.DesiredWorkerID)
		e.WorkerId = &workerId
	}

	return e
}

func NewCancelledTaskOutputEventFromTask(task *sqlcv1.V1Task) *TaskOutputEvent {
	e := baseFromTasksRow(task)
	e.EventType = sqlcv1.V1TaskEventTypeCANCELLED
	return e
}

func baseFromTasksRow(task *sqlcv1.V1Task) *TaskOutputEvent {
	return &TaskOutputEvent{
		TaskExternalId: sqlchelpers.UUIDToStr(task.ExternalID),
		TaskId:         task.ID,
		RetryCount:     task.RetryCount,
		StepReadableID: task.StepReadableID,
	}
}

func NewCompletedTaskOutputEvent(row *sqlcv1.ReleaseTasksRow, output []byte) *TaskOutputEvent {
	e := baseFromReleaseTasksRow(row)
	e.Output = output
	e.EventType = sqlcv1.V1TaskEventTypeCOMPLETED

	if row.WorkerID.Valid {
		workerId := sqlchelpers.UUIDToStr(row.WorkerID)
		e.WorkerId = &workerId
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
		TaskExternalId: sqlchelpers.UUIDToStr(row.ExternalID),
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
