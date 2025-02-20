package dagutils_test

import (
	"testing"

	"github.com/hatchet-dev/hatchet/internal/dagutils"
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

func TestOrderWorkflowSteps(t *testing.T) {
	t.Run("valid ordering", func(t *testing.T) {
		steps := []repository.CreateWorkflowStepOpts{
			{
				ReadableId: "step1",
				Action:     "action1",
				Parents:    []string{},
			},
			{
				ReadableId: "step2",
				Action:     "action2",
				Parents:    []string{"step1"},
			},
			{
				ReadableId: "step3",
				Action:     "action3",
				Parents:    []string{"step2"},
			},
		}

		ordered, err := dagutils.OrderWorkflowSteps(steps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Validate that each step appears after its parents.
		orderIndex := make(map[string]int)
		for i, step := range ordered {
			orderIndex[step.ReadableId] = i
		}

		for _, step := range steps {
			for _, parent := range step.Parents {
				if orderIndex[parent] > orderIndex[step.ReadableId] {
					t.Errorf("step %q appears before its parent %q", step.ReadableId, parent)
				}
			}
		}
	})

	t.Run("unknown parent", func(t *testing.T) {
		steps := []repository.CreateWorkflowStepOpts{
			{
				ReadableId: "step1",
				Action:     "action1",
				Parents:    []string{"nonexistent"},
			},
		}

		_, err := dagutils.OrderWorkflowSteps(steps)
		if err == nil {
			t.Fatal("expected error for unknown parent, got nil")
		}
	})

	t.Run("cycle detection", func(t *testing.T) {
		steps := []repository.CreateWorkflowStepOpts{
			{
				ReadableId: "step1",
				Action:     "action1",
				Parents:    []string{"step3"},
			},
			{
				ReadableId: "step2",
				Action:     "action2",
				Parents:    []string{"step1"},
			},
			{
				ReadableId: "step3",
				Action:     "action3",
				Parents:    []string{"step2"},
			},
		}

		_, err := dagutils.OrderWorkflowSteps(steps)
		if err == nil {
			t.Fatal("expected error for cycle detection, got nil")
		}
	})
}
