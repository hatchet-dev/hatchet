package repository

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"

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

// implements comparable for the lru cache
type streamTopicKey struct {
	tenantId  uuid.UUID
	namespace string
	topic     string
}

type eventKeyTenantIdTuple struct {
	tenantId uuid.UUID
	eventKey string
}

type sharedRepository struct {
	pool    *fairpool.Pool
	ddlPool *pgxpool.Pool // bypasses pgbouncer for DDL operations
	v       validator.Validator
	l       *zerolog.Logger
	queries *sqlcv1.Queries

	limitConfig limits.LimitConfigFile

	queueCache               *cache.Cache
	concurrencyStrategyCache *cache.Cache

	tenantIdWorkflowNameCache   *expirable.LRU[string, *sqlcv1.ListWorkflowsByNamesRow]
	workflowIdNameCache         *expirable.LRU[uuid.UUID, string]
	stepsInWorkflowVersionCache *expirable.LRU[uuid.UUID, []*sqlcv1.ListStepsByWorkflowVersionIdsRow]
	stepIdConfigCache           *expirable.LRU[uuid.UUID, *sqlcv1.ListStepsByIdsRow]
	stepIdLabelsCache           *expirable.LRU[uuid.UUID, []*sqlcv1.GetDesiredLabelsRow]
	stepIdSlotRequestsCache     *expirable.LRU[uuid.UUID, map[string]int32]
	stepIdHasBatchConfigCache   *expirable.LRU[uuid.UUID, bool]
	stepIdMatchConditionsCache  *expirable.LRU[uuid.UUID, []*sqlcv1.V1StepMatchCondition]
	stepIdExpressionsCache      *expirable.LRU[uuid.UUID, []*sqlcv1.StepExpression]

	streamTopicSeenCache *expirable.LRU[streamTopicKey, struct{}]

	eventKeyWorkflowsCache *expirable.LRU[eventKeyTenantIdTuple, []*sqlcv1.ListWorkflowsForEventsRow]

	// eventKeyWorkflowsDrops counts InvalidateEventKeyWorkflows calls. A lookup caches its result
	// only if no drop happened while it ran, since it may have read Postgres before the commit.
	eventKeyWorkflowsMu    sync.Mutex
	eventKeyWorkflowsDrops uint64

	rateLimitPauseCache *expirable.LRU[uuid.UUID, bool]

	// tenantControls is this process's snapshot of v1_tenant_control, swapped whole by the
	// tenant control refresher. Nil until the first refresh, which means no tenant is controlled.
	tenantControls atomic.Pointer[map[uuid.UUID]*tenantControl]

	celParser         *cel.CELParser
	boolExprEvaluator *cel.BoolExprEvaluator
	taskLookupCache   *lru.Cache[taskExternalIdTenantIdTuple, *sqlcv1.FlattenExternalIdsRow]
	payloadStore      PayloadStoreRepository
	m                 TenantLimitRepository
	entitlements      TenantEntitlementRepository

	// taskKeyCache has no TTL because a task's external id maps to the same primary key for as long as the task exists
	taskKeyCache *lru.Cache[taskExternalIdTenantIdTuple, IdInsertedAt]

	wrapTaskInputInStepRunData bool
	enablePayloadConsolidation bool
	reassignUnackedTasks       bool
}

