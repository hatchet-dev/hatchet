package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

func displayNamePtr(s string) *string { return &s }

// getCreateWorkflowOpts must copy the workflow-level display_name CEL expression
// from the proto request into the repository opts, and getCreateTaskOpts must copy
// the task-level expression into each step. If this conversion hop drops the field,
// display names sent by every SDK never reach validation/storage/evaluation, so the
// feature is silently a no-op through normal gRPC registration. Regression test.
func TestGetCreateWorkflowOpts_ThreadsDisplayNameExpressions(t *testing.T) {
	req := &contracts.CreateWorkflowVersionRequest{
		Name:        "test-wf",
		DisplayName: displayNamePtr("input.customerName"),
		Tasks: []*contracts.CreateTaskOpts{
			{
				ReadableId:  "step1",
				Action:      "test-wf:step1",
				DisplayName: displayNamePtr("'enrich-' + input.name"),
			},
		},
	}

	opts, err := getCreateWorkflowOpts(req)
	require.NoError(t, err)

	require.NotNil(t, opts.DisplayName, "workflow-level display_name must reach the repository opts")
	assert.Equal(t, "input.customerName", *opts.DisplayName)

	require.Len(t, opts.Tasks, 1)
	require.NotNil(t, opts.Tasks[0].DisplayName, "task-level display_name must reach the step opts")
	assert.Equal(t, "'enrich-' + input.name", *opts.Tasks[0].DisplayName)
}

// When the request carries no display_name, the opts must leave it nil so the engine
// falls back to the generated <name>-<timestamp> label rather than an empty string.
func TestGetCreateWorkflowOpts_NilDisplayNameStaysNil(t *testing.T) {
	req := &contracts.CreateWorkflowVersionRequest{
		Name: "test-wf",
		Tasks: []*contracts.CreateTaskOpts{
			{ReadableId: "step1", Action: "test-wf:step1"},
		},
	}

	opts, err := getCreateWorkflowOpts(req)
	require.NoError(t, err)

	assert.Nil(t, opts.DisplayName)
	require.Len(t, opts.Tasks, 1)
	assert.Nil(t, opts.Tasks[0].DisplayName)
}
