package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"io"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// runStreamQueueSize bounds the responses a channel-backed run stream holds for a caller that
// reads slower than the engine sends; the handler's send blocks past it, the way a gRPC
// stream's flow control blocks it.
const runStreamQueueSize = 64

// RegisterRunStream is the in-engine equivalent of one of the run observation streams
// (operator.RunStreamKind), for operators hosted in this process: the same handler runs as
// for the gRPC stream, over a channel pair instead of a transport. The caller writes the
// stream's requests to the returned request channel (nil for a server stream, whose one
// request is first) and reads the responses from the response channel, which closes once the
// handler returned. The handler ends when ctx is cancelled, the request channel is closed, or
// the stream finishes on its own; ctx must carry the tenant, as the auth middleware puts it.
func (d *DispatcherImpl) RegisterRunStream(ctx context.Context, kind operator.RunStreamKind, first proto.Message) (chan<- proto.Message, <-chan proto.Message, error) {
	if _, ok := ctx.Value("tenant").(*sqlcv1.Tenant); !ok {
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, errors.New("tenant not found on context"))
	}

	switch kind {
	case operator.RunStreamWorkflowRuns:
		return registerBidiRunStream(ctx, first, func(ctx context.Context, receive func() (*contracts.SubscribeToWorkflowRunsRequest, error), sender *rpcstream.Sender[contracts.WorkflowRunEvent]) error {
			return d.subscribeToWorkflowRunsV1(ctx, receive, sender)
		})
	case operator.RunStreamDurableEvents:
		return registerBidiRunStream(ctx, first, func(ctx context.Context, receive func() (*v1.ListenForDurableEventRequest, error), sender *rpcstream.Sender[v1.DurableEvent]) error {
			return d.serviceV1.listenForDurableEvent(ctx, receive, sender)
		})
	case operator.RunStreamWorkflowEvents:
		req, ok := first.(*contracts.SubscribeToWorkflowEventsRequest)

		if !ok {
			return nil, nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s takes a SubscribeToWorkflowEventsRequest, not a %T", kind, first))
		}

		respCh := registerServerRunStream(ctx, func(ctx context.Context, sender *rpcstream.Sender[contracts.WorkflowEvent]) error {
			return d.subscribeToWorkflowEventsV1(ctx, req, sender)
		})

		return nil, respCh, nil
	default:
		return nil, nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("run stream kind %d is not served in process", int(kind)))
	}
}

// chanStream is the sending half of a channel-backed stream: Send hands the message to the
// caller's response channel, or fails once the stream's context ended so a handler can always
// return.
type chanStream[T any] struct {
	ctx context.Context
	out chan<- proto.Message
}

func (c chanStream[T]) Send(msg *T) error {
	m, ok := any(msg).(proto.Message)

	if !ok {
		return fmt.Errorf("stream message %T is not a proto message", msg)
	}

	select {
	case c.out <- m:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

// registerBidiRunStream runs handler over a request channel and a response channel. first,
// when set, is delivered as the stream's first request before anything the caller sends.
func registerBidiRunStream[Req any, Resp any](ctx context.Context, first proto.Message, handler func(context.Context, func() (*Req, error), *rpcstream.Sender[Resp]) error) (chan<- proto.Message, <-chan proto.Message, error) {
	if first != nil {
		if _, ok := any(first).(*Req); !ok {
			return nil, nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("stream takes a %T, not a %T", (*Req)(nil), first))
		}
	}

	ctx, cancel := context.WithCancel(ctx)

	reqCh := make(chan proto.Message)
	respCh := make(chan proto.Message, runStreamQueueSize)

	receive := func() (*Req, error) {
		if first != nil {
			req := any(first).(*Req)
			first = nil

			return req, nil
		}

		select {
		case msg, ok := <-reqCh:
			if !ok {
				return nil, io.EOF
			}

			req, ok := any(msg).(*Req)

			if !ok {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("stream takes a %T, not a %T", (*Req)(nil), msg))
			}

			return req, nil
		case <-ctx.Done():
			return nil, io.EOF
		}
	}

	sender := rpcstream.NewSender[Resp](ctx, chanStream[Resp]{ctx: ctx, out: respCh})

	go func() {
		defer close(respCh)
		// Cancel before Close so a Send blocked on a full response channel returns and Close
		// does not wait its grace out.
		defer sender.Close()
		defer cancel()

		_ = handler(ctx, receive, sender)
	}()

	return reqCh, respCh, nil
}

// registerServerRunStream runs handler over a response channel only.
func registerServerRunStream[Resp any](ctx context.Context, handler func(context.Context, *rpcstream.Sender[Resp]) error) <-chan proto.Message {
	ctx, cancel := context.WithCancel(ctx)

	respCh := make(chan proto.Message, runStreamQueueSize)
	sender := rpcstream.NewSender[Resp](ctx, chanStream[Resp]{ctx: ctx, out: respCh})

	go func() {
		defer close(respCh)
		defer sender.Close()
		defer cancel()

		_ = handler(ctx, sender)
	}()

	return respCh
}
