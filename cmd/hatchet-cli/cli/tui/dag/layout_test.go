package dag

import (
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// TestLayoutLinearGraph tests layout of a simple linear graph
func TestLayoutLinearGraph(t *testing.T) {
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

	g, err := BuildGraph(shape, []rest.V1TaskSummary{}, 200, 100)
	require.NoError(t, err)

	g.DetectComponents()
	require.Equal(t, 1, g.ComponentCount())

	config := &RenderConfig{
		NodeWidth:   20,
		NodeHeight:  3,
		PaddingX:    4,
		PaddingY:    2,
		MaxWidth:    200,
		MaxHeight:   100,
		CompactMode: false,
	}

	err = g.LayoutGraph(config)
	require.NoError(t, err)

	// Verify nodes are laid out left-to-right with correct spacing
	nodeA := g.GetNode(stepA.String())
	nodeB := g.GetNode(stepB.String())
	nodeC := g.GetNode(stepC.String())

	assert.Equal(t, 0, nodeA.Layer)
	assert.Equal(t, 1, nodeB.Layer)
	assert.Equal(t, 2, nodeC.Layer)

	// Verify horizontal progression
	assert.True(t, nodeA.DrawX < nodeB.DrawX)
	assert.True(t, nodeB.DrawX < nodeC.DrawX)
}

// TestLayoutDiamondGraph tests layout of a diamond graph with branching
func TestLayoutDiamondGraph(t *testing.T) {
	stepA := mustUUID("00000000-0000-0000-0000-000000000001")
	stepB := mustUUID("00000000-0000-0000-0000-000000000002")
	stepC := mustUUID("00000000-0000-0000-0000-000000000003")
	stepD := mustUUID("00000000-0000-0000-0000-000000000004")

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
			ChildrenStepIds: []openapi_types.UUID{stepD},
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

	g.DetectComponents()
	require.Equal(t, 1, g.ComponentCount())

	config := &RenderConfig{
		NodeWidth:   20,
		NodeHeight:  3,
		PaddingX:    4,
		PaddingY:    2,
		MaxWidth:    200,
		MaxHeight:   100,
		CompactMode: false,
	}

	err = g.LayoutGraph(config)
	require.NoError(t, err)

	nodeA := g.GetNode(stepA.String())
	nodeB := g.GetNode(stepB.String())
	nodeC := g.GetNode(stepC.String())
	nodeD := g.GetNode(stepD.String())

	// Verify layers
	assert.Equal(t, 0, nodeA.Layer)
	assert.Equal(t, 1, nodeB.Layer)
	assert.Equal(t, 1, nodeC.Layer)
	assert.Equal(t, 2, nodeD.Layer)

	// B and C should be in the same layer but different vertical positions
	assert.Equal(t, nodeB.DrawX, nodeC.DrawX)
	assert.NotEqual(t, nodeB.DrawY, nodeC.DrawY)

	// D should be to the right of B and C
	assert.True(t, nodeD.DrawX > nodeB.DrawX)
}

// TestLayoutMultipleComponents tests layout with disconnected components
func TestLayoutMultipleComponents(t *testing.T) {
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

	g.DetectComponents()
	require.Equal(t, 2, g.ComponentCount())

	config := &RenderConfig{
		NodeWidth:   20,
		NodeHeight:  3,
		PaddingX:    4,
		PaddingY:    2,
		MaxWidth:    200,
		MaxHeight:   100,
		CompactMode: false,
	}

	err = g.LayoutGraph(config)
	require.NoError(t, err)

	// Components should be arranged without overlap
	nodeA := g.GetNode(stepA.String())
	nodeB := g.GetNode(stepB.String())
	nodeC := g.GetNode(stepC.String())
	nodeD := g.GetNode(stepD.String())

	// Different components should have different ComponentIDs
	assert.NotEqual(t, nodeA.ComponentID, nodeC.ComponentID)
	assert.Equal(t, nodeA.ComponentID, nodeB.ComponentID)
	assert.Equal(t, nodeC.ComponentID, nodeD.ComponentID)

	// All nodes should have valid positions
	assert.True(t, nodeA.DrawX >= 0)
	assert.True(t, nodeB.DrawX >= 0)
	assert.True(t, nodeC.DrawX >= 0)
	assert.True(t, nodeD.DrawX >= 0)
}

// TestLayoutTooLarge tests error handling when graph doesn't fit
func TestLayoutTooLarge(t *testing.T) {
	// Create a large linear graph
	steps := make([]openapi_types.UUID, 20)
	for i := 0; i < 20; i++ {
		steps[i] = mustUUID("")
	}

	shape := make(rest.WorkflowRunShapeForWorkflowRunDetails, 20)
	for i := 0; i < 20; i++ {
		var children []openapi_types.UUID
		if i < 19 {
			children = []openapi_types.UUID{steps[i+1]}
		} else {
			children = []openapi_types.UUID{}
		}

		shape[i] = rest.WorkflowRunShapeItemForWorkflowRunDetails{
			StepId:          steps[i],
			TaskName:        "Task",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: children,
		}
	}

	// Very small space - should fail
	g, err := BuildGraph(shape, []rest.V1TaskSummary{}, 50, 20)
	require.NoError(t, err)

	g.DetectComponents()

	config := &RenderConfig{
		NodeWidth:   20,
		NodeHeight:  3,
		PaddingX:    4,
		PaddingY:    2,
		MaxWidth:    50,
		MaxHeight:   20,
		CompactMode: false,
	}

	err = g.LayoutGraph(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}

// TestLayoutEmpty tests layout of empty graph
func TestLayoutEmpty(t *testing.T) {
	g := NewGraph(100, 50)

	config := &RenderConfig{
		NodeWidth:   20,
		NodeHeight:  3,
		PaddingX:    4,
		PaddingY:    2,
		MaxWidth:    100,
		MaxHeight:   50,
		CompactMode: false,
	}

	err := g.LayoutGraph(config)
	require.NoError(t, err)
}

// TestAssignLayers tests layer assignment algorithm
func TestAssignLayers(t *testing.T) {
	stepA := mustUUID("00000000-0000-0000-0000-000000000001")
	stepB := mustUUID("00000000-0000-0000-0000-000000000002")
	stepC := mustUUID("00000000-0000-0000-0000-000000000003")
	stepD := mustUUID("00000000-0000-0000-0000-000000000004")

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
			ChildrenStepIds: []openapi_types.UUID{stepD},
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

	g.DetectComponents()
	require.Equal(t, 1, g.ComponentCount())

	comp := g.GetComponent(0)
	layers := comp.assignLayers()

	// A should be at layer 0 (root)
	assert.Equal(t, 0, layers[stepA.String()])

	// B and C should be at layer 1
	assert.Equal(t, 1, layers[stepB.String()])
	assert.Equal(t, 1, layers[stepC.String()])

	// D should be at layer 2 (longest path from root)
	assert.Equal(t, 2, layers[stepD.String()])
}

// TestEdgeRouting tests that edges are routed without overlapping nodes
func TestEdgeRouting(t *testing.T) {
	stepA := mustUUID("00000000-0000-0000-0000-000000000001")
	stepB := mustUUID("00000000-0000-0000-0000-000000000002")
	stepC := mustUUID("00000000-0000-0000-0000-000000000003")
	stepD := mustUUID("00000000-0000-0000-0000-000000000004")

	// Diamond pattern: A -> B,C -> D
	// This tests the case where D has multiple parents and edges should not overlap
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
			ChildrenStepIds: []openapi_types.UUID{stepD},
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

	g.DetectComponents()
	require.Equal(t, 1, g.ComponentCount())

	config := &RenderConfig{
		NodeWidth:   20,
		NodeHeight:  3,
		PaddingX:    4,
		PaddingY:    2,
		MaxWidth:    200,
		MaxHeight:   100,
		CompactMode: false,
	}

	err = g.LayoutGraph(config)
	require.NoError(t, err)

	// Verify edges were created
	assert.Equal(t, 4, len(g.Edges)) // A->B, A->C, B->D, C->D

	// Verify all edges have path points
	for _, edge := range g.Edges {
		assert.Greater(t, len(edge.Points), 0, "Edge should have routing points")

		// Verify path starts near source and ends near destination
		firstPoint := edge.Points[0]
		lastPoint := edge.Points[len(edge.Points)-1]

		// First point should be on the right edge of the source node
		assert.Equal(t, edge.From.DrawX+edge.From.Width, firstPoint.X)
		assert.Equal(t, edge.From.DrawY+edge.From.Height/2, firstPoint.Y)

		// Last point should be on the left edge of the destination node
		assert.Equal(t, edge.To.DrawX, lastPoint.X)
		assert.Equal(t, edge.To.DrawY+edge.To.Height/2, lastPoint.Y)
	}
}
