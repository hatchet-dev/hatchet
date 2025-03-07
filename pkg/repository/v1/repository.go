package v1

import (
	"time"

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
	Logs() LogLineRepository
}

type repositoryImpl struct {
	triggers  TriggerRepository
	tasks     TaskRepository
	scheduler SchedulerRepository
	matches   MatchRepository
	olap      OLAPRepository
	logs      LogLineRepository
}

func NewRepository(pool *pgxpool.Pool, l *zerolog.Logger, taskRetentionPeriod, olapRetentionPeriod time.Duration) (Repository, func() error) {
	v := validator.NewDefaultValidator()

	shared, cleanupShared := newSharedRepository(pool, v, l)

	matchRepo, err := newMatchRepository(shared)

	if err != nil {
		l.Fatal().Err(err).Msg("cannot create match repository")
	}

	impl := &repositoryImpl{
		triggers:  newTriggerRepository(shared),
		tasks:     newTaskRepository(shared, taskRetentionPeriod),
		scheduler: newSchedulerRepository(shared),
		matches:   matchRepo,
		olap:      newOLAPRepository(shared, olapRetentionPeriod),
		logs:      newLogLineRepository(shared),
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

func (r *repositoryImpl) Logs() LogLineRepository {
	return r.logs
}
