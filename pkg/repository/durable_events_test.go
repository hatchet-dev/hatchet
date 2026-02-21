//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string { return &s }

func key(t *testing.T, opts IngestDurableTaskEventOpts) string {
	t.Helper()
	r := &durableEventsRepository{}
	k, err := r.createIdempotencyKey(opts)
	assert.NoError(t, err)
	return string(k)
}

func TestCreateIdempotencyKey_ConditionOrderInvariant(t *testing.T) {
	condA := CreateExternalSignalConditionOpt{
		Kind:            CreateExternalSignalConditionKindSLEEP,
		Expression:      "aaa",
		ReadableDataKey: "output",
		SleepFor:        strPtr("10s"),
	}
	condB := CreateExternalSignalConditionOpt{
		Kind:            CreateExternalSignalConditionKindUSEREVENT,
		Expression:      "bbb",
		ReadableDataKey: "output",
		UserEventKey:    strPtr("some-event"),
	}

	optsAB := IngestDurableTaskEventOpts{
		Kind:              sqlcv1.V1DurableEventLogKindWAITFOR,
		WaitForConditions: []CreateExternalSignalConditionOpt{condA, condB},
	}
	optsBA := IngestDurableTaskEventOpts{
		Kind:              sqlcv1.V1DurableEventLogKindWAITFOR,
		WaitForConditions: []CreateExternalSignalConditionOpt{condB, condA},
	}

	assert.Equal(t, key(t, optsAB), key(t, optsBA))
}

func TestCreateIdempotencyKey_DifferentConditions(t *testing.T) {
	base := IngestDurableTaskEventOpts{
		Kind: sqlcv1.V1DurableEventLogKindWAITFOR,
		WaitForConditions: []CreateExternalSignalConditionOpt{
			{Kind: CreateExternalSignalConditionKindSLEEP, Expression: "true", ReadableDataKey: "output", SleepFor: strPtr("5s")},
		},
	}
	different := IngestDurableTaskEventOpts{
		Kind: sqlcv1.V1DurableEventLogKindWAITFOR,
		WaitForConditions: []CreateExternalSignalConditionOpt{
			{Kind: CreateExternalSignalConditionKindSLEEP, Expression: "true", ReadableDataKey: "output", SleepFor: strPtr("30s")},
		},
	}

	assert.NotEqual(t, key(t, base), key(t, different))
}

func TestCreateIdempotencyKey_DifferentKind(t *testing.T) {
	run := IngestDurableTaskEventOpts{Kind: sqlcv1.V1DurableEventLogKindRUN}
	waitFor := IngestDurableTaskEventOpts{Kind: sqlcv1.V1DurableEventLogKindWAITFOR}
	memo := IngestDurableTaskEventOpts{Kind: sqlcv1.V1DurableEventLogKindMEMO}

	assert.NotEqual(t, key(t, run), key(t, waitFor))
	assert.NotEqual(t, key(t, run), key(t, memo))
	assert.NotEqual(t, key(t, waitFor), key(t, memo))
}

func TestCreateIdempotencyKey_DifferentWorkflowName(t *testing.T) {
	optsA := IngestDurableTaskEventOpts{
		Kind: sqlcv1.V1DurableEventLogKindRUN,
		TriggerOpts: &WorkflowNameTriggerOpts{
			TriggerTaskData: &TriggerTaskData{WorkflowName: "workflow-a"},
		},
	}
	optsB := IngestDurableTaskEventOpts{
		Kind: sqlcv1.V1DurableEventLogKindRUN,
		TriggerOpts: &WorkflowNameTriggerOpts{
			TriggerTaskData: &TriggerTaskData{WorkflowName: "workflow-b"},
		},
	}

	assert.NotEqual(t, key(t, optsA), key(t, optsB))
}

func TestCreateIdempotencyKey_DifferentTriggerData(t *testing.T) {
	optsA := IngestDurableTaskEventOpts{
		Kind: sqlcv1.V1DurableEventLogKindRUN,
		TriggerOpts: &WorkflowNameTriggerOpts{
			TriggerTaskData: &TriggerTaskData{WorkflowName: "my-workflow", Data: []byte(`{"x":1}`)},
		},
	}
	optsB := IngestDurableTaskEventOpts{
		Kind: sqlcv1.V1DurableEventLogKindRUN,
		TriggerOpts: &WorkflowNameTriggerOpts{
			TriggerTaskData: &TriggerTaskData{WorkflowName: "my-workflow", Data: []byte(`{"x":2}`)},
		},
	}

	assert.NotEqual(t, key(t, optsA), key(t, optsB))
}

func TestCreateIdempotencyKey_WithAndWithoutTriggerOpts(t *testing.T) {
	without := IngestDurableTaskEventOpts{Kind: sqlcv1.V1DurableEventLogKindRUN}
	with := IngestDurableTaskEventOpts{
		Kind: sqlcv1.V1DurableEventLogKindRUN,
		TriggerOpts: &WorkflowNameTriggerOpts{
			TriggerTaskData: &TriggerTaskData{WorkflowName: "my-workflow"},
		},
	}

	assert.NotEqual(t, key(t, without), key(t, with))
}

func int32Ptr(i int32) *int32 { return &i }

func TestCreateIdempotencyKey_PriorityIgnored(t *testing.T) {
	base := IngestDurableTaskEventOpts{
		Kind: sqlcv1.V1DurableEventLogKindRUN,
		TriggerOpts: &WorkflowNameTriggerOpts{
			TriggerTaskData: &TriggerTaskData{WorkflowName: "my-workflow", Data: []byte(`{"x":1}`)},
		},
	}
	withPriority := IngestDurableTaskEventOpts{
		Kind: sqlcv1.V1DurableEventLogKindRUN,
		TriggerOpts: &WorkflowNameTriggerOpts{
			TriggerTaskData: &TriggerTaskData{WorkflowName: "my-workflow", Data: []byte(`{"x":1}`), Priority: int32Ptr(3)},
		},
	}

	assert.Equal(t, key(t, base), key(t, withPriority))
}

func TestCreateIdempotencyKey_AdditionalMetadataIgnored(t *testing.T) {
	base := IngestDurableTaskEventOpts{
		Kind: sqlcv1.V1DurableEventLogKindRUN,
		TriggerOpts: &WorkflowNameTriggerOpts{
			TriggerTaskData: &TriggerTaskData{WorkflowName: "my-workflow", Data: []byte(`{"x":1}`)},
		},
	}
	withMeta := IngestDurableTaskEventOpts{
		Kind: sqlcv1.V1DurableEventLogKindRUN,
		TriggerOpts: &WorkflowNameTriggerOpts{
			TriggerTaskData: &TriggerTaskData{
				WorkflowName:       "my-workflow",
				Data:               []byte(`{"x":1}`),
				AdditionalMetadata: []byte(`{"env":"prod"}`),
			},
		},
	}

	assert.Equal(t, key(t, base), key(t, withMeta))
}
