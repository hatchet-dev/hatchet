package postgres

import (
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type apiRepository struct {
	tenant         repository.TenantAPIRepository
	tenantAlerting repository.TenantAlertingRepository
	tenantInvite   repository.TenantInviteRepository
	workflow       repository.WorkflowAPIRepository
	workflowRun    repository.WorkflowRunAPIRepository
	stepRun        repository.StepRunAPIRepository
	step           repository.StepRepository
	slack          repository.SlackRepository
	sns            repository.SNSRepository
	worker         repository.WorkerAPIRepository
	userSession    repository.UserSessionRepository
	user           repository.UserRepository
	webhookWorker  repository.WebhookWorkerRepository
}

type PostgresRepositoryOpt func(*PostgresRepositoryOpts)

type PostgresRepositoryOpts struct {
	v       validator.Validator
	l       *zerolog.Logger
	cache   cache.Cacheable
	metered *metered.Metered
}

func defaultPostgresRepositoryOpts() *PostgresRepositoryOpts {
	return &PostgresRepositoryOpts{
		v: validator.NewDefaultValidator(),
	}
}

func WithValidator(v validator.Validator) PostgresRepositoryOpt {
	return func(opts *PostgresRepositoryOpts) {
		opts.v = v
	}
}

func WithLogger(l *zerolog.Logger) PostgresRepositoryOpt {
	return func(opts *PostgresRepositoryOpts) {
		opts.l = l
	}
}

func WithCache(cache cache.Cacheable) PostgresRepositoryOpt {
	return func(opts *PostgresRepositoryOpts) {
		opts.cache = cache
	}
}

func WithMetered(metered *metered.Metered) PostgresRepositoryOpt {
	return func(opts *PostgresRepositoryOpts) {
		opts.metered = metered
	}
}

func NewAPIRepository(pool *pgxpool.Pool, cf *server.ConfigFileRuntime, fs ...PostgresRepositoryOpt) (repository.APIRepository, func() error, error) {
	opts := defaultPostgresRepositoryOpts()

	for _, f := range fs {
		f(opts)
	}

	newLogger := opts.l.With().Str("service", "database").Logger()
	opts.l = &newLogger

	if opts.cache == nil {
		opts.cache = cache.New(1 * time.Millisecond)
	}

	shared, cleanup, err := newSharedRepository(pool, opts.v, opts.l, cf)

	if err != nil {
		return nil, nil, err
	}

	defaultEngineVersion := dbsqlc.TenantMajorEngineVersionV0

	switch strings.ToLower(cf.DefaultEngineVersion) {
	case "v0":
		defaultEngineVersion = dbsqlc.TenantMajorEngineVersionV0
	case "v1":
		defaultEngineVersion = dbsqlc.TenantMajorEngineVersionV1
	}

	return &apiRepository{
		tenant:         NewTenantAPIRepository(shared, opts.cache, defaultEngineVersion),
		tenantAlerting: NewTenantAlertingRepository(shared, opts.cache),
		tenantInvite:   NewTenantInviteRepository(shared),
		workflow:       NewWorkflowRepository(shared),
		workflowRun:    NewWorkflowRunRepository(shared, opts.metered, cf),
		stepRun:        NewStepRunAPIRepository(shared),
		step:           NewStepRepository(pool, opts.v, opts.l),
		slack:          NewSlackRepository(shared),
		sns:            NewSNSRepository(shared),
		worker:         NewWorkerAPIRepository(shared, opts.metered),
		userSession:    NewUserSessionRepository(shared),
		user:           NewUserRepository(shared),
		webhookWorker:  NewWebhookWorkerRepository(shared),
	}, cleanup, err
}

func (r *apiRepository) Tenant() repository.TenantAPIRepository {
	return r.tenant
}

func (r *apiRepository) TenantAlertingSettings() repository.TenantAlertingRepository {
	return r.tenantAlerting
}

func (r *apiRepository) TenantInvite() repository.TenantInviteRepository {
	return r.tenantInvite
}

func (r *apiRepository) Workflow() repository.WorkflowAPIRepository {
	return r.workflow
}

func (r *apiRepository) WorkflowRun() repository.WorkflowRunAPIRepository {
	return r.workflowRun
}

func (r *apiRepository) StepRun() repository.StepRunAPIRepository {
	return r.stepRun
}

func (r *apiRepository) Slack() repository.SlackRepository {
	return r.slack
}

func (r *apiRepository) SNS() repository.SNSRepository {
	return r.sns
}

func (r *apiRepository) Step() repository.StepRepository {
	return r.step
}

func (r *apiRepository) Worker() repository.WorkerAPIRepository {
	return r.worker
}

func (r *apiRepository) UserSession() repository.UserSessionRepository {
	return r.userSession
}

func (r *apiRepository) User() repository.UserRepository {
	return r.user
}

func (r *apiRepository) WebhookWorker() repository.WebhookWorkerRepository {
	return r.webhookWorker
}

type engineRepository struct {
	step           repository.StepRepository
	stepRun        repository.StepRunEngineRepository
	tenant         repository.TenantEngineRepository
	tenantAlerting repository.TenantAlertingRepository
	ticker         repository.TickerEngineRepository
	worker         repository.WorkerEngineRepository
	workflow       repository.WorkflowEngineRepository
	workflowRun    repository.WorkflowRunEngineRepository
	streamEvent    repository.StreamEventsEngineRepository
	webhookWorker  repository.WebhookWorkerEngineRepository
}

func (r *engineRepository) StepRun() repository.StepRunEngineRepository {
	return r.stepRun
}

func (r *engineRepository) Step() repository.StepRepository {
	return r.step
}

func (r *engineRepository) Tenant() repository.TenantEngineRepository {
	return r.tenant
}

func (r *engineRepository) TenantAlertingSettings() repository.TenantAlertingRepository {
	return r.tenantAlerting
}

func (r *engineRepository) Ticker() repository.TickerEngineRepository {
	return r.ticker
}

func (r *engineRepository) Worker() repository.WorkerEngineRepository {
	return r.worker
}

func (r *engineRepository) Workflow() repository.WorkflowEngineRepository {
	return r.workflow
}

func (r *engineRepository) WorkflowRun() repository.WorkflowRunEngineRepository {
	return r.workflowRun
}

func (r *engineRepository) StreamEvent() repository.StreamEventsEngineRepository {
	return r.streamEvent
}

func (r *engineRepository) WebhookWorker() repository.WebhookWorkerEngineRepository {
	return r.webhookWorker
}

func NewEngineRepository(pool *pgxpool.Pool, cf *server.ConfigFileRuntime, fs ...PostgresRepositoryOpt) (func() error, repository.EngineRepository, error) {
	opts := defaultPostgresRepositoryOpts()

	for _, f := range fs {
		f(opts)
	}

	buffer.SetDefaults(cf.FlushPeriodMilliseconds, cf.FlushItemsThreshold)

	newLogger := opts.l.With().Str("service", "database").Logger()
	opts.l = &newLogger

	if opts.cache == nil {
		opts.cache = cache.New(1 * time.Millisecond)
	}

	rlCache := cache.New(5 * time.Minute)
	queueCache := cache.New(5 * time.Minute)

	shared, cleanup, err := newSharedRepository(pool, opts.v, opts.l, cf)

	if err != nil {
		return nil, nil, err
	}

	return func() error {
			rlCache.Stop()
			queueCache.Stop()

			return cleanup()
		}, &engineRepository{
			stepRun:        NewStepRunEngineRepository(shared, cf, rlCache, queueCache),
			step:           NewStepRepository(pool, opts.v, opts.l),
			tenant:         NewTenantEngineRepository(pool, opts.v, opts.l, opts.cache),
			tenantAlerting: NewTenantAlertingRepository(shared, opts.cache),
			ticker:         NewTickerRepository(pool, opts.v, opts.l),
			worker:         NewWorkerEngineRepository(pool, opts.v, opts.l, opts.metered),
			workflow:       NewWorkflowEngineRepository(shared, opts.metered, opts.cache),
			workflowRun:    NewWorkflowRunEngineRepository(shared, opts.metered, cf),
			streamEvent:    NewStreamEventsEngineRepository(pool, opts.v, opts.l),
			webhookWorker:  NewWebhookWorkerEngineRepository(pool, opts.v, opts.l),
		},
		err
}

type entitlementRepository struct {
	tenantLimit repository.TenantLimitRepository
}

func (r *entitlementRepository) TenantLimit() repository.TenantLimitRepository {
	return r.tenantLimit
}

func NewEntitlementRepository(pool *pgxpool.Pool, s *server.ConfigFileRuntime, fs ...PostgresRepositoryOpt) repository.EntitlementsRepository {
	opts := defaultPostgresRepositoryOpts()

	for _, f := range fs {
		f(opts)
	}

	newLogger := opts.l.With().Str("service", "database").Logger()
	opts.l = &newLogger

	if opts.cache == nil {
		opts.cache = cache.New(1 * time.Millisecond)
	}

	return &entitlementRepository{
		tenantLimit: NewTenantLimitRepository(pool, opts.v, opts.l, s),
	}
}
