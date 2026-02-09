package dag

import (
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// Helper to create UUIDs for testing
func mustUUID(s string) openapi_types.UUID {
	if s == "" {
		return openapi_types.UUID(uuid.New())
	}
	u, _ := uuid.Parse(s)
	return openapi_types.UUID(u)
}

// TestNewGraph tests graph creation
func TestNewGraph(t *testing.T) {
	g := NewGraph(100, 50)

	assert.NotNil(t, g)
	assert.Equal(t, 100, g.MaxWidth)
	assert.Equal(t, 50, g.MaxHeight)
	assert.Equal(t, "LR", g.Direction)
	assert.Equal(t, 0, g.NodeCount())
	assert.Equal(t, 0, g.EdgeCount())
	assert.True(t, g.IsEmpty())
}

// TestBuildGraphEmpty tests building an empty graph
func TestBuildGraphEmpty(t *testing.T) {
	shape := rest.WorkflowRunShapeForWorkflowRunDetails{}
	tasks := []rest.V1TaskSummary{}

	g, err := BuildGraph(shape, tasks, 100, 50)

	require.NoError(t, err)
	assert.True(t, g.IsEmpty())
	assert.Equal(t, 0, g.NodeCount())
}

// TestBuildGraphLinear tests building a simple linear graph (A -> B -> C)
func TestBuildGraphLinear(t *testing.T) {
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

	g, err := BuildGraph(shape, []rest.V1TaskSummary{}, 100, 50)

	require.NoError(t, err)
	assert.Equal(t, 3, g.NodeCount())
	assert.Equal(t, 2, g.EdgeCount())

	// Check nodes exist
	nodeA := g.GetNode(stepA.String())
	nodeB := g.GetNode(stepB.String())
	nodeC := g.GetNode(stepC.String())

	require.NotNil(t, nodeA)
	require.NotNil(t, nodeB)
	require.NotNil(t, nodeC)

	// Check topology
	assert.Equal(t, 0, len(nodeA.Parents))
	assert.Equal(t, 1, len(nodeA.Children))
	assert.Equal(t, nodeB, nodeA.Children[0])

	assert.Equal(t, 1, len(nodeB.Parents))
	assert.Equal(t, nodeA, nodeB.Parents[0])
	assert.Equal(t, 1, len(nodeB.Children))
	assert.Equal(t, nodeC, nodeB.Children[0])

	assert.Equal(t, 1, len(nodeC.Parents))
	assert.Equal(t, nodeB, nodeC.Parents[0])
	assert.Equal(t, 0, len(nodeC.Children))
}

// TestBuildGraphDiamond tests building a diamond graph (A -> B,C -> D)
func TestBuildGraphDiamond(t *testing.T) {
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

	g, err := BuildGraph(shape, []rest.V1TaskSummary{}, 100, 50)

	require.NoError(t, err)
	assert.Equal(t, 4, g.NodeCount())
	assert.Equal(t, 4, g.EdgeCount()) // A->B, A->C, B->D, C->D

	nodeD := g.GetNode(stepD.String())
	require.NotNil(t, nodeD)
	assert.Equal(t, 2, len(nodeD.Parents)) // B and C
}

// TestBuildGraphInvalidChild tests error handling for invalid child reference
func TestBuildGraphInvalidChild(t *testing.T) {
	stepA := mustUUID("00000000-0000-0000-0000-000000000001")
	stepB := mustUUID("00000000-0000-0000-0000-000000000002")
	stepC := mustUUID("00000000-0000-0000-0000-000000000099") // Doesn't exist

	shape := rest.WorkflowRunShapeForWorkflowRunDetails{
		{
			StepId:          stepA,
			TaskName:        "Task A",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{stepB, stepC}, // stepC doesn't exist
		},
		{
			StepId:          stepB,
			TaskName:        "Task B",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{},
		},
	}

	_, err := BuildGraph(shape, []rest.V1TaskSummary{}, 100, 50)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-existent child")
}

// TestDetectComponentsSingle tests component detection with a single connected graph
func TestDetectComponentsSingle(t *testing.T) {
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

	g, err := BuildGraph(shape, []rest.V1TaskSummary{}, 100, 50)
	require.NoError(t, err)

	count := g.DetectComponents()

	assert.Equal(t, 1, count)
	assert.Equal(t, 1, g.ComponentCount())

	comp := g.GetComponent(0)
	require.NotNil(t, comp)
	assert.Equal(t, 3, len(comp.Nodes))
	assert.Equal(t, 1, len(comp.RootNodes)) // Only A has no parents

	// Check all nodes have correct component ID
	for _, node := range g.Nodes {
		assert.Equal(t, 0, node.ComponentID)
	}
}

// TestDetectComponentsMultiple tests component detection with disconnected components
func TestDetectComponentsMultiple(t *testing.T) {
	// Component 1: A -> B
	stepA := mustUUID("00000000-0000-0000-0000-000000000001")
	stepB := mustUUID("00000000-0000-0000-0000-000000000002")

	// Component 2: C -> D
	stepC := mustUUID("00000000-0000-0000-0000-000000000003")
	stepD := mustUUID("00000000-0000-0000-0000-000000000004")

	// Component 3: E (isolated)
	stepE := mustUUID("00000000-0000-0000-0000-000000000005")

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
		{
			StepId:          stepE,
			TaskName:        "Task E",
			TaskExternalId:  mustUUID(""),
			ChildrenStepIds: []openapi_types.UUID{},
		},
	}

	g, err := BuildGraph(shape, []rest.V1TaskSummary{}, 100, 50)
	require.NoError(t, err)

	count := g.DetectComponents()

	assert.Equal(t, 3, count)
	assert.Equal(t, 3, g.ComponentCount())

	// Check component stats
	stats := g.GetComponentStats()
	assert.Equal(t, 3, stats.TotalComponents)
	assert.Equal(t, 2, stats.LargestComponent) // A-B or C-D
	assert.Equal(t, 1, stats.SmallestComponent) // E
	assert.Equal(t, 1, stats.IsolatedNodes)     // E
}

// TestHasCycleNoCycle tests cycle detection on an acyclic graph
func TestHasCycleNoCycle(t *testing.T) {
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

	g, err := BuildGraph(shape, []rest.V1TaskSummary{}, 100, 50)
	require.NoError(t, err)

	assert.False(t, g.HasCycle())
}

// TestHasCycleCycle tests cycle detection on a cyclic graph
// Note: Real workflow graphs shouldn't have cycles, but we test the detector
func TestHasCycleCycle(t *testing.T) {
	g := NewGraph(100, 50)

	// Create nodes
	nodeA := &Node{StepID: "A", TaskName: "Task A", Parents: make([]*Node, 0), Children: make([]*Node, 0)}
	nodeB := &Node{StepID: "B", TaskName: "Task B", Parents: make([]*Node, 0), Children: make([]*Node, 0)}
	nodeC := &Node{StepID: "C", TaskName: "Task C", Parents: make([]*Node, 0), Children: make([]*Node, 0)}

	g.AddNode(nodeA)
	g.AddNode(nodeB)
	g.AddNode(nodeC)

	// Create cycle: A -> B -> C -> A
	g.AddEdge(nodeA, nodeB)
	g.AddEdge(nodeB, nodeC)
	g.AddEdge(nodeC, nodeA) // Cycle!

	assert.True(t, g.HasCycle())
}
