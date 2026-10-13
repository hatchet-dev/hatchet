package repository

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type TaskInput struct {
	Input map[string]interface{} `json:"input"`

	TriggerData *MatchData `json:"trigger_datas"`

	FilterPayload map[string]interface{} `json:"filter_payload"`

	// task run external IDs of parent tasks in a durable DAG orchestration
	DagParentTaskRunIds []uuid.UUID `json:"dag_parent_task_run_ids,omitempty"`

	// run-level desired worker labels, carried on the DAG orchestrator task so it can
	// propagate them to the child steps it triggers (child steps route to the same worker pool)
	DesiredWorkerLabels []*sqlcv1.GetDesiredLabelsRow `json:"desired_worker_labels,omitempty"`
}

func (s *sharedRepository) DesiredWorkerId(t *TaskInput) *uuid.UUID {
	if t.TriggerData != nil {
		for _, stepReadableId := range t.TriggerData.DataKeys() {
			data := t.TriggerData.DataValueAsTaskOutputEvent(stepReadableId)

			return data.WorkerId
		}
	}

	return nil
}

func (s *sharedRepository) newTaskInputFromTaskInputPayload(taskInputPayload []byte) *TaskInput {
	i := &TaskInput{}

	err := json.Unmarshal(UserInputFromTaskInputPayload(taskInputPayload), &i.Input)

	if err != nil {
		s.l.Error().Err(err).Msg("failed to unmarshal input bytes")
	}

	return i
}

func (s *sharedRepository) newTaskInput(inputBytes []byte, triggerData *MatchData, filterPayload []byte, dagParentTaskRunIds []uuid.UUID) *TaskInput {
	var input map[string]interface{}

	if len(inputBytes) > 0 {
		err := json.Unmarshal(inputBytes, &input)

		if err != nil {
			s.l.Error().Err(err).Msg("failed to unmarshal input bytes")
		}
	}

	var filterPayloadMap map[string]interface{}
	if len(filterPayload) > 0 {
		err := json.Unmarshal(filterPayload, &filterPayloadMap)
		if err != nil {
			s.l.Error().Err(err).Msg("failed to unmarshal event filter payload bytes")
		}
	}

	return &TaskInput{
		Input:               input,
		TriggerData:         triggerData,
		FilterPayload:       filterPayloadMap,
		DagParentTaskRunIds: dagParentTaskRunIds,
	}
}

func (t *TaskInput) Bytes() []byte {
	if t == nil {
		return nil
	}

	out, err := json.Marshal(t)

	if err != nil {
		return nil
	}

	return out
}

func (s *sharedRepository) ToV1StepRunData(t *TaskInput) *V1StepRunData {
	if t == nil {
		return nil
	}

	parents := make(map[string]map[string]interface{})
	triggers := make(map[string]map[string]interface{})

	if t.TriggerData != nil {
		for _, stepReadableId := range t.TriggerData.DataKeys() {
			data := t.TriggerData.DataValueAsTaskOutputEvent(stepReadableId)

			if !data.IsCompleted() {
				continue
			}

			dataMap := make(map[string]interface{})

			err := json.Unmarshal(data.Output, &dataMap)

			if err != nil {
				s.l.Warn().Err(err).Msg("failed to unmarshal output")
			}

			parents[stepReadableId] = dataMap
		}

		for _, key := range t.TriggerData.TriggerDataKeys() {
			dataMap := t.TriggerData.TriggerDataValue(key)

			triggers[key] = dataMap
		}
	}

	triggers["filter_payload"] = t.FilterPayload

	input, err := json.Marshal(t.Input)

	if err != nil {
		s.l.Warn().Err(err).Msg("failed to marshal input")
	}

	return &V1StepRunData{
		Input:               input,
		Parents:             parents,
		Triggers:            triggers,
		DagParentTaskRunIds: t.DagParentTaskRunIds,
		DesiredWorkerLabels: t.DesiredWorkerLabels,
	}
}

