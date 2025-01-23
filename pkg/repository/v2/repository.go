package v2

import (
	"github.com/hatchet-dev/hatchet/pkg/validator"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Repository interface {
	Events() EventRepository
	Tasks() TaskRepository
	Scheduler() SchedulerRepository
}

type repositoryImpl struct {
	events    EventRepository
	tasks     TaskRepository
	scheduler SchedulerRepository
}

func NewRepository(pool *pgxpool.Pool, l *zerolog.Logger) Repository {
	v := validator.NewDefaultValidator()

	shared := newSharedRepository(pool, v, l)

	impl := &repositoryImpl{
		events:    newEventRepository(shared),
		tasks:     newTaskRepository(shared),
		scheduler: newSchedulerRepository(shared),
	}

	return impl
}

func (r *repositoryImpl) Events() EventRepository {
	return r.events
}

func (r *repositoryImpl) Tasks() TaskRepository {
	return r.tasks
}

func (r *repositoryImpl) Scheduler() SchedulerRepository {
	return r.scheduler
}
