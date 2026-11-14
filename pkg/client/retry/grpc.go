package retry

import (
	"context"
	"math/rand/v2"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/rs/zerolog"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func grpcRetryableCodes() []codes.Code {
	return []codes.Code{
		codes.ResourceExhausted,
		codes.DeadlineExceeded,
		codes.Internal,
		codes.Unavailable,
	}
}

// GRPCDialOptions returns gRPC dial options that install unary and stream retry interceptors.
// When enabled is false, it returns nil.
func GRPCDialOptions(l *zerolog.Logger, enabled bool) []grpc.DialOption {
	if !enabled {
		return nil
	}

	return grpcDialOptions(l, rand.Int64N)
}

func grpcDialOptions(l *zerolog.Logger, jitter func(int64) int64) []grpc.DialOption {
	retryOnCodes := grpcRetryableCodes()

	retryOpts := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpcFullJitterBackoff(jitter)),
		grpc_retry.WithMax(grpcMaxRetries),
		grpc_retry.WithPerRetryTimeout(30 * time.Second),
		grpc_retry.WithCodes(retryOnCodes...),
		grpc_retry.WithOnRetryCallback(grpc_retry.OnRetryCallback(func(ctx context.Context, attempt uint, err error) {
			l.Debug().Msgf("grpc_retry attempt: %d, backoff for %v", attempt, err)
		})),
	}

	return []grpc.DialOption{
		grpc.WithChainStreamInterceptor(grpc_retry.StreamClientInterceptor(retryOpts...)),
		// The retry interceptor's backoff sees only the call's context, not the attempt's error.
		// The interceptors around it pass the server's RetryInfo delay to it through that context.
		grpc.WithChainUnaryInterceptor(
			withServerRetryDelay,
			grpc_retry.UnaryClientInterceptor(retryOpts...),
			recordServerRetryDelay,
		),
	}
}

type serverRetryDelayKey struct{}

// withServerRetryDelay gives each unary call a slot for the RetryInfo delay of its latest attempt.
func withServerRetryDelay(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return invoker(context.WithValue(ctx, serverRetryDelayKey{}, new(time.Duration)), method, req, reply, cc, opts...)
}

// recordServerRetryDelay runs once per attempt and stores the RetryInfo delay of its error, or zero.
func recordServerRetryDelay(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	err := invoker(ctx, method, req, reply, cc, opts...)
	*ctx.Value(serverRetryDelayKey{}).(*time.Duration) = retryInfoDelay(err)
	return err
}

// retryInfoDelay returns the delay of the RetryInfo detail on a gRPC status error, or zero.
func retryInfoDelay(err error) time.Duration {
	for _, detail := range status.Convert(err).Details() {
		if info, ok := detail.(*errdetails.RetryInfo); ok {
			return info.GetRetryDelay().AsDuration()
		}
	}

	return 0
}

// grpcMaxServerRetryDelay is the longest wait a server's RetryInfo delay can ask for before one retry.
const grpcMaxServerRetryDelay = 5 * time.Second

// grpcFullJitterBackoff waits the server's RetryInfo delay, capped at grpcMaxServerRetryDelay, when
// the previous attempt carried one. Otherwise it waits a full-jitter delay.
func grpcFullJitterBackoff(jitter func(int64) int64) grpc_retry.BackoffFunc {
	return func(ctx context.Context, attempt uint) time.Duration {
		// Stream calls have no slot; their backoff is always the jittered one.
		if serverDelay, ok := ctx.Value(serverRetryDelayKey{}).(*time.Duration); ok && *serverDelay > 0 {
			return min(*serverDelay, grpcMaxServerRetryDelay)
		}

		// go-grpc-middleware passes attempt=1 for the first retry backoff.
		zeroBasedAttempt := int(attempt) - 1
		return fullJitterDelay(zeroBasedAttempt, grpcBaseDelay, grpcBackoffFactor, grpcMaxDelay, jitter)
	}
}
