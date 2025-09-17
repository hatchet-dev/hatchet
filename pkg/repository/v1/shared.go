package v1

import (
	"log"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

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
	pool                      *pgxpool.Pool
	v                         validator.Validator
	l                         *zerolog.Logger
	queries                   *sqlcv1.Queries
	queueCache                *cache.Cache
	stepExpressionCache       *cache.Cache
	tenantIdWorkflowNameCache *cache.Cache
	celParser                 *cel.CELParser
	env                       *celgo.Env
	taskLookupCache           *lru.Cache[taskExternalIdTenantIdTuple, *sqlcv1.FlattenExternalIdsRow]
	m                         *metered.Metered
	payloadStore              PayloadStoreRepository
}

func newSharedRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, entitlements repository.EntitlementsRepository, enablePayloadDualWrites bool) (*sharedRepository, func() error) {
	m := metered.NewMetered(entitlements, l)

	queries := sqlcv1.New()
	queueCache := cache.New(5 * time.Minute)
	stepExpressionCache := cache.New(5 * time.Minute)
	tenantIdWorkflowNameCache := cache.New(5 * time.Minute)
	payloadStore := NewPayloadStoreRepository(pool, l, queries, enablePayloadDualWrites)

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
			pool:                      pool,
			v:                         v,
			l:                         l,
			queries:                   queries,
			queueCache:                queueCache,
			stepExpressionCache:       stepExpressionCache,
			tenantIdWorkflowNameCache: tenantIdWorkflowNameCache,
			celParser:                 celParser,
			env:                       env,
			taskLookupCache:           lookupCache,
			m:                         m,
			payloadStore:              payloadStore,
		}, func() error {
			queueCache.Stop()
			stepExpressionCache.Stop()
			tenantIdWorkflowNameCache.Stop()
			m.Stop()
			return nil
		}
}
