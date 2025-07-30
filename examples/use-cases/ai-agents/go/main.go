package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	v1worker "github.com/hatchet-dev/hatchet/pkg/v1/worker"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type CustomerQuery struct {
	QueryID    string         `json:"query_id"`
	CustomerID string         `json:"customer_id"`
	Message    string         `json:"message"`
	Priority   string         `json:"priority"`
	Context    map[string]any `json:"context"`
}

type AutonomousResearchRequest struct {
	QueryID      string        `json:"query_id"`
	ResearchGoal string        `json:"research_goal"`
	MaxDuration  time.Duration `json:"max_duration"`
	SafetyLimits SafetyLimits  `json:"safety_limits"`
	Tools        []string      `json:"tools"`
}

type SafetyLimits struct {
	MaxAPIRequests    int           `json:"max_api_requests"`
	MaxExecutionTime  time.Duration `json:"max_execution_time"`
	RestrictedDomains []string      `json:"restricted_domains"`
	RequiresApproval  []string      `json:"requires_approval"`
	AutoStop          bool          `json:"auto_stop"`
}

type AgentState struct {
	QueryID           string                 `json:"query_id"`
	CurrentPhase      string                 `json:"current_phase"`
	ToolExecutions    []ToolExecution        `json:"tool_executions"`
	SafetyViolations  []string               `json:"safety_violations"`
	HumanReviewNeeded bool                   `json:"human_review_needed"`
	ResearchProgress  map[string]interface{} `json:"research_progress"`
	TotalAPIRequests  int                    `json:"total_api_requests"`
}

type ToolExecution struct {
	Name        string        `json:"name"`
	StartTime   time.Time     `json:"start_time"`
	Duration    time.Duration `json:"duration"`
	Success     bool          `json:"success"`
	Result      interface{}   `json:"result"`
	SafetyCheck bool          `json:"safety_check"`
}

type HumanLoopSignal struct {
	QueryID    string     `json:"query_id"`
	Reason     string     `json:"reason"`
	Urgency    string     `json:"urgency"`
	Context    AgentState `json:"context"`
	Timestamp  time.Time  `json:"timestamp"`
	RequiredBy time.Time  `json:"required_by"`
}

type AIAgentInput struct {
	Query           CustomerQuery              `json:"query"`
	ResearchRequest *AutonomousResearchRequest `json:"research_request,omitempty"`
}

type AIAgentResult struct {
	QueryID         string           `json:"query_id"`
	Status          string           `json:"status"`
	Response        string           `json:"response"`
	ConfidenceScore float64          `json:"confidence_score"`
	HumanLoopSignal *HumanLoopSignal `json:"human_loop_signal,omitempty"`
	ProcessingTime  time.Duration    `json:"processing_time"`
	SafetyCompliant bool             `json:"safety_compliant"`
}

func AIAgentWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[AIAgentInput, AIAgentResult] {
	wf := factory.NewWorkflow[AIAgentInput, AIAgentResult](
		create.WorkflowCreateOpts[AIAgentInput]{
			Name: "ai-agent-workflow",
		},
		hatchet,
	)

	step1 := wf.Task(
		create.WorkflowTask[AIAgentInput, AIAgentResult]{
			Name: "assess-and-route",
		},
		func(ctx worker.HatchetContext, input AIAgentInput) (interface{}, error) {
			log.Printf("Assessing query %s with priority %s", input.Query.QueryID, input.Query.Priority)

			riskLevel := assessSecurityRisk(input.Query)
			route := determineRoute(input.Query, riskLevel)
			safetyLimits := defineSafetyLimits(input.Query, riskLevel)

			result := map[string]interface{}{
				"query_id":       input.Query.QueryID,
				"risk_level":     riskLevel,
				"route":          route,
				"safety_limits":  safetyLimits,
				"requires_human": riskLevel == "high" || input.Query.Priority == "urgent",
			}

			return result, nil
		},
	)

	step2 := wf.Task(
		create.WorkflowTask[AIAgentInput, AIAgentResult]{
			Name:    "autonomous-research",
			Parents: []create.NamedTask{step1},
		},
		func(ctx worker.HatchetContext, input AIAgentInput) (interface{}, error) {
			var routeMap map[string]interface{}
			err := ctx.ParentOutput(step1, &routeMap)
			if err != nil {
				return nil, fmt.Errorf("failed to get route data: %w", err)
			}
			queryID := routeMap["query_id"].(string)
			safetyLimits := routeMap["safety_limits"].(map[string]interface{})

			log.Printf("Starting autonomous research for query %s", queryID)

			state := AgentState{
				QueryID:          queryID,
				CurrentPhase:     "research",
				ToolExecutions:   []ToolExecution{},
				SafetyViolations: []string{},
				ResearchProgress: make(map[string]interface{}),
			}

			researchTools := []string{"knowledge_search", "api_documentation", "code_analysis", "system_health"}

			for _, toolName := range researchTools {
				if !checkToolSafety(toolName, safetyLimits, &state) {
					state.SafetyViolations = append(state.SafetyViolations,
						fmt.Sprintf("Tool %s blocked by safety constraints", toolName))
					continue
				}

				execution := executeToolSafely(toolName, &state)
				state.ToolExecutions = append(state.ToolExecutions, execution)

				maxRequests := int(safetyLimits["max_api_requests"].(float64))
				if state.TotalAPIRequests >= maxRequests {
					state.HumanReviewNeeded = true
					break
				}
			}

			return state, nil
		},
	)

	step3 := wf.Task(
		create.WorkflowTask[AIAgentInput, AIAgentResult]{
			Name:    "human-loop-check",
			Parents: []create.NamedTask{step2},
		},
		func(ctx worker.HatchetContext, input AIAgentInput) (interface{}, error) {
			var state AgentState
			err := ctx.ParentOutput(step2, &state)
			if err != nil {
				return nil, fmt.Errorf("failed to get research state: %w", err)
			}

			log.Printf("Checking if human intervention needed for query %s", state.QueryID)

			needsHuman := state.HumanReviewNeeded ||
				len(state.SafetyViolations) > 0 ||
				getFailedToolCount(state.ToolExecutions) > 2

			result := map[string]interface{}{
				"needs_human": needsHuman,
				"state":       state,
			}

			if needsHuman {
				signal := HumanLoopSignal{
					QueryID:    state.QueryID,
					Reason:     determineHumanReviewReason(state),
					Urgency:    determineUrgency(state),
					Context:    state,
					Timestamp:  time.Now(),
					RequiredBy: time.Now().Add(4 * time.Hour),
				}

				result["human_signal"] = signal
				log.Printf("Human intervention required for query %s: %s", state.QueryID, signal.Reason)
			}

			return result, nil
		},
	)

	wf.Task(
		create.WorkflowTask[AIAgentInput, AIAgentResult]{
			Name:    "generate-response",
			Parents: []create.NamedTask{step3},
		},
		func(ctx worker.HatchetContext, input AIAgentInput) (interface{}, error) {
			var humanCheckMap map[string]interface{}
			err := ctx.ParentOutput(step3, &humanCheckMap)
			if err != nil {
				return nil, fmt.Errorf("failed to get human check result: %w", err)
			}
			needsHuman := humanCheckMap["needs_human"].(bool)
			stateMap := humanCheckMap["state"].(map[string]interface{})
			
			state := AgentState{
				QueryID:           stateMap["query_id"].(string),
				CurrentPhase:      stateMap["current_phase"].(string),
				SafetyViolations:  convertToStringSlice(stateMap["safety_violations"]),
				HumanReviewNeeded: stateMap["human_review_needed"].(bool),
				TotalAPIRequests:  int(stateMap["total_api_requests"].(float64)),
			}

			startTime := time.Now()

			if needsHuman {
				signalMap := humanCheckMap["human_signal"].(map[string]interface{})
				signal := HumanLoopSignal{
					QueryID:    signalMap["query_id"].(string),
					Reason:     signalMap["reason"].(string),
					Urgency:    signalMap["urgency"].(string),
					Timestamp:  time.Now(),
					RequiredBy: time.Now().Add(4 * time.Hour),
				}

				return &AIAgentResult{
					QueryID:         state.QueryID,
					Status:          "human_review_required",
					Response:        "Your query requires expert human review. A specialist will respond within 4 hours.",
					ConfidenceScore: 0.0,
					HumanLoopSignal: &signal,
					ProcessingTime:  time.Since(startTime),
					SafetyCompliant: len(state.SafetyViolations) == 0,
				}, nil
			}

			response := generateAutomatedResponse(state)
			confidence := calculateConfidence(state)

			log.Printf("Generated response for query %s with confidence %.2f", state.QueryID, confidence)

			return &AIAgentResult{
				QueryID:         state.QueryID,
				Status:          "completed",
				Response:        response,
				ConfidenceScore: confidence,
				ProcessingTime:  time.Since(startTime),
				SafetyCompliant: len(state.SafetyViolations) == 0,
			}, nil
		},
	)

	return wf
}

