package v1

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

func Test_newTriggerOpt_normalizes_display_name(t *testing.T) {
	a := &AdminServiceImpl{}
	name := "  Acme Corp  "

	opt, err := a.newTriggerOpt(context.Background(), uuid.New(), &contracts.TriggerWorkflowRunRequest{
		WorkflowName: "my-workflow",
		DisplayName:  &name,
	})
	require.NoError(t, err)
	require.NotNil(t, opt.DisplayName, "REST display name should be carried onto the trigger opt")
	assert.Equal(t, "Acme Corp", *opt.DisplayName, "REST display name is trimmed via NormalizeDisplayName")
}

func Test_newTriggerOpt_omits_display_name_when_unset(t *testing.T) {
	a := &AdminServiceImpl{}

	opt, err := a.newTriggerOpt(context.Background(), uuid.New(), &contracts.TriggerWorkflowRunRequest{
		WorkflowName: "my-workflow",
	})
	require.NoError(t, err)
	assert.Nil(t, opt.DisplayName, "omitted display name stays unset so the run falls back to a generated name")
}
