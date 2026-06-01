//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type dagPayloadInput struct {
	WorkflowKey string `json:"workflow_key"`
}

// Regression test for empty payloads on task replay. Runs 10 at the same time because the original bug was from
// go's map iteration ordering, which is nondeterministic, so would be flaky otherwise.
func TestDAGPayloadFreshRunConcurrent(t *testing.T) {
	const n = 10
	ctx := newTestContext(t)

	refs := make([]*hatchet.WorkflowRunRef, n)
	for i := range n {
		ref, err := testDAGPayloadWorkflow.RunNoWait(ctx, dagPayloadInput{WorkflowKey: "test-value"})
		require.NoError(t, err)
		refs[i] = ref
	}

	for i, ref := range refs {
		_, err := ref.Result()
		require.NoError(t, err, "run %d: step B failed (payload not propagated)", i)
	}
}
