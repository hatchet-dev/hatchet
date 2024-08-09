package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
)

type partitioner struct {
	s    gocron.Scheduler
	repo repository.TenantEngineRepository
	l    *zerolog.Logger
}

func newPartitioner(repo repository.TenantEngineRepository, l *zerolog.Logger) (partitioner, error) {
	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return partitioner{}, err
	}

	return partitioner{s: s, repo: repo, l: l}, nil
}

func (p *partitioner) withControllers(ctx context.Context) (*Teardown, string, error) {
	partitionId := uuid.New().String()

	err := p.repo.CreateControllerPartition(ctx, partitionId)

	if err != nil {
		return nil, "", fmt.Errorf("could not create engine partition: %w", err)
	}

	// rebalance partitions 10 seconds after startup
	_, err = p.s.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(time.Now().Add(time.Second*10)),
		),
		gocron.NewTask(
			func() {
				rebalanceAllControllerPartitions(ctx, p.l, p.repo) // nolint: errcheck
			},
		),
	)

	if err != nil {
		return nil, "", fmt.Errorf("could not create rebalance all controller partitions job: %w", err)
	}

	_, err = p.s.NewJob(
		gocron.DurationJob(time.Minute*1),
		gocron.NewTask(
			func() {
				rebalanceInactiveControllerPartitions(ctx, p.l, p.repo) // nolint: errcheck
			},
		),
	)

	if err != nil {
		return nil, "", fmt.Errorf("could not create rebalance inactive controller partitions job: %w", err)
	}

	return &Teardown{
		Name: "partition teardown",
		Fn: func() error {
			deleteCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := p.repo.DeleteControllerPartition(deleteCtx, partitionId)

			if err != nil {
				return fmt.Errorf("could not delete controller partition: %w", err)
			}

			return p.repo.RebalanceAllControllerPartitions(deleteCtx)
		},
	}, partitionId, nil
}

func (p *partitioner) withTenantWorkers(ctx context.Context) (*Teardown, string, error) {
	partitionId := uuid.New().String()

	err := p.repo.CreateTenantWorkerPartition(ctx, partitionId)

	if err != nil {
		return nil, "", fmt.Errorf("could not create engine partition: %w", err)
	}

	// rebalance partitions 30 seconds after startup
	_, err = p.s.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(time.Now().Add(time.Second*30)),
		),
		gocron.NewTask(
			func() {
				rebalanceAllTenantWorkerPartitions(ctx, p.l, p.repo) // nolint: errcheck
			},
		),
	)

	if err != nil {
		return nil, "", fmt.Errorf("could not create rebalance all tenant worker partitions job: %w", err)
	}

	_, err = p.s.NewJob(
		gocron.DurationJob(time.Minute*1),
		gocron.NewTask(
			func() {
				rebalanceInactiveTenantWorkerPartitions(ctx, p.l, p.repo) // nolint: errcheck
			},
		),
	)

	if err != nil {
		return nil, "", fmt.Errorf("could not create rebalance inactive tenant worker partitions job: %w", err)
	}

	return &Teardown{
		Name: "partition teardown",
		Fn: func() error {
			deleteCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := p.repo.DeleteTenantWorkerPartition(deleteCtx, partitionId)

			if err != nil {
				return fmt.Errorf("could not delete worker partition: %w", err)
			}

			return p.repo.RebalanceAllTenantWorkerPartitions(deleteCtx)
		},
	}, partitionId, nil
}

func (p *partitioner) start() {
	p.s.Start()
}

func (p *partitioner) shutdown() error {
	return p.s.Shutdown()
}

func rebalanceAllControllerPartitions(ctx context.Context, l *zerolog.Logger, r repository.TenantEngineRepository) error {
	err := r.RebalanceAllControllerPartitions(ctx)

	if err != nil {
		l.Err(err).Msg("could not rebalance controller partitions")
	}

	return err
}

func rebalanceAllTenantWorkerPartitions(ctx context.Context, l *zerolog.Logger, r repository.TenantEngineRepository) error {
	err := r.RebalanceAllTenantWorkerPartitions(ctx)

	if err != nil {
		l.Err(err).Msg("could not rebalance tenant worker partitions")
	}

	return err
}

func rebalanceInactiveControllerPartitions(ctx context.Context, l *zerolog.Logger, r repository.TenantEngineRepository) error {
	err := r.RebalanceInactiveControllerPartitions(ctx)

	if err != nil {
		l.Err(err).Msg("could not rebalance inactive controller partitions")
	}

	return err
}

func rebalanceInactiveTenantWorkerPartitions(ctx context.Context, l *zerolog.Logger, r repository.TenantEngineRepository) error {
	err := r.RebalanceInactiveTenantWorkerPartitions(ctx)

	if err != nil {
		l.Err(err).Msg("could not rebalance inactive tenant worker partitions")
	}

	return err
}
