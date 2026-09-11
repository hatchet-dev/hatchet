package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator"
)

// serverlessLinkName labels the in-engine mode's metrics.
const serverlessLinkName = "engine"

// serverlessRestartBackoff is the first delay before the core is restarted after it stops on
// its own; it doubles up to serverlessRestartBackoffMax. Tests shorten it.
var serverlessRestartBackoff = time.Second

const serverlessRestartBackoffMax = 30 * time.Second

// startServerlessOperator runs the serverless operator core inside the dispatcher process
// when SERVER_SERVERLESS_OPERATOR_ENABLED is set, with the dispatcher id as its process id
// and its sessions opened on the engine's in-process operator host, the one the claimed
// operators use, as self-leased GRPC operators whose workers are named after the
// dispatcher. The core is
// supervised: a stop that the engine did not ask for (a database that was unreachable at
// startup, a failing loop) is logged and the core restarted with backoff, so a startup fault
// cannot leave the engine running without its operator. It returns a stop function that
// shuts the core down (release leases, drain deliveries, close registrations) and blocks
// until it has; the caller runs it before the dispatcher drains its workers so the
// registrations' workers are deactivated while the dispatcher can still take their events.
// Stop never reports the core's earlier failures, which are logged when they happen, so the
// engine's cleanup chain continues past it. running reports whether the core is up at the
// moment, for the engine's readiness probe. When disabled the stop function is a no-op and
// running is nil.
//
// The core runs on its own context rather than the engine's so that shutdown is ordered by
// the cleanup chain, not by the engine context's cancellation.
func startServerlessOperator(sc *server.ServerConfig, d *dispatcher.DispatcherImpl, host operator.Host) (stop func() error, running func() bool, err error) {
	if !sc.Runtime.ServerlessOperatorEnabled {
		return func() error { return nil }, nil, nil
	}

	l := sc.Logger.With().Str("service", "serverless-operator").Logger()
	cf := sc.Runtime.ServerlessOperator

	sender, err := safeclient.New(safeclient.Config{
		InfraBlockedCIDRs:    sc.Runtime.OperatorInfraBlockedCIDRs,
		AllowEmptyInfraCIDRs: cf.AllowEmptyInfraCIDRs || cf.InsecureDestinations,
		InsecureDestinations: cf.InsecureDestinations,
		MaxIdleConns:         cf.HTTPMaxIdleConns,
		MaxIdleConnsPerHost:  cf.HTTPMaxIdleConnsPerHost,
		IdleConnTimeout:      cf.HTTPIdleConnTimeout,
	}, &l)

	if err != nil {
		return nil, nil, fmt.Errorf("could not build serverless operator request sender: %w", err)
	}

	cfg := serverlessConfigFromServer(cf)
	hostname, _ := os.Hostname()

	sup := superviseServerless(context.Background(), &l, func(ctx context.Context) error {
		return serverlessoperator.Run(ctx, serverlessoperator.Deps{
			Repo:       sc.V1.Serverless(),
			Host:       host,
			WorkerName: serverlessWorkerName(d.DispatcherId()),
			Encryption: sc.Encryption,
			Sender:     sender,
			Logger:     &l,
			Version:    sc.Version,
			Hostname:   hostname,
			ProcessId:  d.DispatcherId(),
			Config:     cfg,
		})
	})

	return func() error {
		err := sup.stop()
		// the core's deliveries are drained once stop returns, so the pool holds nothing
		// the engine still needs.
		sender.CloseIdleConnections()

		return err
	}, sup.Running, nil
}

// serverlessSupervisor keeps the core running until stopped. Running reports whether the core
// is up at the moment, for a readiness probe to consult.
type serverlessSupervisor struct {
	cancel  context.CancelFunc
	done    chan struct{}
	running atomic.Bool
}

// superviseServerless starts run and restarts it with backoff whenever it returns while ctx
// is still live. The delay resets after a run that lasted longer than the maximum backoff,
// so a long-lived core that hits a transient fault comes back quickly.
func superviseServerless(parent context.Context, l *zerolog.Logger, run func(context.Context) error) *serverlessSupervisor {
	ctx, cancel := context.WithCancel(parent)
	sup := &serverlessSupervisor{cancel: cancel, done: make(chan struct{})}

	go func() {
		defer close(sup.done)

		backoff := serverlessRestartBackoff

		for {
			started := time.Now()
			sup.running.Store(true)
			err := run(ctx)
			sup.running.Store(false)

			if ctx.Err() != nil {
				if err != nil && !errors.Is(err, context.Canceled) {
					l.Error().Err(err).Msg("serverless operator stopped with an error during shutdown")
				}

				return
			}

			if time.Since(started) > serverlessRestartBackoffMax {
				backoff = serverlessRestartBackoff
			}

			l.Error().Err(err).Dur("restart_in", backoff).Msg("serverless operator stopped unexpectedly; restarting")

			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}

			backoff = min(backoff*2, serverlessRestartBackoffMax)
		}
	}()

	return sup
}

// Running reports whether the core is up.
func (s *serverlessSupervisor) Running() bool {
	return s.running.Load()
}

// stop ends the core and waits for it to finish.
func (s *serverlessSupervisor) stop() error {
	s.cancel()
	<-s.done

	return nil
}

// serverlessWorkerName is "serverless-<dispatcher id>": one worker per tenant per engine
// replica, whatever the number of units, told apart by the replica that runs it.
func serverlessWorkerName(dispatcherId uuid.UUID) string {
	return fmt.Sprintf("serverless-%s", dispatcherId)
}

// serverlessConfigFromServer maps the engine's SERVER_SERVERLESS_OPERATOR_* settings onto the
// core's Config. The health server is disabled: the engine serves its own health and metrics
// endpoints, and the core's metrics are registered on the default Prometheus registry either
// way. Zero values fall through to the core's defaults.
func serverlessConfigFromServer(cf server.ServerlessOperatorConfigFile) serverlessoperator.Config {
	return serverlessoperator.Config{
		OperatorName:                 cf.OperatorName,
		LinkName:                     serverlessLinkName,
		DefaultSlots:                 cf.DefaultSlots,
		DurableSlots:                 cf.DurableSlots,
		LeaseTTL:                     cf.LeaseTTL,
		HeartbeatInterval:            cf.HeartbeatInterval,
		RebalanceInterval:            cf.RebalanceInterval,
		ShedHysteresis:               cf.ShedHysteresis,
		DrainTimeout:                 cf.DrainTimeout,
		RoutingRefreshInterval:       cf.RoutingRefreshInterval,
		HealthcheckTimeout:           cf.HealthcheckTimeout,
		HealthcheckConcurrency:       cf.HealthcheckConcurrency,
		HealthcheckTenantConcurrency: cf.HealthcheckTenantConcurrency,
		HealthcheckApplyTimeout:      cf.HealthcheckApplyTimeout,
		MaxWorkflowsPerEndpoint:      cf.MaxWorkflowsPerEndpoint,
		MaxActionsPerEndpoint:        cf.MaxActionsPerEndpoint,
		MaintenanceConcurrency:       cf.MaintenanceConcurrency,
		LeaseMaxClaimPerTick:         cf.LeaseMaxClaimPerTick,
		WSMaxFrameBytes:              cf.WSMaxFrameBytes,
		WSPingInterval:               cf.WSPingInterval,
		WSMaxUpgradeHeaderBytes:      cf.WSMaxUpgradeHeaderBytes,
		WSMaxQueuedBytes:             cf.WSMaxQueuedBytes,
		HealthPort:                   0,
	}
}
