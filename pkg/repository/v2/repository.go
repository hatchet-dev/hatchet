package v2

import (
	"github.com/hatchet-dev/hatchet/pkg/validator"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Repository interface {
	Triggers() TriggerRepository
	Tasks() TaskRepository
	Scheduler() SchedulerRepository
}

type repositoryImpl struct {
	triggers  TriggerRepository
	tasks     TaskRepository
	scheduler SchedulerRepository
}

func NewRepository(pool *pgxpool.Pool, l *zerolog.Logger) Repository {
	v := validator.NewDefaultValidator()

	shared := newSharedRepository(pool, v, l)

	impl := &repositoryImpl{
		triggers:  newTriggerRepository(shared),
		tasks:     newTaskRepository(shared),
		scheduler: newSchedulerRepository(shared),
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
