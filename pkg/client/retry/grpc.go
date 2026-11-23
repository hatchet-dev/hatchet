package retry

import (
	"context"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/rs/zerolog"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
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
	retryOpts := func(retriable grpc_retry.CallOption) []grpc_retry.CallOption {
		return []grpc_retry.CallOption{
			grpc_retry.WithBackoff(grpcFullJitterBackoff(jitter)),
			grpc_retry.WithMax(grpcMaxRetries),
			grpc_retry.WithPerRetryTimeout(30 * time.Second),
			retriable,
			grpc_retry.WithOnRetryCallback(grpc_retry.OnRetryCallback(func(ctx context.Context, attempt uint, err error) {
				l.Debug().Msgf("grpc_retry attempt: %d, backoff for %v", attempt, err)
			})),
		}
	}

	byCode := retryOpts(grpc_retry.WithCodes(grpcRetryableCodes()...))
	withinServerDelayCap := retryOpts(grpc_retry.WithRetriable(retriableWithinServerDelayCap))

	return []grpc.DialOption{
		grpc.WithChainStreamInterceptor(grpc_retry.StreamClientInterceptor(byCode...)),
		grpc.WithChainUnaryInterceptor(unaryRetry(
			grpc_retry.UnaryClientInterceptor(byCode...),
			grpc_retry.UnaryClientInterceptor(withinServerDelayCap...),
		)),
	}
}

// dispatcherMethodPrefixes match the dispatcher services, which carry a worker's task results. A
// lost result makes the engine time the task out and run it again, so these calls keep retrying on
// the jittered backoff, whose window is longer than one bounded by grpcMaxServerRetryDelay.
var dispatcherMethodPrefixes = []string{
	"/" + dispatchercontracts.Dispatcher_ServiceDesc.ServiceName + "/",
	"/" + v1contracts.V1Dispatcher_ServiceDesc.ServiceName + "/",
}

type serverRetryDelayKey struct{}

// unaryRetry sends dispatcher calls through byCode. Every other call goes through
// withinServerDelayCap, whose backoff reads the RetryInfo delay of the latest attempt. The retry
// interceptor's backoff sees only the call's context, not the attempt's error, so each attempt
// stores its delay, or zero, in a slot on that context.
func unaryRetry(byCode, withinServerDelayCap grpc.UnaryClientInterceptor) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if slices.ContainsFunc(dispatcherMethodPrefixes, func(prefix string) bool { return strings.HasPrefix(method, prefix) }) {
			return byCode(ctx, method, req, reply, cc, invoker, opts...)
		}

		serverDelay := new(time.Duration)
		recordingInvoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			err := invoker(ctx, method, req, reply, cc, opts...)
			*serverDelay = retryInfoDelay(err)
			return err
		}

		return withinServerDelayCap(context.WithValue(ctx, serverRetryDelayKey{}, serverDelay), method, req, reply, cc, recordingInvoker, opts...)
	}
}

// retriableWithinServerDelayCap retries the retryable codes, unless the server asked for a wait
// longer than grpcMaxServerRetryDelay. Retrying sooner would only be refused again, so the call
// fails with the server's error, which still carries the status and RetryInfo for the caller.
func retriableWithinServerDelayCap(err error) bool {
	return slices.Contains(grpcRetryableCodes(), status.Code(err)) && retryInfoDelay(err) <= grpcMaxServerRetryDelay
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

// grpcMaxServerRetryDelay is the longest RetryInfo delay the client accepts, and the cap on that
// delay as it doubles across retries. A server that asks for a longer one fails the call, since the
// wait blocks the caller.
const grpcMaxServerRetryDelay = 5 * time.Second

// grpcServerRetryDelayJitterPercent is the most a wait on a RetryInfo delay is extended, as a
// percentage of the wait. Clients throttled at the same moment get the same delay; the random
// extension keeps them from all coming back at the same moment. It only extends the wait, since a
// retry before the server's delay would be refused.
const grpcServerRetryDelayJitterPercent = 20

// grpcFullJitterBackoff waits a full-jitter delay, unless the previous attempt carried a RetryInfo
// delay. Then it waits that delay, doubled for each retry the call has already made and capped at
// grpcMaxServerRetryDelay, which is the client behavior RetryInfo documents, and extended by up to
// grpcServerRetryDelayJitterPercent. A server that keeps throttling repeats the same delay, so
// without the doubling the client would not back off.
func grpcFullJitterBackoff(jitter func(int64) int64) grpc_retry.BackoffFunc {
	return func(ctx context.Context, attempt uint) time.Duration {
		// go-grpc-middleware passes attempt=1 for the first retry backoff.
		// Stream and dispatcher calls have no slot; their backoff is always the jittered one.
		if serverDelay, ok := ctx.Value(serverRetryDelayKey{}).(*time.Duration); ok && *serverDelay > 0 {
			wait := *serverDelay
			for retry := uint(1); retry < attempt && wait < grpcMaxServerRetryDelay; retry++ {
				wait *= 2
			}

			wait = min(wait, grpcMaxServerRetryDelay)

			return wait + time.Duration(jitter(int64(wait)*grpcServerRetryDelayJitterPercent/100+1))
		}

		zeroBasedAttempt := int(attempt) - 1
		return fullJitterDelay(zeroBasedAttempt, grpcBaseDelay, grpcBackoffFactor, grpcMaxDelay, jitter)
	}
}
