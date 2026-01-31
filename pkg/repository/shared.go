package repository

import (
	"context"
	"log"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/pkg/config/limits"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	celgo "github.com/google/cel-go/cel"
	"github.com/google/uuid"
)

// implements comparable for the lru cache
type taskExternalIdTenantIdTuple struct {
	externalId uuid.UUID
	tenantId   uuid.UUID
}

type sharedRepository struct {
	pool                      *pgxpool.Pool
	v                         validator.Validator
	l                         *zerolog.Logger
	queries                   *sqlcv1.Queries
	queueCache                *cache.Cache
	stepExpressionCache       *cache.Cache
	concurrencyStrategyCache  *cache.Cache
	tenantIdWorkflowNameCache *cache.Cache
	celParser                 *cel.CELParser
	env                       *celgo.Env
	taskLookupCache           *lru.Cache[taskExternalIdTenantIdTuple, *sqlcv1.FlattenExternalIdsRow]
	payloadStore              PayloadStoreRepository
	m                         TenantLimitRepository
}

func newSharedRepository(
	pool *pgxpool.Pool,
	v validator.Validator,
	l *zerolog.Logger,
	payloadStoreOpts PayloadStoreRepositoryOpts,
	c limits.LimitConfigFile,
	shouldEnforceLimits bool,
	enforceLimitsFunc func(ctx context.Context, tenantId string) (bool, error),
	cacheDuration time.Duration,
) (*sharedRepository, func() error) {
	queries := sqlcv1.New()
	queueCache := cache.New(5 * time.Minute)
	stepExpressionCache := cache.New(5 * time.Minute)
	concurrencyStrategyCache := cache.New(5 * time.Minute)
	tenantIdWorkflowNameCache := cache.New(5 * time.Minute)
	payloadStore := NewPayloadStoreRepository(pool, l, queries, payloadStoreOpts)

	celParser := cel.NewCELParser()

	env, err := celgo.NewEnv(
		celgo.Variable("input", celgo.MapType(celgo.StringType, celgo.DynType)),
		celgo.Variable("output", celgo.MapType(celgo.StringType, celgo.DynType)),
	)

	if err != nil {
		log.Fatalf("failed to create CEL environment: %v", err)
	}

	lookupCache, err := lru.New[taskExternalIdTenantIdTuple, *sqlcv1.FlattenExternalIdsRow](20000)

	if err != nil {
		log.Fatalf("failed to create LRU cache: %v", err)
	}

	s := &sharedRepository{
		pool:                      pool,
		v:                         v,
		l:                         l,
		queries:                   queries,
		queueCache:                queueCache,
		stepExpressionCache:       stepExpressionCache,
		concurrencyStrategyCache:  concurrencyStrategyCache,
		tenantIdWorkflowNameCache: tenantIdWorkflowNameCache,
		celParser:                 celParser,
		env:                       env,
		taskLookupCache:           lookupCache,
		payloadStore:              payloadStore,
	}

	tenantLimitRepository := newTenantLimitRepository(s, c, shouldEnforceLimits, enforceLimitsFunc, cacheDuration)

	s.m = tenantLimitRepository

	return s, func() error {
		queueCache.Stop()
		stepExpressionCache.Stop()
		concurrencyStrategyCache.Stop()
		tenantIdWorkflowNameCache.Stop()
		s.m.Stop()
		return nil
	}
}
