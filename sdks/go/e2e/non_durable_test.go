//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNonDurableStandaloneTaskRun(t *testing.T) {
	ctx := newTestContext(t)

	result, err := testSpawnChildTask.Run(ctx, DurableBulkSpawnInput{N: 7})
	require.NoError(t, err)

	var output map[string]string
	err = result.Into(&output)
	require.NoError(t, err)
	assert.Equal(t, "hello from child 7", output["message"])
}

func TestNonDurableStandaloneTaskRunNoWaitResult(t *testing.T) {
	ctx := newTestContext(t)

	ref, err := testSpawnChildTask.RunNoWait(ctx, DurableBulkSpawnInput{N: 8})
	require.NoError(t, err)

	result := workflowResultWithin(t, ref, 20*time.Second)
	var output map[string]string
	err = result.TaskOutput("spawn-child-task").Into(&output)
	require.NoError(t, err)
	assert.Equal(t, "hello from child 8", output["message"])
}