func runExamples(ctx context.Context, aiWorkflow workflow.WorkflowDeclaration[AIAgentInput, AIAgentResult]) {
	result1, err := aiWorkflow.Run(ctx, AIAgentInput{
		Query: CustomerQuery{
			QueryID:    "tech-safety-001",
			CustomerID: "enterprise-123",
			Message:    "I need to debug our production API that's returning 500 errors and potentially delete some corrupted database entries.",
			Priority:   "urgent",
			Context: map[string]interface{}{
				"customer_tier":  "enterprise",
				"user_role":      "admin",
				"system_access":  true,
				"production_env": true,
			},
		},
	})
	if err != nil {
		log.Printf("Error running technical query: %v", err)
	} else {
		log.Printf("Technical query result: %s", result1.Status)
	}

	// Example 2: Autonomous research request
	result2, err := aiWorkflow.Run(ctx, AIAgentInput{
		Query: CustomerQuery{
			QueryID:    "research-002",
			CustomerID: "startup-456",
			Message:    "I need comprehensive research on implementing microservices architecture with Hatchet workflows.",
			Priority:   "normal",
			Context: map[string]interface{}{
				"customer_tier": "starter",
				"user_role":     "developer",
				"research_task": true,
			},
		},
		ResearchRequest: &AutonomousResearchRequest{
			QueryID:      "research-002",
			ResearchGoal: "Gather microservices implementation patterns and best practices",
			MaxDuration:  5 * time.Minute,
			SafetyLimits: SafetyLimits{
				MaxAPIRequests:    15,
				MaxExecutionTime:  4 * time.Minute,
				RestrictedDomains: []string{"internal.company.com"},
				AutoStop:          true,
			},
			Tools: []string{"knowledge_search", "api_documentation", "code_analysis"},
		},
	})
	if err != nil {
		log.Printf("Error running research query: %v", err)
	} else {
		log.Printf("Research query result: %s", result2.Status)
	}

	// Example 3: Standard billing query
	result3, err := aiWorkflow.Run(ctx, AIAgentInput{
		Query: CustomerQuery{
			QueryID:    "billing-003",
			CustomerID: "pro-789",
			Message:    "Can you explain my latest billing statement? I see some API usage charges.",
			Priority:   "normal",
			Context: map[string]interface{}{
				"customer_tier": "pro",
				"user_role":     "admin",
				"billing_query": true,
			},
		},
	})
	if err != nil {
		log.Printf("Error running billing query: %v", err)
	} else {
		log.Printf("Billing query result: %s", result3.Status)
	}

	log.Println("All workflow examples completed")
}

func convertToStringSlice(val interface{}) []string {
	if val == nil {
		return []string{}
	}
	if slice, ok := val.([]interface{}); ok {
		result := make([]string, len(slice))
		for i, v := range slice {
			result[i] = v.(string)
		}
		return result
	}
	return []string{}
}

func assessSecurityRisk(query CustomerQuery) string {
	message := strings.ToLower(query.Message)

	highRiskKeywords := []string{"delete", "admin", "root", "system", "database", "production"}
	for _, keyword := range highRiskKeywords {
		if strings.Contains(message, keyword) {
			return "high"
		}
	}

	if strings.Contains(message, "api") || strings.Contains(message, "integration") {
		return "medium"
	}

	return "low"
}

func determineRoute(query CustomerQuery, riskLevel string) string {
	if riskLevel == "high" {
		return "senior-security-agent"
	}

	message := strings.ToLower(query.Message)
	if strings.Contains(message, "billing") {
		return "billing-specialist"
	}
	if strings.Contains(message, "technical") || strings.Contains(message, "api") {
		return "technical-expert"
	}

	return "general-agent"
}

func defineSafetyLimits(_ CustomerQuery, riskLevel string) map[string]interface{} {
	limits := map[string]interface{}{
		"max_api_requests":   10,
		"max_execution_time": 300,
		"auto_stop":          true,
	}

	if riskLevel == "high" {
		limits["max_api_requests"] = 5
		limits["max_execution_time"] = 120
		limits["requires_approval"] = []string{"system_admin", "database_query"}
	}

	return limits
}

