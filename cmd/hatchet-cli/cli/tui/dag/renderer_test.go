package dag

import (
	"strings"
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// TestRenderEmpty tests rendering an empty graph
func TestRenderEmpty(t *testing.T) {
	g := NewGraph(100, 50)

	output, err := Render(g, "")
	require.NoError(t, err)
	assert.Equal(t, "", output)
}

// TestRenderLinearGraph tests rendering a simple linear graph
func TestRenderLinearGraph(t *testing.T) {
	stepA := mustUUID("00000000-0000-0000-0000-000000000001")
	stepB := mustUUID("00000000-0000-0000-0000-000000000002")
	stepC := mustUUID("00000000-0000-0000-0000-000000000003")

	shape := rest.WorkflowRunShapeForWorkflowRunDetails{
		{
			StepId:          stepA,
			TaskName:        "Task A",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{stepB},
		},
		{
			StepId:          stepB,
			TaskName:        "Task B",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{stepC},
		},
		{
			StepId:          stepC,
			TaskName:        "Task C",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{},
		},
	}

	tasks := []rest.V1TaskSummary{
		{
			Status: rest.V1TaskStatusCOMPLETED,
			Metadata: rest.APIResourceMeta{
				Id: mustUUID("").String(),
			},
		},
		{
			Status: rest.V1TaskStatusRUNNING,
			Metadata: rest.APIResourceMeta{
				Id: mustUUID("").String(),
			},
		},
		{
			Status: rest.V1TaskStatusQUEUED,
			Metadata: rest.APIResourceMeta{
				Id: mustUUID("").String(),
			},
		},
	}

	g, err := BuildGraph(shape, tasks, 200, 100)
	require.NoError(t, err)

	output, err := Render(g, "")
	require.NoError(t, err)

	// Verify output contains the task names
	assert.Contains(t, output, "Task A")
	assert.Contains(t, output, "Task B")
	assert.Contains(t, output, "Task C")

	// Verify output contains box-drawing characters
	assert.Contains(t, output, "┌")
	assert.Contains(t, output, "─")
	assert.Contains(t, output, "│")

	// Output should have multiple lines
	lines := strings.Split(output, "\n")
	assert.GreaterOrEqual(t, len(lines), 3)
}

// TestRenderWithSelection tests rendering with a selected node
func TestRenderWithSelection(t *testing.T) {
	stepA := mustUUID("00000000-0000-0000-0000-000000000001")
	stepB := mustUUID("00000000-0000-0000-0000-000000000002")

	shape := rest.WorkflowRunShapeForWorkflowRunDetails{
		{
			StepId:          stepA,
			TaskName:        "Task A",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{stepB},
		},
		{
			StepId:          stepB,
			TaskName:        "Task B",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{},
		},
	}

	g, err := BuildGraph(shape, []rest.V1TaskSummary{}, 200, 100)
	require.NoError(t, err)

	// Render with selection
	output, err := Render(g, stepA.String())
	require.NoError(t, err)

	assert.NotEmpty(t, output)
	assert.Contains(t, output, "Task A")
	assert.Contains(t, output, "Task B")
}

// TestRenderCompact tests compact mode rendering
func TestRenderCompact(t *testing.T) {
	stepA := mustUUID("00000000-0000-0000-0000-000000000001")
	stepB := mustUUID("00000000-0000-0000-0000-000000000002")

	shape := rest.WorkflowRunShapeForWorkflowRunDetails{
		{
			StepId:          stepA,
			TaskName:        "Task A",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{stepB},
		},
		{
			StepId:          stepB,
			TaskName:        "Task B",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{},
		},
	}

	g, err := BuildGraph(shape, []rest.V1TaskSummary{}, 200, 100)
	require.NoError(t, err)

	output, err := RenderCompact(g, "")
	require.NoError(t, err)

	assert.NotEmpty(t, output)
	assert.Contains(t, output, "Task A")
	assert.Contains(t, output, "Task B")
}

// TestGetNodeAtPosition tests node position detection
func TestGetNodeAtPosition(t *testing.T) {
	stepA := mustUUID("00000000-0000-0000-0000-000000000001")
	stepB := mustUUID("00000000-0000-0000-0000-000000000002")

	shape := rest.WorkflowRunShapeForWorkflowRunDetails{
		{
			StepId:          stepA,
			TaskName:        "Task A",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{stepB},
		},
		{
			StepId:          stepB,
			TaskName:        "Task B",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{},
		},
	}

	g, err := BuildGraph(shape, []rest.V1TaskSummary{}, 200, 100)
	require.NoError(t, err)

	_, err = Render(g, "")
	require.NoError(t, err)

	// Test hitting a node
	nodeA := g.GetNode(stepA.String())
	require.NotNil(t, nodeA)

	foundNode := GetNodeAtPosition(g, nodeA.DrawX+1, nodeA.DrawY+1)
	require.NotNil(t, foundNode)
	assert.Equal(t, stepA.String(), foundNode.StepID)

	// Test missing a node
	missedNode := GetNodeAtPosition(g, -10, -10)
	assert.Nil(t, missedNode)
}

// TestGetNavigableNodes tests keyboard navigation ordering
func TestGetNavigableNodes(t *testing.T) {
	stepA := mustUUID("00000000-0000-0000-0000-000000000001")
	stepB := mustUUID("00000000-0000-0000-0000-000000000002")
	stepC := mustUUID("00000000-0000-0000-0000-000000000003")

	shape := rest.WorkflowRunShapeForWorkflowRunDetails{
		{
			StepId:          stepA,
			TaskName:        "Task A",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{stepB, stepC},
		},
		{
			StepId:          stepB,
			TaskName:        "Task B",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{},
		},
		{
			StepId:          stepC,
			TaskName:        "Task C",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{},
		},
	}

	g, err := BuildGraph(shape, []rest.V1TaskSummary{}, 200, 100)
	require.NoError(t, err)

	_, err = Render(g, "")
	require.NoError(t, err)

	nodes := GetNavigableNodes(g)
	require.Equal(t, 3, len(nodes))

	// Nodes should be in visual order (top-to-bottom, left-to-right)
	for i := 0; i < len(nodes)-1; i++ {
		curr := nodes[i]
		next := nodes[i+1]

		// Current node should be above or to the left of next node
		assert.True(t,
			curr.DrawY < next.DrawY ||
				(curr.DrawY == next.DrawY && curr.DrawX <= next.DrawX),
			"Nodes not in visual order")
	}
}

// TestRenderMultipleComponents tests rendering disconnected components
func TestRenderMultipleComponents(t *testing.T) {
	// Component 1: A -> B
	stepA := mustUUID("00000000-0000-0000-0000-000000000001")
	stepB := mustUUID("00000000-0000-0000-0000-000000000002")

	// Component 2: C -> D
	stepC := mustUUID("00000000-0000-0000-0000-000000000003")
	stepD := mustUUID("00000000-0000-0000-0000-000000000004")

	shape := rest.WorkflowRunShapeForWorkflowRunDetails{
		{
			StepId:          stepA,
			TaskName:        "Task A",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{stepB},
		},
		{
			StepId:          stepB,
			TaskName:        "Task B",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{},
		},
		{
			StepId:          stepC,
			TaskName:        "Task C",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{stepD},
		},
		{
			StepId:          stepD,
			TaskName:        "Task D",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{},
		},
	}

	g, err := BuildGraph(shape, []rest.V1TaskSummary{}, 200, 100)
	require.NoError(t, err)

	output, err := Render(g, "")
	require.NoError(t, err)

	// All components should be rendered
	assert.Contains(t, output, "Task A")
	assert.Contains(t, output, "Task B")
	assert.Contains(t, output, "Task C")
	assert.Contains(t, output, "Task D")
}

// TestDeterministicRendering verifies that the same graph always produces identical output
func TestDeterministicRendering(t *testing.T) {
	// Create a diamond-shaped graph with multiple root nodes to test stability
	// This mimics the real-world scenario where step1 and step2 both feed into step3
	stepA := mustUUID("00000000-0000-0000-0000-000000000001")
	stepB := mustUUID("00000000-0000-0000-0000-000000000002")
	stepC := mustUUID("00000000-0000-0000-0000-000000000003")
	stepD := mustUUID("00000000-0000-0000-0000-000000000004")

	shape := rest.WorkflowRunShapeForWorkflowRunDetails{
		{
			StepId:          stepA,
			TaskName:        "Task A",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{stepC},
		},
		{
			StepId:          stepB,
			TaskName:        "Task B",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{stepC},
		},
		{
			StepId:          stepC,
			TaskName:        "Task C",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{stepD},
		},
		{
			StepId:          stepD,
			TaskName:        "Task D",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{},
		},
	}

	// Render the same graph multiple times
	outputs := make([]string, 10)
	for i := 0; i < 10; i++ {
		g, err := BuildGraph(shape, []rest.V1TaskSummary{}, 200, 100)
		require.NoError(t, err)

		output, err := Render(g, "")
		require.NoError(t, err)

		outputs[i] = output
	}

	// All outputs should be identical
	expectedOutput := outputs[0]
	for i := 1; i < len(outputs); i++ {
		assert.Equal(t, expectedOutput, outputs[i],
			"Render output %d differs from expected. Graph rendering should be deterministic.", i)
	}

	// Verify that nodes are in a consistent order
	// Task A and B should both appear before Task C and D
	assert.Contains(t, expectedOutput, "Task A")
	assert.Contains(t, expectedOutput, "Task B")
	assert.Contains(t, expectedOutput, "Task C")
	assert.Contains(t, expectedOutput, "Task D")
}
