//go:build !e2e && !load && !rampup && !integration

package hatchet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WithWorkflowDisplayName threads a CEL expression into
// CreateWorkflowVersionRequest.display_name at registration.
func TestWithWorkflowDisplayName_ThreadsIntoRequest(t *testing.T) {
	c := newTestClient()
	wf := c.NewWorkflow("dn-wf", WithWorkflowDisplayName("input.customerName"))
	wf.NewTask("step", sampleTaskFn)

	req, _, _, _ := wf.Dump()

	require.NotNil(t, req.DisplayName)
	assert.Equal(t, "input.customerName", *req.DisplayName)
}

// WithDisplayName on a task threads a CEL expression into
// CreateTaskOpts.display_name at registration.
func TestWithDisplayName_ThreadsIntoTaskOpts(t *testing.T) {
	c := newTestClient()
	wf := c.NewWorkflow("dn-task-wf")
	wf.NewTask("step", sampleTaskFn, WithDisplayName("input.stepName"))

	req, _, _, _ := wf.Dump()

	require.Len(t, req.Tasks, 1)
	require.NotNil(t, req.Tasks[0].DisplayName)
	assert.Equal(t, "input.stepName", *req.Tasks[0].DisplayName)
}

// A standalone task accepts both a workflow-level and a task-level display-name
// expression and threads each into the correct proto field.
func TestStandaloneTask_DisplayName(t *testing.T) {
	c := newTestClient()
	task := c.NewStandaloneTask("dn-standalone", sampleTaskFn,
		WithWorkflowDisplayName("input.run"),
		WithDisplayName("input.step"),
	)

	req, _, _, _ := task.Dump()

	require.NotNil(t, req.DisplayName)
	assert.Equal(t, "input.run", *req.DisplayName)

	require.Len(t, req.Tasks, 1)
	require.NotNil(t, req.Tasks[0].DisplayName)
	assert.Equal(t, "input.step", *req.Tasks[0].DisplayName)
}

// Omitting the display-name options leaves both proto fields unset (nil), so the
// run falls back to a generated name — the pre-feature behavior.
func TestDisplayName_OmittedLeavesNil(t *testing.T) {
	c := newTestClient()
	wf := c.NewWorkflow("dn-none-wf")
	wf.NewTask("step", sampleTaskFn)

	req, _, _, _ := wf.Dump()

	assert.Nil(t, req.DisplayName)
	require.Len(t, req.Tasks, 1)
	assert.Nil(t, req.Tasks[0].DisplayName)
}
