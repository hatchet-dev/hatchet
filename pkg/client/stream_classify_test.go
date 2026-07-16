package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewStreamClassifierTransientCodes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	classify := newStreamClassifier(func(context.Context) bool { return false })

	codesToRetry := []codes.Code{
		codes.Unavailable,
		codes.Internal,
		codes.DeadlineExceeded,
		codes.ResourceExhausted,
	}

	for _, code := range codesToRetry {
		err := status.Error(code, "transient")
		assert.Equal(t, verdictRetry, classify(ctx, err), "code %s", code)
	}
}

func TestNewStreamClassifierPermanentCodes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	classify := newStreamClassifier(func(context.Context) bool { return false })

	codesToStop := []codes.Code{
		codes.Unauthenticated,
		codes.PermissionDenied,
		codes.InvalidArgument,
		codes.FailedPrecondition,
		codes.NotFound,
		codes.Unimplemented,
	}

	for _, code := range codesToStop {
		err := status.Error(code, "permanent")
		assert.Equal(t, verdictStopError, classify(ctx, err), "code %s", code)
	}
}

func TestNewStreamClassifierCleanStopConditions(t *testing.T) {
	t.Parallel()

	classify := newStreamClassifier(func(context.Context) bool { return false })

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.Equal(t, verdictStopClean, classify(ctx, status.Error(codes.Unavailable, "transient")))
	})

	t.Run("errListenerClosed", func(t *testing.T) {
		assert.Equal(t, verdictStopClean, classify(context.Background(), errListenerClosed))
	})

	t.Run("context.Canceled", func(t *testing.T) {
		assert.Equal(t, verdictStopClean, classify(context.Background(), context.Canceled))
	})

	t.Run("grpc Canceled", func(t *testing.T) {
		assert.Equal(t, verdictStopClean, classify(context.Background(), status.Error(codes.Canceled, "cancelled")))
	})
}

func TestNewStreamClassifierEOFCallback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	calls := 0
	classify := newStreamClassifier(func(context.Context) bool {
		calls++
		return calls%2 == 1
	})

	assert.Equal(t, verdictRetry, classify(ctx, io.EOF))
	assert.Equal(t, verdictStopClean, classify(ctx, io.EOF))
	assert.Equal(t, 2, calls)
}

func TestNewStreamClassifierPlainError(t *testing.T) {
	t.Parallel()

	classify := newStreamClassifier(func(context.Context) bool { return false })
	assert.Equal(t, verdictNoProgress, classify(context.Background(), fmt.Errorf("plain error")))
}

func TestNoProgressFatalClassifierWrapper(t *testing.T) {
	t.Parallel()

	base := newStreamClassifier(func(context.Context) bool { return false })
	classify := func(ctx context.Context, err error) streamVerdict {
		if v := base(ctx, err); v != verdictNoProgress {
			return v
		}
		return verdictStopError
	}

	assert.Equal(t, verdictStopError, classify(context.Background(), fmt.Errorf("plain error")))
	assert.Equal(t, verdictRetry, classify(context.Background(), status.Error(codes.Unavailable, "transient")))
}

func TestShouldLogReconnectMilestone(t *testing.T) {
	t.Parallel()

	assert.True(t, shouldLogReconnectMilestone(1))
	assert.False(t, shouldLogReconnectMilestone(2))
	assert.False(t, shouldLogReconnectMilestone(4))
	assert.True(t, shouldLogReconnectMilestone(5))
	assert.True(t, shouldLogReconnectMilestone(10))
}

func TestSendListenerError(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)
	sendListenerError(context.Background(), errCh, errors.New("boom"))
	require.Equal(t, "boom", (<-errCh).Error())
}
