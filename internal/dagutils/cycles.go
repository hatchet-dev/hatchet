package dagutils

import "github.com/hatchet-dev/hatchet/pkg/repository"

func HasCycle(steps []repository.CreateWorkflowStepOpts) bool {
	graph := make(map[string][]string)
	for _, step := range steps {
		graph[step.ReadableId] = step.Parents
	}

	visited := make(map[string]bool)
	var dfs func(string) bool

	dfs = func(node string) bool {
		if seen, ok := visited[node]; ok && seen {
			return true
		}
		if _, ok := graph[node]; !ok {
			return false
		}
		visited[node] = true
		for _, parent := range graph[node] {
			if dfs(parent) {
				return true
			}
		}
		visited[node] = false
		return false
	}

	for _, step := range steps {
		if dfs(step.ReadableId) {
			return true
		}
	}
	return false
}
