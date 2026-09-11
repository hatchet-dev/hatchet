//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

func TestDAGOrchestratorActionId(t *testing.T) {
	id := DAGOrchestratorActionId("py314-d54c50e_CancelWorkflow")

	assert.Equal(t, "py314-d54c50e_cancelworkflow_orchestrator", id)
	assert.True(t, IsDAGOrchestratorActionId(id))

	_, err := types.ParseActionID(id)
	assert.Error(t, err, "an orchestrator id is not an action id a worker can register")

	for _, bad := range []string{"", "_orchestrator", "svc:run", "svc:run_orchestrator", "a;b_orchestrator", "A_orchestrator", "a_orchestrator ", "a_orchestrator_x"} {
		assert.False(t, IsDAGOrchestratorActionId(bad), bad)
	}
}