// legacyStepRunDataFromTaskInputPayload recognises task inputs stored before the payload held only the user's
// input. Those rows hold a whole V1StepRunData, which always has these three keys. A user's input would have to
// contain all three to be mistaken for one. This can be deleted once no such rows remain in the payload store.
func legacyStepRunDataFromTaskInputPayload(taskInputPayload []byte) *V1StepRunData {
	var topLevelKeys map[string]json.RawMessage

	if err := json.Unmarshal(taskInputPayload, &topLevelKeys); err != nil {
		return nil
	}

	for _, key := range []string{"input", "parents", "triggers"} {
		if _, ok := topLevelKeys[key]; !ok {
			return nil
		}
	}

	legacyStepRunData := &V1StepRunData{}

	if err := json.Unmarshal(taskInputPayload, legacyStepRunData); err != nil {
		return nil
	}

	return legacyStepRunData
}

func UserInputFromTaskInputPayload(taskInputPayload []byte) []byte {
	if legacyStepRunData := legacyStepRunDataFromTaskInputPayload(taskInputPayload); legacyStepRunData != nil {
		return legacyStepRunData.Input
	}

	return taskInputPayload
}

// NewV1StepRunDataFromTaskAndInputPayload leaves Parents empty and StepRunErrors unset. The dispatcher fills both in from the
// task events of the task's parents.
func NewV1StepRunDataFromTaskAndInputPayload(task *sqlcv1.V1Task, taskInputPayload []byte) (*V1StepRunData, error) {
	if legacyStepRunData := legacyStepRunDataFromTaskInputPayload(taskInputPayload); legacyStepRunData != nil {
		return legacyStepRunData, nil
	}

	triggers := map[string]map[string]interface{}{"filter_payload": nil}

	if len(task.Triggers) > 0 {
		if err := json.Unmarshal(task.Triggers, &triggers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal triggers: %w", err)
		}
	}

	var childTaskDesiredWorkerLabels []*sqlcv1.GetDesiredLabelsRow

	if len(task.ChildTaskDesiredWorkerLabels) > 0 {
		if err := json.Unmarshal(task.ChildTaskDesiredWorkerLabels, &childTaskDesiredWorkerLabels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal child task desired worker labels: %w", err)
		}
	}

	input := json.RawMessage(taskInputPayload)

	if len(input) == 0 {
		input = json.RawMessage("{}")
	}

	return &V1StepRunData{
		Input:               input,
		Parents:             make(map[string]map[string]interface{}),
		Triggers:            triggers,
		DagParentTaskRunIds: task.DagParentTaskRunIds,
		DesiredWorkerLabels: childTaskDesiredWorkerLabels,
	}, nil
}

type V1StepRunData struct {
	Input    json.RawMessage                   `json:"input"`
	Parents  map[string]map[string]interface{} `json:"parents"`
	Triggers map[string]map[string]interface{} `json:"triggers"`

	// errors in upstream steps, read from their task events when the task is dispatched
	StepRunErrors map[string]string `json:"step_run_errors,omitempty"`

	// task run external IDs of parent tasks in a durable DAG orchestration
	DagParentTaskRunIds []uuid.UUID `json:"dag_parent_task_run_ids,omitempty"`

	// run-level desired worker labels, carried on the DAG orchestrator task so it can
	// propagate them to the child steps it triggers
	DesiredWorkerLabels []*sqlcv1.GetDesiredLabelsRow `json:"desired_worker_labels,omitempty"`
}

// userInput is null for a task created without an input, which matches what Bytes returns for it.
func (v1 *V1StepRunData) userInput() []byte {
	if v1 == nil {
		return []byte("null")
	}

	return v1.Input
}

func (v1 *V1StepRunData) triggersColumn() ([]byte, error) {
	if v1 == nil {
		return nil, nil
	}

	hasOnlyAnEmptyFilterPayload := len(v1.Triggers) == 1 && v1.Triggers["filter_payload"] == nil

	if hasOnlyAnEmptyFilterPayload {
		return nil, nil
	}

	return json.Marshal(v1.Triggers)
}

func (v1 *V1StepRunData) dagParentTaskRunIdsColumn() ([]byte, error) {
	if v1 == nil || len(v1.DagParentTaskRunIds) == 0 {
		return nil, nil
	}

	return json.Marshal(v1.DagParentTaskRunIds)
}

func (v1 *V1StepRunData) childTaskDesiredWorkerLabelsColumn() ([]byte, error) {
	if v1 == nil || len(v1.DesiredWorkerLabels) == 0 {
		return nil, nil
	}

	return json.Marshal(v1.DesiredWorkerLabels)
}

func (v1 *V1StepRunData) Bytes() []byte {
	out, err := json.Marshal(v1)

	if err != nil {
		return []byte("{}")
	}

	return out
}
