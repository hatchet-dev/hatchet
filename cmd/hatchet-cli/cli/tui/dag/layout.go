package dag

import (
	"fmt"
	"sort"
)

// LayoutGraph computes positions for all nodes within the available space
func (g *Graph) LayoutGraph(config *RenderConfig) error {
	if g.IsEmpty() {
		return nil
	}

	if g.ComponentCount() == 0 {
		return fmt.Errorf("components must be detected before layout (call DetectComponents)")
	}

	if g.HasCycle() {
		return fmt.Errorf("graph contains cycles, cannot layout")
	}

	for _, component := range g.Components {
		if err := component.layoutComponent(config); err != nil {
			return fmt.Errorf("failed to layout component %d: %w", component.ID, err)
		}
	}

	if err := g.arrangeComponents(config); err != nil {
		return fmt.Errorf("failed to arrange components: %w", err)
	}

	g.routeEdges()

	return nil
}

// layoutComponent performs hierarchical layout for a single component
func (c *Component) layoutComponent(config *RenderConfig) error {
	if len(c.Nodes) == 0 {
		return nil
	}

	layers := c.assignLayers()
	c.orderWithinLayers(layers)

	return c.assignCoordinates(layers, config)
}

// assignLayers places nodes into layers using topological sort
// Returns a map from stepID to layer number
func (c *Component) assignLayers() map[string]int {
	layers := make(map[string]int)

	for _, node := range c.Nodes {
		layers[node.StepID] = -1
	}

	for _, root := range c.RootNodes {
		layers[root.StepID] = 0
	}

	changed := true
	for changed {
		changed = false
		for _, node := range c.Nodes {
			if layers[node.StepID] == -1 {
				continue
			}

			for _, child := range node.Children {
				if child.ComponentID != c.ID {
					continue
				}

				newLayer := layers[node.StepID] + 1
				if layers[child.StepID] < newLayer {
					layers[child.StepID] = newLayer
					changed = true
				}
			}
		}
	}

	for _, node := range c.Nodes {
		node.Layer = layers[node.StepID]
	}

	return layers
}

// orderWithinLayers minimizes edge crossings using barycenter heuristic
func (c *Component) orderWithinLayers(layers map[string]int) {
	maxLayer := 0
	for _, layer := range layers {
		if layer > maxLayer {
			maxLayer = layer
		}
	}

	nodesPerLayer := make([][]*Node, maxLayer+1)
	for _, node := range c.Nodes {
		layer := layers[node.StepID]
		nodesPerLayer[layer] = append(nodesPerLayer[layer], node)
	}

	// Sort nodes within each layer by StepID for deterministic initial order
	for _, layerNodes := range nodesPerLayer {
		sort.Slice(layerNodes, func(i, j int) bool {
			return layerNodes[i].StepID < layerNodes[j].StepID
		})
	}

	for pass := 0; pass < 3; pass++ {
		for layer := 1; layer <= maxLayer; layer++ {
			c.sortLayerByBarycenter(nodesPerLayer[layer], nodesPerLayer[layer-1])
		}

		for layer := maxLayer - 1; layer >= 0; layer-- {
			if layer < maxLayer {
				c.sortLayerByBarycenter(nodesPerLayer[layer], nodesPerLayer[layer+1])
			}
		}
	}

	for _, nodes := range nodesPerLayer {
		for i, node := range nodes {
			node.GridY = i
		}
	}
}

// sortLayerByBarycenter sorts nodes in a layer by their barycenter relative to neighbors
func (c *Component) sortLayerByBarycenter(layer, neighborLayer []*Node) {
	type nodeWithBarycenter struct {
		node       *Node
		barycenter float64
	}

	withBarycenters := make([]nodeWithBarycenter, len(layer))

	for i, node := range layer {
		barycenter := c.calculateBarycenter(node, neighborLayer)
		withBarycenters[i] = nodeWithBarycenter{node, barycenter}
	}

	// Sort by barycenter, with StepID as tiebreaker for deterministic ordering
	sort.Slice(withBarycenters, func(i, j int) bool {
		if withBarycenters[i].barycenter != withBarycenters[j].barycenter {
			return withBarycenters[i].barycenter < withBarycenters[j].barycenter
		}
		// Use StepID as tiebreaker for stable sorting
		return withBarycenters[i].node.StepID < withBarycenters[j].node.StepID
	})

	for i, item := range withBarycenters {
		layer[i] = item.node
	}
}

