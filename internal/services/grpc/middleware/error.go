package middleware

import (
	"context"
	goerrors "errors"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/errors"
)

// expectedStatus maps known, non-internal errors to their canonical gRPC codes.
// These should be returned to the client without error-logging or alerting.
func expectedStatus(err error) error {
	if goerrors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "request was canceled")
	}

	if goerrors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "request deadline exceeded")
	}

	if goerrors.Is(err, pgx.ErrNoRows) {
		// A missing row is a client-visible miss (task already complete, webhook
		// deleted, etc.), not an engine failure.
		return status.Error(codes.NotFound, "not found")
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

// UnaryServerInterceptor returns a new unary server interceptor for panic recovery.
func (e *ErrorInterceptor) ErrorUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ any, err error) {
		res, err := handler(ctx, req)

		// if this is not a grpc error already, convert it to an internal grpc error
		if err != nil && status.Code(err) == codes.Unknown {
			if statusErr := expectedStatus(err); statusErr != nil {
				return res, statusErr
			}

			e.l.Err(err).Ctx(ctx).Msg("")
			e.a.SendAlert(context.Background(), err, nil)

			err = status.Errorf(codes.Internal, "An internal error occurred.")
		}

		return res, err
	}
}

// StreamServerInterceptor returns a new streaming server interceptor for panic recovery.
func (e *ErrorInterceptor) ErrorStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		err = handler(srv, stream)

		// if this is not a grpc error already, convert it to an internal grpc error
		if err != nil && status.Code(err) == codes.Unknown {
			if statusErr := expectedStatus(err); statusErr != nil {
				return statusErr
			}

			e.l.Err(err).Ctx(stream.Context()).Msg("")
			e.a.SendAlert(context.Background(), err, nil)

			err = status.Errorf(codes.Internal, "An internal error occurred.")
		}

		return err
	}
}
