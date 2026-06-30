package client

import (
	"context"
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

func disableStreamBackoffForTest(t *testing.T) {
	t.Helper()

	retry.SetStreamSleepHookForTesting(func(ctx context.Context, attempt int) error {
		return nil
	})
	t.Cleanup(retry.ResetStreamSleepHookForTesting)
}
