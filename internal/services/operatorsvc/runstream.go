package operatorsvc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// runStreamCloseDrainTimeout bounds how long Close waits for the engine to close the response
// side after the stream is cancelled.
const runStreamCloseDrainTimeout = 5 * time.Second

// OpenRunStream opens one run observation stream on the dispatcher's channel-backed entry,
// with the session's tenant on the context the way the auth middleware puts it for a gRPC
// caller. The stream is detached from ctx once open and ends on Close, or when the engine ends
// it, which Recv reports as ErrStreamEnded.
func (ss *Session) OpenRunStream(ctx context.Context, kind operator.RunStreamKind, first proto.Message) (operator.RunStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if kind == operator.RunStreamWorkflowEvents && first == nil {
		return nil, fmt.Errorf("%s requires its request as the opening message", kind)
	}

	sctx, cancel := context.WithCancel(WithTenant(context.WithoutCancel(ctx), ss.tenant))

	reqCh, respCh, err := ss.svc.dispatcher.RegisterRunStream(sctx, kind, first)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not open %s: %w", kind, err)
	}

	return &runStream{
		reqCh:  reqCh,
		respCh: respCh,
		ctx:    sctx,
		cancel: cancel,
		closed: make(chan struct{}),
	}, nil
}

// runStream is one channel-backed engine stream. The engine's handler delivers responses on
// its own goroutine, blocking on a full channel, so Recv reads straight from it; Send hands the
// request to the handler's receive loop and waits for it to be taken.
type runStream struct {
	reqCh  chan<- proto.Message
	respCh <-chan proto.Message
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
	if rs.reqCh == nil {
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
	case rs.reqCh <- msg:
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
	case msg, ok := <-rs.respCh:
		if !ok {
			rs.sendMu.Lock()
			rs.ended = true
			rs.sendMu.Unlock()

			if rs.isClosed() {
				return nil, ErrChannelClosed
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
			case _, ok := <-rs.respCh:
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
