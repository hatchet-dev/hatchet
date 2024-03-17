package prisma

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/cache"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type prismaRepository struct {
	apiToken       repository.APITokenRepository
	event          repository.EventRepository
	log            repository.LogsRepository
	tenant         repository.TenantRepository
	tenantInvite   repository.TenantInviteRepository
	workflow       repository.WorkflowRepository
	workflowRun    repository.WorkflowRunRepository
	jobRun         repository.JobRunRepository
	stepRun        repository.StepRunRepository
	getGroupKeyRun repository.GetGroupKeyRunRepository
	github         repository.GithubRepository
	step           repository.StepRepository
	sns            repository.SNSRepository
	dispatcher     repository.DispatcherRepository
	worker         repository.WorkerRepository
	ticker         repository.TickerRepository
	userSession    repository.UserSessionRepository
	user           repository.UserRepository
	health         repository.HealthRepository
}

type PrismaRepositoryOpt func(*PrismaRepositoryOpts)

type PrismaRepositoryOpts struct {
	v     validator.Validator
	l     *zerolog.Logger
	cache cache.Cacheable
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

func WithLogger(l *zerolog.Logger) PrismaRepositoryOpt {
	return func(opts *PrismaRepositoryOpts) {
		opts.l = l
	}
}

func WithCache(cache cache.Cacheable) PrismaRepositoryOpt {
	return func(opts *PrismaRepositoryOpts) {
		opts.cache = cache
	}
}

func NewPrismaRepository(client *db.PrismaClient, pool *pgxpool.Pool, fs ...PrismaRepositoryOpt) repository.Repository {
	opts := defaultPrismaRepositoryOpts()

	for _, f := range fs {
		f(opts)
	}

	newLogger := opts.l.With().Str("service", "database").Logger()
	opts.l = &newLogger

	if opts.cache == nil {
		opts.cache = cache.New(1 * time.Millisecond)
	}

	return &prismaRepository{
		apiToken:       NewAPITokenRepository(client, opts.v, opts.cache),
		event:          NewEventRepository(client, pool, opts.v, opts.l),
		log:            NewLogRepository(client, pool, opts.v, opts.l),
		tenant:         NewTenantRepository(client, opts.v, opts.cache),
		tenantInvite:   NewTenantInviteRepository(client, opts.v),
		workflow:       NewWorkflowRepository(client, pool, opts.v, opts.l),
		workflowRun:    NewWorkflowRunRepository(client, pool, opts.v, opts.l),
		jobRun:         NewJobRunRepository(client, pool, opts.v, opts.l),
		stepRun:        NewStepRunRepository(client, pool, opts.v, opts.l),
		getGroupKeyRun: NewGetGroupKeyRunRepository(client, pool, opts.v, opts.l),
		github:         NewGithubRepository(client, opts.v),
		step:           NewStepRepository(client, opts.v),
		sns:            NewSNSRepository(client, opts.v),
		dispatcher:     NewDispatcherRepository(client, pool, opts.v, opts.l),
		worker:         NewWorkerRepository(client, pool, opts.v, opts.l),
		ticker:         NewTickerRepository(client, pool, opts.v, opts.l),
		userSession:    NewUserSessionRepository(client, opts.v),
		user:           NewUserRepository(client, opts.v),
		health:         NewHealthRepository(client, pool),
	}
}

func (r *prismaRepository) Health() repository.HealthRepository {
	return r.health
}

func (r *prismaRepository) APIToken() repository.APITokenRepository {
	return r.apiToken
}

func (r *prismaRepository) Event() repository.EventRepository {
	return r.event
}

func (r *prismaRepository) Log() repository.LogsRepository {
	return r.log
}

func (r *prismaRepository) Tenant() repository.TenantRepository {
	return r.tenant
}

func (r *prismaRepository) TenantInvite() repository.TenantInviteRepository {
	return r.tenantInvite
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

func (r *prismaRepository) SNS() repository.SNSRepository {
	return r.sns
}

func (r *prismaRepository) GetGroupKeyRun() repository.GetGroupKeyRunRepository {
	return r.getGroupKeyRun
}

func (r *prismaRepository) Github() repository.GithubRepository {
	return r.github
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
