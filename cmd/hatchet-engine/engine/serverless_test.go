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

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator"
)

func TestServerlessConfigFromServer(t *testing.T) {
	cf := server.ServerlessOperatorConfigFile{
		OperatorName:                 "custom",
		DefaultSlots:                 11,
		DurableSlots:                 12,
		LeaseTTL:                     13 * time.Second,
		HeartbeatInterval:            14 * time.Second,
		RebalanceInterval:            15 * time.Second,
		ShedHysteresis:               0.3,
		DrainTimeout:                 16 * time.Second,
		RoutingRefreshInterval:       17 * time.Second,
		HealthcheckTimeout:           18 * time.Second,
		HealthcheckConcurrency:       19,
		WSMaxFrameBytes:              20,
		WSPingInterval:               21 * time.Second,
		HealthcheckTenantConcurrency: 22,
		HealthcheckApplyTimeout:      23 * time.Second,
		MaxWorkflowsPerEndpoint:      24,
		MaxActionsPerEndpoint:        25,
		MaintenanceConcurrency:       26,
		LeaseMaxClaimPerTick:         27,
		WSMaxUpgradeHeaderBytes:      28,
		WSMaxQueuedBytes:             29,
	}

	got := serverlessConfigFromServer(cf)

	assert.Equal(t, serverlessoperator.Config{
		OperatorName:                 "custom",
		LinkName:                     serverlessLinkName,
		DefaultSlots:                 11,
		DurableSlots:                 12,
		LeaseTTL:                     13 * time.Second,
		HeartbeatInterval:            14 * time.Second,
		RebalanceInterval:            15 * time.Second,
		ShedHysteresis:               0.3,
		DrainTimeout:                 16 * time.Second,
		RoutingRefreshInterval:       17 * time.Second,
		HealthcheckTimeout:           18 * time.Second,
		HealthcheckConcurrency:       19,
		WSMaxFrameBytes:              20,
		WSPingInterval:               21 * time.Second,
		HealthcheckTenantConcurrency: 22,
		HealthcheckApplyTimeout:      23 * time.Second,
		MaxWorkflowsPerEndpoint:      24,
		MaxActionsPerEndpoint:        25,
		MaintenanceConcurrency:       26,
		LeaseMaxClaimPerTick:         27,
		WSMaxUpgradeHeaderBytes:      28,
		WSMaxQueuedBytes:             29,
		HealthPort:                   0,
	}, got)

	// the engine serves health and metrics itself; the core's server stays off even when
	// the config file is empty
	assert.Equal(t, 0, serverlessConfigFromServer(server.ServerlessOperatorConfigFile{}).HealthPort)
	assert.Equal(t, serverlessLinkName, serverlessConfigFromServer(server.ServerlessOperatorConfigFile{}).LinkName)
}

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
