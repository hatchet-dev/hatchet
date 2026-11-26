package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
)

// NewRequestGate authenticates and authorizes a call and applies the per-token rate limit once the
// request headers are in, before any message is read or decoded. A rejected call never reaches the
// interceptor chain, so the gate logs and traces it the way the chain would have.
func NewRequestGate(authn *GRPCAuthN, authz *GRPCAuthZ, limiter *HatchetRateLimiter, l *zerolog.Logger) connect.RequestGateFunc {
	return newRequestGate(authn, authz, limiter, l, newGlobalTelemetryRecorder())
}

func newRequestGate(authn *GRPCAuthN, authz *GRPCAuthZ, limiter *HatchetRateLimiter, l *zerolog.Logger, recorder *telemetryRecorder) connect.RequestGateFunc {
	return func(ctx context.Context, spec connect.Spec, peer connect.Peer, header http.Header) (context.Context, error) {
		start := callStart{at: time.Now(), peer: peer}

		authenticated, err := admit(ctx, header, spec.Procedure, authn, authz, limiter)
		if err != nil {
			logStartedCall(ctx, l, spec, start)
			logFinishedCall(l, spec, start, err)

			// admit returns the authenticated context even when authz or the rate limit rejects the call,
			// so those spans still carry the tenant.
			recorder.recordRejected(authenticated, ctx, spec, header, start.at, err)

			return nil, err
		}

		return authenticated, nil
	}
}

func admit(ctx context.Context, header http.Header, procedure string, authn *GRPCAuthN, authz *GRPCAuthZ, limiter *HatchetRateLimiter) (context.Context, error) {
	authenticated, err := authn.Middleware(ctx, header)
	if err != nil {
		return authenticated, err
	}

	if err := authz.Authorize(authenticated, procedure); err != nil {
		return authenticated, err
	}

	return authenticated, rateLimit(authenticated, limiter, procedure)
}

// rateLimit reports every limiter error as resource exhausted with the message clients have
// always received, which names the method and embeds the limiter's own error.
func rateLimit(ctx context.Context, limiter *HatchetRateLimiter, procedure string) error {
	err := limiter.Limit(ctx, procedure)

	if err == nil {
		return nil
	}

	return connect.NewError(
		connect.CodeResourceExhausted,
		fmt.Errorf("%s is rejected by grpc_ratelimit middleware, please retry later. %s", procedure, grpcErrorString(err)),
	)
}
