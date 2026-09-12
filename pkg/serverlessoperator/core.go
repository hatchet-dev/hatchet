// Package serverlessoperator is the serverless operator core: it owns (tenant, shard) lease
// units, polls their endpoints' healthchecks, registers their workflows with the engine and
// delivers assigned tasks to the endpoints over signed HTTP or, for durable tasks, over a
// websocket relay.
//
// The core reaches the engine through the operator host contract (pkg/operator.Host,
// Session and DurableChannel): one session per served tenant, opened by name and kind with
// the tenant's action union, its handler the registration that routes assigned actions to
// endpoints. The operator is a GRPC contract operator that leases itself in both modes
// (kind GRPC, leasing manager SELF): its rows are kept alive by the sessions it holds, never by the
// engine's claimer. The same core runs in either host as a deployment choice. Out of
// process, hatchet-serverless-operator gives it pkg/operator/hostgrpc, which speaks
// OperatorService with a per-tenant token; inside the engine, cmd/hatchet-engine gives it
// the dispatcher's in-process host, which upserts the rows itself and delivers by direct
// call. Nothing in the core distinguishes the two beyond Deps.Host.
package serverlessoperator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/lease"
)

// Deps is everything Run needs. The binary and the in-engine mode build it differently: the
// host and the worker name are hosting facts the wiring decides.
type Deps struct {
	Repo repository.ServerlessRepository

	// Host opens the engine session of each served tenant.
	Host operator.Host

	// WorkerName names the worker rows the sessions back; empty means the operator name.
	// Only a host that can name workers accepts it (the in-process host does, the gRPC host
	// refuses), so it is set by the engine wiring only.
	WorkerName string

	Encryption encryption.EncryptionService
	Sender     RequestSender
	Logger     *zerolog.Logger
	Version    string
	Hostname   string
	Config     Config
	ProcessId  uuid.UUID
}

func (d Deps) validate() error {
	switch {
	case d.Repo == nil:
		return errors.New("serverless operator: repository is required")
	case d.Host == nil:
		return errors.New("serverless operator: operator host is required")
	case d.Sender == nil:
		return errors.New("serverless operator: request sender is required")
	case d.Logger == nil:
		return errors.New("serverless operator: logger is required")
	case d.ProcessId == uuid.Nil:
		return errors.New("serverless operator: process id is required")
	}

	return nil
}

// shutdownGrace is added to DrainTimeout to bound the whole shutdown sequence.
const shutdownGrace = 10 * time.Second

// Run starts the leaser, the maintenance loop and the health server and blocks until ctx is
// done or a loop fails, then shuts down gracefully: release every lease, drain deliveries
// up to DrainTimeout, close registrations, delete the process row.
func Run(ctx context.Context, deps Deps) error {
	if err := deps.validate(); err != nil {
		return err
	}

	cfg := deps.Config.withDefaults()
	m := newMetrics(cfg.LinkName)
	l := deps.Logger.With().Str("process_id", deps.ProcessId.String()).Logger()
	deps.Logger = &l

	r := newRunner(deps, cfg, m)

	leaser := lease.New(deps.Repo, r, lease.Config{
		ProcessId:         deps.ProcessId,
		Hostname:          deps.Hostname,
		Version:           deps.Version,
		TTL:               cfg.LeaseTTL,
		HeartbeatInterval: cfg.HeartbeatInterval,
		RebalanceInterval: cfg.RebalanceInterval,
		SweepInterval:     processSweepInterval,
		SweepCutoff:       processExpiryCutoff,
		ShedHysteresis:    cfg.ShedHysteresis,
		MaxClaimPerTick:   cfg.LeaseMaxClaimPerTick,
	}, &l, lease.Hooks{
		Claimed:    m.claimed,
		Shed:       m.shed,
		Rebalanced: m.rebalanced,
		Owned:      m.setOwned,
	})

	var health *healthServer

	if cfg.HealthPort > 0 {
		health = newHealthServer(cfg.HealthPort, &l)
	}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return leaser.Run(gctx)
	})

	g.Go(func() error {
		r.maintain(gctx)
		return nil
	})

	if health != nil {
		g.Go(func() error {
			return health.serve()
		})

		g.Go(func() error {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-gctx.Done():
					return nil
				case <-ticker.C:
					if leaser.Ready() {
						health.setReady()
						return nil
					}
				}
			}
		})
	}

	l.Info().Str("version", deps.Version).Str("link", cfg.LinkName).Msg("serverless operator started")

	<-gctx.Done()

	runErr := context.Cause(gctx)

	if errors.Is(runErr, context.Canceled) {
		runErr = nil
	}

	l.Info().Msg("serverless operator shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.DrainTimeout+shutdownGrace)
	defer cancel()

	if health != nil {
		health.shutdown(shutdownCtx)
	}

	_ = g.Wait()

	if _, err := leaser.Release(shutdownCtx); err != nil {
		l.Error().Err(err).Msg("could not release serverless leases")
	}

	r.Shutdown()

	if err := leaser.DeleteProcess(shutdownCtx); err != nil {
		l.Error().Err(err).Msg("could not delete serverless process row")
	}

	l.Info().Msg("serverless operator stopped")

	if runErr != nil {
		return fmt.Errorf("serverless operator: %w", runErr)
	}

	return nil
}
