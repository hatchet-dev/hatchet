package operatorsvc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// A bidi run stream hands the opening message and later sends to the dispatcher's entry and
// yields what the engine answers; Close ends the engine's side.
func TestOpenRunStreamBidi(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, _ := durableSession(t, svc, tenant)

	first := &contracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "run-1"}
	rs, err := session.OpenRunStream(t.Context(), operator.RunStreamWorkflowRuns, first)
	require.NoError(t, err)

	streams := svc.dispatcher.RunStreams()
	require.Len(t, streams, 1)
	assert.Equal(t, operator.RunStreamWorkflowRuns, streams[0].Kind)
	assert.Same(t, first, streams[0].First)

	require.NoError(t, rs.Send(t.Context(), &contracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "run-2"}))

	select {
	case sent := <-streams[0].Requests:
		assert.Equal(t, "run-2", sent.(*contracts.SubscribeToWorkflowRunsRequest).WorkflowRunId)
	case <-time.After(5 * time.Second):
		t.Fatal("the request never reached the dispatcher")
	}

	event := &contracts.WorkflowRunEvent{WorkflowRunId: "run-2"}
	streams[0].Responses <- event

	got, err := rs.Recv(t.Context())
	require.NoError(t, err)
	assert.Same(t, event, got)

	require.NoError(t, rs.Close())

	select {
	case <-streams[0].Done():
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not end the engine's side")
	}

	_, err = rs.Recv(t.Context())
	assert.ErrorIs(t, err, operatorsvc.ErrChannelClosed)
	assert.ErrorIs(t, rs.Send(t.Context(), first), operatorsvc.ErrChannelClosed)
}

// A server stream takes no sends; the engine ending it is ErrStreamEnded on Recv.
func TestOpenRunStreamServerStreamEnds(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, _ := durableSession(t, svc, tenant)

	_, err := session.OpenRunStream(t.Context(), operator.RunStreamWorkflowEvents, nil)
	require.Error(t, err, "a server stream needs its request")

	runId := "run-1"
	rs, err := session.OpenRunStream(t.Context(), operator.RunStreamWorkflowEvents, &contracts.SubscribeToWorkflowEventsRequest{WorkflowRunId: &runId})
	require.NoError(t, err)

	t.Cleanup(func() { _ = rs.Close() })

	assert.ErrorIs(t, rs.Send(t.Context(), &contracts.SubscribeToWorkflowEventsRequest{}), operator.ErrNotSupported)

	streams := svc.dispatcher.RunStreams()
	require.Len(t, streams, 1)
	assert.Nil(t, streams[0].Requests)

	streams[0].Responses <- &contracts.WorkflowEvent{WorkflowRunId: runId}
	streams[0].End()

	got, err := rs.Recv(t.Context())
	require.NoError(t, err)
	assert.Equal(t, runId, got.(*contracts.WorkflowEvent).WorkflowRunId)

	_, err = rs.Recv(t.Context())
	assert.ErrorIs(t, err, operator.ErrStreamEnded)
}

// A refused registration surfaces from OpenRunStream, and a Recv bounded by its context
// returns the context's error.
func TestOpenRunStreamErrors(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, _ := durableSession(t, svc, tenant)

	rs, err := session.OpenRunStream(t.Context(), operator.RunStreamDurableEvents, nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = rs.Close() })

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()

	_, err = rs.Recv(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	refused := errors.New("no")
	svc.dispatcher.FailRegisterRunStream(refused)

	_, err = session.OpenRunStream(t.Context(), operator.RunStreamDurableEvents, &v1contracts.ListenForDurableEventRequest{})
	assert.ErrorIs(t, err, refused)
}
