package middleware

import (
	"context"
	goerrors "errors"
	"fmt"
	"runtime/debug"
	"strings"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
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

	// The connection cap is throttling, so it maps to ResourceExhausted. The RetryInfo detail
	// marks it as transient, which tells it apart from a tenant quota of the same code.
	var limitErr *fairpool.LimitError
	if goerrors.As(err, &limitErr) {
		throttled := connect.NewError(connect.CodeResourceExhausted, goerrors.New("resource exhausted: too many concurrent database operations for this tenant, retry shortly"))

		if detail, detailErr := connect.NewErrorDetail(&errdetails.RetryInfo{RetryDelay: durationpb.New(fairpool.RetryAfter)}); detailErr == nil {
			throttled.AddDetail(detail)
		}

		return throttled
	}

	// An operator's ingest rate limit is throttling too, and the client learns when its request
	// would be admitted.
	var rateLimited *repository.TenantRateLimitedError
	if goerrors.As(err, &rateLimited) {
		limited := connect.NewError(connect.CodeResourceExhausted, rateLimited)

		if detail, detailErr := connect.NewErrorDetail(&errdetails.RetryInfo{RetryDelay: durationpb.New(rateLimited.RetryAfter)}); detailErr == nil {
			limited.AddDetail(detail)
		}

		return limited
	}

	// A pause lasts until an operator ends it, so the refusal carries no RetryInfo and a client
	// backs off on its own schedule.
	if goerrors.Is(err, repository.ErrTenantPaused) {
		return connect.NewError(connect.CodeResourceExhausted, repository.ErrTenantPaused)
	}

	// Extensions can return gRPC status errors. Convert their code and details here;
	// redactServerError removes diagnostics from server failures before they reach the wire.
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

// Interceptor logs unexpected failures and hides server diagnostics on unary and streaming calls.
func (e *ErrorInterceptor) Interceptor() connect.Interceptor {
	return &handlerInterceptor{
		after: func(ctx context.Context, _ connect.Spec, err error) error {
			// if this is not a coded error already, convert it to an internal error
			if err != nil && connect.CodeOf(err) == connect.CodeUnknown {
				if statusErr := expectedStatus(ctx, err); statusErr != nil {
					return e.redactServerError(ctx, err, statusErr)
				}

				e.l.Err(err).Ctx(ctx).Msg("")
				e.a.SendAlert(context.Background(), err, nil)

				return connect.NewError(connect.CodeInternal, goerrors.New("An internal error occurred."))
			}

			return e.redactServerError(ctx, err, unwrappedCodedError(err))
		},
	}
}

// Server and transport errors may contain backend addresses or debug details even when
// they have an explicit status code. Only client-actionable errors retain their details.
func (e *ErrorInterceptor) redactServerError(ctx context.Context, cause, response error) error {
	if response == nil {
		return nil
	}

	code := connect.CodeOf(response)
	level := zerolog.WarnLevel
	var message string
	//exhaustive:ignore
	switch code {
	case connect.CodeInternal:
		message = "An internal error occurred."
		level = zerolog.ErrorLevel
	case connect.CodeDataLoss:
		message = "An internal data error occurred."
		level = zerolog.ErrorLevel
	case connect.CodeUnavailable:
		message = "service temporarily unavailable"
	case connect.CodeAborted:
		message = "request was aborted"
	case connect.CodeDeadlineExceeded:
		message = "request deadline exceeded"
	case connect.CodeCanceled:
		message = "request was canceled"
		level = zerolog.DebugLevel
	default:
		return response
	}

	e.l.WithLevel(level).Ctx(ctx).Err(cause).Msg("RPC request failed")
	return connect.NewError(code, goerrors.New(message))
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

		return connect.NewError(connect.CodeInternal, goerrors.New("An internal error occurred."))
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
