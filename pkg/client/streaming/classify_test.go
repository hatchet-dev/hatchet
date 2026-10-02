package streaming

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
	classify := NewClassifier(func(context.Context) bool { return false })

	codesToRetry := []codes.Code{
		codes.Unavailable,
		codes.Internal,
		codes.DeadlineExceeded,
		codes.ResourceExhausted,
	}

	for _, code := range codesToRetry {
		err := status.Error(code, "transient")
		assert.Equal(t, VerdictRetry, classify(ctx, err), "code %s", code)
	}
}

func TestNewStreamClassifierPermanentCodes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	classify := NewClassifier(func(context.Context) bool { return false })

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
		assert.Equal(t, VerdictStopError, classify(ctx, err), "code %s", code)
	}
}

func TestNewStreamClassifierCleanStopConditions(t *testing.T) {
	t.Parallel()

	classify := NewClassifier(func(context.Context) bool { return false })

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.Equal(t, VerdictStopClean, classify(ctx, status.Error(codes.Unavailable, "transient")))
	})

	t.Run("ErrListenerClosed", func(t *testing.T) {
		assert.Equal(t, VerdictStopClean, classify(context.Background(), ErrListenerClosed))
	})

	t.Run("context.Canceled", func(t *testing.T) {
		assert.Equal(t, VerdictStopClean, classify(context.Background(), context.Canceled))
	})

	t.Run("grpc Canceled", func(t *testing.T) {
		assert.Equal(t, VerdictStopClean, classify(context.Background(), status.Error(codes.Canceled, "cancelled")))
	})
}

func TestNewStreamClassifierEOFCallback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	calls := 0
	classify := NewClassifier(func(context.Context) bool {
		calls++
		return calls%2 == 1
	})

	assert.Equal(t, VerdictRetry, classify(ctx, io.EOF))
	assert.Equal(t, VerdictStopClean, classify(ctx, io.EOF))
	assert.Equal(t, 2, calls)
}

func TestNewStreamClassifierPlainError(t *testing.T) {
	t.Parallel()

	classify := NewClassifier(func(context.Context) bool { return false })
	assert.Equal(t, VerdictNoProgress, classify(context.Background(), fmt.Errorf("plain error")))
}

func TestNoProgressFatalClassifierWrapper(t *testing.T) {
	t.Parallel()

	base := NewClassifier(func(context.Context) bool { return false })
	classify := func(ctx context.Context, err error) Verdict {
		if v := base(ctx, err); v != VerdictNoProgress {
			return v
		}
		return VerdictStopError
	}

	assert.Equal(t, VerdictStopError, classify(context.Background(), fmt.Errorf("plain error")))
	assert.Equal(t, VerdictRetry, classify(context.Background(), status.Error(codes.Unavailable, "transient")))
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
	SendListenerError(context.Background(), errCh, errors.New("boom"))
	require.Equal(t, "boom", (<-errCh).Error())
}
