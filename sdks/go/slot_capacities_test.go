package hatchet

import (
	"testing"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

func TestResolveWorkerSlotConfig_NoDurable(t *testing.T) {
	dumps := []workflowDump{
		{
			req: &v1.CreateWorkflowVersionRequest{
				Tasks: []*v1.CreateTaskOpts{
					{
						IsDurable:        false,
						SlotRequests: map[string]int32{"default": 1},
					},
				},
			},
		},
	}

	resolved := resolveWorkerSlotConfig(map[slotType]int{}, dumps)

	if resolved[slotTypeDefault] != 100 {
		t.Fatalf("expected default slots to be 100, got %d", resolved[slotTypeDefault])
	}
	if _, ok := resolved[slotTypeDurable]; ok {
		t.Fatalf("expected durable slots to be unset, got %d", resolved[slotTypeDurable])
	}
}

func TestResolveWorkerSlotConfig_OnlyDurable(t *testing.T) {
	dumps := []workflowDump{
		{
			req: &v1.CreateWorkflowVersionRequest{
				Tasks: []*v1.CreateTaskOpts{
					{
						IsDurable:        true,
						SlotRequests: map[string]int32{"durable": 1},
					},
				},
			},
		},
	}

	resolved := resolveWorkerSlotConfig(map[slotType]int{}, dumps)

	if resolved[slotTypeDurable] != 1000 {
		t.Fatalf("expected durable slots to be 1000, got %d", resolved[slotTypeDurable])
	}
	if _, ok := resolved[slotTypeDefault]; ok {
		t.Fatalf("expected default slots to be unset, got %d", resolved[slotTypeDefault])
	}
}

func TestResolveWorkerSlotConfig_Mixed(t *testing.T) {
	dumps := []workflowDump{
		{
			req: &v1.CreateWorkflowVersionRequest{
				Tasks: []*v1.CreateTaskOpts{
					{
						IsDurable:        false,
						SlotRequests: map[string]int32{"default": 1},
					},
					{
						IsDurable:        true,
						SlotRequests: map[string]int32{"durable": 1},
					},
				},
			},
		},
	}

	resolved := resolveWorkerSlotConfig(map[slotType]int{}, dumps)

	if resolved[slotTypeDefault] != 100 {
		t.Fatalf("expected default slots to be 100, got %d", resolved[slotTypeDefault])
	}
	if resolved[slotTypeDurable] != 1000 {
		t.Fatalf("expected durable slots to be 1000, got %d", resolved[slotTypeDurable])
	}
}
