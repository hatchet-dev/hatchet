package retry

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStreamBackoffDelayFullJitter(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewPCG(1, 1))
	delay := fullJitterDelay(3, StreamBaseDelay, StreamBackoffFactor, StreamMaxDelay, r.Int64N)
	assert.GreaterOrEqual(t, delay, time.Duration(0))
	assert.LessOrEqual(t, delay, StreamMaxDelay)
}

func TestStreamBackoffDelayCap(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewPCG(2, 2))
	delay := fullJitterDelay(100, StreamBaseDelay, StreamBackoffFactor, StreamMaxDelay, r.Int64N)
	assert.LessOrEqual(t, delay, StreamMaxDelay)
}

func TestSleepCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Sleep(ctx, time.Second)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestSleepStreamBackoffCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := SleepStreamBackoff(ctx, 0)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestClassifyStreamErrorTransientCodes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	codesToRetry := []codes.Code{
		codes.Unavailable,
		codes.Internal,
		codes.DeadlineExceeded,
		codes.ResourceExhausted,
	}

	for _, code := range codesToRetry {
		err := status.Error(code, "transient")
		assert.Equal(t, StreamDecisionRetry, ClassifyStreamError(ctx, err), "code %s", code)
	}
}

func TestClassifyStreamErrorPermanentCodes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	codesToStop := []codes.Code{
		codes.Canceled,
		codes.Unauthenticated,
		codes.PermissionDenied,
		codes.InvalidArgument,
		codes.FailedPrecondition,
		codes.NotFound,
		codes.Unimplemented,
	}

	for _, code := range codesToStop {
		err := status.Error(code, "permanent")
		assert.Equal(t, StreamDecisionStop, ClassifyStreamError(ctx, err), "code %s", code)
	}
}

func TestClassifyStreamErrorEOFAndUnknown(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	assert.Equal(t, StreamDecisionNoProgress, ClassifyStreamError(ctx, io.EOF))
	assert.Equal(t, StreamDecisionNoProgress, ClassifyStreamError(ctx, fmt.Errorf("plain error")))
}

func TestClassifyStreamErrorContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := status.Error(codes.Unavailable, "transient")
	assert.Equal(t, StreamDecisionStop, ClassifyStreamError(ctx, err))
}
