package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/errors"
)

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
			e.l.Err(err).Msg("")
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
			e.l.Err(err).Msg("")
			e.a.SendAlert(context.Background(), err, nil)

			err = status.Errorf(codes.Internal, "An internal error occurred.")
		}

		return err
	}
}
