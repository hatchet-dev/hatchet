package operatorclient

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// blockedRunStream is a grpcRunStream over a transport whose send and receive return only
// once the stream context is cancelled, the way a gRPC send blocked on flow control does.
func blockedRunStream() (*grpcRunStream, context.Context) {
	ctx, cancel := context.WithCancel(context.Background())

	rs := &grpcRunStream{
		cancel: cancel,
		queue:  make(chan runStreamItem, runStreamQueueSize),
		closed: make(chan struct{}),
		done:   make(chan struct{}),
	}

	rs.send = func(proto.Message) error {
		<-ctx.Done()
		return ctx.Err()
	}
	rs.recv = func() (proto.Message, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	go rs.readLoop()

	return rs, ctx
}

// Close returns promptly while a Send is blocked on the transport: it cancels the stream,
// which is what releases the send, without waiting for the send's lock first.
func TestRunStreamCloseWhileSendBlocked(t *testing.T) {
	rs, ctx := blockedRunStream()

	sent := make(chan error, 1)

	go func() {
		sent <- rs.Send(context.Background(), &dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "run-1"})
	}()

	// Wait for the send to reach the transport.
	require.Eventually(t, func() bool {
		if !rs.sendMu.TryLock() {
			return true
		}

		rs.sendMu.Unlock()

		return false
	}, time.Second, time.Millisecond, "the send never reached the transport")

	closed := make(chan struct{})

	go func() {
		defer close(closed)
		_ = rs.Close()
	}()

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close waited behind the Send blocked on the transport")
	}

	assert.ErrorIs(t, ctx.Err(), context.Canceled, "Close cancelled the stream")

	select {
	case err := <-sent:
		assert.ErrorIs(t, err, context.Canceled, "the blocked send was released by the cancellation")
	case <-time.After(time.Second):
		t.Fatal("the blocked send was not released")
	}

	err := rs.Send(context.Background(), &dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "run-2"})
	assert.ErrorIs(t, err, operator.ErrChannelClosed)

	// The reader's final error may or may not have been queued before the close was seen.
	_, err = rs.Recv(context.Background())
	assert.Error(t, err)
}

// blockingDispatcher is a DispatcherClient whose SubscribeToWorkflowRuns stream holds every
// Send and Recv until the context it was opened with ends, the way a gRPC send blocked on
// flow control is held.
type blockingDispatcher struct {
	dispatchercontracts.DispatcherClient
}

func (blockingDispatcher) SubscribeToWorkflowRuns(ctx context.Context, _ ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
	return &blockingRunsClient{ctx: ctx}, nil
}

type blockingRunsClient struct {
	grpc.ClientStream
	ctx context.Context
}

func (c *blockingRunsClient) Send(*dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
	<-c.ctx.Done()
	return c.ctx.Err()
}

func (c *blockingRunsClient) Recv() (*dispatchercontracts.WorkflowRunEvent, error) {
	<-c.ctx.Done()
	return nil, c.ctx.Err()
}

// The context OpenRunStream receives releases a Send blocked on the transport: the caller
// cancelling it ends the stream, so the caller is never left waiting on flow control for a
// stream it has given up on.
func TestOpenRunStreamCallerContextReleasesBlockedSend(t *testing.T) {
	loopCtx, loopCancel := context.WithCancel(context.Background())
	defer loopCancel()

	s := &session{
		streams: runStreamClients{dispatcher: blockingDispatcher{}},
		md:      newCallMetadata("token", nil),
		loopCtx: loopCtx,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := s.OpenRunStream(ctx, operator.RunStreamWorkflowRuns, nil)
	require.NoError(t, err)

	rs, ok := stream.(*grpcRunStream)
	require.True(t, ok)

	sent := make(chan error, 1)

	go func() {
		sent <- rs.Send(context.Background(), &dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "run-1"})
	}()

	// Wait for the send to reach the transport.
	require.Eventually(t, func() bool {
		if !rs.sendMu.TryLock() {
			return true
		}

		rs.sendMu.Unlock()

		return false
	}, time.Second, time.Millisecond, "the send never reached the transport")

	cancel()

	select {
	case err := <-sent:
		assert.ErrorIs(t, err, context.Canceled, "the blocked send was released by the caller's context")
	case <-time.After(time.Second):
		t.Fatal("the caller's context did not release the send blocked on the transport")
	}

	closed := make(chan struct{})

	go func() {
		defer close(closed)
		_ = rs.Close()
	}()

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after the stream was released")
	}
}
