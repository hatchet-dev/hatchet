package dag

// Graph rendering implementation for Hatchet TUI
//
// This package provides ASCII-based visualization of workflow execution graphs
// (technically forests, as they may contain multiple disconnected components).
//
// Implementation inspired by the mermaid-ascii project:
// https://github.com/AlexanderGrooff/mermaid-ascii

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// Direction represents a cardinal direction for navigation or drawing
type Direction int

const (
	DirectionLeft Direction = iota
	DirectionRight
	DirectionUp
	DirectionDown
)

// Coord represents a position on the canvas or in grid space
type Coord struct {
	X int
	Y int
}

// Rect represents a rectangular region with position and dimensions
type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// Node represents a single task node in the workflow graph
type Node struct {
	// Identity
	StepID         string // Unique step identifier from API
	TaskName       string // Display name for the task
	TaskExternalID string // Link to full task details

	// Status for coloring
	Status rest.V1TaskStatus

	// Graph topology
	Parents  []*Node // Incoming edges
	Children []*Node // Outgoing edges

	// Component membership
	ComponentID int // Which connected component this belongs to

	// Layout coordinates (relative to component, in grid units)
	GridX int
	GridY int
	Layer int // Computed layer in hierarchical layout

	// Drawing coordinates (absolute on canvas, in character units)
	DrawX int
	DrawY int

	// Dimensions in character units
	Width  int
	Height int
}

// Edge represents a directed connection between two nodes
type Edge struct {
	From   *Node   // Source node
	To     *Node   // Target node
	Points []Coord // Computed path points for drawing (set by edge router)
}

// Component represents a connected subgraph (part of the forest)
type Component struct {
	Nodes       []*Node
	RootNodes   []*Node
	BoundingBox Rect
	ID          int
}

// Graph represents the complete workflow execution graph
// May be a forest (multiple disconnected components)
type Graph struct {
	Nodes        map[string]*Node
	Direction    string
	Edges        []*Edge
	Components   []*Component
	MaxWidth     int
	MaxHeight    int
	ActualWidth  int
	ActualHeight int
}

// RenderConfig contains layout and rendering parameters
type RenderConfig struct {
	// Node dimensions in character units
	NodeWidth  int
	NodeHeight int

	// Spacing between nodes
	PaddingX int
	PaddingY int

	// Available space
	MaxWidth  int
	MaxHeight int

	// Compact mode uses minimal spacing
	CompactMode bool
}

// NewGraph creates a new empty graph with the given space constraints
func NewGraph(maxWidth, maxHeight int) *Graph {
	return &Graph{
		Nodes:      make(map[string]*Node),
		Edges:      make([]*Edge, 0),
		Components: make([]*Component, 0),
		Direction:  "LR", // Default to left-to-right
		MaxWidth:   maxWidth,
		MaxHeight:  maxHeight,
	}
}

// AddNode adds a node to the graph
// Returns error if a node with the same StepID already exists
func (g *Graph) AddNode(node *Node) error {
	if node == nil {
		return fmt.Errorf("cannot add nil node")
	}
	if node.StepID == "" {
		return fmt.Errorf("node must have a StepID")
	}
	if _, exists := g.Nodes[node.StepID]; exists {
		return fmt.Errorf("node with StepID %s already exists", node.StepID)
	}

	g.Nodes[node.StepID] = node
	return nil
}

// GetNode retrieves a node by its step ID
func (g *Graph) GetNode(stepID string) *Node {
	return g.Nodes[stepID]
}

// AddEdge adds a directed edge and updates the parent/child relationships
func (g *Graph) AddEdge(from, to *Node) error {
	if from == nil || to == nil {
		return fmt.Errorf("cannot add edge with nil nodes")
	}

	edge := &Edge{
		From:   from,
		To:     to,
		Points: make([]Coord, 0),
	}
	g.Edges = append(g.Edges, edge)

	to.Parents = append(to.Parents, from)
	from.Children = append(from.Children, to)

	return nil
}

// NodeCount returns the total number of nodes in the graph
func (g *Graph) NodeCount() int {
	return len(g.Nodes)
}

// EdgeCount returns the total number of edges in the graph
func (g *Graph) EdgeCount() int {
	return len(g.Edges)
}

// ComponentCount returns the number of connected components
func (g *Graph) ComponentCount() int {
	return len(g.Components)
}

// IsEmpty returns true if the graph has no nodes
func (g *Graph) IsEmpty() bool {
	return len(g.Nodes) == 0
}

// BuildGraph converts API shape and task data into the internal graph representation
func BuildGraph(
	shape rest.WorkflowRunShapeForWorkflowRunDetails,
	tasks []rest.V1TaskSummary,
	maxWidth, maxHeight int,
) (*Graph, error) {
	g := NewGraph(maxWidth, maxHeight)

	if len(shape) == 0 {
		return g, nil
	}

	taskMap := make(map[string]rest.V1TaskSummary)
	for _, task := range tasks {
		taskMap[task.Metadata.Id] = task
	}

	for _, shapeItem := range shape {
		stepID := shapeItem.StepId.String()
		taskExternalID := shapeItem.TaskExternalId.String()

		var status rest.V1TaskStatus
		if task, found := taskMap[taskExternalID]; found {
			status = task.Status
		} else {
			status = rest.V1TaskStatus("PENDING")
		}

		node := &Node{
			StepID:         stepID,
			TaskName:       shapeItem.TaskName,
			TaskExternalID: taskExternalID,
			Status:         status,
			Parents:        make([]*Node, 0),
			Children:       make([]*Node, 0),
			ComponentID:    -1,
		}

		if err := g.AddNode(node); err != nil {
			return nil, fmt.Errorf("failed to add node %s: %w", stepID, err)
		}
	}

	for _, shapeItem := range shape {
		parentStepID := shapeItem.StepId.String()
		parentNode := g.GetNode(parentStepID)

		if parentNode == nil {
			return nil, fmt.Errorf("internal error: node %s not found after creation", parentStepID)
		}

		for _, childStepID := range shapeItem.ChildrenStepIds {
			childStepIDStr := childStepID.String()
			childNode := g.GetNode(childStepIDStr)

			if childNode == nil {
				return nil, fmt.Errorf("invalid graph: node %s references non-existent child %s", parentStepID, childStepIDStr)
			}

			if err := g.AddEdge(parentNode, childNode); err != nil {
				return nil, fmt.Errorf("failed to add edge %s -> %s: %w", parentStepID, childStepIDStr, err)
			}
		}
	}

	return g, nil
}

// HasCycle detects cycles using DFS
func (g *Graph) HasCycle() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for stepID := range g.Nodes {
		if !visited[stepID] {
			if g.hasCycleDFS(stepID, visited, recStack) {
				return true
			}
		}
	}

	return false
}

// hasCycleDFS is a helper for cycle detection using DFS
func (g *Graph) hasCycleDFS(stepID string, visited, recStack map[string]bool) bool {
	visited[stepID] = true
	recStack[stepID] = true

	node := g.Nodes[stepID]
	for _, child := range node.Children {
		if !visited[child.StepID] {
			if g.hasCycleDFS(child.StepID, visited, recStack) {
				return true
			}
		} else if recStack[child.StepID] {
			// Back edge found - cycle detected
			return true
		}
	}

	recStack[stepID] = false
	return false
}
