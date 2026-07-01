package retry

import (
	"context"
	"errors"
	"io"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StreamDecision describes how a stream error should be handled.
type StreamDecision int

const (
	// StreamDecisionRetry indicates the stream should reconnect after a transient failure.
	StreamDecisionRetry StreamDecision = iota
	// StreamDecisionStop indicates the stream should exit without reconnecting.
	StreamDecisionStop
	// StreamDecisionNoProgress indicates reconnect would not advance the stream.
	StreamDecisionNoProgress
)

const (
	// StreamBaseDelay is the base delay for stream reconnect backoff.
	StreamBaseDelay = 1 * time.Second
	// StreamMaxDelay caps stream reconnect backoff delay.
	StreamMaxDelay = 30 * time.Second
	// StreamBackoffFactor is the exponential multiplier for stream reconnect backoff.
	StreamBackoffFactor = 2.0
	// StreamSyncMaxAttempts bounds synchronous subscribe/send reconnect attempts.
	StreamSyncMaxAttempts = 5
)

// StreamBackoffDelay returns a full-jitter delay for the given stream reconnect attempt.
func StreamBackoffDelay(attempt int) time.Duration {
	clk := defaultClock()
	return fullJitterDelay(attempt, StreamBaseDelay, StreamBackoffFactor, StreamMaxDelay, clk.jitter)
}

// Sleep waits for d or until ctx is cancelled.
func Sleep(ctx context.Context, d time.Duration) error {
	return sleepContext(ctx, d)
}

var streamSleepHook func(ctx context.Context, attempt int) error

// SetStreamSleepHookForTesting overrides stream reconnect sleep for tests.
func SetStreamSleepHookForTesting(hook func(ctx context.Context, attempt int) error) {
	streamSleepHook = hook
}

// ResetStreamSleepHookForTesting clears the stream reconnect sleep test override.
func ResetStreamSleepHookForTesting() {
	streamSleepHook = nil
}

// SleepStreamBackoff waits for the stream reconnect backoff delay or until ctx is cancelled.
func SleepStreamBackoff(ctx context.Context, attempt int) error {
	if streamSleepHook != nil {
		return streamSleepHook(ctx, attempt)
	}

	return Sleep(ctx, StreamBackoffDelay(attempt))
}

// ClassifyStreamError maps a stream error to a reconnect decision.
func ClassifyStreamError(ctx context.Context, err error) StreamDecision {
	if err == nil {
		return StreamDecisionStop
	}

	if ctx.Err() != nil {
		return StreamDecisionStop
	}

	if errors.Is(err, io.EOF) {
		return StreamDecisionNoProgress
	}

	st, ok := status.FromError(err)
	if !ok {
		return StreamDecisionNoProgress
	}

	switch st.Code() {
	case codes.Canceled:
		return StreamDecisionStop
	case codes.Unauthenticated, codes.PermissionDenied, codes.InvalidArgument,
		codes.FailedPrecondition, codes.NotFound, codes.Unimplemented:
		return StreamDecisionStop
	case codes.Unavailable, codes.Internal, codes.DeadlineExceeded, codes.ResourceExhausted:
		return StreamDecisionRetry
	default:
		return StreamDecisionNoProgress
	}
}
