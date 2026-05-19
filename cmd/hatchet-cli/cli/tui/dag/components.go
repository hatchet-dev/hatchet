package dag

import "sort"

// Connected component detection for forest handling
//
// Workflow graphs may contain multiple disconnected subgraphs (a forest).
// This module detects and groups nodes into connected components for
// independent layout and positioning.

// DetectComponents finds all connected components using DFS
func (g *Graph) DetectComponents() int {
	if g.IsEmpty() {
		return 0
	}

	g.Components = make([]*Component, 0)
	visited := make(map[string]bool)
	componentID := 0

	// Get sorted list of step IDs for deterministic iteration
	stepIDs := make([]string, 0, len(g.Nodes))
	for stepID := range g.Nodes {
		stepIDs = append(stepIDs, stepID)
	}
	sort.Strings(stepIDs)

	// Iterate in sorted order for deterministic component detection
	for _, stepID := range stepIDs {
		node := g.Nodes[stepID]
		if !visited[stepID] {
			component := &Component{
				ID:        componentID,
				Nodes:     make([]*Node, 0),
				RootNodes: make([]*Node, 0),
			}

			g.exploreComponent(node, visited, component)

			// Sort root nodes deterministically by StepID
			for _, n := range component.Nodes {
				if len(n.Parents) == 0 {
					component.RootNodes = append(component.RootNodes, n)
				}
			}
			sort.Slice(component.RootNodes, func(i, j int) bool {
				return component.RootNodes[i].StepID < component.RootNodes[j].StepID
			})

			g.Components = append(g.Components, component)
			componentID++
		}
	}

	return len(g.Components)
}

// exploreComponent performs DFS to find all nodes in a component
func (g *Graph) exploreComponent(node *Node, visited map[string]bool, component *Component) {
	if visited[node.StepID] {
		return
	}

	visited[node.StepID] = true
	node.ComponentID = component.ID
	component.Nodes = append(component.Nodes, node)

	for _, child := range node.Children {
		g.exploreComponent(child, visited, component)
	}
	for _, parent := range node.Parents {
		g.exploreComponent(parent, visited, component)
	}
}

// GetComponent returns the component with the given ID
func (g *Graph) GetComponent(id int) *Component {
	if id < 0 || id >= len(g.Components) {
		return nil
	}
	return g.Components[id]
}

// GetNodeComponent returns the component containing the node
func (g *Graph) GetNodeComponent(stepID string) *Component {
	node := g.GetNode(stepID)
	if node == nil || node.ComponentID < 0 {
		return nil
	}
	return g.GetComponent(node.ComponentID)
}

// ComponentStats returns statistics about the components
type ComponentStats struct {
	TotalComponents   int
	LargestComponent  int // Number of nodes
	SmallestComponent int // Number of nodes
	IsolatedNodes     int // Single-node components
}

// GetComponentStats returns component statistics
func (g *Graph) GetComponentStats() ComponentStats {
	stats := ComponentStats{
		TotalComponents:   len(g.Components),
		LargestComponent:  0,
		SmallestComponent: 0,
		IsolatedNodes:     0,
	}

	if len(g.Components) == 0 {
		return stats
	}

	// Initialize min with a large value
	stats.SmallestComponent = len(g.Nodes) + 1

	for _, comp := range g.Components {
		nodeCount := len(comp.Nodes)

		if nodeCount > stats.LargestComponent {
			stats.LargestComponent = nodeCount
		}
		if nodeCount < stats.SmallestComponent {
			stats.SmallestComponent = nodeCount
		}
		if nodeCount == 1 {
			stats.IsolatedNodes++
		}
	}

	return stats
}
