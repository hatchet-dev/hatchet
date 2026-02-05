package repository

import (
	"log"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/hashicorp/golang-lru/v2/expirable"
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
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *sqlcv1.Queries

	queueCache               *cache.Cache
	stepExpressionCache      *cache.Cache
	concurrencyStrategyCache *cache.Cache

	tenantIdWorkflowNameCache   *expirable.LRU[string, *sqlcv1.ListWorkflowsByNamesRow]
	stepsInWorkflowVersionCache *expirable.LRU[uuid.UUID, []*sqlcv1.ListStepsByWorkflowVersionIdsRow]
	stepIdLabelsCache           *expirable.LRU[uuid.UUID, []*sqlcv1.GetDesiredLabelsRow]
	stepIdSlotRequestsCache     *expirable.LRU[uuid.UUID, map[string]int32]

	celParser       *cel.CELParser
	env             *celgo.Env
	taskLookupCache *lru.Cache[taskExternalIdTenantIdTuple, *sqlcv1.FlattenExternalIdsRow]
	payloadStore    PayloadStoreRepository
	m               TenantLimitRepository

	enableDurableUserEventLog         bool
	idempotencyKeyTTL                 time.Duration
	idempotencyKeyDenyRecheckInterval time.Duration
}

func newSharedRepository(
	pool *pgxpool.Pool,
	v validator.Validator,
	l *zerolog.Logger,
	payloadStoreOpts PayloadStoreRepositoryOpts,
	c limits.LimitConfigFile,
	shouldEnforceLimits bool,
	cacheDuration time.Duration,
	enableDurableUserEventLog bool,
	idempotencyKeyTTL time.Duration,
	idempotencyKeyDenyRecheckInterval time.Duration,
) (*sharedRepository, func() error) {
	queries := sqlcv1.New()
	queueCache := cache.New(5 * time.Minute)
	stepExpressionCache := cache.New(5 * time.Minute)
	concurrencyStrategyCache := cache.New(5 * time.Minute)
	payloadStore := NewPayloadStoreRepository(pool, l, queries, payloadStoreOpts)

	// 5-second cache because the workflow version id can change when a new workflow is deployed
	tenantIdWorkflowNameCache := expirable.NewLRU(10000, func(key string, value *sqlcv1.ListWorkflowsByNamesRow) {}, 5*time.Second)
	stepsInWorkflowVersionCache := expirable.NewLRU(10000, func(key uuid.UUID, value []*sqlcv1.ListStepsByWorkflowVersionIdsRow) {}, 5*time.Minute)
	stepIdLabelsCache := expirable.NewLRU(10000, func(key uuid.UUID, value []*sqlcv1.GetDesiredLabelsRow) {}, 5*time.Minute)
	stepIdSlotRequestsCache := expirable.NewLRU(10000, func(key uuid.UUID, value map[string]int32) {}, 5*time.Minute)

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
		pool:                              pool,
		v:                                 v,
		l:                                 l,
		queries:                           queries,
		queueCache:                        queueCache,
		stepExpressionCache:               stepExpressionCache,
		concurrencyStrategyCache:          concurrencyStrategyCache,
		tenantIdWorkflowNameCache:         tenantIdWorkflowNameCache,
		stepsInWorkflowVersionCache:       stepsInWorkflowVersionCache,
		stepIdLabelsCache:                 stepIdLabelsCache,
		stepIdSlotRequestsCache:           stepIdSlotRequestsCache,
		celParser:                         celParser,
		env:                               env,
		taskLookupCache:                   lookupCache,
		payloadStore:                      payloadStore,
		enableDurableUserEventLog:         enableDurableUserEventLog,
		idempotencyKeyTTL:                 idempotencyKeyTTL,
		idempotencyKeyDenyRecheckInterval: idempotencyKeyDenyRecheckInterval,
	}

	tenantLimitRepository := newTenantLimitRepository(s, c, shouldEnforceLimits, cacheDuration)

	s.m = tenantLimitRepository

	return s, func() error {
		queueCache.Stop()
		stepExpressionCache.Stop()
		concurrencyStrategyCache.Stop()
		s.m.Stop()
		return nil
	}
}
