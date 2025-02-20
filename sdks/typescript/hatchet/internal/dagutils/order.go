package dagutils

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/repository"
)

func OrderWorkflowSteps(steps []repository.CreateWorkflowStepOpts) ([]repository.CreateWorkflowStepOpts, error) {
	// Build a map of step id to step for quick lookup.
	stepMap := make(map[string]repository.CreateWorkflowStepOpts)
	for _, step := range steps {
		stepMap[step.ReadableId] = step
	}

	// Initialize in-degree map and adjacency list graph.
	inDegree := make(map[string]int)
	graph := make(map[string][]string)
	for _, step := range steps {
		inDegree[step.ReadableId] = 0
	}

	// Build the graph and compute in-degrees.
	for _, step := range steps {
		for _, parent := range step.Parents {
			if _, exists := stepMap[parent]; !exists {
				return nil, fmt.Errorf("unknown parent step: %s", parent)
			}
			graph[parent] = append(graph[parent], step.ReadableId)
			inDegree[step.ReadableId]++
		}
	}

	// Queue for steps with no incoming edges.
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var ordered []repository.CreateWorkflowStepOpts
	// Process the steps in topological order.
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		ordered = append(ordered, stepMap[id])
		for _, child := range graph[id] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	// If not all steps are processed, there is a cycle.
	if len(ordered) != len(steps) {
		return nil, fmt.Errorf("cycle detected in workflow steps")
	}

	return ordered, nil
}
