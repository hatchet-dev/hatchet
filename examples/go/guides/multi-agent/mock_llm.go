package main

import "fmt"

var orchestratorCallCount int

type OrchestratorResponse struct {
	Done     bool
	Content  string
	ToolCall *struct {
		Name string
		Args map[string]string
	}
}

func MockOrchestratorLLM(messages []map[string]interface{}) OrchestratorResponse {
	orchestratorCallCount++
	switch orchestratorCallCount {
	case 1:
		return OrchestratorResponse{Done: false, ToolCall: &struct {
			Name string
			Args map[string]string
		}{Name: "research", Args: map[string]string{"task": "Find key facts about the topic"}}}
	case 2:
		return OrchestratorResponse{Done: false, ToolCall: &struct {
			Name string
			Args map[string]string
		}{Name: "writing", Args: map[string]string{"task": "Write a summary from the research"}}}
	default:
		return OrchestratorResponse{Done: true, Content: "Here is the final report combining research and writing."}
	}
}

func MockSpecialistLLM(task, role string) string {
	return fmt.Sprintf("[%s] Completed: %s", role, task)
}
