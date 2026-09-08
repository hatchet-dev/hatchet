// Package serverlessoperator is the serverless operator core: it owns (tenant, shard) lease
// units, polls their endpoints' healthchecks, registers their workflows with the engine
// through a link, and delivers assigned tasks to the endpoints over signed HTTP. The same
// core runs out of process (grpclink) and, in a later phase, inside the engine.
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
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/lease"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

// Deps is everything Run needs. The binary and the in-engine mode build it differently.
type Deps struct {
	Repo       repository.ServerlessRepository
	Link       link.Link
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
	case d.Link == nil:
		return errors.New("serverless operator: link is required")
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
