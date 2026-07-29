package retry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestGRPCRetryableCodesExcludeFailedPrecondition(t *testing.T) {
	t.Parallel()

	retryable := grpcRetryableCodes()
	assert.Contains(t, retryable, codes.ResourceExhausted)
	assert.Contains(t, retryable, codes.DeadlineExceeded)
	assert.Contains(t, retryable, codes.Internal)
	assert.Contains(t, retryable, codes.Unavailable)
	assert.NotContains(t, retryable, codes.FailedPrecondition)
}

func TestGRPCFullJitterBackoff(t *testing.T) {
	t.Parallel()

	backoff := grpcFullJitterBackoff(func(maxExclusive int64) int64 {
		return maxExclusive - 1
	})

	assert.Equal(t, 5*time.Second, backoff(context.Background(), 1))
	assert.Equal(t, 10*time.Second, backoff(context.Background(), 2))
	assert.Equal(t, grpcMaxDelay, backoff(context.Background(), 99))
}
