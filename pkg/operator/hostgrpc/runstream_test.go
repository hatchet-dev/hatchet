//go:build !e2e && !load && !rampup && !integration

package hostgrpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// OpenRunStream goes to the current client session with the kind and the opening message,
// and the stream it returns is the client's.
func TestOpenRunStreamDelegatesToClientSession(t *testing.T) {
	s, fs := openTestSession(t)

	first := &contracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "run-1"}
	rs, err := s.OpenRunStream(context.Background(), operator.RunStreamWorkflowRuns, first)
	require.NoError(t, err)

	opened := fs.openedRunStreams()
	require.Len(t, opened, 1)
	assert.Equal(t, operator.RunStreamWorkflowRuns, opened[0].kind)
	assert.Same(t, first, opened[0].first)

	require.NoError(t, rs.Send(context.Background(), &contracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "run-2"}))

	msg, err := rs.Recv(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "run-2", msg.(*contracts.SubscribeToWorkflowRunsRequest).WorkflowRunId)

	require.NoError(t, rs.Close())
}

// A closed session opens nothing.
func TestOpenRunStreamRefusedAfterClose(t *testing.T) {
	s, fs := openTestSession(t)

	require.NoError(t, s.Close(context.Background()))

	_, err := s.OpenRunStream(context.Background(), operator.RunStreamDurableEvents, nil)
	assert.ErrorIs(t, err, operator.ErrSessionClosed)
	assert.Empty(t, fs.openedRunStreams())
}
