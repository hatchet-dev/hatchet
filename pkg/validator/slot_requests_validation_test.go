//go:build !e2e && !load && !rampup && !integration

package validator_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

func TestSlotRequests_RejectsNonPositiveUnits(t *testing.T) {
	v := validator.NewDefaultValidator()

	desc := "desc"
	opts := &repository.CreateWorkflowVersionOpts{
		Name: "workflow-1",
		// Description is used unconditionally downstream in PutWorkflowVersion, so set it.
		Description: &desc,
		Tasks: []repository.CreateStepOpts{
			{
				ReadableId: "step-1",
				Action:     "svc:do",
				SlotRequests: map[string]int32{
					repository.SlotTypeDefault: 0,
				},
			},
		},
	}

	err := v.Validate(opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SlotRequests")
	require.Contains(t, err.Error(), "gt")
}

func TestSlotRequests_AllowsPositiveUnits(t *testing.T) {
	v := validator.NewDefaultValidator()

	desc := "desc"
	opts := &repository.CreateWorkflowVersionOpts{
		Name:        "workflow-1",
		Description: &desc,
		Tasks: []repository.CreateStepOpts{
			{
				ReadableId: "step-1",
				Action:     "svc:do",
				SlotRequests: map[string]int32{
					repository.SlotTypeDefault: 1,
					"gpu":                     2,
				},
			},
		},
	}

	require.NoError(t, v.Validate(opts))
}

func TestSlotRequests_RejectsEmptySlotTypeKey(t *testing.T) {
	v := validator.NewDefaultValidator()

	desc := "desc"
	opts := &repository.CreateWorkflowVersionOpts{
		Name:        "workflow-1",
		Description: &desc,
		Tasks: []repository.CreateStepOpts{
			{
				ReadableId: "step-1",
				Action:     "svc:do",
				SlotRequests: map[string]int32{
					"": 1,
				},
			},
		},
	}

	err := v.Validate(opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SlotRequests")
	require.Contains(t, err.Error(), "required")
}
