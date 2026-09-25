package operatorclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/protobuf/proto"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// runStreamQueueSize bounds the messages read ahead of the caller's Recv; the reader blocks
// past it, which is the stream's flow control.
const runStreamQueueSize = 64

// runStreamClients are the dispatcher clients the run streams are opened on, built over the
// same connection as the operator client.
type runStreamClients struct {
	dispatcher   dispatchercontracts.DispatcherClient
	v1dispatcher v1.V1DispatcherClient
}

// OpenRunStream implements Session: the stream is opened on the session's connection with
// its bearer token and lives until Close, the engine ending it, or the session's loops
// ending. The dispatcher services authenticate with the token alone; the operator id
// metadata is only meaningful to OperatorService.
func (s *session) OpenRunStream(ctx context.Context, kind operator.RunStreamKind, first proto.Message) (operator.RunStream, error) {
	if s.isClosed() {
		return nil, operator.ErrSessionClosed
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	streamCtx, cancel := context.WithCancel(s.loopCtx)
	mdCtx := s.md.context(streamCtx)

	rs := &grpcRunStream{
		cancel: cancel,
		queue:  make(chan runStreamItem, runStreamQueueSize),
		closed: make(chan struct{}),
		done:   make(chan struct{}),
	}

	switch kind {
	case operator.RunStreamWorkflowRuns:
		c, err := s.streams.dispatcher.SubscribeToWorkflowRuns(mdCtx, grpc_retry.Disable())

		if err != nil {
			cancel()
			return nil, err
		}

		rs.send = func(msg proto.Message) error {
			req, ok := msg.(*dispatchercontracts.SubscribeToWorkflowRunsRequest)

			if !ok {
				return fmt.Errorf("%s takes a SubscribeToWorkflowRunsRequest, not a %T", kind, msg)
			}

			return c.Send(req)
		}
		rs.recv = func() (proto.Message, error) { return c.Recv() }
		rs.closeSend = c.CloseSend
	case operator.RunStreamWorkflowEvents:
		req, ok := first.(*dispatchercontracts.SubscribeToWorkflowEventsRequest)

		if !ok {
			cancel()
			return nil, fmt.Errorf("%s takes a SubscribeToWorkflowEventsRequest, not a %T", kind, first)
		}

		c, err := s.streams.dispatcher.SubscribeToWorkflowEvents(mdCtx, req, grpc_retry.Disable())

		if err != nil {
			cancel()
			return nil, err
		}

		first = nil
		rs.recv = func() (proto.Message, error) { return c.Recv() }
	case operator.RunStreamDurableEvents:
		c, err := s.streams.v1dispatcher.ListenForDurableEvent(mdCtx, grpc_retry.Disable())

		if err != nil {
			cancel()
			return nil, err
		}

		rs.send = func(msg proto.Message) error {
			req, ok := msg.(*v1.ListenForDurableEventRequest)

			if !ok {
				return fmt.Errorf("%s takes a ListenForDurableEventRequest, not a %T", kind, msg)
			}

			return c.Send(req)
		}
		rs.recv = func() (proto.Message, error) { return c.Recv() }
		rs.closeSend = c.CloseSend
	default:
		cancel()
		return nil, fmt.Errorf("unknown run stream kind %d", int(kind))
	}

	go rs.readLoop()

	if first != nil {
		if err := rs.send(first); err != nil {
			_ = rs.Close()
			return nil, err
		}
	}

	return rs, nil
}

func (s *session) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.closed
}

type runStreamItem struct {
	msg proto.Message
	err error
}

// grpcRunStream is one gRPC stream behind operator.RunStream: a reader goroutine turns the
// stream's Recv into a bounded queue so the caller's Recv honours its context, and Close
// cancels the stream context, which is what unblocks a pending Recv or Send on the transport.
type grpcRunStream struct {
	send      func(proto.Message) error
	recv      func() (proto.Message, error)
	closeSend func() error
	cancel    context.CancelFunc
	queue     chan runStreamItem
	closed    chan struct{}
	done      chan struct{}
	sendMu    sync.Mutex
	closeOnce sync.Once
}

func (rs *grpcRunStream) readLoop() {
	defer close(rs.done)

	for {
		msg, err := rs.recv()

		if err != nil {
			if errors.Is(err, io.EOF) {
				err = operator.ErrStreamEnded
			}

			select {
			case rs.queue <- runStreamItem{err: err}:
			case <-rs.closed:
			}

			return
		}

		select {
		case rs.queue <- runStreamItem{msg: msg}:
		case <-rs.closed:
			return
		}
	}
}

func (rs *grpcRunStream) Send(_ context.Context, msg proto.Message) error {
	if rs.send == nil {
		return fmt.Errorf("sending on a server stream: %w", operator.ErrNotSupported)
	}

	select {
	case <-rs.closed:
		return operator.ErrChannelClosed
	default:
	}

	// grpc-go allows one sender at a time on a stream.
	rs.sendMu.Lock()
	defer rs.sendMu.Unlock()

	return rs.send(msg)
}

func (rs *grpcRunStream) Recv(ctx context.Context) (proto.Message, error) {
	select {
	case item := <-rs.queue:
		return item.msg, item.err
	case <-rs.closed:
		return nil, operator.ErrChannelClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close half-closes the send side, cancels the stream and waits for the reader to leave.
func (rs *grpcRunStream) Close() error {
	rs.closeOnce.Do(func() {
		close(rs.closed)

		if rs.closeSend != nil {
			rs.sendMu.Lock()
			_ = rs.closeSend()
			rs.sendMu.Unlock()
		}

		rs.cancel()
	})

	<-rs.done

	return nil
}
