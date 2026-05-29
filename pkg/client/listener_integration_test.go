// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

const listenerIntegrationBufSize = 1024 * 1024

type scriptedSubscribeServer struct {
	dispatchercontracts.UnimplementedDispatcherServer
	v1contracts.UnimplementedV1DispatcherServer
	workflowEvents       func(*dispatchercontracts.SubscribeToWorkflowEventsRequest, dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsServer) error
	workflowRunsHandlers []func(dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsServer) error
	durableEventHandlers []func(v1contracts.V1Dispatcher_ListenForDurableEventServer) error
	mu                   sync.Mutex
}

func (s *scriptedSubscribeServer) SubscribeToWorkflowRuns(stream dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsServer) error {
	s.mu.Lock()
	if len(s.workflowRunsHandlers) == 0 {
		s.mu.Unlock()
		return status.Error(codes.Unimplemented, "workflow run handler not configured")
	}

	handler := s.workflowRunsHandlers[0]
	if len(s.workflowRunsHandlers) > 1 {
		s.workflowRunsHandlers = s.workflowRunsHandlers[1:]
	}
	s.mu.Unlock()

	return handler(stream)
}

func (s *scriptedSubscribeServer) ListenForDurableEvent(stream v1contracts.V1Dispatcher_ListenForDurableEventServer) error {
	s.mu.Lock()
	if len(s.durableEventHandlers) == 0 {
		s.mu.Unlock()
		return status.Error(codes.Unimplemented, "durable event handler not configured")
	}

	handler := s.durableEventHandlers[0]
	if len(s.durableEventHandlers) > 1 {
		s.durableEventHandlers = s.durableEventHandlers[1:]
	}
	s.mu.Unlock()

	return handler(stream)
}

func (s *scriptedSubscribeServer) SubscribeToWorkflowEvents(
	req *dispatchercontracts.SubscribeToWorkflowEventsRequest,
	stream dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsServer,
) error {
	s.mu.Lock()
	handler := s.workflowEvents
	s.mu.Unlock()

	if handler == nil {
		return status.Error(codes.Unimplemented, "workflow events handler not configured")
	}

	return handler(req, stream)
}

func newScriptedSubscribeClient(t *testing.T, server *scriptedSubscribeServer) SubscribeClient {
	t.Helper()

	listener := bufconn.Listen(listenerIntegrationBufSize)
	grpcServer := grpc.NewServer()

	dispatchercontracts.RegisterDispatcherServer(grpcServer, server)
	v1contracts.RegisterV1DispatcherServer(grpcServer, server)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(listener)
	}()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, conn.Close())
		grpcServer.Stop()
		require.NoError(t, listener.Close())

		select {
		case <-serveErr:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for scripted grpc server to stop")
		}
	})

	l := zerolog.Nop()

	return newSubscribe(conn, &sharedClientOpts{
		l:         &l,
		v:         validator.NewDefaultValidator(),
		ctxLoader: newContextLoader("", nil),
	})
}

type workflowRunStreamScript struct {
	requests chan *dispatchercontracts.SubscribeToWorkflowRunsRequest
	events   chan *dispatchercontracts.WorkflowRunEvent
}

func newWorkflowRunStreamScript() *workflowRunStreamScript {
	return &workflowRunStreamScript{
		requests: make(chan *dispatchercontracts.SubscribeToWorkflowRunsRequest, 10),
		events:   make(chan *dispatchercontracts.WorkflowRunEvent, 10),
	}
}