// calculateBarycenter computes the average position of neighboring nodes
func (c *Component) calculateBarycenter(node *Node, neighborLayer []*Node) float64 {
	positions := make([]int, 0)

	for _, neighbor := range neighborLayer {
		for _, child := range neighbor.Children {
			if child.StepID == node.StepID {
				positions = append(positions, neighbor.GridY)
			}
		}
		for _, parent := range neighbor.Parents {
			if parent.StepID == node.StepID {
				positions = append(positions, neighbor.GridY)
			}
		}
	}

	if len(positions) == 0 {
		return float64(node.GridY)
	}

	sum := 0
	for _, pos := range positions {
		sum += pos
	}

	return float64(sum) / float64(len(positions))
}

// assignCoordinates converts layers to drawing coordinates with space checking
func (c *Component) assignCoordinates(layers map[string]int, config *RenderConfig) error {
	maxLayer := 0
	maxNodesInLayer := 0

	layerCounts := make(map[int]int)
	for _, layer := range layers {
		if layer > maxLayer {
			maxLayer = layer
		}
		layerCounts[layer]++
		if layerCounts[layer] > maxNodesInLayer {
			maxNodesInLayer = layerCounts[layer]
		}
	}

	requiredWidth := (maxLayer+1)*(config.NodeWidth+config.PaddingX) - config.PaddingX
	requiredHeight := maxNodesInLayer*(config.NodeHeight+config.PaddingY) - config.PaddingY

	if requiredWidth > config.MaxWidth || requiredHeight > config.MaxHeight {
		if config.CompactMode {
			return fmt.Errorf("component too large even in compact mode: needs %dx%d but only %dx%d available",
				requiredWidth, requiredHeight, config.MaxWidth, config.MaxHeight)
		}
		return fmt.Errorf("component too large: needs %dx%d but only %dx%d available (try compact mode)",
			requiredWidth, requiredHeight, config.MaxWidth, config.MaxHeight)
	}

	for _, node := range c.Nodes {
		node.Width = config.NodeWidth
		node.Height = config.NodeHeight

		node.GridX = node.Layer * (config.NodeWidth + config.PaddingX)
		node.GridY *= (config.NodeHeight + config.PaddingY)
	}

	c.BoundingBox = Rect{
		X:      0,
		Y:      0,
		Width:  requiredWidth,
		Height: requiredHeight,
	}

	return nil
}

// arrangeComponents positions multiple components on the canvas
func (g *Graph) arrangeComponents(config *RenderConfig) error {
	if len(g.Components) == 0 {
		return nil
	}

	if len(g.Components) == 1 {
		comp := g.Components[0]
		for _, node := range comp.Nodes {
			node.DrawX = node.GridX
			node.DrawY = node.GridY
		}
		g.ActualWidth = comp.BoundingBox.Width
		g.ActualHeight = comp.BoundingBox.Height
		return nil
	}

	sort.Slice(g.Components, func(i, j int) bool {
		sizeI := g.Components[i].BoundingBox.Width * g.Components[i].BoundingBox.Height
		sizeJ := g.Components[j].BoundingBox.Width * g.Components[j].BoundingBox.Height
		return sizeI > sizeJ
	})

	totalWidth := 0
	maxHeight := 0
	for _, comp := range g.Components {
		totalWidth += comp.BoundingBox.Width
		if comp.BoundingBox.Height > maxHeight {
			maxHeight = comp.BoundingBox.Height
		}
	}

	componentSpacing := config.PaddingX * 2
	totalWidthWithSpacing := totalWidth + componentSpacing*(len(g.Components)-1)

	if totalWidthWithSpacing <= config.MaxWidth {
		currentX := 0
		for _, comp := range g.Components {
			comp.BoundingBox.X = currentX
			comp.BoundingBox.Y = 0

			for _, node := range comp.Nodes {
				node.DrawX = currentX + node.GridX
				node.DrawY = node.GridY
			}

			currentX += comp.BoundingBox.Width + componentSpacing
		}

		g.ActualWidth = currentX - componentSpacing
		g.ActualHeight = maxHeight
		return nil
	}

	totalHeight := 0
	maxWidth := 0
	for _, comp := range g.Components {
		totalHeight += comp.BoundingBox.Height
		if comp.BoundingBox.Width > maxWidth {
			maxWidth = comp.BoundingBox.Width
		}
	}

	totalHeightWithSpacing := totalHeight + componentSpacing*(len(g.Components)-1)

	if totalHeightWithSpacing <= config.MaxHeight && maxWidth <= config.MaxWidth {
		currentY := 0
		for _, comp := range g.Components {
			comp.BoundingBox.X = 0
			comp.BoundingBox.Y = currentY

			for _, node := range comp.Nodes {
				node.DrawX = node.GridX
				node.DrawY = currentY + node.GridY
			}

			currentY += comp.BoundingBox.Height + componentSpacing
		}

		g.ActualWidth = maxWidth
		g.ActualHeight = currentY - componentSpacing
		return nil
	}

	return fmt.Errorf("cannot fit %d components in available space %dx%d",
		len(g.Components), config.MaxWidth, config.MaxHeight)
}

