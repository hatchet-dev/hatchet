//go:build !e2e && !load && !rampup && !integration

package engine

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An operator core that fails at startup (a database that is not reachable yet) is restarted
// with backoff until it runs, and a later unexpected stop is restarted too; stop ends the
// core, waits for it and never reports the earlier failures.
func TestServerlessSupervisorRestartsTheCore(t *testing.T) {
	serverlessRestartBackoff = 5 * time.Millisecond
	t.Cleanup(func() { serverlessRestartBackoff = time.Second })

	var starts atomic.Int32
	l := zerolog.Nop()

	sup := superviseServerless(context.Background(), &l, func(ctx context.Context) error {
		n := starts.Add(1)

		if n <= 2 {
			return errors.New("initial heartbeat: database unreachable")
		}

		<-ctx.Done()

		return nil
	})

	require.Eventually(t, func() bool { return starts.Load() == 3 }, 3*time.Second, time.Millisecond, "the core is restarted after each early failure")
	assert.True(t, sup.Running(), "readiness follows the running core")

	stopped := make(chan error, 1)

	go func() { stopped <- sup.stop() }()

	select {
	case err := <-stopped:
		assert.NoError(t, err, "earlier failures are not reported by stop")
	case <-time.After(3 * time.Second):
		t.Fatal("stop did not return")
	}

	assert.False(t, sup.Running())
	assert.Equal(t, int32(3), starts.Load(), "no restart after stop")
}

// A core that fails during shutdown does not make stop fail: the engine's cleanup chain must
// continue past the operator.
func TestServerlessSupervisorStopSwallowsShutdownErrors(t *testing.T) {
	l := zerolog.Nop()

	sup := superviseServerless(context.Background(), &l, func(ctx context.Context) error {
		<-ctx.Done()

		return errors.New("could not release serverless leases")
	})

	require.Eventually(t, sup.Running, time.Second, time.Millisecond)
	assert.NoError(t, sup.stop())
}
