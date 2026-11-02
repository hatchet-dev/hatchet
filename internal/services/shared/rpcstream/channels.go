package rpcstream

import (
	"context"
	"fmt"
	"io"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
)

// channelQueueSize bounds the responses a channel-backed stream holds for a caller that reads
// slower than the handler sends; the handler's send blocks past it, the way a transport's flow
// control blocks it.
const channelQueueSize = 64

// ChannelStream is a streaming handler served over channels in place of a transport, to a
// caller in the same process. The handler runs on its own goroutine with the context it was
// served with, which must carry what the transport's middleware would put there (the tenant,
// for the engine's handlers). It ends when that context is cancelled, when Requests is
// closed, or when it finishes on its own; Responses closes once it returned and Err then
// reports how it ended.
type ChannelStream struct {
	// Requests takes the caller's messages on a bidi stream; closing it half-closes the
	// stream, the way a client's CloseSend does. It is nil on a server stream, whose one
	// request the handler was served with.
	Requests chan<- proto.Message

	// Responses carries the handler's messages and closes once the handler returned.
	Responses <-chan proto.Message

	// Err is the handler's result: nil when it finished the stream, the error it refused or
	// failed with otherwise. It is written before Responses closes and is only meaningful
	// once it has.
	Err error
}

// chanSink is the sending half of a channel-backed stream: Send hands the message to the
// caller's response channel, or fails once the stream's context ended so a handler can always
// return.
type chanSink[T any] struct {
	ctx context.Context
	out chan<- proto.Message
}

func (c chanSink[T]) Send(msg *T) error {
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

// ServeBidi runs handler over a request channel and a response channel. first, when set, is
// delivered as the stream's first request before anything the caller sends; a first message
// that is not a Req is refused before the handler runs.
func ServeBidi[Req any, Resp any](ctx context.Context, first proto.Message, handler func(context.Context, func() (*Req, error), *Sender[Resp]) error) (*ChannelStream, error) {
	if first != nil {
		if _, ok := any(first).(*Req); !ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("stream takes a %T, not a %T", (*Req)(nil), first))
		}
	}

	ctx, cancel := context.WithCancel(ctx)

	reqCh := make(chan proto.Message)
	respCh := make(chan proto.Message, channelQueueSize)

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

	s := &ChannelStream{Requests: reqCh, Responses: respCh}
	sender := NewSender[Resp](ctx, chanSink[Resp]{ctx: ctx, out: respCh})

	go func() {
		defer close(respCh)
		// Cancel before Close so a Send blocked on a full response channel returns and Close
		// does not wait its grace out.
		defer sender.Close()
		defer cancel()

		s.Err = handler(ctx, receive, sender)
	}()

	return s, nil
}

// ServeServer runs handler over a response channel only.
func ServeServer[Resp any](ctx context.Context, handler func(context.Context, *Sender[Resp]) error) *ChannelStream {
	ctx, cancel := context.WithCancel(ctx)

	respCh := make(chan proto.Message, channelQueueSize)
	s := &ChannelStream{Responses: respCh}
	sender := NewSender[Resp](ctx, chanSink[Resp]{ctx: ctx, out: respCh})

	go func() {
		defer close(respCh)
		defer sender.Close()
		defer cancel()

		s.Err = handler(ctx, sender)
	}()

	return s
}
