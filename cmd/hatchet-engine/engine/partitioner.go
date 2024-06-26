package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository"
)

type partitioner struct {
	s    gocron.Scheduler
	repo repository.TenantEngineRepository
}

func newPartitioner(repo repository.TenantEngineRepository) (partitioner, error) {
	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return partitioner{}, err
	}

	return partitioner{s: s, repo: repo}, nil
}

func (p *partitioner) withControllers(ctx context.Context) (*Teardown, string, error) {
	partitionId := uuid.New().String()

	err := p.repo.CreateControllerPartition(ctx, partitionId)

	if err != nil {
		return nil, "", fmt.Errorf("could not create engine partition: %w", err)
	}

	// rebalance partitions on startup
	err = p.repo.RebalanceAllControllerPartitions(ctx)

	if err != nil {
		return nil, "", fmt.Errorf("could not rebalance engine partitions: %w", err)
	}

	_, err = p.s.NewJob(
		gocron.DurationJob(time.Minute*1),
		gocron.NewTask(
			func() {
				rebalanceControllerPartitions(ctx, p.repo) // nolint: errcheck
			},
		),
	)

	if err != nil {
		return nil, "", fmt.Errorf("could not create rebalance controller partitions job: %w", err)
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

	// rebalance partitions on startup
	err = p.repo.RebalanceAllTenantWorkerPartitions(ctx)

	if err != nil {
		return nil, "", fmt.Errorf("could not rebalance engine partitions: %w", err)
	}

	_, err = p.s.NewJob(
		gocron.DurationJob(time.Minute*1),
		gocron.NewTask(
			func() {
				rebalanceTenantWorkerPartitions(ctx, p.repo) // nolint: errcheck
			},
		),
	)

	if err != nil {
		return nil, "", fmt.Errorf("could not create rebalance tenant worker partitions job: %w", err)
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

func rebalanceControllerPartitions(ctx context.Context, r repository.TenantEngineRepository) error {
	return r.RebalanceInactiveControllerPartitions(ctx)
}

func rebalanceTenantWorkerPartitions(ctx context.Context, r repository.TenantEngineRepository) error {
	return r.RebalanceInactiveTenantWorkerPartitions(ctx)
}