func newSharedRepository(
	pool *fairpool.Pool,
	ddlPool *pgxpool.Pool,
	v validator.Validator,
	l *zerolog.Logger,
	payloadStoreOpts PayloadStoreRepositoryOpts,
	c limits.LimitConfigFile,
	shouldEnforceLimits bool,
	cacheDuration time.Duration,
	reassignUnackedTasks bool,
) (*sharedRepository, func() error) {
	queries := sqlcv1.New()
	queueCache := cache.New(5 * time.Minute)
	concurrencyStrategyCache := cache.New(5 * time.Minute)
	payloadStore := NewPayloadStoreRepository(pool, l, queries, payloadStoreOpts)

	// 5-second cache because the workflow version id can change when a new workflow is deployed
	tenantIdWorkflowNameCache := expirable.NewLRU(10000, func(key string, value *sqlcv1.ListWorkflowsByNamesRow) {}, 5*time.Second)
	// a workflow id always maps to the same name, so a long TTL is safe
	workflowIdNameCache := expirable.NewLRU(10000, func(key uuid.UUID, value string) {}, time.Hour)
	stepsInWorkflowVersionCache := expirable.NewLRU(10000, func(key uuid.UUID, value []*sqlcv1.ListStepsByWorkflowVersionIdsRow) {}, 5*time.Minute)
	stepIdConfigCache := expirable.NewLRU(10000, func(key uuid.UUID, value *sqlcv1.ListStepsByIdsRow) {}, 5*time.Minute)
	stepIdLabelsCache := expirable.NewLRU(10000, func(key uuid.UUID, value []*sqlcv1.GetDesiredLabelsRow) {}, 5*time.Minute)
	stepIdSlotRequestsCache := expirable.NewLRU(10000, func(key uuid.UUID, value map[string]int32) {}, 5*time.Minute)
	stepIdHasBatchConfigCache := expirable.NewLRU(10000, func(key uuid.UUID, value bool) {}, 5*time.Minute)
	stepIdMatchConditionsCache := expirable.NewLRU(10000, func(key uuid.UUID, value []*sqlcv1.V1StepMatchCondition) {}, 5*time.Minute)
	stepIdExpressionsCache := expirable.NewLRU(10000, func(key uuid.UUID, value []*sqlcv1.StepExpression) {}, 5*time.Minute)
	streamTopicSeenCache := expirable.NewLRU(10000, func(key streamTopicKey, value struct{}) {}, time.Hour)
	eventKeyWorkflowsCache := expirable.NewLRU(10000, func(key eventKeyTenantIdTuple, value []*sqlcv1.ListWorkflowsForEventsRow) {}, 5*time.Second)
	// short TTL since a pause set through another engine is only seen once this expires; the scheduler holds anything created in between
	rateLimitPauseCache := expirable.NewLRU(10000, func(key uuid.UUID, value bool) {}, 5*time.Second)

	celParser := cel.NewCELParser()

	boolExprEvaluator, err := cel.NewBoolExprEvaluator()

	if err != nil {
		log.Fatalf("failed to create CEL bool expr evaluator: %v", err)
	}

	lookupCache, err := lru.New[taskExternalIdTenantIdTuple, *sqlcv1.FlattenExternalIdsRow](20000)

	if err != nil {
		log.Fatalf("failed to create LRU cache: %v", err)
	}

	taskKeyCache, err := lru.New[taskExternalIdTenantIdTuple, IdInsertedAt](100000)

	if err != nil {
		log.Fatalf("failed to create LRU cache: %v", err)
	}

	s := &sharedRepository{
		pool:                        pool,
		ddlPool:                     ddlPool,
		v:                           v,
		l:                           l,
		queries:                     queries,
		limitConfig:                 c,
		queueCache:                  queueCache,
		concurrencyStrategyCache:    concurrencyStrategyCache,
		tenantIdWorkflowNameCache:   tenantIdWorkflowNameCache,
		workflowIdNameCache:         workflowIdNameCache,
		stepsInWorkflowVersionCache: stepsInWorkflowVersionCache,
		stepIdConfigCache:           stepIdConfigCache,
		stepIdLabelsCache:           stepIdLabelsCache,
		stepIdSlotRequestsCache:     stepIdSlotRequestsCache,
		stepIdHasBatchConfigCache:   stepIdHasBatchConfigCache,
		stepIdMatchConditionsCache:  stepIdMatchConditionsCache,
		stepIdExpressionsCache:      stepIdExpressionsCache,
		streamTopicSeenCache:        streamTopicSeenCache,
		rateLimitPauseCache:         rateLimitPauseCache,
		eventKeyWorkflowsCache:      eventKeyWorkflowsCache,
		celParser:                   celParser,
		boolExprEvaluator:           boolExprEvaluator,
		taskLookupCache:             lookupCache,
		taskKeyCache:                taskKeyCache,
		payloadStore:                payloadStore,
		wrapTaskInputInStepRunData:  payloadStoreOpts.WrapTaskInputInStepRunData,
		enablePayloadConsolidation:  payloadStoreOpts.EnablePayloadConsolidation,
		reassignUnackedTasks:        reassignUnackedTasks,
	}

	tenantLimitRepository := newTenantLimitRepository(s, c, shouldEnforceLimits, cacheDuration)

	s.m = tenantLimitRepository
	s.entitlements = newTenantEntitlementRepository(s)

	return s, s.cleanup
}

// tenantControl is one tenant's entry in the snapshot.
type tenantControl struct {
	row *sqlcv1.V1TenantControl

	// ingest meters the tenant's new runs and events; nil when they are not limited.
	ingest *ingestBucket
}

func (s *sharedRepository) tenantControl(tenantId uuid.UUID) *tenantControl {
	controls := s.tenantControls.Load()
	if controls == nil {
		return nil
	}

	return (*controls)[tenantId]
}

func (s *sharedRepository) tenantPaused(tenantId uuid.UUID) bool {
	control := s.tenantControl(tenantId)

	return control != nil && control.row.Paused
}

// admitTenant applies the operator's controls to a request for items new runs or events of the
// tenant: ErrTenantPaused for a paused tenant, a *TenantRateLimitedError (which takes nothing from
// the bucket) for one over its ingest rate, else nil.
func (s *sharedRepository) admitTenant(tenantId uuid.UUID, items int) error {
	control := s.tenantControl(tenantId)

	switch {
	case control == nil:
		return nil
	case control.row.Paused:
		return ErrTenantPaused
	case control.ingest != nil:
		if wait := control.ingest.take(items, time.Now()); wait > 0 {
			return &TenantRateLimitedError{RetryAfter: wait}
		}
	}

	return nil
}

func (s *sharedRepository) cleanup() error {
	s.queueCache.Stop()
	s.concurrencyStrategyCache.Stop()
	s.m.Stop()
	return nil
}
