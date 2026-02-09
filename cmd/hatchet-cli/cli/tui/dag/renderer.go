package dag

import (
	"fmt"
)

// Render performs the complete graph rendering pipeline
// Returns the ASCII visualization as a string, or an error if the graph cannot be rendered
func Render(g *Graph, selectedStepID string) (string, error) {
	if g.IsEmpty() {
		return "", nil
	}

	config := &RenderConfig{
		NodeWidth:   20,
		NodeHeight:  3,
		PaddingX:    4,
		PaddingY:    2,
		MaxWidth:    g.MaxWidth,
		MaxHeight:   g.MaxHeight,
		CompactMode: false,
	}

	if err := prepareGraph(g, config); err != nil {
		return "", err
	}

	canvas := NewCanvas(g.ActualWidth, g.ActualHeight)

	renderNodes(canvas, g, selectedStepID)
	renderEdges(canvas, g)

	return canvas.ToString(), nil
}

// RenderCompact attempts rendering in compact mode when normal mode fails
func RenderCompact(g *Graph, selectedStepID string) (string, error) {
	if g.IsEmpty() {
		return "", nil
	}

	config := &RenderConfig{
		NodeWidth:   16,
		NodeHeight:  3,
		PaddingX:    2,
		PaddingY:    1,
		MaxWidth:    g.MaxWidth,
		MaxHeight:   g.MaxHeight,
		CompactMode: true,
	}

	if err := prepareGraph(g, config); err != nil {
		return "", err
	}

	canvas := NewCanvas(g.ActualWidth, g.ActualHeight)

	renderNodes(canvas, g, selectedStepID)
	renderEdges(canvas, g)

	return canvas.ToString(), nil
}

// prepareGraph runs the layout pipeline on the graph
func prepareGraph(g *Graph, config *RenderConfig) error {
	if g.ComponentCount() == 0 {
		componentCount := g.DetectComponents()
		if componentCount == 0 {
			return fmt.Errorf("no components detected in graph")
		}
	}

	if err := g.LayoutGraph(config); err != nil {
		return fmt.Errorf("layout failed: %w", err)
	}

	return nil
}

// renderNodes draws all nodes on the canvas
func renderNodes(canvas *Canvas, g *Graph, selectedStepID string) {
	for _, node := range g.Nodes {
		selected := node.StepID == selectedStepID
		canvas.DrawBox(
			node.DrawX,
			node.DrawY,
			node.Width,
			node.Height,
			node.TaskName,
			node.Status,
			selected,
		)
	}
}

// renderEdges draws all edges on the canvas
func renderEdges(canvas *Canvas, g *Graph) {
	for _, edge := range g.Edges {
		if len(edge.Points) > 0 {
			canvas.DrawPath(edge.Points)
		}
	}
}

// GetNodeAtPosition returns the node at the given canvas position, or nil if none
func GetNodeAtPosition(g *Graph, x, y int) *Node {
	for _, node := range g.Nodes {
		if x >= node.DrawX && x < node.DrawX+node.Width &&
			y >= node.DrawY && y < node.DrawY+node.Height {
			return node
		}
	}
	return nil
}

// GetNavigableNodes returns nodes in visual order for keyboard navigation
func GetNavigableNodes(g *Graph) []*Node {
	if g.IsEmpty() {
		return nil
	}

	nodes := make([]*Node, 0, len(g.Nodes))
	for _, node := range g.Nodes {
		nodes = append(nodes, node)
	}

	// Sort by vertical position, then horizontal
	// This creates a natural reading order for keyboard navigation
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			if nodes[i].DrawY > nodes[j].DrawY ||
				(nodes[i].DrawY == nodes[j].DrawY && nodes[i].DrawX > nodes[j].DrawX) {
				nodes[i], nodes[j] = nodes[j], nodes[i]
			}
		}
	}

	return nodes
}
