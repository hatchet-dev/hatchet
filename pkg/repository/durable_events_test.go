//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string { return &s }

func keyFromKind(t *testing.T, kind sqlcv1.V1DurableEventLogKind, triggerOpts *WorkflowNameTriggerOpts, waitForConditions []CreateExternalSignalConditionOpt) string {
	t.Helper()
	r := &durableEventsRepository{}
	k, err := r.createIdempotencyKey(kind, triggerOpts, waitForConditions)
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

	keyAB := keyFromKind(t, sqlcv1.V1DurableEventLogKindWAITFOR, nil, []CreateExternalSignalConditionOpt{condA, condB})
	keyBA := keyFromKind(t, sqlcv1.V1DurableEventLogKindWAITFOR, nil, []CreateExternalSignalConditionOpt{condB, condA})

	assert.Equal(t, keyAB, keyBA)
}

func TestCreateIdempotencyKey_DifferentConditions(t *testing.T) {
	base := keyFromKind(t, sqlcv1.V1DurableEventLogKindWAITFOR, nil, []CreateExternalSignalConditionOpt{
		{Kind: CreateExternalSignalConditionKindSLEEP, Expression: "true", ReadableDataKey: "output", SleepFor: strPtr("5s")},
	})
	different := keyFromKind(t, sqlcv1.V1DurableEventLogKindWAITFOR, nil, []CreateExternalSignalConditionOpt{
		{Kind: CreateExternalSignalConditionKindSLEEP, Expression: "true", ReadableDataKey: "output", SleepFor: strPtr("30s")},
	})

	assert.NotEqual(t, base, different)
}

func TestCreateIdempotencyKey_DifferentKind(t *testing.T) {
	run := keyFromKind(t, sqlcv1.V1DurableEventLogKindRUN, nil, nil)
	waitFor := keyFromKind(t, sqlcv1.V1DurableEventLogKindWAITFOR, nil, nil)
	memo := keyFromKind(t, sqlcv1.V1DurableEventLogKindMEMO, nil, nil)

	assert.NotEqual(t, run, waitFor)
	assert.NotEqual(t, run, memo)
	assert.NotEqual(t, waitFor, memo)
}

func TestCreateIdempotencyKey_DifferentWorkflowName(t *testing.T) {
	keyA := keyFromKind(t, sqlcv1.V1DurableEventLogKindRUN, &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{WorkflowName: "workflow-a"},
	}, nil)
	keyB := keyFromKind(t, sqlcv1.V1DurableEventLogKindRUN, &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{WorkflowName: "workflow-b"},
	}, nil)

	assert.NotEqual(t, keyA, keyB)
}

func TestCreateIdempotencyKey_DifferentTriggerData(t *testing.T) {
	keyA := keyFromKind(t, sqlcv1.V1DurableEventLogKindRUN, &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{WorkflowName: "my-workflow", Data: []byte(`{"x":1}`)},
	}, nil)
	keyB := keyFromKind(t, sqlcv1.V1DurableEventLogKindRUN, &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{WorkflowName: "my-workflow", Data: []byte(`{"x":2}`)},
	}, nil)

	assert.NotEqual(t, keyA, keyB)
}

func TestCreateIdempotencyKey_WithAndWithoutTriggerOpts(t *testing.T) {
	without := keyFromKind(t, sqlcv1.V1DurableEventLogKindRUN, nil, nil)
	with := keyFromKind(t, sqlcv1.V1DurableEventLogKindRUN, &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{WorkflowName: "my-workflow"},
	}, nil)

	assert.NotEqual(t, without, with)
}

func int32Ptr(i int32) *int32 { return &i }

func TestCreateIdempotencyKey_PriorityIgnored(t *testing.T) {
	base := keyFromKind(t, sqlcv1.V1DurableEventLogKindRUN, &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{WorkflowName: "my-workflow", Data: []byte(`{"x":1}`)},
	}, nil)
	withPriority := keyFromKind(t, sqlcv1.V1DurableEventLogKindRUN, &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{WorkflowName: "my-workflow", Data: []byte(`{"x":1}`), Priority: int32Ptr(3)},
	}, nil)

	assert.Equal(t, base, withPriority)
}

func TestCreateIdempotencyKey_AdditionalMetadataIgnored(t *testing.T) {
	base := keyFromKind(t, sqlcv1.V1DurableEventLogKindRUN, &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{WorkflowName: "my-workflow", Data: []byte(`{"x":1}`)},
	}, nil)
	withMeta := keyFromKind(t, sqlcv1.V1DurableEventLogKindRUN, &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName:       "my-workflow",
			Data:               []byte(`{"x":1}`),
			AdditionalMetadata: []byte(`{"env":"prod"}`),
		},
	}, nil)

	assert.Equal(t, base, withMeta)
}
