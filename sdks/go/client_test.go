package hatchet

import (
	"encoding/json"
	"strings"
	"testing"
)

// Mock Task implementation for testing
type mockTask struct {
	name string
}

func (m *mockTask) GetName() string {
	return m.name
}

type TestOutputStruct struct {
	Value  string `json:"value"`
	Number int    `json:"number"`
}

func TestTaskOutput_WithMapResult(t *testing.T) {
	// Test the core functionality with a map result (most common case)
	expectedOutput := TestOutputStruct{
		Value:  "test-value",
		Number: 42,
	}

	// Create a workflow result with task-specific outputs
	taskResults := map[string]any{
		"task1": expectedOutput,
		"task2": map[string]any{"other": "data"},
	}

	workflowResult := &WorkflowResult{
		Result: taskResults,
	}

	// Create mock task
	task1 := &mockTask{name: "task1"}

	// Test TaskOutput method
	taskResult := workflowResult.TaskOutput(task1.GetName())
	if taskResult == nil {
		t.Fatal("TaskOutput returned nil")
	}

	// Test Into method
	var actualOutput TestOutputStruct
	err := taskResult.Into(&actualOutput)
	if err != nil {
		t.Fatalf("TaskResult.Into() failed: %v", err)
	}

	// Verify the output
	if actualOutput.Value != expectedOutput.Value {
		t.Errorf("Expected Value %s, got %s", expectedOutput.Value, actualOutput.Value)
	}
	if actualOutput.Number != expectedOutput.Number {
		t.Errorf("Expected Number %d, got %d", expectedOutput.Number, actualOutput.Number)
	}
}

func TestTaskOutput_WithJSONStringResult(t *testing.T) {
	// Test with JSON string result (pointer to string)
	expectedOutput := TestOutputStruct{
		Value:  "json-test",
		Number: 456,
	}

	jsonBytes, err := json.Marshal(expectedOutput)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}
	jsonString := string(jsonBytes)

	taskResults := map[string]any{
		"json-task": &jsonString,
	}

	workflowResult := &WorkflowResult{
		Result: taskResults,
	}

	task := &mockTask{name: "json-task"}

	// Test TaskOutput and Into
	taskResult := workflowResult.TaskOutput(task.GetName())
	var actualOutput TestOutputStruct
	err = taskResult.Into(&actualOutput)
	if err != nil {
		t.Fatalf("TaskResult.Into() failed: %v", err)
	}

	if actualOutput.Value != expectedOutput.Value {
		t.Errorf("Expected Value %s, got %s", expectedOutput.Value, actualOutput.Value)
	}
	if actualOutput.Number != expectedOutput.Number {
		t.Errorf("Expected Number %d, got %d", expectedOutput.Number, actualOutput.Number)
	}
}

func TestTaskOutput_TaskNotFound(t *testing.T) {
	// Test behavior when task is not found in results
	taskResults := map[string]any{
		"existing-task": TestOutputStruct{Value: "exists", Number: 1},
	}

	workflowResult := &WorkflowResult{
		Result: taskResults,
	}

	// Request a task that doesn't exist
	nonExistentTask := &mockTask{name: "non-existent-task"}
	taskResult := workflowResult.TaskOutput(nonExistentTask.GetName())

	// Should return the entire result when task not found
	var actualOutput map[string]any
	err := taskResult.Into(&actualOutput)
	if err != nil {
		t.Fatalf("TaskResult.Into() failed: %v", err)
	}

	// Should contain all task results
	if len(actualOutput) != 1 {
		t.Errorf("Expected 1 result, got %d", len(actualOutput))
	}
}

func TestTaskOutput_SingleTaskWorkflow(t *testing.T) {
	// Test behavior with single task (common case)
	expectedOutput := TestOutputStruct{
		Value:  "single-task",
		Number: 789,
	}

	// Single task result map
	taskResults := map[string]any{
		"only-task": expectedOutput,
	}

	workflowResult := &WorkflowResult{
		Result: taskResults,
	}

	task := &mockTask{name: "only-task"}

	// Test TaskOutput and Into
	taskResult := workflowResult.TaskOutput(task.GetName())
	var actualOutput TestOutputStruct
	err := taskResult.Into(&actualOutput)
	if err != nil {
		t.Fatalf("TaskResult.Into() failed: %v", err)
	}

	if actualOutput.Value != expectedOutput.Value {
		t.Errorf("Expected Value %s, got %s", expectedOutput.Value, actualOutput.Value)
	}
	if actualOutput.Number != expectedOutput.Number {
		t.Errorf("Expected Number %d, got %d", expectedOutput.Number, actualOutput.Number)
	}
}

func TestTaskResult_Into_InvalidJSON(t *testing.T) {
	// Test error handling with invalid data
	taskResult := &TaskResult{
		Result: make(chan int), // Cannot be marshaled to JSON
	}

	var output TestOutputStruct
	err := taskResult.Into(&output)
	if err == nil {
		t.Fatal("Expected error when marshaling invalid data, got nil")
	}

	if !strings.Contains(err.Error(), "marshal") {
		t.Errorf("Expected marshal error, got: %v", err)
	}
}

func TestTaskResult_Into_WithPointerToInterface(t *testing.T) {
	// Test with pointer to interface{} (common internal representation)
	expectedOutput := TestOutputStruct{
		Value:  "pointer-test",
		Number: 999,
	}

	var data any = expectedOutput
	taskResult := &TaskResult{
		Result: &data,
	}

	var actualOutput TestOutputStruct
	err := taskResult.Into(&actualOutput)
	if err != nil {
		t.Fatalf("TaskResult.Into() failed: %v", err)
	}

	if actualOutput.Value != expectedOutput.Value {
		t.Errorf("Expected Value %s, got %s", expectedOutput.Value, actualOutput.Value)
	}
	if actualOutput.Number != expectedOutput.Number {
		t.Errorf("Expected Number %d, got %d", expectedOutput.Number, actualOutput.Number)
	}
}
