package main

// CallLLM is a mock - no external LLM API.
// First call returns tool_calls; second returns final answer.
var llmCallCount int

type LLMResponse struct {
	Content   string
	ToolCalls []ToolCall
	Done      bool
}

type ToolCall struct {
	Name string
	Args map[string]interface{}
}

func CallLLM(messages []map[string]interface{}) LLMResponse {
	llmCallCount++
	if llmCallCount == 1 {
		return LLMResponse{
			Content:   "",
			ToolCalls: []ToolCall{{Name: "get_weather", Args: map[string]interface{}{"location": "SF"}}},
			Done:      false,
		}
	}
	return LLMResponse{Content: "It's 72°F and sunny in SF.", ToolCalls: nil, Done: true}
}

// RunTool is a mock - returns canned results.
func RunTool(name string, args map[string]interface{}) string {
	if name == "get_weather" {
		loc := "unknown"
		if v, ok := args["location"]; ok {
			loc = v.(string)
		}
		return "Weather in " + loc + ": 72°F, sunny"
	}
	return "Unknown tool: " + name
}
