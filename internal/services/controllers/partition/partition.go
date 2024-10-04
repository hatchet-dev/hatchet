package partition

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

const (
	heartbeatTimeout = time.Second * 5
)

type Partition struct {
	controllerPartitionId string
	workerPartitionId     string
	controllerScheduler   gocron.Scheduler
	workerScheduler       gocron.Scheduler
	repo                  repository.TenantEngineRepository
	l                     *zerolog.Logger

	controllerMu sync.Mutex
	workerMu     sync.Mutex
}

func NewPartition(l *zerolog.Logger, repo repository.TenantEngineRepository) (*Partition, error) {
	s1, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, err
	}

	s2, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, err
	}

	return &Partition{
		repo:                repo,
		l:                   l,
		controllerScheduler: s1,
		workerScheduler:     s2,
	}, nil
}

func (p *Partition) GetControllerPartitionId() string {
	return p.controllerPartitionId
}

func (p *Partition) GetWorkerPartitionId() string {
	return p.workerPartitionId
}

func (p *Partition) Shutdown() error {
	err := p.controllerScheduler.Shutdown()

	if err != nil {
		return fmt.Errorf("could not shutdown controller scheduler: %w", err)
	}

	err = p.workerScheduler.Shutdown()

	if err != nil {
		return fmt.Errorf("could not shutdown worker scheduler: %w", err)
	}

	// wait for heartbeat timeout duration
	time.Sleep(heartbeatTimeout)

	return nil
}

func (p *Partition) StartControllerPartition(ctx context.Context) (func() error, error) {
	partitionId, err := p.repo.CreateControllerPartition(ctx)

	if err != nil {
		return nil, err
	}

	cleanup := func() error {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = p.repo.DeleteControllerPartition(deleteCtx, p.GetControllerPartitionId())

		if err != nil {
			return fmt.Errorf("could not delete controller partition: %w", err)
		}

		return p.repo.RebalanceAllControllerPartitions(deleteCtx)
	}

	p.controllerPartitionId = partitionId

	// start the schedules
	_, err = p.controllerScheduler.NewJob(
		gocron.DurationJob(time.Second*20),
		gocron.NewTask(
			p.runControllerPartitionHeartbeat(ctx), // nolint: errcheck
		),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create controller partition heartbeat job: %w", err)
	}

	// rebalance partitions 10 seconds after startup
	_, err = p.controllerScheduler.NewJob(
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
		return nil, fmt.Errorf("could not create rebalance all controller partitions job: %w", err)
	}

	_, err = p.controllerScheduler.NewJob(
		gocron.DurationJob(time.Minute*1),
		gocron.NewTask(
			func() {
				rebalanceInactiveControllerPartitions(ctx, p.l, p.repo) // nolint: errcheck
			},
		),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create rebalance inactive controller partitions job: %w", err)
	}

	p.controllerScheduler.Start()

	return cleanup, nil
}

func (p *Partition) runControllerPartitionHeartbeat(ctx context.Context) func() {
	return func() {
		if !p.controllerMu.TryLock() {
			p.l.Warn().Msg("could not acquire lock on controller partition")
			return
		}

		defer p.controllerMu.Unlock()

		ctx, cancel := context.WithTimeout(ctx, heartbeatTimeout)
		defer cancel()

		ctx, span := telemetry.NewSpan(ctx, "run-partition-heartbeat")
		defer span.End()

		p.l.Debug().Msg("running controller partition heartbeat")

		partitionId, err := p.repo.UpdateControllerPartitionHeartbeat(ctx, p.GetControllerPartitionId())

		if err != nil {
			p.l.Err(err).Msg("could not heartbeat partition")
			return
		}

		if partitionId != p.GetControllerPartitionId() {
			p.controllerPartitionId = partitionId
		}
	}
}

func (p *Partition) StartTenantWorkerPartition(ctx context.Context) (func() error, error) {
	partitionId, err := p.repo.CreateTenantWorkerPartition(ctx)

	if err != nil {
		return nil, err
	}

	p.workerPartitionId = partitionId

	cleanup := func() error {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = p.repo.DeleteTenantWorkerPartition(deleteCtx, p.GetWorkerPartitionId())

		if err != nil {
			return fmt.Errorf("could not delete worker partition: %w", err)
		}

		return p.repo.RebalanceAllTenantWorkerPartitions(deleteCtx)
	}

	// start the schedules
	_, err = p.workerScheduler.NewJob(
		gocron.DurationJob(time.Second*20),
		gocron.NewTask(
			p.runTenantWorkerPartitionHeartbeat(ctx), // nolint: errcheck
		),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create controller partition heartbeat job: %w", err)
	}

	// rebalance partitions 30 seconds after startup
	_, err = p.workerScheduler.NewJob(
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
		return nil, fmt.Errorf("could not create rebalance all tenant worker partitions job: %w", err)
	}

	_, err = p.workerScheduler.NewJob(
		gocron.DurationJob(time.Minute*1),
		gocron.NewTask(
			func() {
				rebalanceInactiveTenantWorkerPartitions(ctx, p.l, p.repo) // nolint: errcheck
			},
		),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create rebalance inactive tenant worker partitions job: %w", err)
	}

	p.workerScheduler.Start()

	return cleanup, nil
}

func (p *Partition) runTenantWorkerPartitionHeartbeat(ctx context.Context) func() {
	return func() {
		if !p.workerMu.TryLock() {
			p.l.Warn().Msg("could not acquire lock on worker partition")
			return
		}

		defer p.workerMu.Unlock()

		ctx, cancel := context.WithTimeout(ctx, heartbeatTimeout)
		defer cancel()

		ctx, span := telemetry.NewSpan(ctx, "run-partition-heartbeat")
		defer span.End()

		p.l.Debug().Msg("running worker partition heartbeat")

		partitionId, err := p.repo.UpdateWorkerPartitionHeartbeat(ctx, p.GetWorkerPartitionId())

		if err != nil {
			p.l.Err(err).Msg("could not heartbeat partition")
			return
		}

		if partitionId != p.GetWorkerPartitionId() {
			p.workerPartitionId = partitionId
		}
	}
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
