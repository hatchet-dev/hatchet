//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	"github.com/stretchr/testify/assert"
)

func TestResolveBranchForNode_NoBranchPoints(t *testing.T) {
	// Single branch, no forks. All nodes resolve to branch 1.
	branchPoints := map[int64]*sqlcv1.V1DurableEventLogBranchPoint{}

	for _, nodeId := range []int64{1, 2, 3, 4, 5, 6} {
		assert.Equal(t, resolveBranchForNode(nodeId, 1, branchPoints), int64(1), "nodeId=%d", nodeId)
	}
}

func TestResolveBranchForNode_SingleForkFromNode1(t *testing.T) {
	// Branch 1 forked at node 1 → branch 2.
	// Nodes >= 1 should resolve to branch 2.
	branchPoints := map[int64]*sqlcv1.V1DurableEventLogBranchPoint{
		2: {FirstNodeIDInNewBranch: 1, ParentBranchID: 1, NextBranchID: 2},
	}

	assert.Equal(t, resolveBranchForNode(1, 2, branchPoints), int64(2))
	assert.Equal(t, resolveBranchForNode(2, 2, branchPoints), int64(2))
	assert.Equal(t, resolveBranchForNode(3, 2, branchPoints), int64(2))
}

func TestResolveBranchForNode_SingleForkFromNode2(t *testing.T) {
	// Branch 1 forked at node 2 → branch 2.
	// Node 1 should resolve to branch 1 (cached), nodes >= 2 to branch 2.
	branchPoints := map[int64]*sqlcv1.V1DurableEventLogBranchPoint{
		2: {FirstNodeIDInNewBranch: 2, ParentBranchID: 1, NextBranchID: 2},
	}

	assert.Equal(t, resolveBranchForNode(1, 2, branchPoints), int64(1))
	assert.Equal(t, resolveBranchForNode(2, 2, branchPoints), int64(2))
	assert.Equal(t, resolveBranchForNode(3, 2, branchPoints), int64(2))
}

func TestResolveBranchForNode_BranchOffBranch(t *testing.T) {
	// Branch 1 forked at node 1 → branch 2.
	// Branch 2 forked at node 2 → branch 3.
	// Node 1 should use branch 2, nodes >= 2 should use branch 3.
	branchPoints := map[int64]*sqlcv1.V1DurableEventLogBranchPoint{
		2: {FirstNodeIDInNewBranch: 1, ParentBranchID: 1, NextBranchID: 2},
		3: {FirstNodeIDInNewBranch: 2, ParentBranchID: 2, NextBranchID: 3},
	}

	assert.Equal(t, resolveBranchForNode(1, 3, branchPoints), int64(2))
	assert.Equal(t, resolveBranchForNode(2, 3, branchPoints), int64(3))
	assert.Equal(t, resolveBranchForNode(3, 3, branchPoints), int64(3))
}

func TestResolveBranchForNode_DeepChain(t *testing.T) {
	// Chain: branch 1 → 2 (at node 1) → 3 (at node 2) → 4 (at node 3)
	branchPoints := map[int64]*sqlcv1.V1DurableEventLogBranchPoint{
		2: {FirstNodeIDInNewBranch: 1, ParentBranchID: 1, NextBranchID: 2},
		3: {FirstNodeIDInNewBranch: 2, ParentBranchID: 2, NextBranchID: 3},
		4: {FirstNodeIDInNewBranch: 3, ParentBranchID: 3, NextBranchID: 4},
	}

	assert.Equal(t, resolveBranchForNode(1, 4, branchPoints), int64(2))
	assert.Equal(t, resolveBranchForNode(2, 4, branchPoints), int64(3))
	assert.Equal(t, resolveBranchForNode(3, 4, branchPoints), int64(4))
	assert.Equal(t, resolveBranchForNode(4, 4, branchPoints), int64(4))
}

func TestResolveBranchForNode_ForkAtSameNode(t *testing.T) {
	// Two successive forks both at node 1: branch 1 → 2, then branch 2 → 3.
	// All nodes on branch 3 should resolve to branch 3 (since fork point is node 1).
	branchPoints := map[int64]*sqlcv1.V1DurableEventLogBranchPoint{
		2: {FirstNodeIDInNewBranch: 1, ParentBranchID: 1, NextBranchID: 2},
		3: {FirstNodeIDInNewBranch: 1, ParentBranchID: 2, NextBranchID: 3},
	}

	assert.Equal(t, resolveBranchForNode(1, 3, branchPoints), int64(3))
	assert.Equal(t, resolveBranchForNode(2, 3, branchPoints), int64(3))
	assert.Equal(t, resolveBranchForNode(3, 3, branchPoints), int64(3))
}

