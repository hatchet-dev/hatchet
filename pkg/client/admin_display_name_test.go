package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

func TestWithDisplayName_setsRequestField(t *testing.T) {
	req := &v1contracts.TriggerWorkflowRequest{Name: "wf"}

	err := WithDisplayName("Acme Corp")(req)
	require.NoError(t, err)

	require.NotNil(t, req.DisplayName)
	assert.Equal(t, "Acme Corp", *req.DisplayName)
}

func TestNewChildWorkflowTriggerRequest_setsDisplayName(t *testing.T) {
	name := "Acme Corp"

	req, err := NewChildWorkflowTriggerRequest("child-wf", map[string]any{}, &ChildWorkflowOpts{DisplayName: &name}, nil)
	require.NoError(t, err)

	require.NotNil(t, req.DisplayName)
	assert.Equal(t, "Acme Corp", *req.DisplayName, "child spawn carries the display name onto the request")
}

func TestNewChildWorkflowTriggerRequest_omitsDisplayName(t *testing.T) {
	req, err := NewChildWorkflowTriggerRequest("child-wf", map[string]any{}, &ChildWorkflowOpts{}, nil)
	require.NoError(t, err)

	assert.Nil(t, req.DisplayName, "an unset display name leaves the request field nil")
}
