package v1

import (
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Repository interface {
	Triggers() TriggerRepository
	Tasks() TaskRepository
	Scheduler() SchedulerRepository
	Matches() MatchRepository
	OLAP() OLAPRepository
	OverwriteOLAPRepository(o OLAPRepository)
	Logs() LogLineRepository
	OverwriteLogsRepository(l LogLineRepository)
	Workers() WorkerRepository
	Workflows() WorkflowRepository
	Ticker() TickerRepository
}

type repositoryImpl struct {
	triggers  TriggerRepository
	tasks     TaskRepository
	scheduler SchedulerRepository
	matches   MatchRepository
	olap      OLAPRepository
	logs      LogLineRepository
	workers   WorkerRepository
	workflows WorkflowRepository
	ticker    TickerRepository
}

func NewRepository(pool *pgxpool.Pool, l *zerolog.Logger, taskRetentionPeriod, olapRetentionPeriod time.Duration, maxInternalRetryCount int32, entitlements repository.EntitlementsRepository) (Repository, func() error) {
	v := validator.NewDefaultValidator()

	shared, cleanupShared := newSharedRepository(pool, v, l, entitlements)

	matchRepo, err := newMatchRepository(shared)

	if err != nil {
		l.Fatal().Err(err).Msg("cannot create match repository")
	}

	impl := &repositoryImpl{
		triggers:  newTriggerRepository(shared),
		tasks:     newTaskRepository(shared, taskRetentionPeriod, maxInternalRetryCount),
		scheduler: newSchedulerRepository(shared),
		matches:   matchRepo,
		olap:      newOLAPRepository(shared, olapRetentionPeriod),
		logs:      newLogLineRepository(shared),
		workers:   newWorkerRepository(shared),
		workflows: newWorkflowRepository(shared),
		ticker:    newTickerRepository(shared),
	}

	return impl, func() error {
		return cleanupShared()
	}
}

func (r *repositoryImpl) Triggers() TriggerRepository {
	return r.triggers
}

func (r *repositoryImpl) Tasks() TaskRepository {
	return r.tasks
}

func (r *repositoryImpl) Scheduler() SchedulerRepository {
	return r.scheduler
}

func (r *repositoryImpl) Matches() MatchRepository {
	return r.matches
}

func (r *repositoryImpl) OLAP() OLAPRepository {
	return r.olap
}

func (r *repositoryImpl) OverwriteOLAPRepository(o OLAPRepository) {
	r.olap = o
}

func (r *repositoryImpl) Logs() LogLineRepository {
	return r.logs
}

func (r *repositoryImpl) OverwriteLogsRepository(l LogLineRepository) {
	r.logs = l
}

func (r *repositoryImpl) Workers() WorkerRepository {
	return r.workers
}

func (r *repositoryImpl) Workflows() WorkflowRepository {
	return r.workflows
}

func (r *repositoryImpl) Ticker() TickerRepository {
	return r.ticker
}
