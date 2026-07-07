// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"io"
	"sync/atomic"
	"testing"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

func disableStreamBackoff[C any](t *testing.T, s *reconnectingStream[C]) {
	t.Helper()
	s.sleep = func(context.Context, int) error { return nil }
}

func newTestWorkflowStream(
	t *testing.T,
	client dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient,
	constructor func(context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error),
) *reconnectingStream[dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient] {
	t.Helper()

	logger := zerolog.Nop()
	stream := newReconnectingStream(
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
		stream.setInitialClient(client)
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
) *reconnectingStream[*testListenClient] {
	t.Helper()

	logger := zerolog.Nop()
	stream := newReconnectingStream(
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
		stream.setInitialClient(initial)
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

type mockDispatcherClient struct {
	subscribeToWorkflowRunsFn   func(ctx context.Context, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error)
	subscribeToWorkflowEventsFn func(ctx context.Context, in *dispatchercontracts.SubscribeToWorkflowEventsRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient, error)
}

func (m *mockDispatcherClient) Register(ctx context.Context, in *dispatchercontracts.WorkerRegisterRequest, opts ...grpc.CallOption) (*dispatchercontracts.WorkerRegisterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) Listen(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenClient, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) ListenV2(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) Heartbeat(ctx context.Context, in *dispatchercontracts.HeartbeatRequest, opts ...grpc.CallOption) (*dispatchercontracts.HeartbeatResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) SubscribeToWorkflowEvents(ctx context.Context, in *dispatchercontracts.SubscribeToWorkflowEventsRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient, error) {
	if m.subscribeToWorkflowEventsFn != nil {
		return m.subscribeToWorkflowEventsFn(ctx, in, opts...)
	}

	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) SubscribeToWorkflowRuns(ctx context.Context, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
	if m.subscribeToWorkflowRunsFn != nil {
		return m.subscribeToWorkflowRunsFn(ctx, opts...)
	}

	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) SendStepActionEvent(ctx context.Context, in *dispatchercontracts.StepActionEvent, opts ...grpc.CallOption) (*dispatchercontracts.ActionEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) SendGroupKeyActionEvent(ctx context.Context, in *dispatchercontracts.GroupKeyActionEvent, opts ...grpc.CallOption) (*dispatchercontracts.ActionEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) PutOverridesData(ctx context.Context, in *dispatchercontracts.OverridesData, opts ...grpc.CallOption) (*dispatchercontracts.OverridesDataResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) Unsubscribe(ctx context.Context, in *dispatchercontracts.WorkerUnsubscribeRequest, opts ...grpc.CallOption) (*dispatchercontracts.WorkerUnsubscribeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) RefreshTimeout(ctx context.Context, in *dispatchercontracts.RefreshTimeoutRequest, opts ...grpc.CallOption) (*dispatchercontracts.RefreshTimeoutResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) ReleaseSlot(ctx context.Context, in *dispatchercontracts.ReleaseSlotRequest, opts ...grpc.CallOption) (*dispatchercontracts.ReleaseSlotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) RestoreEvictedTask(ctx context.Context, in *dispatchercontracts.RestoreEvictedTaskRequest, opts ...grpc.CallOption) (*dispatchercontracts.RestoreEvictedTaskResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) UpsertWorkerLabels(ctx context.Context, in *dispatchercontracts.UpsertWorkerLabelsRequest, opts ...grpc.CallOption) (*dispatchercontracts.UpsertWorkerLabelsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) GetVersion(ctx context.Context, in *dispatchercontracts.GetVersionRequest, opts ...grpc.CallOption) (*dispatchercontracts.GetVersionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
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

type mockWorkflowEventsClient struct {
	recvFn      func() (*dispatchercontracts.WorkflowEvent, error)
	recvCh      chan *dispatchercontracts.WorkflowEvent
	establishFn func(ctx context.Context, opts ...grpc.CallOption) error
}

func (m *mockWorkflowEventsClient) Recv() (*dispatchercontracts.WorkflowEvent, error) {
	if m.recvFn != nil {
		return m.recvFn()
	}
	event, ok := <-m.recvCh
	if !ok {
		return nil, io.EOF
	}
	return event, nil
}

func (m *mockWorkflowEventsClient) CloseSend() error              { return nil }
func (m *mockWorkflowEventsClient) Header() (metadata.MD, error)  { return nil, nil }
func (m *mockWorkflowEventsClient) Trailer() metadata.MD          { return nil }
func (m *mockWorkflowEventsClient) Context() context.Context      { return context.Background() }
func (m *mockWorkflowEventsClient) SendMsg(msg interface{}) error { return nil }
func (m *mockWorkflowEventsClient) RecvMsg(msg interface{}) error { return nil }
func newTestWorkflowRunsListener(
	t *testing.T,
	logger *zerolog.Logger,
	constructor func(context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error),
	client dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient,
) *WorkflowRunsListener {
	t.Helper()

	l := newWorkflowRunsListener(logger, constructor)
	if client != nil {
		l.stream.setInitialClient(client)
	}
	disableStreamBackoff(t, l.stream)
	return l
}

type mockDurableEventClient struct {
	sendFn      func(req *contracts.ListenForDurableEventRequest) error
	recvFn      func() (*contracts.DurableEvent, error)
	closeSendFn func() error
	recvCh      chan *contracts.DurableEvent
	sendCount   atomic.Int32
}

type mockV1DispatcherClient struct {
	listenForDurableEventFn func(ctx context.Context, opts ...grpc.CallOption) (contracts.V1Dispatcher_ListenForDurableEventClient, error)
}

func (m *mockV1DispatcherClient) DurableTask(ctx context.Context, opts ...grpc.CallOption) (contracts.V1Dispatcher_DurableTaskClient, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockV1DispatcherClient) RegisterDurableEvent(ctx context.Context, in *contracts.RegisterDurableEventRequest, opts ...grpc.CallOption) (*contracts.RegisterDurableEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockV1DispatcherClient) ListenForDurableEvent(ctx context.Context, opts ...grpc.CallOption) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
	if m.listenForDurableEventFn != nil {
		return m.listenForDurableEventFn(ctx, opts...)
	}

	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDurableEventClient) Send(req *contracts.ListenForDurableEventRequest) error {
	m.sendCount.Add(1)
	if m.sendFn != nil {
		return m.sendFn(req)
	}
	return nil
}

func (m *mockDurableEventClient) Recv() (*contracts.DurableEvent, error) {
	if m.recvFn != nil {
		return m.recvFn()
	}
	if m.recvCh == nil {
		return nil, io.EOF
	}
	event, ok := <-m.recvCh
	if !ok {
		return nil, io.EOF
	}
	return event, nil
}

func (m *mockDurableEventClient) CloseSend() error {
	if m.closeSendFn != nil {
		return m.closeSendFn()
	}
	return nil
}

func (m *mockDurableEventClient) Header() (metadata.MD, error) {
	return nil, nil
}

func (m *mockDurableEventClient) Trailer() metadata.MD {
	return nil
}

func (m *mockDurableEventClient) Context() context.Context {
	return context.Background()
}

func (m *mockDurableEventClient) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockDurableEventClient) RecvMsg(msg interface{}) error {
	return nil
}

func newTestDurableEventsListener(
	t *testing.T,
	logger *zerolog.Logger,
	constructor func(context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error),
	client contracts.V1Dispatcher_ListenForDurableEventClient,
) *DurableEventsListener {
	t.Helper()

	l := newDurableEventsListener(logger, constructor)
	if client != nil {
		l.stream.setInitialClient(client)
	}
	disableStreamBackoff(t, l.stream)
	return l
}

type mockListenV2Client struct {
	recvFn func() (*dispatchercontracts.AssignedAction, error)
}

func (m *mockListenV2Client) Recv() (*dispatchercontracts.AssignedAction, error) {
	if m.recvFn != nil {
		return m.recvFn()
	}
	return nil, io.EOF
}

func (m *mockListenV2Client) CloseSend() error              { return nil }
func (m *mockListenV2Client) Header() (metadata.MD, error)  { return nil, nil }
func (m *mockListenV2Client) Trailer() metadata.MD          { return nil }
func (m *mockListenV2Client) Context() context.Context      { return context.Background() }
func (m *mockListenV2Client) SendMsg(msg interface{}) error { return nil }
func (m *mockListenV2Client) RecvMsg(msg interface{}) error { return nil }

type mockWorkerDispatcherClient struct {
	listenV2Fn func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error)
	listenFn   func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenClient, error)
}