// routeEdges calculates paths for all edges
func (g *Graph) routeEdges() {
	for _, edge := range g.Edges {
		if edge.From.ComponentID != edge.To.ComponentID {
			continue
		}

		edge.Points = g.calculateEdgePath(edge.From, edge.To)
	}
}

// calculateEdgePath computes a Manhattan path between two nodes that avoids overlapping other nodes
func (g *Graph) calculateEdgePath(from, to *Node) []Coord {
	path := make([]Coord, 0)

	if g.Direction == "LR" {
		startX := from.DrawX + from.Width
		startY := from.DrawY + from.Height/2

		endX := to.DrawX
		endY := to.DrawY + to.Height/2

		path = append(path, Coord{X: startX, Y: startY})

		if startX != endX {
			// Find the best routing channel to avoid overlapping nodes
			midX := g.findBestRoutingChannel(from, to, startX, endX)

			path = append(path, Coord{X: midX, Y: startY})
			path = append(path, Coord{X: midX, Y: endY})
		}

		path = append(path, Coord{X: endX, Y: endY})
	}

	return path
}

// findBestRoutingChannel finds an X coordinate for the vertical routing segment that avoids nodes
func (g *Graph) findBestRoutingChannel(from, to *Node, startX, endX int) int {
	// Default to midpoint
	midX := startX + (endX-startX)/2

	// Calculate Y coordinates for the routing path
	startY := from.DrawY + from.Height/2
	endY := to.DrawY + to.Height/2

	// Get the Y range for the vertical segment
	minY := startY
	maxY := endY
	if startY > endY {
		minY = endY
		maxY = startY
	}

	// Find nodes in the same layer between source and destination
	intermediateNodes := make([]*Node, 0)
	for _, node := range g.Nodes {
		// Skip source and destination nodes
		if node.StepID == from.StepID || node.StepID == to.StepID {
			continue
		}

		// Check if node is in the same component
		if node.ComponentID != from.ComponentID {
			continue
		}

		// Check if node is between source and destination horizontally
		nodeLeft := node.DrawX
		nodeRight := node.DrawX + node.Width
		if nodeLeft < endX && nodeRight > startX {
			// Check if node overlaps vertically with our routing path
			nodeTop := node.DrawY
			nodeBottom := node.DrawY + node.Height
			if nodeTop < maxY && nodeBottom > minY {
				intermediateNodes = append(intermediateNodes, node)
			}
		}
	}

	// If no intermediate nodes, use simple midpoint
	if len(intermediateNodes) == 0 {
		return midX
	}

	// Try to find a channel that doesn't overlap any nodes
	// First, try routing on the right side of all intermediate nodes
	rightChannel := startX
	for _, node := range intermediateNodes {
		nodeRight := node.DrawX + node.Width
		if nodeRight > rightChannel {
			rightChannel = nodeRight + 1
		}
	}

	// Check if right channel is still before the destination
	if rightChannel < endX {
		return rightChannel
	}

	// Otherwise, try routing on the left side of all intermediate nodes
	leftChannel := endX
	for _, node := range intermediateNodes {
		nodeLeft := node.DrawX
		if nodeLeft < leftChannel {
			leftChannel = nodeLeft - 1
		}
	}

	// Check if left channel is still after the source
	if leftChannel > startX {
		return leftChannel
	}

	// If we can't find a clear channel, use the midpoint as fallback
	return midX
}