func (s *workflowRunStreamScript) handle(stream dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsServer) error {
	recvErr := make(chan error, 1)

	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				recvErr <- err
				return
			}

			s.requests <- req
		}
	}()

	for {
		select {
		case event, ok := <-s.events:
			if !ok {
				return nil
			}

			if err := stream.Send(event); err != nil {
				return err
			}
		case err := <-recvErr:
			if err == io.EOF {
				return nil
			}

			return err
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func waitForTestValue[T any](t *testing.T, ch <-chan T) T {
	t.Helper()

	select {
	case value := <-ch:
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for value")
	}

	var zero T
	return zero
}

func assertNoTestValue[T any](t *testing.T, ch <-chan T) {
	t.Helper()

	select {
	case value := <-ch:
		t.Fatalf("received unexpected value: %#v", value)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestWorkflowRunsListenerIntegrationDispatchesMultipleHandlersAndRemove(t *testing.T) {
	stream := newWorkflowRunStreamScript()
	subscriber := newScriptedSubscribeClient(t, &scriptedSubscribeServer{
		workflowRunsHandlers: []func(dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsServer) error{
			stream.handle,
		},
	})

	listener, err := subscriber.SubscribeToWorkflowRunEvents(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, listener.Close())
	})

	first := make(chan WorkflowRunEvent, 2)
	second := make(chan WorkflowRunEvent, 2)

	require.NoError(t, listener.AddWorkflowRun("run-1", "session-1", func(event WorkflowRunEvent) error {
		first <- event
		return nil
	}))
	require.Equal(t, "run-1", waitForTestValue(t, stream.requests).WorkflowRunId)

	require.NoError(t, listener.AddWorkflowRun("run-1", "session-2", func(event WorkflowRunEvent) error {
		second <- event
		return nil
	}))
	require.Equal(t, "run-1", waitForTestValue(t, stream.requests).WorkflowRunId)

	stream.events <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-1",
		EventType:     dispatchercontracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
	}

	require.Equal(t, "run-1", waitForTestValue(t, first).WorkflowRunId)
	require.Equal(t, "run-1", waitForTestValue(t, second).WorkflowRunId)

	listener.RemoveWorkflowRun("run-1", "session-1")

	stream.events <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-1",
		EventType:     dispatchercontracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
	}

	require.Equal(t, "run-1", waitForTestValue(t, second).WorkflowRunId)
	assertNoTestValue(t, first)
}

func TestWorkflowRunsListenerIntegrationReconnectsAfterRecvDrop(t *testing.T) {
	firstRequest := make(chan *dispatchercontracts.SubscribeToWorkflowRunsRequest, 1)
	secondRequest := make(chan *dispatchercontracts.SubscribeToWorkflowRunsRequest, 1)

	subscriber := newScriptedSubscribeClient(t, &scriptedSubscribeServer{
		workflowRunsHandlers: []func(dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsServer) error{
			func(stream dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsServer) error {
				req, err := stream.Recv()
				if err != nil {
					return err
				}

				firstRequest <- req
				return status.Error(codes.Internal, "scripted stream drop")
			},
			func(stream dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsServer) error {
				req, err := stream.Recv()
				if err != nil {
					return err
				}

				secondRequest <- req
				return stream.Send(&dispatchercontracts.WorkflowRunEvent{
					WorkflowRunId: req.WorkflowRunId,
					EventType:     dispatchercontracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
				})
			},
		},
	})

	listener, err := subscriber.SubscribeToWorkflowRunEvents(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, listener.Close())
	})

	received := make(chan WorkflowRunEvent, 1)
	require.NoError(t, listener.AddWorkflowRun("run-reconnect", "session-1", func(event WorkflowRunEvent) error {
		received <- event
		return nil
	}))

	require.Equal(t, "run-reconnect", waitForTestValue(t, firstRequest).WorkflowRunId)
	require.Equal(t, "run-reconnect", waitForTestValue(t, secondRequest).WorkflowRunId)
	require.Equal(t, "run-reconnect", waitForTestValue(t, received).WorkflowRunId)
}

func TestWorkflowRunsListenerIntegrationStreamExitAllowsNewListener(t *testing.T) {
	firstStarted := make(chan struct{}, 1)
	secondStream := newWorkflowRunStreamScript()

	subscriber := newScriptedSubscribeClient(t, &scriptedSubscribeServer{
		workflowRunsHandlers: []func(dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsServer) error{
			func(stream dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsServer) error {
				firstStarted <- struct{}{}
				return nil
			},
			secondStream.handle,
		},
	})

	firstListener, err := subscriber.SubscribeToWorkflowRunEvents(context.Background())
	require.NoError(t, err)
	_ = firstListener
	waitForTestValue(t, firstStarted)

	var secondListener *WorkflowRunsListener
	require.Eventually(t, func() bool {
		var err error
		secondListener, err = subscriber.SubscribeToWorkflowRunEvents(context.Background())
		return err == nil && secondListener != firstListener
	}, time.Second, 10*time.Millisecond)
	t.Cleanup(func() {
		require.NoError(t, secondListener.Close())
	})

	received := make(chan WorkflowRunEvent, 1)
	require.NoError(t, secondListener.AddWorkflowRun("run-after-eof", "session-1", func(event WorkflowRunEvent) error {
		received <- event
		return nil
	}))
	require.Equal(t, "run-after-eof", waitForTestValue(t, secondStream.requests).WorkflowRunId)

	secondStream.events <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-after-eof",
		EventType:     dispatchercontracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
	}

	require.Equal(t, "run-after-eof", waitForTestValue(t, received).WorkflowRunId)
}

