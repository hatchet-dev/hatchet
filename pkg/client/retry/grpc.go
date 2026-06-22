package retry

import (
	"context"
	"math/rand/v2"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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

	retryOnCodes := grpcRetryableCodes()

	retryOpts := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpcFullJitterBackoff(rand.Int64N)),
		grpc_retry.WithMax(grpcMaxRetries),
		grpc_retry.WithPerRetryTimeout(30 * time.Second),
		grpc_retry.WithCodes(retryOnCodes...),
		grpc_retry.WithOnRetryCallback(grpc_retry.OnRetryCallback(func(ctx context.Context, attempt uint, err error) {
			l.Debug().Msgf("grpc_retry attempt: %d, backoff for %v", attempt, err)
		})),
	}

	return []grpc.DialOption{
		grpc.WithChainStreamInterceptor(grpc_retry.StreamClientInterceptor(retryOpts...)),
		grpc.WithChainUnaryInterceptor(grpc_retry.UnaryClientInterceptor(retryOpts...)),
	}
}

func grpcFullJitterBackoff(jitter func(int64) int64) grpc_retry.BackoffFunc {
	return func(_ context.Context, attempt uint) time.Duration {
		return fullJitterDelay(int(attempt), grpcBaseDelay, grpcBackoffFactor, grpcMaxDelay, jitter)
	}
}
