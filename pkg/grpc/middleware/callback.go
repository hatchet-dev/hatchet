package middleware

import (
	"context"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

type callbackKey struct{}
type CallbackFunc func(method, correlationID string)

func WithCallback(ctx context.Context, callback CallbackFunc) context.Context {
	return context.WithValue(ctx, callbackKey{}, callback)
}

func TriggerCallback(ctx context.Context, correlationID string) {
	if callback, ok := ctx.Value(callbackKey{}).(CallbackFunc); ok {
		if method, ok := ctx.Value("grpc_method").(string); ok {
			callback(method, correlationID)
		}
	}
}

// CallbackInterceptor creates an interceptor that can receive callbacks from handlers
func CallbackInterceptor(logger *zerolog.Logger, onCallback func(ctx context.Context, method, correlationID string) error) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Store method name in context
		ctx = context.WithValue(ctx, "grpc_method", info.FullMethod)

		ctx = WithCallback(ctx, func(method, correlationID string) {
			if err := onCallback(ctx, method, correlationID); err != nil {
				logger.Error().Err(err).Msg("Callback error")
			}
		})

		return handler(ctx, req)
	}
}
