package dispatcher

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func newDagChildInput(t *testing.T, taskId int64, parentExternalIds ...uuid.UUID) *dagChildTaskInput {
	t.Helper()

	return &dagChildTaskInput{
		payloadKey: v1.RetrievePayloadOpts{Id: taskId},
		currInput: &v1.V1StepRunData{
			DagParentTaskRunIds: parentExternalIds,
		},
	}
}

func parentOutputEvent(t *testing.T, externalId uuid.UUID, stepReadableId string, output any) *v1.TaskOutputEvent {
	t.Helper()

	b, err := json.Marshal(output)
	require.NoError(t, err)

	return &v1.TaskOutputEvent{
		TaskExternalId: externalId,
		StepReadableID: stepReadableId,
		Output:         b,
	}
}

// unmarshalParents round-trips inputs[key] the same way a worker would when it receives the
// dispatched task, confirming resolveDagParentOutputs' effect actually survives serialization.
func unmarshalParents(t *testing.T, inputs map[v1.RetrievePayloadOpts][]byte, key v1.RetrievePayloadOpts) map[string]map[string]interface{} {
	t.Helper()

	raw, ok := inputs[key]
	require.True(t, ok, "expected inputs to contain an entry for %+v", key)

	var data v1.V1StepRunData
	require.NoError(t, json.Unmarshal(raw, &data))

	return data.Parents
}

func TestResolveDagParentOutputs_PopulatesParentsFromBatch(t *testing.T) {
	parentA := uuid.New()
	parentB := uuid.New()

	// taskX depends on both parentA and parentB; taskY depends only on parentB. This mirrors
	// what populateTaskData builds: a single dagParentOutputs batch shared across every
	// dag-child task in the dispatch batch, each pulling out only its own parents.
	taskX := newDagChildInput(t, 1, parentA, parentB)
	taskY := newDagChildInput(t, 2, parentB)

	dagParentOutputs := map[uuid.UUID]*v1.TaskOutputEvent{
		parentA: parentOutputEvent(t, parentA, "step_a", map[string]int{"random_number": 1}),
		parentB: parentOutputEvent(t, parentB, "step_b", map[string]int{"random_number": 2}),
	}

	inputs := make(map[v1.RetrievePayloadOpts][]byte)

	resolveDagParentOutputs(context.Background(), zerologNop(), []*dagChildTaskInput{taskX, taskY}, dagParentOutputs, inputs)

	xParents := unmarshalParents(t, inputs, taskX.payloadKey)
	assert.Equal(t, map[string]interface{}{"random_number": float64(1)}, xParents["step_a"])
	assert.Equal(t, map[string]interface{}{"random_number": float64(2)}, xParents["step_b"])
	assert.Len(t, xParents, 2)

	yParents := unmarshalParents(t, inputs, taskY.payloadKey)
	assert.Equal(t, map[string]interface{}{"random_number": float64(2)}, yParents["step_b"])
	assert.Len(t, yParents, 1, "taskY never listed parentA, so it must not see step_a's output")
}

// TestResolveDagParentOutputs_DoesNotLeakAcrossDifferentDagRuns is the regression test for the
// bug the batching redesign fixes: two unrelated DAG runs can both have a step named "start",
// so the shared dagParentOutputs batch must be keyed by parent external ID, never by step
// readable ID, or one run's output would silently clobber the other's.
func TestResolveDagParentOutputs_DoesNotLeakAcrossDifferentDagRuns(t *testing.T) {
	runOneParent := uuid.New()
	runTwoParent := uuid.New()

	runOneChild := newDagChildInput(t, 1, runOneParent)
	runTwoChild := newDagChildInput(t, 2, runTwoParent)

	dagParentOutputs := map[uuid.UUID]*v1.TaskOutputEvent{
		runOneParent: parentOutputEvent(t, runOneParent, "start", map[string]int{"random_number": 1}),
		runTwoParent: parentOutputEvent(t, runTwoParent, "start", map[string]int{"random_number": 99}),
	}

	inputs := make(map[v1.RetrievePayloadOpts][]byte)

	resolveDagParentOutputs(context.Background(), zerologNop(), []*dagChildTaskInput{runOneChild, runTwoChild}, dagParentOutputs, inputs)

	runOneParents := unmarshalParents(t, inputs, runOneChild.payloadKey)
	runTwoParents := unmarshalParents(t, inputs, runTwoChild.payloadKey)

	assert.Equal(t, map[string]interface{}{"random_number": float64(1)}, runOneParents["start"])
	assert.Equal(t, map[string]interface{}{"random_number": float64(99)}, runTwoParents["start"])
}

// TestResolveDagParentOutputs_IsolatesMissingOrUnparsableParents checks that a gap in
// dagParentOutputs (e.g. a parent the batched/fallback lookup couldn't find) or a malformed
// payload for one parent only drops that one entry rather than the whole task's Parents map,
// and never affects a sibling task in the same batch.
func TestResolveDagParentOutputs_IsolatesMissingOrUnparsableParents(t *testing.T) {
	goodParent := uuid.New()
	missingParent := uuid.New()
	malformedParent := uuid.New()

	affected := newDagChildInput(t, 1, goodParent, missingParent, malformedParent)
	sibling := newDagChildInput(t, 2, goodParent)

	dagParentOutputs := map[uuid.UUID]*v1.TaskOutputEvent{
		goodParent: parentOutputEvent(t, goodParent, "good", map[string]int{"random_number": 7}),
		malformedParent: {
			TaskExternalId: malformedParent,
			StepReadableID: "malformed",
			Output:         []byte("{not valid json"),
		},
		// missingParent intentionally absent, simulating a lookup that never resolved it.
	}

	inputs := make(map[v1.RetrievePayloadOpts][]byte)

	resolveDagParentOutputs(context.Background(), zerologNop(), []*dagChildTaskInput{affected, sibling}, dagParentOutputs, inputs)

	affectedParents := unmarshalParents(t, inputs, affected.payloadKey)
	assert.Equal(t, map[string]interface{}{"random_number": float64(7)}, affectedParents["good"])
	assert.NotContains(t, affectedParents, "missing")
	assert.NotContains(t, affectedParents, "malformed")
	assert.Len(t, affectedParents, 1)

	siblingParents := unmarshalParents(t, inputs, sibling.payloadKey)
	assert.Equal(t, map[string]interface{}{"random_number": float64(7)}, siblingParents["good"])
	assert.Len(t, siblingParents, 1, "the malformed/missing parents on a different task must not affect this sibling")
}

func zerologNop() *zerolog.Logger {
	l := zerolog.Nop()
	return &l
}
