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

func (s *sharedRepository) newTaskInputFromTaskInputPayload(version taskInputPayloadVersion, taskInputPayload []byte) *TaskInput {
	i := &TaskInput{}

	var userInput []byte

	switch version {
	case taskInputPayloadVersionStepRunData:
		stepRunData, err := stepRunDataFromVersion1TaskInputPayload(taskInputPayload)

		if err != nil {
			s.l.Error().Err(err).Msg("failed to unmarshal step run data")
			return i
		}

		userInput = stepRunData.Input
		i.DagParentTaskRunIds = stepRunData.DagParentTaskRunIds
		i.DesiredWorkerLabels = stepRunData.DesiredWorkerLabels
	case taskInputPayloadVersionUserInputOnly:
		userInput = taskInputPayload
	default:
		s.l.Error().Msgf("unknown task input payload version %d", version)
		return i
	}

	if len(userInput) == 0 {
		return i
	}

	if err := json.Unmarshal(userInput, &i.Input); err != nil {
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

	// empty or `null` input bytes leave the map nil, which would be stored as a `null` TASK_INPUT row
	if input == nil {
		input = map[string]interface{}{}
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

func (s *sharedRepository) sharedRunInputPayloadReferenceId(runPayloadReferenceId *uuid.UUID) *uuid.UUID {
	if !s.enablePayloadConsolidation {
		return nil
	}

	return runPayloadReferenceId
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
	stepRunErrors := make(map[string]string)

	if t.TriggerData != nil {
		for _, stepReadableId := range t.TriggerData.DataKeys() {
			data := t.TriggerData.DataValueAsTaskOutputEvent(stepReadableId)

			switch {
			case data.IsCompleted():
				dataMap := make(map[string]interface{})

				err := json.Unmarshal(data.Output, &dataMap)

				if err != nil {
					s.l.Warn().Err(err).Msg("failed to unmarshal output")
				}

				parents[stepReadableId] = dataMap
			case data.IsFailed():
				stepRunErrors[stepReadableId] = data.ErrorMessage
			}
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
		StepRunErrors:       stepRunErrors,
		DagParentTaskRunIds: t.DagParentTaskRunIds,
		DesiredWorkerLabels: t.DesiredWorkerLabels,
	}
}

type taskInputPayloadVersion int16

const (
	taskInputPayloadVersionStepRunData   taskInputPayloadVersion = 1
	taskInputPayloadVersionUserInputOnly taskInputPayloadVersion = 2
)

// Payload consolidation shares one input row across a run, which only works once every row holds just the user input.
func (s *sharedRepository) taskInputPayloadVersionToWrite(isDagOrchestrator bool) taskInputPayloadVersion {
	if s.wrapTaskInputInStepRunData && !s.enablePayloadConsolidation && !isDagOrchestrator {
		return taskInputPayloadVersionStepRunData
	}

	return taskInputPayloadVersionUserInputOnly
}

func stepRunDataFromVersion1TaskInputPayload(taskInputPayload []byte) (*V1StepRunData, error) {
	stepRunData := &V1StepRunData{}

	if len(taskInputPayload) == 0 {
		return stepRunData, nil
	}

	if err := json.Unmarshal(taskInputPayload, stepRunData); err != nil {
		return nil, err
	}

	return stepRunData, nil
}

func userInputFromTaskInputPayload(version taskInputPayloadVersion, taskInputPayload []byte) ([]byte, error) {
	switch version {
	case taskInputPayloadVersionStepRunData:
		stepRunData, err := stepRunDataFromVersion1TaskInputPayload(taskInputPayload)

		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal step run data: %w", err)
		}

		return stepRunData.Input, nil
	case taskInputPayloadVersionUserInputOnly:
		return taskInputPayload, nil
	default:
		return nil, fmt.Errorf("unknown task input payload version %d", version)
	}
}

func NewV1StepRunDataFromTaskAndInputPayload(task *sqlcv1.V1Task, taskInputPayload []byte) (*V1StepRunData, error) {
	switch version := taskInputPayloadVersion(task.TaskInputPayloadVersion); version {
	case taskInputPayloadVersionStepRunData:
		return stepRunDataFromVersion1TaskInputPayload(taskInputPayload)
	case taskInputPayloadVersionUserInputOnly:
		return newV1StepRunDataFromTaskAndUserInput(task, taskInputPayload)
	default:
		return nil, fmt.Errorf("unknown task input payload version %d", version)
	}
}

// newV1StepRunDataFromTaskAndUserInput leaves Parents empty and StepRunErrors unset. The dispatcher fills both in
// from the task events of the task's parents.
func newV1StepRunDataFromTaskAndUserInput(task *sqlcv1.V1Task, userInput []byte) (*V1StepRunData, error) {
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

	input := json.RawMessage(userInput)

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

	// errors in upstream steps. Set at task creation for engines that read the wrapped payload, and overwritten
	// from the parents' task events when the task is dispatched.
	StepRunErrors map[string]string `json:"step_run_errors,omitempty"`

	// task run external IDs of parent tasks in a durable DAG orchestration
	DagParentTaskRunIds []uuid.UUID `json:"dag_parent_task_run_ids,omitempty"`

	// run-level desired worker labels, carried on the DAG orchestrator task so it can
	// propagate them to the child steps it triggers
	DesiredWorkerLabels []*sqlcv1.GetDesiredLabelsRow `json:"desired_worker_labels,omitempty"`
}

func (v1 *V1StepRunData) taskInputPayload(version taskInputPayloadVersion) []byte {
	if v1 == nil {
		return nil
	}

	switch version {
	case taskInputPayloadVersionStepRunData:
		return v1.Bytes()
	case taskInputPayloadVersionUserInputOnly:
		return v1.Input
	default:
		panic(fmt.Sprintf("unknown task input payload version %d", version))
	}
}

type stepRunDataTaskColumns struct {
	triggers                     []byte
	dagParentTaskRunIds          []byte
	childTaskDesiredWorkerLabels []byte
}

func (v1 *V1StepRunData) taskColumns() (stepRunDataTaskColumns, error) {
	columns := stepRunDataTaskColumns{}

	if v1 == nil {
		return columns, nil
	}

	var err error

	if len(v1.Triggers) > 0 {
		if columns.triggers, err = json.Marshal(v1.Triggers); err != nil {
			return columns, fmt.Errorf("failed to marshal triggers: %w", err)
		}
	}

	if len(v1.DagParentTaskRunIds) > 0 {
		if columns.dagParentTaskRunIds, err = json.Marshal(v1.DagParentTaskRunIds); err != nil {
			return columns, fmt.Errorf("failed to marshal dag parent task run ids: %w", err)
		}
	}

	if len(v1.DesiredWorkerLabels) > 0 {
		if columns.childTaskDesiredWorkerLabels, err = json.Marshal(v1.DesiredWorkerLabels); err != nil {
			return columns, fmt.Errorf("failed to marshal child task desired worker labels: %w", err)
		}
	}

	return columns, nil
}

func (v1 *V1StepRunData) Bytes() []byte {
	out, err := json.Marshal(v1)

	if err != nil {
		return []byte("{}")
	}

	return out
}
