package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
)

// NewRequestGate authenticates a call and applies the per-token rate limit once the request
// headers are in, before any message is read or decoded. A rejected call never reaches the
// interceptor chain, so the gate logs and traces it the way the chain would have.
func NewRequestGate(authn *GRPCAuthN, limiter *HatchetRateLimiter, l *zerolog.Logger) connect.RequestGateFunc {
	return newRequestGate(authn, limiter, l, newGlobalTelemetryRecorder())
}

func newRequestGate(authn *GRPCAuthN, limiter *HatchetRateLimiter, l *zerolog.Logger, recorder *telemetryRecorder) connect.RequestGateFunc {
	return func(ctx context.Context, spec connect.Spec, peer connect.Peer, header http.Header) (context.Context, error) {
		start := callStart{at: time.Now(), peer: peer}

		authenticated, err := authn.Middleware(ctx, header)

		if err == nil {
			err = rateLimit(authenticated, limiter, spec.Procedure)
		}

		if err != nil {
			logStartedCall(ctx, l, spec, start)
			logFinishedCall(l, spec, start, err)

			// a call rejected for its rate limit is authenticated, so its span carries the tenant
			recorder.recordRejected(authenticated, ctx, spec, header, start.at, err)

			return nil, err
		}

		return authenticated, nil
	}
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