func checkToolSafety(toolName string, safetyLimits map[string]interface{}, state *AgentState) bool {
	maxRequests := int(safetyLimits["max_api_requests"].(float64))
	if state.TotalAPIRequests >= maxRequests {
		return false
	}

	if requiresApproval, exists := safetyLimits["requires_approval"]; exists {
		if approvalList, ok := requiresApproval.([]string); ok {
			for _, restricted := range approvalList {
				if toolName == restricted {
					return false
				}
			}
		}
	}

	return true
}

func executeToolSafely(toolName string, state *AgentState) ToolExecution {
	startTime := time.Now()
	execution := ToolExecution{
		Name:        toolName,
		StartTime:   startTime,
		SafetyCheck: true,
	}

	switch toolName {
	case "knowledge_search":
		execution.Result = map[string]interface{}{
			"results":    []string{"Documentation Link 1", "FAQ Entry 2"},
			"confidence": 0.85,
		}
		execution.Success = true
		state.TotalAPIRequests += 2
	case "api_documentation":
		execution.Result = map[string]interface{}{
			"endpoint":   "/api/v1/example",
			"method":     "GET",
			"parameters": []string{"id", "format"},
		}
		execution.Success = true
		state.TotalAPIRequests += 1
	case "code_analysis":
		execution.Result = map[string]interface{}{
			"analysis":    "Code structure appears valid",
			"suggestions": []string{"Add error handling", "Implement rate limiting"},
		}
		execution.Success = true
		state.TotalAPIRequests += 3
	case "system_health":
		execution.Result = map[string]interface{}{
			"status":        "operational",
			"uptime":        "99.9%",
			"response_time": "45ms",
		}
		execution.Success = true
		state.TotalAPIRequests += 1
	default:
		execution.Result = "Unknown tool"
		execution.Success = false
	}

	execution.Duration = time.Since(startTime)
	return execution
}

func getFailedToolCount(executions []ToolExecution) int {
	count := 0
	for _, exec := range executions {
		if !exec.Success {
			count++
		}
	}
	return count
}

func determineHumanReviewReason(state AgentState) string {
	if len(state.SafetyViolations) > 0 {
		return "Safety constraints violated"
	}
	if getFailedToolCount(state.ToolExecutions) > 2 {
		return "Multiple tool failures detected"
	}
	if state.HumanReviewNeeded {
		return "Complex query requires expert review"
	}
	return "Manual review requested"
}

func determineUrgency(state AgentState) string {
	if len(state.SafetyViolations) > 0 {
		return "high"
	}
	return "normal"
}

func generateAutomatedResponse(state AgentState) string {
	successfulTools := 0
	for _, exec := range state.ToolExecutions {
		if exec.Success {
			successfulTools++
		}
	}

	if successfulTools == 0 {
		return "I apologize, but I encountered issues while processing your request. Please contact support for assistance."
	}

	return fmt.Sprintf("Based on my analysis using %d tools, I can help you with your query. Here's what I found...", successfulTools)
}

func calculateConfidence(state AgentState) float64 {
	if len(state.ToolExecutions) == 0 {
		return 0.0
	}

	successfulTools := float64(getSuccessfulToolCount(state.ToolExecutions))
	totalTools := float64(len(state.ToolExecutions))

	confidence := successfulTools / totalTools

	// Penalize for safety violations
	if len(state.SafetyViolations) > 0 {
		confidence *= 0.7
	}

	return confidence
}

func getSuccessfulToolCount(executions []ToolExecution) int {
	count := 0
	for _, exec := range executions {
		if exec.Success {
			count++
		}
	}
	return count
}

func main() {
	hatchet, err := v1.NewHatchetClient()
	if err != nil {
		log.Fatalf("error creating hatchet client: %v", err)
	}

	aiWorkflow := AIAgentWorkflow(hatchet)

	worker, err := hatchet.Worker(v1worker.WorkerOpts{
		Name: "ai-agent-worker",
		Workflows: []workflow.WorkflowBase{
			aiWorkflow,
		},
	})
	if err != nil {
		log.Fatalf("error creating worker: %v", err)
	}

	go func() {
		log.Println("AI Agent worker started")
		err := worker.StartBlocking(context.Background())
		if err != nil {
			log.Fatalf("error starting worker: %v", err)
		}
	}()

	ctx := context.Background()
	log.Println("Running workflow examples...")
	runExamples(ctx, aiWorkflow)

	select {}
}
