# Hatchet Go SDK v1 Examples

This directory contains comprehensive examples for the new Hatchet Go SDK v1, which provides a clean, reflection-based API for building workflows in Go.

## Overview

The new SDK eliminates the need for generics while maintaining type safety through runtime validation. It offers a simple, intuitive API that's perfect for Go developers who want to get started quickly with workflow orchestration.

## Examples

### 1. Simple Workflow (`simple/`)
A basic workflow with a single task demonstrating the fundamental concepts.

**Features:**
- Basic workflow creation
- Simple task definition with input/output types
- Worker creation and execution

```bash
cd simple && go run main.go
```

### 2. DAG Workflow (`dag/`)
A complex workflow demonstrating task dependencies and parallel execution.

**Features:**
- Multi-step workflows with dependencies
- Parent task output access
- Parallel task execution
- Task result aggregation

```bash
cd dag && go run main.go
```

### 3. Durable Tasks (`durable/`)
Long-running tasks that can survive worker restarts.

**Features:**
- Durable context for persistent execution
- Sleep operations that survive restarts
- Configurable durable slots

```bash
cd durable && go run main.go
```

### 4. Event-Triggered Workflows (`events/`)
Workflows that respond to external events.

**Features:**
- Event-based workflow triggers
- Multiple event types
- Event payload access
- Filter conditions

```bash
cd events && go run main.go
```

### 5. Retries and Concurrency (`retries-concurrency/`)
Advanced task configuration with retry logic and concurrency controls.

**Features:**
- Exponential backoff retries
- Concurrency limiting per category
- Round-robin task distribution
- Timeout configuration
- Failure simulation

```bash
cd retries-concurrency && go run main.go
```

### 6. Complex Conditions (`conditions/`)
Advanced workflow control flow with conditional execution.

**Features:**
- Sleep conditions
- User event conditions
- Parent task condition evaluation
- Task skipping based on conditions
- OR/AND condition combinations
- Dynamic task execution paths

```bash
cd conditions && go run main.go
```

### 7. Cron Workflows (`cron/`)
Scheduled workflows that run on cron expressions.

**Features:**
- Multiple cron schedules
- Business hours scheduling
- Daily, hourly, and weekly jobs
- Complex cron expressions

```bash
cd cron && go run main.go
```

## Key Features of the New SDK

### Simple API
```go
client, err := hatchet.NewClient()
workflow := client.NewWorkflow("my-workflow")
workflow.NewTask("my-task", func(ctx hatchet.Context, input MyInput) (MyOutput, error) {
    // Task logic here
    return MyOutput{}, nil
})
```

### Type Safety
The SDK uses reflection to validate function signatures at runtime, ensuring type safety without generics:
- Context must implement `hatchet.Context` or `hatchet.DurableContext`
- Input and output can be any serializable Go types
- Return values must be `(output, error)`

### Task Dependencies
```go
step1 := workflow.AddTask("step-1", taskFunc1)
step2 := workflow.AddTask("step-2", taskFunc2, hatchet.WithParents(step1.namedTask))
```

### Advanced Configuration
```go
workflow.NewTask("my-task", taskFunc,
    hatchet.WithRetries(3),
    hatchet.WithRetryBackoff(2.0, 60),
    hatchet.WithTimeout(30*time.Second),
    hatchet.WithConcurrency(&types.Concurrency{...}),
    hatchet.WithWaitFor(hatchet.SleepCondition(10*time.Second)),
    hatchet.WithSkipIf(hatchet.ParentCondition(parentTask, "output.value > 100")),
)
```

### Worker Configuration
```go
worker, err := client.NewWorker("my-worker",
    hatchet.WithWorkflows(workflow1, workflow2),
    hatchet.WithSlots(10),
    hatchet.WithDurableSlots(5),
    hatchet.WithLabels(worker.WorkerLabels{"env": "production"}),
)
```

## Migration from Old SDK

If you're migrating from the older generic-based SDK, the main differences are:

1. **No generics**: Use `any` types with runtime validation instead
2. **Simplified API**: Direct method calls instead of factory patterns
3. **Reflection-based**: Function signatures validated at runtime
4. **Unified context**: Single context types for all operations

## Running Examples

Each example can be run independently:

```bash
# Make sure you have Hatchet running locally or configure connection
export HATCHET_CLIENT_TOKEN="your-token"
export HATCHET_CLIENT_HOST="your-hatchet-host"

cd <example-directory>
go run main.go
```

## Prerequisites

- Go 1.21 or later
- Running Hatchet instance (local or cloud)
- Proper environment configuration

For more information, see the [Hatchet documentation](https://docs.hatchet.run).