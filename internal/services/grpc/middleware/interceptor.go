package middleware

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
)

// handlerInterceptor runs the same hooks for unary and streaming handlers. before runs ahead of
// the handler and may replace the context or reject the call; after sees the handler's error and
// may replace it.
type handlerInterceptor struct {
	before func(ctx context.Context, spec connect.Spec, header http.Header, peer connect.Peer) (context.Context, error)
	after  func(ctx context.Context, spec connect.Spec, err error) error
}

func (i *handlerInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Spec().IsClient {
			return next(ctx, req)
		}

		if i.before != nil {
			var err error

			if ctx, err = i.before(ctx, req.Spec(), req.Header(), req.Peer()); err != nil {
				return nil, err
			}
		}

		res, err := next(ctx, req)

		if i.after != nil {
			err = i.after(ctx, req.Spec(), err)
		}

		if err != nil {
			return nil, err
		}

		return res, nil
	}
}

func (i *handlerInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *handlerInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if i.before != nil {
			var err error

			if ctx, err = i.before(ctx, conn.Spec(), conn.RequestHeader(), conn.Peer()); err != nil {
				return err
			}
		}

		err := next(ctx, conn)

		if i.after != nil {
			err = i.after(ctx, conn.Spec(), err)
		}

		return err
	}
}
