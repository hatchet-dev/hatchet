package middleware

import (
	"context"
	goerrors "errors"
	"fmt"
	"runtime/debug"
	"strings"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/errors"
)

// expectedStatus maps known, non-internal errors to their canonical codes.
// These should be returned to the client without error-logging or alerting.
func expectedStatus(ctx context.Context, err error) error {
	// A caller that went away surfaces as a transport read or write error before the handler
	// sees the cancelled context; that is the caller's doing, not an engine failure.
	if ctxErr := ctx.Err(); ctxErr != nil && !goerrors.Is(err, context.Canceled) && !goerrors.Is(err, context.DeadlineExceeded) {
		err = ctxErr
	}

	if goerrors.Is(err, context.Canceled) {
		return connect.NewError(connect.CodeCanceled, goerrors.New("request was canceled"))
	}

	if goerrors.Is(err, context.DeadlineExceeded) {
		return connect.NewError(connect.CodeDeadlineExceeded, goerrors.New("request deadline exceeded"))
	}

	if goerrors.Is(err, pgx.ErrNoRows) {
		// A missing row is a client-visible miss (task already complete, webhook
		// deleted, etc.), not an engine failure.
		return connect.NewError(connect.CodeNotFound, goerrors.New("not found"))
	}

	// Extensions built against google.golang.org/grpc return status errors; keep
	// their code, message and details on the wire. The message of a wrapped status error is the
	// text of the whole chain, as status.FromError reports it.
	if st, ok := status.FromError(err); ok && st.Code() != codes.Unknown && st.Code() != codes.OK {
		statusErr := connect.NewError(connect.Code(st.Code()), goerrors.New(st.Message()))

		for _, detail := range st.Proto().GetDetails() {
			// an Any is carried over as is, so a type this server does not know survives
			if connectDetail, detailErr := connect.NewErrorDetail(detail); detailErr == nil {
				statusErr.AddDetail(connectDetail)
			}
		}

		return statusErr
	}

	return nil
}

type ErrorInterceptor struct {
	a errors.Alerter
	l *zerolog.Logger
}

func NewErrorInterceptor(a errors.Alerter,
	l *zerolog.Logger) *ErrorInterceptor {
	return &ErrorInterceptor{
		a, l,
	}
}

// Interceptor converts errors without a code into internal errors, for unary and streaming calls.
func (e *ErrorInterceptor) Interceptor() connect.Interceptor {
	return &handlerInterceptor{
		after: func(ctx context.Context, _ connect.Spec, err error) error {
			// if this is not a coded error already, convert it to an internal error
			if err != nil && connect.CodeOf(err) == connect.CodeUnknown {
				if statusErr := expectedStatus(ctx, err); statusErr != nil {
					return statusErr
				}

				e.l.Err(err).Ctx(ctx).Msg("")
				e.a.SendAlert(context.Background(), err, nil)

				return connect.NewError(connect.CodeInternal, goerrors.New("An internal error occurred."))
			}

			return unwrappedCodedError(err)
		},
	}
}

// unwrappedCodedError keeps the message of a coded error that a handler wrapped, as in
// fmt.Errorf("could not create trigger opt: %w", codedErr). connect would send the inner error
// alone; clients have always received the text of the whole chain, with the coded error rendered
// the way google.golang.org/grpc renders a status error.
func unwrappedCodedError(err error) error {
	coded := new(connect.Error)

	if err == nil || !goerrors.As(err, &coded) || error(coded) == err {
		return err
	}

	message := err.Error()

	if i := strings.LastIndex(message, coded.Error()); i >= 0 {
		message = message[:i] + grpcErrorString(coded) + message[i+len(coded.Error()):]
	}

	unwrapped := connect.NewError(coded.Code(), goerrors.New(message))

	for _, detail := range coded.Details() {
		unwrapped.AddDetail(detail)
	}

	for key, values := range coded.Meta() {
		unwrapped.Meta()[key] = values
	}

	return unwrapped
}

// RecoveryInterceptor converts a panic in a handler into an internal error, for unary and
// streaming calls.
func RecoveryInterceptor(a errors.Alerter, l *zerolog.Logger) connect.Interceptor {
	handlePanic := func(p any) error {
		panicErr, ok := p.(error)

		var panicStr string

		if !ok {
			panicStr, ok = p.(string)

			if !ok {
				panicStr = "Could not determine panic error"
			}
		} else {
			panicStr = panicErr.Error()
		}

		err := fmt.Errorf("recovered from panic: %s. Stack: %s", panicStr, string(debug.Stack()))

		l.Err(err).Msg("")
		a.SendAlert(context.Background(), err, nil)

		return connect.NewError(connect.CodeInternal, goerrors.New("An internal error occurred"))
	}

	return &recoveryInterceptor{handlePanic: handlePanic}
}

type recoveryInterceptor struct {
	handlePanic func(p any) error
}

func (r *recoveryInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (_ connect.AnyResponse, err error) {
		if req.Spec().IsClient {
			return next(ctx, req)
		}

		defer func() {
			if p := recover(); p != nil {
				err = r.handlePanic(p)
			}
		}()

		return next(ctx, req)
	}
}

func (r *recoveryInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (r *recoveryInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) (err error) {
		defer func() {
			if p := recover(); p != nil {
				err = r.handlePanic(p)
			}
		}()

		return next(ctx, conn)
	}
}
