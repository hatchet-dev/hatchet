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
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

const (
	heartbeatTimeout = time.Second * 5
)

type Partition struct {
	controllerPartitionId string
	workerPartitionId     string
	schedulerPartitionId  string

	controllerCron gocron.Scheduler
	workerCron     gocron.Scheduler
	schedulerCron  gocron.Scheduler

	repo repository.TenantEngineRepository
	l    *zerolog.Logger

	controllerMu sync.Mutex
	workerMu     sync.Mutex
	schedulerMu  sync.Mutex
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

	s3, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, err
	}

	return &Partition{
		repo:           repo,
		l:              l,
		controllerCron: s1,
		workerCron:     s2,
		schedulerCron:  s3,
	}, nil
}

func (p *Partition) GetControllerPartitionId() string {
	return p.controllerPartitionId
}

func (p *Partition) GetWorkerPartitionId() string {
	return p.workerPartitionId
}

func (p *Partition) GetSchedulerPartitionId() string {
	return p.schedulerPartitionId
}

func (p *Partition) Shutdown() error {
	err := p.controllerCron.Shutdown()

	if err != nil {
		return fmt.Errorf("could not shutdown controller cron: %w", err)
	}

	err = p.workerCron.Shutdown()

	if err != nil {
		return fmt.Errorf("could not shutdown worker cron: %w", err)
	}

	err = p.schedulerCron.Shutdown()

	if err != nil {
		return fmt.Errorf("could not shutdown scheduler cron: %w", err)
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
	_, err = p.controllerCron.NewJob(
		gocron.DurationJob(time.Second*20),
		gocron.NewTask(
			p.runControllerPartitionHeartbeat(ctx), // nolint: errcheck
		),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create controller partition heartbeat job: %w", err)
	}

	// rebalance partitions 10 seconds after startup
	_, err = p.controllerCron.NewJob(
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

	_, err = p.controllerCron.NewJob(
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

	p.controllerCron.Start()

	return cleanup, nil
}

func (p *Partition) ListTenantsForController(ctx context.Context) ([]*dbsqlc.Tenant, error) {
	return p.repo.ListTenantsByControllerPartition(ctx, p.GetControllerPartitionId())
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

func (p *Partition) StartSchedulerPartition(ctx context.Context) (func() error, error) {
	partitionId, err := p.repo.CreateSchedulerPartition(ctx)

	if err != nil {
		return nil, err
	}

	cleanup := func() error {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = p.repo.DeleteSchedulerPartition(deleteCtx, p.GetSchedulerPartitionId())

		if err != nil {
			return fmt.Errorf("could not delete scheduler partition: %w", err)
		}

		return p.repo.RebalanceAllSchedulerPartitions(deleteCtx)
	}

	p.schedulerPartitionId = partitionId

	// start the schedules
	_, err = p.schedulerCron.NewJob(
		gocron.DurationJob(time.Second*20),
		gocron.NewTask(
			p.runSchedulerPartitionHeartbeat(ctx), // nolint: errcheck
		),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler partition heartbeat job: %w", err)
	}

	// rebalance partitions 10 seconds after startup
	_, err = p.schedulerCron.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(time.Now().Add(time.Second*10)),
		),
		gocron.NewTask(
			func() {
				rebalanceAllSchedulerPartitions(ctx, p.l, p.repo) // nolint: errcheck
			},
		),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create rebalance all scheduler partitions job: %w", err)
	}

	_, err = p.schedulerCron.NewJob(
		gocron.DurationJob(time.Minute*1),
		gocron.NewTask(
			func() {
				rebalanceInactiveSchedulerPartitions(ctx, p.l, p.repo) // nolint: errcheck
			},
		),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create rebalance inactive scheduler partitions job: %w", err)
	}

	p.schedulerCron.Start()

	return cleanup, nil
}

func (p *Partition) runSchedulerPartitionHeartbeat(ctx context.Context) func() {
	return func() {
		if !p.schedulerMu.TryLock() {
			p.l.Warn().Msg("could not acquire lock on scheduler partition")
			return
		}

		defer p.schedulerMu.Unlock()

		ctx, cancel := context.WithTimeout(ctx, heartbeatTimeout)
		defer cancel()

		ctx, span := telemetry.NewSpan(ctx, "run-partition-heartbeat")
		defer span.End()

		p.l.Debug().Msg("running scheduler partition heartbeat")

		partitionId, err := p.repo.UpdateSchedulerPartitionHeartbeat(ctx, p.GetSchedulerPartitionId())

		if err != nil {
			p.l.Err(err).Msg("could not heartbeat partition")
			return
		}

		if partitionId != p.GetSchedulerPartitionId() {
			p.schedulerPartitionId = partitionId
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
	_, err = p.workerCron.NewJob(
		gocron.DurationJob(time.Second*20),
		gocron.NewTask(
			p.runTenantWorkerPartitionHeartbeat(ctx), // nolint: errcheck
		),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create controller partition heartbeat job: %w", err)
	}

	// rebalance partitions 30 seconds after startup
	_, err = p.workerCron.NewJob(
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

	_, err = p.workerCron.NewJob(
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

	p.workerCron.Start()

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

func rebalanceAllSchedulerPartitions(ctx context.Context, l *zerolog.Logger, r repository.TenantEngineRepository) error {
	err := r.RebalanceAllSchedulerPartitions(ctx)

	if err != nil {
		l.Err(err).Msg("could not rebalance scheduler partitions")
	}

	return err
}

func rebalanceInactiveSchedulerPartitions(ctx context.Context, l *zerolog.Logger, r repository.TenantEngineRepository) error {
	err := r.RebalanceInactiveSchedulerPartitions(ctx)

	if err != nil {
		l.Err(err).Msg("could not rebalance inactive scheduler partitions")
	}

	return err
}
