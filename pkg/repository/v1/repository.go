package v1

import (
	"github.com/hatchet-dev/hatchet/pkg/validator"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Repository interface {
	Triggers() TriggerRepository
	Tasks() TaskRepository
	Scheduler() SchedulerRepository
	Matches() MatchRepository
}

type repositoryImpl struct {
	triggers  TriggerRepository
	tasks     TaskRepository
	scheduler SchedulerRepository
	matches   MatchRepository
}

func NewRepository(pool *pgxpool.Pool, l *zerolog.Logger) Repository {
	v := validator.NewDefaultValidator()

	shared := newSharedRepository(pool, v, l)

	matchRepo, err := newMatchRepository(shared)

	if err != nil {
		l.Fatal().Err(err).Msg("cannot create match repository")
	}

	impl := &repositoryImpl{
		triggers:  newTriggerRepository(shared),
		tasks:     newTaskRepository(shared),
		scheduler: newSchedulerRepository(shared),
		matches:   matchRepo,
	}

	return impl
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
