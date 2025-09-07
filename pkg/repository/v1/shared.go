package v1

import (
	"log"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	celgo "github.com/google/cel-go/cel"
)

// implements comparable for the lru cache
type taskExternalIdTenantIdTuple struct {
	externalId string
	tenantId   string
}

type sharedRepository struct {
	pool                        *pgxpool.Pool
	v                           validator.Validator
	l                           *zerolog.Logger
	queries                     *sqlcv1.Queries
	queueCache                  *cache.Cache
	stepExpressionCache         *cache.Cache
	tenantIdWorkflowNameCache   *expirable.LRU[string, *sqlcv1.ListWorkflowsByNamesRow]
	stepsInWorkflowVersionCache *expirable.LRU[string, []*sqlcv1.ListStepsByWorkflowVersionIdsRow]
	stepIdLabelsCache           *expirable.LRU[string, []*sqlcv1.GetDesiredLabelsRow]
	celParser                   *cel.CELParser
	env                         *celgo.Env
	taskLookupCache             *lru.Cache[taskExternalIdTenantIdTuple, *sqlcv1.FlattenExternalIdsRow]
	m                           *metered.Metered
}

func newSharedRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, entitlements repository.EntitlementsRepository) (*sharedRepository, func() error) {
	m := metered.NewMetered(entitlements, l)

	queries := sqlcv1.New()
	queueCache := cache.New(5 * time.Minute)
	stepExpressionCache := cache.New(5 * time.Minute)
	tenantIdWorkflowNameCache := expirable.NewLRU(10000, func(key string, value *sqlcv1.ListWorkflowsByNamesRow) {}, 5*time.Second)
	stepsInWorkflowVersionCache := expirable.NewLRU(10000, func(key string, value []*sqlcv1.ListStepsByWorkflowVersionIdsRow) {}, 5*time.Second)
	stepIdLabelsCache := expirable.NewLRU(10000, func(key string, value []*sqlcv1.GetDesiredLabelsRow) {}, 5*time.Minute)

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

	return &sharedRepository{
			pool:                        pool,
			v:                           v,
			l:                           l,
			queries:                     queries,
			queueCache:                  queueCache,
			stepExpressionCache:         stepExpressionCache,
			tenantIdWorkflowNameCache:   tenantIdWorkflowNameCache,
			stepsInWorkflowVersionCache: stepsInWorkflowVersionCache,
			stepIdLabelsCache:           stepIdLabelsCache,
			celParser:                   celParser,
			env:                         env,
			taskLookupCache:             lookupCache,
			m:                           m,
		}, func() error {
			queueCache.Stop()
			stepExpressionCache.Stop()
			m.Stop()
			return nil
		}
}
