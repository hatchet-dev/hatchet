package dagutils_test

import (
	"testing"

	"github.com/hatchet-dev/hatchet/internal/dagutils"
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

func TestHasCycle(t *testing.T) {
	tests := []struct {
		name     string
		steps    []repository.CreateWorkflowStepOpts
		expected bool
	}{
		{
			name: "No cycle",
			steps: []repository.CreateWorkflowStepOpts{
				{ReadableId: "Step1", Action: "Action1", Parents: []string{"Step2"}},
				{ReadableId: "Step2", Action: "Action2"},
			},
			expected: false,
		},
		{
			name: "Self referential cycle",
			steps: []repository.CreateWorkflowStepOpts{
				{ReadableId: "Step1", Action: "Action1", Parents: []string{"Step1"}},
			},
			expected: true,
		},
		{
			name: "Simple cycle",
			steps: []repository.CreateWorkflowStepOpts{
				{ReadableId: "Step1", Action: "Action1", Parents: []string{"Step2"}},
				{ReadableId: "Step2", Action: "Action2", Parents: []string{"Step1"}},
			},
			expected: true,
		},
		{
			name: "Complex cycle",
			steps: []repository.CreateWorkflowStepOpts{
				{ReadableId: "Step1", Action: "Action1", Parents: []string{"Step3"}},
				{ReadableId: "Step2", Action: "Action2", Parents: []string{"Step1"}},
				{ReadableId: "Step3", Action: "Action3", Parents: []string{"Step2"}},
			},
			expected: true,
		},
		{
			name:     "No Steps",
			steps:    []repository.CreateWorkflowStepOpts{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dagutils.HasCycle(tt.steps); got != tt.expected {
				t.Errorf("HasCycle() = %v, expected %v", got, tt.expected)
			}
		})
	}
}
