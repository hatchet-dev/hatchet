package streaming

import (
	"context"
	"io"
	"sync/atomic"
	"testing"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/metadata"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

func disableStreamBackoff[C any](t *testing.T, s *ReconnectingStream[C]) {
	t.Helper()
	s.SetSleep(func(context.Context, int) error { return nil })
}

func newTestWorkflowStream(
	t *testing.T,
	client dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient,
	constructor func(context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error),
) *ReconnectingStream[dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient] {
	t.Helper()

	logger := zerolog.Nop()
	stream := NewReconnectingStream(
		&logger,
		"workflow run listener",
		constructor,
		func(c dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
			return c.CloseSend()
		},
		nil,
	)
	disableStreamBackoff(t, stream)
	if client != nil {
		stream.SetInitialClient(client)
	}
	return stream
}

type testListenEvent struct {
	value string
}

type testListenClient struct {
	recvFn      func() (testListenEvent, error)
	closeSendFn func() error
	closeCalled atomic.Bool
}

func (c *testListenClient) Recv() (testListenEvent, error) {
	if c.recvFn != nil {
		return c.recvFn()
	}
	return testListenEvent{}, io.EOF
}

func (c *testListenClient) CloseSend() error {
	c.closeCalled.Store(true)
	if c.closeSendFn != nil {
		return c.closeSendFn()
	}
	return nil
}

func newTestListenStream(
	t *testing.T,
	initial *testListenClient,
	constructor func(context.Context) (*testListenClient, error),
) *ReconnectingStream[*testListenClient] {
	t.Helper()

	logger := zerolog.Nop()
	stream := NewReconnectingStream(
		&logger,
		"test listener",
		constructor,
		func(client *testListenClient) error {
			return client.CloseSend()
		},
		nil,
	)
	disableStreamBackoff(t, stream)
	if initial != nil {
		stream.SetInitialClient(initial)
	}
	return stream
}

// mockSubscribeClient implements dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient
type mockSubscribeClient struct {
	sendErr     error
	sendCount   atomic.Int32
	recvErr     error
	recvChan    chan *dispatchercontracts.WorkflowRunEvent
	closeSendFn func() error
	sendFn      func(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error
	recvFn      func() (*dispatchercontracts.WorkflowRunEvent, error)
}

func (m *mockSubscribeClient) Send(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
	m.sendCount.Add(1)
	if m.sendFn != nil {
		return m.sendFn(req)
	}
	return m.sendErr
}

func (m *mockSubscribeClient) Recv() (*dispatchercontracts.WorkflowRunEvent, error) {
	if m.recvFn != nil {
		return m.recvFn()
	}
	if m.recvErr != nil {
		return nil, m.recvErr
	}
	event, ok := <-m.recvChan
	if !ok {
		return nil, io.EOF
	}
	return event, nil
}

func (m *mockSubscribeClient) CloseSend() error {
	if m.closeSendFn != nil {
		return m.closeSendFn()
	}
	return nil
}

func (m *mockSubscribeClient) Header() (metadata.MD, error) {
	return nil, nil
}

func (m *mockSubscribeClient) Trailer() metadata.MD {
	return nil
}

func (m *mockSubscribeClient) Context() context.Context {
	return context.Background()
}

func (m *mockSubscribeClient) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockSubscribeClient) RecvMsg(msg interface{}) error {
	return nil
}
