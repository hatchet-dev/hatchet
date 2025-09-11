package middleware

import (
	"context"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	"github.com/hatchet-dev/hatchet/pkg/constants"
)

type callbackKey struct{}
type CallbackFunc func(ctx context.Context)

func WithCallback(ctx context.Context, callback CallbackFunc) context.Context {
	return context.WithValue(ctx, callbackKey{}, callback)
}

func TriggerCallback(ctx context.Context) {
	if callback, ok := ctx.Value(callbackKey{}).(CallbackFunc); ok {
		callback(ctx)
	}
}

// CallbackInterceptor creates an interceptor that can receive callbacks from handlers
func CallbackInterceptor(logger *zerolog.Logger, onCallback func(ctx context.Context) error) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Store method name in context for callback
		ctx = context.WithValue(ctx, constants.GRPCMethodKey, info.FullMethod)

		ctx = WithCallback(ctx, func(callbackCtx context.Context) {
			if err := onCallback(callbackCtx); err != nil {
				logger.Error().Err(err).Msg("Callback error")
			}
		})

		return handler(ctx, req)
	}
}
