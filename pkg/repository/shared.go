package repository

import (
	"context"
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

	"github.com/google/uuid"
)

// implements comparable for the lru cache
type taskExternalIdTenantIdTuple struct {
	externalId uuid.UUID
	tenantId   uuid.UUID
}

type sharedRepository struct {
	pool    *pgxpool.Pool
	ddlPool *pgxpool.Pool // bypasses pgbouncer for DDL operations
	v       validator.Validator
	l       *zerolog.Logger
	queries *sqlcv1.Queries

	dagOperatorEnabled bool
	limitConfig        limits.LimitConfigFile

	queueCache               *cache.Cache
	stepExpressionCache      *cache.Cache
	concurrencyStrategyCache *cache.Cache

	tenantIdWorkflowNameCache   *expirable.LRU[string, *sqlcv1.ListWorkflowsByNamesRow]
	stepsInWorkflowVersionCache *expirable.LRU[uuid.UUID, []*sqlcv1.ListStepsByWorkflowVersionIdsRow]
	stepIdLabelsCache           *expirable.LRU[uuid.UUID, []*sqlcv1.GetDesiredLabelsRow]
	stepIdSlotRequestsCache     *expirable.LRU[uuid.UUID, map[string]int32]

	celParser         *cel.CELParser
	boolExprEvaluator *cel.BoolExprEvaluator
	taskLookupCache   *lru.Cache[taskExternalIdTenantIdTuple, *sqlcv1.FlattenExternalIdsRow]
	payloadStore      PayloadStoreRepository
	m                 TenantLimitRepository
}

func newSharedRepository(
	pool, ddlPool *pgxpool.Pool,
	v validator.Validator,
	l *zerolog.Logger,
	payloadStoreOpts PayloadStoreRepositoryOpts,
	c limits.LimitConfigFile,
	shouldEnforceLimits bool,
	cacheDuration time.Duration,
	dagOperatorEnabled bool,
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

	boolExprEvaluator, err := cel.NewBoolExprEvaluator()

	if err != nil {
		log.Fatalf("failed to create CEL bool expr evaluator: %v", err)
	}

	lookupCache, err := lru.New[taskExternalIdTenantIdTuple, *sqlcv1.FlattenExternalIdsRow](20000)

	if err != nil {
		log.Fatalf("failed to create LRU cache: %v", err)
	}

	s := &sharedRepository{
		pool:                        pool,
		ddlPool:                     ddlPool,
		v:                           v,
		l:                           l,
		queries:                     queries,
		dagOperatorEnabled:          dagOperatorEnabled,
		limitConfig:                 c,
		queueCache:                  queueCache,
		stepExpressionCache:         stepExpressionCache,
		concurrencyStrategyCache:    concurrencyStrategyCache,
		tenantIdWorkflowNameCache:   tenantIdWorkflowNameCache,
		stepsInWorkflowVersionCache: stepsInWorkflowVersionCache,
		stepIdLabelsCache:           stepIdLabelsCache,
		stepIdSlotRequestsCache:     stepIdSlotRequestsCache,
		celParser:                   celParser,
		boolExprEvaluator:           boolExprEvaluator,
		taskLookupCache:             lookupCache,
		payloadStore:                payloadStore,
	}

	tenantLimitRepository := newTenantLimitRepository(s, c, shouldEnforceLimits, cacheDuration)

	s.m = tenantLimitRepository

	return s, s.cleanup
}

func (s *sharedRepository) hasDAGOperator(ctx context.Context, tenantId uuid.UUID) (bool, error) {
	// fixme: can probably cache this?
	if !s.dagOperatorEnabled {
		return false, nil
	}

	return s.queries.TenantHasDAGOperator(ctx, s.pool, tenantId)
}

func (s *sharedRepository) cleanup() error {
	s.queueCache.Stop()
	s.stepExpressionCache.Stop()
	s.concurrencyStrategyCache.Stop()
	s.m.Stop()
	return nil
}
