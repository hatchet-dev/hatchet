package prisma

import (
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/jackc/pgx/v5/pgxpool"
)

type prismaRepository struct {
	event       repository.EventRepository
	tenant      repository.TenantRepository
	workflow    repository.WorkflowRepository
	workflowRun repository.WorkflowRunRepository
	jobRun      repository.JobRunRepository
	stepRun     repository.StepRunRepository
	step        repository.StepRepository
	dispatcher  repository.DispatcherRepository
	worker      repository.WorkerRepository
	ticker      repository.TickerRepository
	userSession repository.UserSessionRepository
	user        repository.UserRepository
}

type PrismaRepositoryOpt func(*PrismaRepositoryOpts)

type PrismaRepositoryOpts struct {
	v validator.Validator
}

func defaultPrismaRepositoryOpts() *PrismaRepositoryOpts {
	return &PrismaRepositoryOpts{
		v: validator.NewDefaultValidator(),
	}
}

func WithValidator(v validator.Validator) PrismaRepositoryOpt {
	return func(opts *PrismaRepositoryOpts) {
		opts.v = v
	}
}

func NewPrismaRepository(client *db.PrismaClient, pool *pgxpool.Pool, fs ...PrismaRepositoryOpt) repository.Repository {
	opts := defaultPrismaRepositoryOpts()

	for _, f := range fs {
		f(opts)
	}

	return &prismaRepository{
		event:       NewEventRepository(client, pool, opts.v),
		tenant:      NewTenantRepository(client, opts.v),
		workflow:    NewWorkflowRepository(client, pool, opts.v),
		workflowRun: NewWorkflowRunRepository(client, pool, opts.v),
		jobRun:      NewJobRunRepository(client, opts.v),
		stepRun:     NewStepRunRepository(client, pool, opts.v),
		step:        NewStepRepository(client, opts.v),
		dispatcher:  NewDispatcherRepository(client, pool, opts.v),
		worker:      NewWorkerRepository(client, opts.v),
		ticker:      NewTickerRepository(client, pool, opts.v),
		userSession: NewUserSessionRepository(client, opts.v),
		user:        NewUserRepository(client, opts.v),
	}
}

func (r *prismaRepository) Event() repository.EventRepository {
	return r.event
}

func (r *prismaRepository) Tenant() repository.TenantRepository {
	return r.tenant
}

func (r *prismaRepository) Workflow() repository.WorkflowRepository {
	return r.workflow
}

func (r *prismaRepository) WorkflowRun() repository.WorkflowRunRepository {
	return r.workflowRun
}

func (r *prismaRepository) JobRun() repository.JobRunRepository {
	return r.jobRun
}

func (r *prismaRepository) StepRun() repository.StepRunRepository {
	return r.stepRun
}

func (r *prismaRepository) Step() repository.StepRepository {
	return r.step
}

func (r *prismaRepository) Dispatcher() repository.DispatcherRepository {
	return r.dispatcher
}

func (r *prismaRepository) Worker() repository.WorkerRepository {
	return r.worker
}

func (r *prismaRepository) Ticker() repository.TickerRepository {
	return r.ticker
}

func (r *prismaRepository) UserSession() repository.UserSessionRepository {
	return r.userSession
}

func (r *prismaRepository) User() repository.UserRepository {
	return r.user
}
