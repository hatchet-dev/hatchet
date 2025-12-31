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
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type apiRepository struct {
	tenant      repository.TenantAPIRepository
	workflow    repository.WorkflowAPIRepository
	workflowRun repository.WorkflowRunAPIRepository
	userSession repository.UserSessionRepository
	user        repository.UserRepository
}

type PostgresRepositoryOpt func(*PostgresRepositoryOpts)

type PostgresRepositoryOpts struct {
	v     validator.Validator
	l     *zerolog.Logger
	cache cache.Cacheable
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
		tenant:      NewTenantAPIRepository(shared, opts.cache, defaultEngineVersion),
		workflow:    NewWorkflowRepository(shared),
		workflowRun: NewWorkflowRunRepository(shared, cf),
		userSession: NewUserSessionRepository(shared),
		user:        NewUserRepository(shared),
	}, cleanup, err
}

func (r *apiRepository) Tenant() repository.TenantAPIRepository {
	return r.tenant
}

func (r *apiRepository) Workflow() repository.WorkflowAPIRepository {
	return r.workflow
}

func (r *apiRepository) WorkflowRun() repository.WorkflowRunAPIRepository {
	return r.workflowRun
}

func (r *apiRepository) UserSession() repository.UserSessionRepository {
	return r.userSession
}

func (r *apiRepository) User() repository.UserRepository {
	return r.user
}

type engineRepository struct {
	tenant      repository.TenantEngineRepository
	ticker      repository.TickerEngineRepository
	workflow    repository.WorkflowEngineRepository
	workflowRun repository.WorkflowRunEngineRepository
	streamEvent repository.StreamEventsEngineRepository
}

func (r *engineRepository) Tenant() repository.TenantEngineRepository {
	return r.tenant
}

func (r *engineRepository) Ticker() repository.TickerEngineRepository {
	return r.ticker
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
			tenant:      NewTenantEngineRepository(pool, opts.v, opts.l, opts.cache),
			ticker:      NewTickerRepository(pool, opts.v, opts.l),
			workflow:    NewWorkflowEngineRepository(shared, opts.cache),
			workflowRun: NewWorkflowRunEngineRepository(shared, cf),
			streamEvent: NewStreamEventsEngineRepository(pool, opts.v, opts.l),
		},
		err
}
