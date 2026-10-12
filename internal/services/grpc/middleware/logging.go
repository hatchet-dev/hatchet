package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
)

type callStartKey struct{}

// LoggingInterceptor logs the start and finish of every call. The messages, field names and
// levels are a contract with log queries and alerts: grpc.code carries the canonical gRPC code
// name and grpc.error the gRPC rendering of the error.
func LoggingInterceptor(l *zerolog.Logger) connect.Interceptor {
	return &handlerInterceptor{
		before: func(ctx context.Context, spec connect.Spec, _ http.Header, peer connect.Peer) (context.Context, error) {
			start := callStart{at: time.Now(), peer: peer}

			logStartedCall(ctx, l, spec, start)

			return context.WithValue(ctx, callStartKey{}, start), nil
		},
		after: func(ctx context.Context, spec connect.Spec, err error) error {
			start, _ := ctx.Value(callStartKey{}).(callStart)

			logFinishedCall(l, spec, start, err)

			return err
		},
	}
}

func logStartedCall(ctx context.Context, l *zerolog.Logger, spec connect.Spec, start callStart) {
	ev := callFields(l.Info(), spec, start.peer).Str("grpc.start_time", start.at.Format(time.RFC3339))

	if d, ok := ctx.Deadline(); ok {
		ev = ev.Str("grpc.request.deadline", d.Format(time.RFC3339))
	}

	ev.Msg("started call")
}

func logFinishedCall(l *zerolog.Logger, spec connect.Spec, start callStart, err error) {
	code := "OK"

	if err != nil {
		code = codeName(connect.CodeOf(err))
	}

	ev := callFields(l.WithLevel(codeToLevel(err)), spec, start.peer).
		Str("grpc.start_time", start.at.Format(time.RFC3339)).
		Str("grpc.code", code)

	if err != nil {
		ev = ev.Str("grpc.error", grpcErrorString(err))
	}

	ev.Str("grpc.time_ms", fmt.Sprintf("%v", float32(time.Since(start.at).Nanoseconds()/1000)/1000)).
		Msg("finished call")
}

type callStart struct {
	at   time.Time
	peer connect.Peer
}

func callFields(ev *zerolog.Event, spec connect.Spec, peer connect.Peer) *zerolog.Event {
	service, method, _ := strings.Cut(strings.TrimPrefix(spec.Procedure, "/"), "/")

	ev = ev.Str("protocol", "grpc").
		Str("grpc.component", "server").
		Str("grpc.service", service).
		Str("grpc.method", method).
		Str("grpc.method_type", methodType(spec.StreamType))

	if peer.Addr != "" {
		ev = ev.Str("peer.address", peer.Addr)
	}

	return ev
}

func methodType(t connect.StreamType) string {
	switch t {
	case connect.StreamTypeClient:
		return "client_stream"
	case connect.StreamTypeServer:
		return "server_stream"
	case connect.StreamTypeBidi:
		return "bidi_stream"
	default:
		return "unary"
	}
}

func codeToLevel(err error) zerolog.Level {
	if err == nil {
		return zerolog.InfoLevel
	}

	switch connect.CodeOf(err) {
	case connect.CodeNotFound, connect.CodeCanceled, connect.CodeAlreadyExists, connect.CodeInvalidArgument, connect.CodeUnauthenticated:
		return zerolog.InfoLevel
	case connect.CodeDeadlineExceeded, connect.CodePermissionDenied, connect.CodeResourceExhausted, connect.CodeFailedPrecondition,
		connect.CodeAborted, connect.CodeOutOfRange, connect.CodeUnavailable:
		return zerolog.WarnLevel
	default:
		return zerolog.ErrorLevel
	}
}

// codeName renders a code the way google.golang.org/grpc/codes does (NotFound, not not_found),
// so log queries on grpc.code keep matching.
func codeName(c connect.Code) string {
	switch c {
	case connect.CodeCanceled:
		return "Canceled"
	case connect.CodeUnknown:
		return "Unknown"
	case connect.CodeInvalidArgument:
		return "InvalidArgument"
	case connect.CodeDeadlineExceeded:
		return "DeadlineExceeded"
	case connect.CodeNotFound:
		return "NotFound"
	case connect.CodeAlreadyExists:
		return "AlreadyExists"
	case connect.CodePermissionDenied:
		return "PermissionDenied"
	case connect.CodeResourceExhausted:
		return "ResourceExhausted"
	case connect.CodeFailedPrecondition:
		return "FailedPrecondition"
	case connect.CodeAborted:
		return "Aborted"
	case connect.CodeOutOfRange:
		return "OutOfRange"
	case connect.CodeUnimplemented:
		return "Unimplemented"
	case connect.CodeInternal:
		return "Internal"
	case connect.CodeUnavailable:
		return "Unavailable"
	case connect.CodeDataLoss:
		return "DataLoss"
	case connect.CodeUnauthenticated:
		return "Unauthenticated"
	default:
		return fmt.Sprintf("Code(%d)", c)
	}
}

// grpcErrorString renders err the way google.golang.org/grpc renders a status error, which is the
// text logs and wrapped error messages have always carried.
func grpcErrorString(err error) string {
	return fmt.Sprintf("rpc error: code = %s desc = %s", codeName(connect.CodeOf(err)), errorMessage(err))
}

// errorMessage is the message of err without connect's code prefix.
func errorMessage(err error) string {
	if connectErr := new(connect.Error); errors.As(err, &connectErr) && error(connectErr) == err {
		return connectErr.Message()
	}

	return err.Error()
}
