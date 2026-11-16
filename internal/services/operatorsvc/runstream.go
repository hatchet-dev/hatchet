package operatorsvc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// runStreamCloseDrainTimeout bounds how long Close waits for the engine to close the response
// side after the stream is cancelled.
const runStreamCloseDrainTimeout = 5 * time.Second

// OpenRunStream opens one run observation stream on the engine's channel-backed entry for the
// kind (the dispatcher's for the run streams, the streams service's for V1Streams.Subscribe),
// with the session's tenant on the context the way the auth middleware puts it for a gRPC
// caller, so every check the RPC makes runs here too. The stream is detached from ctx once
// open and ends on Close, or when the engine ends it, which Recv reports as ErrStreamEnded
// when the handler finished and as the handler's own error when it refused or failed the
// stream.
func (ss *Session) OpenRunStream(ctx context.Context, kind operator.RunStreamKind, first proto.Message) (operator.RunStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if !kind.Bidi() && first == nil {
		return nil, fmt.Errorf("%s requires its request as the opening message", kind)
	}

	sctx, cancel := context.WithCancel(WithTenant(context.WithoutCancel(ctx), ss.tenant))

	cs, err := ss.register(sctx, kind, first)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not open %s: %w", kind, err)
	}

	return &runStream{
		cs:     cs,
		ctx:    sctx,
		cancel: cancel,
		closed: make(chan struct{}),
	}, nil
}

// register serves the kind over channels on the service that owns it.
func (ss *Session) register(ctx context.Context, kind operator.RunStreamKind, first proto.Message) (*rpcstream.ChannelStream, error) {
	if kind != operator.RunStreamDurableStreams {
		return ss.svc.dispatcher.RegisterRunStream(ctx, kind, first)
	}

	req, ok := first.(*v1contracts.SubscribeStreamRequest)

	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s takes a SubscribeStreamRequest, not a %T", kind, first))
	}

	if ss.svc.streams == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("%s is not served in process here", kind))
	}

	return ss.svc.streams.RegisterSubscribe(ctx, req)
}

// runStream is one channel-backed engine stream. The engine's handler delivers responses on
// its own goroutine, blocking on a full channel, so Recv reads straight from it; Send hands the
// request to the handler's receive loop and waits for it to be taken.
type runStream struct {
	cs     *rpcstream.ChannelStream
	ctx    context.Context
	cancel context.CancelFunc
	closed chan struct{}

	closeOnce sync.Once
	sendMu    sync.Mutex
	// ended records that the engine closed its side; guarded by sendMu.
	ended bool
}

func (rs *runStream) isClosed() bool {
	select {
	case <-rs.closed:
		return true
	default:
		return false
	}
}

func (rs *runStream) Send(ctx context.Context, msg proto.Message) error {
	if rs.cs.Requests == nil {
		return fmt.Errorf("sending on a server stream: %w", operator.ErrNotSupported)
	}

	if rs.isClosed() {
		return ErrChannelClosed
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// the handler stops receiving once it ended; a send would block on it for good
	rs.sendMu.Lock()
	defer rs.sendMu.Unlock()

	if rs.ended {
		return ErrSessionEnded
	}

	select {
	case rs.cs.Requests <- msg:
		return nil
	case <-rs.ctx.Done():
		return ErrChannelClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (rs *runStream) Recv(ctx context.Context) (proto.Message, error) {
	if rs.isClosed() {
		return nil, ErrChannelClosed
	}

	select {
	case msg, ok := <-rs.cs.Responses:
		if !ok {
			rs.sendMu.Lock()
			rs.ended = true
			rs.sendMu.Unlock()

			if rs.isClosed() {
				return nil, ErrChannelClosed
			}

			if err := rs.cs.Err; err != nil {
				return nil, err
			}

			return nil, operator.ErrStreamEnded
		}

		return msg, nil
	case <-rs.closed:
		return nil, ErrChannelClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close cancels the stream, unblocks Send and Recv, and waits for the engine to close its
// side, bounded, so the handler has left before Close returns.
func (rs *runStream) Close() error {
	rs.closeOnce.Do(func() {
		close(rs.closed)
		rs.cancel()

		timeout := time.NewTimer(runStreamCloseDrainTimeout)
		defer timeout.Stop()

		for {
			select {
			case _, ok := <-rs.cs.Responses:
				if !ok {
					return
				}
			case <-timeout.C:
				return
			}
		}
	})

	return nil
}
