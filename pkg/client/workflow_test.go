// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

func TestWorkflowResultSingleAddWorkflowRunAttempt(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		constructorCalls.Add(1)
		return nil, status.Error(codes.Unavailable, "engine down")
	}, nil)

	workflow := NewWorkflow("run-1", listener)
	_, err := workflow.Result()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen for workflow events")
	assert.LessOrEqual(t, constructorCalls.Load(), int32(retry.StreamSyncMaxAttempts))
	assert.Greater(t, constructorCalls.Load(), int32(0))

	assert.False(t, listener.reg.hasAny())
}

func TestWorkflowResultFailsWhenListenerDiesPermanently(t *testing.T) {
	logger := zerolog.Nop()
	runID := "run-permanent-fail"

	client := &mockSubscribeClient{
		recvFn: func() (*dispatchercontracts.WorkflowRunEvent, error) {
			return nil, status.Error(codes.Unauthenticated, "auth failed")
		},
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return client, nil
	}, client)

	workflow := NewWorkflow(runID, listener)

	resultDone := make(chan struct{})
	var resultErr error
	go func() {
		_, resultErr = workflow.Result()
		close(resultDone)
	}()

	require.Eventually(t, func() bool {
		select {
		case <-resultDone:
			return true
		default:
			return false
		}
	}, 5*time.Second, 10*time.Millisecond)

	require.Error(t, resultErr)
	assert.Contains(t, resultErr.Error(), runID)
	assert.ErrorIs(t, resultErr, status.Error(codes.Unauthenticated, "auth failed"))

	require.NoError(t, listener.Close())
}
