# AI Agent Workflow with Safety Constraints

This example demonstrates how to build a sophisticated AI agent system using Hatchet workflows that includes safety constraints, human-in-the-loop decision making, and autonomous research capabilities.

## Overview

The AI agent workflow processes customer queries through a secure, multi-step pipeline that automatically routes requests based on risk assessment and implements safety guardrails to prevent potentially harmful operations.

## Architecture

### Workflow Steps

The workflow consists of 4 sequential steps:

1. **Assess and Route** - Security risk assessment and routing logic
2. **Autonomous Research** - Safe tool execution with constraints  
3. **Human Loop Check** - Decision point for human intervention
4. **Generate Response** - Final response generation or escalation

### Key Features

- **Risk-based Routing**: Automatically categorizes queries as low, medium, or high risk
- **Safety Constraints**: Configurable limits on API requests, execution time, and tool access
- **Human-in-the-Loop**: Automatic escalation for high-risk or complex queries
- **Tool Safety**: Sandbox execution of research tools with monitoring
- **Confidence Scoring**: Response quality assessment

## Data Types

### CustomerQuery
```go
type CustomerQuery struct {
    QueryID    string         // Unique identifier
    CustomerID string         // Customer identifier  
    Message    string         // The actual query text
    Priority   string         // urgent, high, normal, low
    Context    map[string]any // Additional context data
}
```

### AgentState
Tracks the agent's current processing state:
```go
type AgentState struct {
    QueryID           string
    CurrentPhase      string
    ToolExecutions    []ToolExecution
    SafetyViolations  []string
    HumanReviewNeeded bool
    ResearchProgress  map[string]interface{}
    TotalAPIRequests  int
}
```

### AIAgentResult
The final workflow output:
```go
type AIAgentResult struct {
    QueryID         string
    Status          string          // "completed" or "human_review_required"
    Response        string
    ConfidenceScore float64
    HumanLoopSignal *HumanLoopSignal
    ProcessingTime  time.Duration
    SafetyCompliant bool
}
```

## Workflow Logic

### 1. Assess and Route

**Purpose**: Analyze the incoming query to determine risk level and appropriate routing.

**Risk Assessment**:
- **High Risk**: Contains keywords like "delete", "admin", "root", "system", "database", "production"
- **Medium Risk**: Technical queries containing "api", "integration" 
- **Low Risk**: All other queries

**Safety Limits by Risk Level**:
- **High Risk**: 5 max API requests, 2 minute timeout, requires approval for system operations
- **Medium/Low Risk**: 10 max API requests, 5 minute timeout

### 2. Autonomous Research

**Purpose**: Execute research tools safely within defined constraints.

**Available Tools**:
- `knowledge_search` - Search documentation and FAQs
- `api_documentation` - Look up API reference information  
- `code_analysis` - Analyze code structure and suggest improvements
- `system_health` - Check system status and performance metrics

**Safety Mechanisms**:
- Pre-execution safety checks for each tool
- API request counting and limits
- Automatic termination when limits exceeded
- Safety violation tracking

### 3. Human Loop Check

**Purpose**: Determine if human intervention is required.

**Escalation Triggers**:
- Safety constraints violated
- More than 2 tool failures occurred
- High-risk query flagged for manual review
- Agent explicitly requests human review

**Human Loop Signal**:
When escalation is needed, creates a `HumanLoopSignal` with:
- Reason for escalation
- Urgency level (high/normal)
- Complete agent state context
- 4-hour response SLA

### 4. Generate Response

**Purpose**: Create the final response or escalation notice.

**Automated Response Path**:
- Generates response based on successful tool executions
- Calculates confidence score based on success rate
- Penalizes confidence for safety violations

**Human Escalation Path**:
- Returns standardized escalation message
- Includes human loop signal for tracking
- Sets confidence score to 0.0

## Safety Features

### Request Limits
- Configurable maximum API requests per workflow execution
- Different limits based on risk assessment
- Automatic termination when limits exceeded

### Tool Restrictions  
- Approval-required tool lists for high-risk queries
- Safety checks before each tool execution
- Violation tracking and reporting

### Human Oversight
- Automatic escalation for high-risk scenarios
- Clear escalation reasons and urgency levels
- Complete context preservation for human reviewers

## Usage Examples

The example includes three different query types:

### 1. Technical Safety Query (High Risk)
```go
Query: "I need to debug our production API that's returning 500 errors and potentially delete some corrupted database entries."
Result: Escalated to human review due to high-risk keywords
```

### 2. Research Query (Normal Risk)
```go  
Query: "I need comprehensive research on implementing microservices architecture with Hatchet workflows."
Result: Autonomous completion with research findings
```

### 3. Billing Query (Low Risk)
```go
Query: "Can you explain my latest billing statement? I see some API usage charges."
Result: Autonomous completion with explanation
```

## Running the Example

### Prerequisites
- Go 1.24+
- Hatchet server running locally or accessible endpoint
- Proper Hatchet client configuration (environment variables or config file)

### Execution
```bash
go run main.go
```

The example will:
1. Start a Hatchet worker in the background
2. Execute three different query examples
3. Display results and processing information
4. Keep running to handle additional workflow triggers

### Expected Output
```
AI Agent worker started
Running workflow examples...
Assessing query tech-safety-001 with priority urgent
Human intervention required for query tech-safety-001: Safety constraints violated
Technical query result: human_review_required

Assessing query research-002 with priority normal  
Starting autonomous research for query research-002
Generated response for query research-002 with confidence 0.75
Research query result: completed

...
```

## Configuration

### Environment Variables
Standard Hatchet configuration applies:
- `HATCHET_CLIENT_TOKEN` - API authentication token
- `HATCHET_CLIENT_TLS_STRATEGY` - TLS configuration  
- `HATCHET_CLIENT_SERVER_URL` - Hatchet server endpoint

### Safety Limits
Modify the `defineSafetyLimits()` function to adjust:
- Maximum API requests per execution
- Execution timeouts
- Restricted tool lists
- Auto-stop behavior

### Risk Assessment
Customize the `assessSecurityRisk()` function to:
- Add/remove risk keywords
- Modify risk categorization logic
- Implement custom risk scoring

## Extension Points

### Adding New Tools
1. Add new case to `executeToolSafely()` function
2. Implement tool-specific logic and API request counting
3. Add tool name to research tools list in step 2

### Custom Risk Assessment
1. Modify `assessSecurityRisk()` function
2. Add new risk levels if needed
3. Update safety limits configuration accordingly  

### Enhanced Human Loop
1. Extend `HumanLoopSignal` with additional context
2. Implement external notification systems
3. Add approval workflow integration

## Best Practices

### Security
- Always validate and sanitize query inputs
- Implement proper authentication and authorization
- Log all safety violations for audit purposes
- Regularly review and update risk keywords

### Performance  
- Set appropriate timeouts for different query types
- Monitor tool execution times and success rates
- Implement circuit breakers for external API calls
- Cache frequently accessed information

### Monitoring
- Track safety violation rates
- Monitor human escalation frequency  
- Measure workflow completion times
- Alert on unusual patterns or failures

This example provides a solid foundation for building production AI agent systems with proper safety guardrails and human oversight capabilities.