func TestResolveBranchForNode_QueriedFromOlderBranch(t *testing.T) {
	// Branch points exist for branches 2 and 3, but we're resolving from branch 2.
	// Branch 3's branch point should be irrelevant.
	branchPoints := map[int64]*sqlcv1.V1DurableEventLogBranchPoint{
		2: {FirstNodeIDInNewBranch: 2, ParentBranchID: 1, NextBranchID: 2},
		3: {FirstNodeIDInNewBranch: 3, ParentBranchID: 2, NextBranchID: 3},
	}

	assert.Equal(t, resolveBranchForNode(1, 2, branchPoints), int64(1))
	assert.Equal(t, resolveBranchForNode(2, 2, branchPoints), int64(2))
	assert.Equal(t, resolveBranchForNode(3, 2, branchPoints), int64(2))
}

func TestResolveBranchForNode_SiblingBranches(t *testing.T) {
	// Branch 1 forks at node 3 → branch 2, then branch 1 forks at node 2 → branch 3.
	// Branch 3's ancestry is just branch 1, so branch 2 is irrelevant.
	// Nodes 1 should use branch 1, nodes >= 2 should use branch 3.
	branchPoints := map[int64]*sqlcv1.V1DurableEventLogBranchPoint{
		2: {FirstNodeIDInNewBranch: 3, ParentBranchID: 1, NextBranchID: 2},
		3: {FirstNodeIDInNewBranch: 2, ParentBranchID: 1, NextBranchID: 3},
	}

	assert.Equal(t, resolveBranchForNode(1, 3, branchPoints), int64(1))
	assert.Equal(t, resolveBranchForNode(2, 3, branchPoints), int64(3))
	assert.Equal(t, resolveBranchForNode(3, 3, branchPoints), int64(3))
	assert.Equal(t, resolveBranchForNode(4, 3, branchPoints), int64(3))

	// And from branch 2's perspective, branch 3 is irrelevant.
	// Nodes 1-2 should use branch 1, nodes >= 3 should use branch 2.
	assert.Equal(t, resolveBranchForNode(1, 2, branchPoints), int64(1))
	assert.Equal(t, resolveBranchForNode(2, 2, branchPoints), int64(1))
	assert.Equal(t, resolveBranchForNode(3, 2, branchPoints), int64(2))
	assert.Equal(t, resolveBranchForNode(4, 2, branchPoints), int64(2))
}

func TestResolveBranchForNode_SiblingBranchForkAfterSibling(t *testing.T) {
	// Branch 1 forks at node 3 → branch 2, then branch 1 forks at node 4 → branch 3.
	// Branch 3's ancestry is just branch 1, so branch 2 is irrelevant.
	// Nodes 1-3 should use branch 1, nodes >= 4 should use branch 3.
	branchPoints := map[int64]*sqlcv1.V1DurableEventLogBranchPoint{
		2: {FirstNodeIDInNewBranch: 3, ParentBranchID: 1, NextBranchID: 2},
		3: {FirstNodeIDInNewBranch: 4, ParentBranchID: 1, NextBranchID: 3},
	}

	assert.Equal(t, resolveBranchForNode(1, 3, branchPoints), int64(1))
	assert.Equal(t, resolveBranchForNode(2, 3, branchPoints), int64(1))
	assert.Equal(t, resolveBranchForNode(3, 3, branchPoints), int64(1))
	assert.Equal(t, resolveBranchForNode(4, 3, branchPoints), int64(3))
	assert.Equal(t, resolveBranchForNode(5, 3, branchPoints), int64(3))

	// From branch 2's perspective, branch 3 is irrelevant.
	// Nodes 1-2 should use branch 1, nodes >= 3 should use branch 2.
	assert.Equal(t, resolveBranchForNode(1, 2, branchPoints), int64(1))
	assert.Equal(t, resolveBranchForNode(2, 2, branchPoints), int64(1))
	assert.Equal(t, resolveBranchForNode(3, 2, branchPoints), int64(2))
	assert.Equal(t, resolveBranchForNode(4, 2, branchPoints), int64(2))
}

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