func (m *mockWorkerDispatcherClient) Register(ctx context.Context, in *dispatchercontracts.WorkerRegisterRequest, opts ...grpc.CallOption) (*dispatchercontracts.WorkerRegisterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) Listen(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenClient, error) {
	if m.listenFn != nil {
		return m.listenFn(ctx, in, opts...)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) ListenV2(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error) {
	if m.listenV2Fn != nil {
		return m.listenV2Fn(ctx, in, opts...)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) Heartbeat(ctx context.Context, in *dispatchercontracts.HeartbeatRequest, opts ...grpc.CallOption) (*dispatchercontracts.HeartbeatResponse, error) {
	return &dispatchercontracts.HeartbeatResponse{}, nil
}

func (m *mockWorkerDispatcherClient) SubscribeToWorkflowEvents(ctx context.Context, in *dispatchercontracts.SubscribeToWorkflowEventsRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) SubscribeToWorkflowRuns(ctx context.Context, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) SendStepActionEvent(ctx context.Context, in *dispatchercontracts.StepActionEvent, opts ...grpc.CallOption) (*dispatchercontracts.ActionEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) SendGroupKeyActionEvent(ctx context.Context, in *dispatchercontracts.GroupKeyActionEvent, opts ...grpc.CallOption) (*dispatchercontracts.ActionEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) PutOverridesData(ctx context.Context, in *dispatchercontracts.OverridesData, opts ...grpc.CallOption) (*dispatchercontracts.OverridesDataResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) Unsubscribe(ctx context.Context, in *dispatchercontracts.WorkerUnsubscribeRequest, opts ...grpc.CallOption) (*dispatchercontracts.WorkerUnsubscribeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) RefreshTimeout(ctx context.Context, in *dispatchercontracts.RefreshTimeoutRequest, opts ...grpc.CallOption) (*dispatchercontracts.RefreshTimeoutResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) ReleaseSlot(ctx context.Context, in *dispatchercontracts.ReleaseSlotRequest, opts ...grpc.CallOption) (*dispatchercontracts.ReleaseSlotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) RestoreEvictedTask(ctx context.Context, in *dispatchercontracts.RestoreEvictedTaskRequest, opts ...grpc.CallOption) (*dispatchercontracts.RestoreEvictedTaskResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) UpsertWorkerLabels(ctx context.Context, in *dispatchercontracts.UpsertWorkerLabelsRequest, opts ...grpc.CallOption) (*dispatchercontracts.UpsertWorkerLabelsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) GetVersion(ctx context.Context, in *dispatchercontracts.GetVersionRequest, opts ...grpc.CallOption) (*dispatchercontracts.GetVersionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func newTestActionListener(t *testing.T, listenClient dispatchercontracts.Dispatcher_ListenV2Client) *actionListenerImpl {
	t.Helper()

	logger := zerolog.Nop()
	listener := &actionListenerImpl{
		client:           &mockWorkerDispatcherClient{},
		listenClient:     listenClient,
		workerId:         "worker-1",
		l:                &logger,
		listenerStrategy: ListenerStrategyV2,
		ctx:              newContextLoader("", nil),
	}
	disableStreamBackoff(t, listener.actionStreamCore())
	return listener
}