func TestSubscribeClientIntegrationOnFiltersStreamEvents(t *testing.T) {
	requests := make(chan *dispatchercontracts.SubscribeToWorkflowEventsRequest, 1)
	subscriber := newScriptedSubscribeClient(t, &scriptedSubscribeServer{
		workflowEvents: func(req *dispatchercontracts.SubscribeToWorkflowEventsRequest, stream dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
			requests <- req

			if err := stream.Send(&dispatchercontracts.WorkflowEvent{
				WorkflowRunId: "run-on",
				EventType:     dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM,
				EventPayload:  "ignored",
			}); err != nil {
				return err
			}

			return stream.Send(&dispatchercontracts.WorkflowEvent{
				WorkflowRunId: "run-on",
				EventType:     dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED,
				EventPayload:  "handled",
			})
		},
	})

	received := make(chan WorkflowEvent, 1)
	done := make(chan error, 1)
	go func() {
		done <- subscriber.On(context.Background(), "run-on", func(event WorkflowEvent) error {
			received <- event
			return nil
		})
	}()

	req := waitForTestValue(t, requests)
	require.NotNil(t, req.WorkflowRunId)
	require.Equal(t, "run-on", *req.WorkflowRunId)
	require.Equal(t, "handled", waitForTestValue(t, received).EventPayload)
	require.NoError(t, waitForTestValue(t, done))
	assertNoTestValue(t, received)
}

func TestSubscribeClientIntegrationStreamFiltersNonStreamEvents(t *testing.T) {
	requests := make(chan *dispatchercontracts.SubscribeToWorkflowEventsRequest, 1)
	subscriber := newScriptedSubscribeClient(t, &scriptedSubscribeServer{
		workflowEvents: func(req *dispatchercontracts.SubscribeToWorkflowEventsRequest, stream dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
			requests <- req

			if err := stream.Send(&dispatchercontracts.WorkflowEvent{
				WorkflowRunId: "run-stream",
				EventType:     dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED,
				EventPayload:  "ignored",
			}); err != nil {
				return err
			}

			return stream.Send(&dispatchercontracts.WorkflowEvent{
				WorkflowRunId: "run-stream",
				EventType:     dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM,
				EventPayload:  "handled-stream",
			})
		},
	})

	received := make(chan StreamEvent, 1)
	done := make(chan error, 1)
	go func() {
		done <- subscriber.Stream(context.Background(), "run-stream", func(event StreamEvent) error {
			received <- event
			return nil
		})
	}()

	req := waitForTestValue(t, requests)
	require.NotNil(t, req.WorkflowRunId)
	require.Equal(t, "run-stream", *req.WorkflowRunId)
	require.Equal(t, []byte("handled-stream"), waitForTestValue(t, received).Message)
	require.NoError(t, waitForTestValue(t, done))
	assertNoTestValue(t, received)
}

func TestSubscribeClientIntegrationStreamByAdditionalMetadata(t *testing.T) {
	requests := make(chan *dispatchercontracts.SubscribeToWorkflowEventsRequest, 1)
	subscriber := newScriptedSubscribeClient(t, &scriptedSubscribeServer{
		workflowEvents: func(req *dispatchercontracts.SubscribeToWorkflowEventsRequest, stream dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
			requests <- req

			return stream.Send(&dispatchercontracts.WorkflowEvent{
				EventType:    dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM,
				EventPayload: "metadata-stream",
			})
		},
	})

	received := make(chan StreamEvent, 1)
	done := make(chan error, 1)
	go func() {
		done <- subscriber.StreamByAdditionalMetadata(context.Background(), "key", "value", func(event StreamEvent) error {
			received <- event
			return nil
		})
	}()

	req := waitForTestValue(t, requests)
	require.NotNil(t, req.AdditionalMetaKey)
	require.NotNil(t, req.AdditionalMetaValue)
	require.Equal(t, "key", *req.AdditionalMetaKey)
	require.Equal(t, "value", *req.AdditionalMetaValue)
	require.Equal(t, []byte("metadata-stream"), waitForTestValue(t, received).Message)
	require.NoError(t, waitForTestValue(t, done))
}
