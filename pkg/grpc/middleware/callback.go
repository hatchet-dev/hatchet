package middleware

import (
	"context"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"

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

// TriggerResourceCallback reports one committed resource to the request's callback. Each call
// builds its own context, so resources reported in a loop don't share values.
func TriggerResourceCallback(ctx context.Context, resourceType constants.ResourceTypeValue, resourceID string, correlationID *string) {
	ctx = context.WithValue(ctx, constants.ResourceIdKey, resourceID)
	ctx = context.WithValue(ctx, constants.ResourceTypeKey, resourceType)

	if correlationID != nil {
		ctx = context.WithValue(ctx, constants.CorrelationIdKey, *correlationID)
	}

	TriggerCallback(ctx)
}

// CallbackInterceptor creates an interceptor that can receive callbacks from handlers
func CallbackInterceptor(logger *zerolog.Logger, onCallback func(ctx context.Context) error) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Store method name in context for callback
			ctx = context.WithValue(ctx, constants.GRPCMethodKey, req.Spec().Procedure)

			ctx = WithCallback(ctx, func(callbackCtx context.Context) {
				if err := onCallback(callbackCtx); err != nil {
					logger.Error().Err(err).Msg("Callback error")
				}
			})

			return next(ctx, req)
		}
	}
}
