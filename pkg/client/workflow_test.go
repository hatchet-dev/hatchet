// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

func TestWorkflowResultSingleAddWorkflowRunAttempt(t *testing.T) {
	retry.SetStreamSleepHookForTesting(func(ctx context.Context, attempt int) error {
		return nil
	})
	t.Cleanup(retry.ResetStreamSleepHookForTesting)

	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
			return nil, status.Error(codes.Unavailable, "engine down")
		},
		l: &logger,
	}

	workflow := NewWorkflow("run-1", listener)
	_, err := workflow.Result()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen for workflow events")
	assert.LessOrEqual(t, constructorCalls.Load(), int32(retry.StreamSyncMaxAttempts))
	assert.Greater(t, constructorCalls.Load(), int32(0))

	_, loaded := listener.handlers.Load("run-1")
	assert.False(t, loaded)
}
