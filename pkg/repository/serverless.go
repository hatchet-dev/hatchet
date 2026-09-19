package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/config/limits"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// ServerlessRepository is the data access layer of the serverless operator. Endpoints and tenant
// settings are written by the API server; process rows and leases by the operator (out of
// process or in-engine). Steady-state writes are one process heartbeat per process: lease rows
// change only on ownership changes and endpoint rows only on status transitions and workflow
// changes.
type ServerlessRepository interface {
	Endpoints() ServerlessEndpointRepository
	Tenants() ServerlessTenantRepository
	Processes() ServerlessProcessRepository
	Leases() ServerlessLeaseRepository
}

// ServerlessUnit is one lease unit: the (tenant, shard) pair a process owns.
type ServerlessUnit struct {
	TenantId uuid.UUID
	Shard    int32
}

// ServerlessEndpointVersion is one endpoint's id and row version (the later of updated_at
// and status_changed_at), the keyset ListVersions pages on.
type ServerlessEndpointVersion struct {
	Version time.Time
	ID      uuid.UUID
}

type ServerlessEndpointRepository interface {
	// Create inserts the endpoint and, in the same transaction, upserts the tenant's settings
	// row, creates the endpoint's lease unit if it is the first endpoint on that unit, and
	// increments the unit's endpoint_count. The endpoint's namespace is assigned by the database
	// and its shard is derived from its id and the tenant's current shard_count.
	Create(ctx context.Context, tenantId uuid.UUID, opts CreateServerlessEndpointOpts) (*sqlcv1.V1ServerlessEndpoint, error)
	Get(ctx context.Context, tenantId, endpointId uuid.UUID) (*sqlcv1.V1ServerlessEndpoint, error)
	// GetById resolves an endpoint by id alone, for the API's resource populator, which sees
	// the endpoint id before the tenant and checks the returned tenant against the caller's.
	GetById(ctx context.Context, endpointId uuid.UUID) (*sqlcv1.V1ServerlessEndpoint, error)
	List(ctx context.Context, tenantId uuid.UUID, opts ListServerlessEndpointsOpts) ([]*sqlcv1.V1ServerlessEndpoint, int64, error)
	// Update changes configuration only; namespace and shard are immutable.
	Update(ctx context.Context, tenantId, endpointId uuid.UUID, opts UpdateServerlessEndpointOpts) (*sqlcv1.V1ServerlessEndpoint, error)
	// Delete removes the endpoint and decrements its lease unit's endpoint_count in the same
	// transaction. The lease row itself is kept.
	Delete(ctx context.Context, tenantId, endpointId uuid.UUID) (*sqlcv1.V1ServerlessEndpoint, error)

	// ListForUnits returns the endpoints of the given units, keyset-paged by id: pass uuid.Nil
	// for the first page and the last returned id afterwards.
	ListForUnits(ctx context.Context, units []ServerlessUnit, afterId uuid.UUID, limit int64) ([]*sqlcv1.V1ServerlessEndpoint, error)
	// ListForTenant loads a tenant's routing cache.
	ListForTenant(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.V1ServerlessEndpoint, error)
	// ListUpdatedSince refreshes a tenant's routing cache incrementally: the rows whose version
	// (the later of updated_at and status_changed_at) and id are past the (since, sinceId)
	// keyset, in that order. Configuration, registered_actions and status changes all surface.
	ListUpdatedSince(ctx context.Context, tenantId uuid.UUID, since time.Time, sinceId uuid.UUID) ([]*sqlcv1.V1ServerlessEndpoint, error)
	// GetByNamespace resolves the tenant's endpoint an action namespace names, for a routing
	// miss; pgx.ErrNoRows when there is none.
	GetByNamespace(ctx context.Context, tenantId, namespace uuid.UUID) (*sqlcv1.V1ServerlessEndpoint, error)
	// ListVersions is the anti-entropy pass of a tenant's routing cache: every endpoint's id
	// and version, keyset-paged on (version, id) from after, in that order, and nothing else,
	// so the cache can find the rows it must fetch and the ids that are gone without
	// transferring the tenant's rows.
	ListVersions(ctx context.Context, tenantId uuid.UUID, after ServerlessEndpointVersion, limit int64) ([]ServerlessEndpointVersion, error)
	// ListByIds returns the given endpoints, by id.
	ListByIds(ctx context.Context, ids []uuid.UUID) ([]*sqlcv1.V1ServerlessEndpoint, error)

	// UpdateStatus records a healthy/unhealthy transition and returns the database's
	// status_changed_at of the write. It is written by the owning process on transitions only,
	// never per poll.
	UpdateStatus(ctx context.Context, endpointId uuid.UUID, healthy bool, statusError *string) (time.Time, error)
	// UpdateRegisteredActions records the namespaced action set the owner registered after a
	// healthcheck changed the endpoint's workflows.
	UpdateRegisteredActions(ctx context.Context, endpointId uuid.UUID, actions []string) error
}

type ServerlessTenantRepository interface {
	// Upsert creates the tenant's settings row with defaults if absent and returns it.
	Upsert(ctx context.Context, tenantId uuid.UUID) (*sqlcv1.V1ServerlessTenant, error)
	Get(ctx context.Context, tenantId uuid.UUID) (*sqlcv1.V1ServerlessTenant, error)
	// UpdateShardCount sets shard_count and creates lease rows for any new shards. Existing
	// endpoints keep the shard they were inserted with; only new endpoints hash over the new
	// count.
	UpdateShardCount(ctx context.Context, tenantId uuid.UUID, shardCount int32) (*sqlcv1.V1ServerlessTenant, error)
}

type ServerlessProcessRepository interface {
	// Upsert is the process heartbeat: one row write per process per interval.
	Upsert(ctx context.Context, opts UpsertServerlessProcessOpts) error
	// ListLive returns the processes whose row has not expired and the ids of those whose row
	// has. Dead ids feed ServerlessLeaseRepository.Claim so their units can be taken over.
	ListLive(ctx context.Context) (live []*sqlcv1.V1ServerlessProcess, dead []*sqlcv1.V1ServerlessProcess, err error)
	// DeleteExpired sweeps process rows that expired before cutoff, releasing in the same
	// statement every unit those rows still own, and returns both counts.
	DeleteExpired(ctx context.Context, cutoff time.Time) (deletedProcesses, releasedUnits int64, err error)
	// Delete removes the process's own row on graceful shutdown, releasing in the same
	// statement any unit the row still owns.
	Delete(ctx context.Context, processId uuid.UUID) error
}

type ServerlessLeaseRepository interface {
	// Claim takes up to limit units for processId: units with no owner, walked in (tenant,
	// shard) order from after (exclusive; pass the zero unit to start from the beginning),
	// then units owned by processes whose heartbeat row has expired. The statement decides
	// liveness in its own snapshot and requires processId to be live itself, and uses
	// FOR UPDATE SKIP LOCKED so concurrent claimers never block or double-claim.
	Claim(ctx context.Context, processId uuid.UUID, after ServerlessUnit, limit int32) ([]*sqlcv1.ClaimServerlessLeasesRow, error)
	// Shed releases the given units if, and only if, processId still owns them.
	Shed(ctx context.Context, processId uuid.UUID, units []ServerlessUnit) ([]*sqlcv1.ShedServerlessLeasesRow, error)
	// ReleaseAll releases every unit processId owns and returns how many.
	ReleaseAll(ctx context.Context, processId uuid.UUID) (int64, error)
	ListOwned(ctx context.Context, processId uuid.UUID) ([]*sqlcv1.V1ServerlessLease, error)
	// CountClaimable returns the number of units Claim would consider and the sum of their
	// endpoint counts, under Claim's liveness rule, each side counted over at most limit
	// units. The claiming process sizes its fair share from it.
	CountClaimable(ctx context.Context, limit int64) (*sqlcv1.CountClaimableServerlessLeasesRow, error)
	// InsertIfAbsent creates the lease row of a unit. Endpoint creation does this itself; it is
	// exposed for callers that add shards.
	InsertIfAbsent(ctx context.Context, unit ServerlessUnit) error
	// IncrementEndpointCount adjusts a unit's fair-share weight by delta.
	IncrementEndpointCount(ctx context.Context, unit ServerlessUnit, delta int32) error
}

type serverlessRepository struct {
	endpoints ServerlessEndpointRepository
	tenants   ServerlessTenantRepository
	processes ServerlessProcessRepository
	leases    ServerlessLeaseRepository
}

func newServerlessRepository(shared *sharedRepository) ServerlessRepository {
	return &serverlessRepository{
		endpoints: &serverlessEndpointRepository{sharedRepository: shared},
		tenants:   &serverlessTenantRepository{sharedRepository: shared},
		processes: &serverlessProcessRepository{sharedRepository: shared},
		leases:    &serverlessLeaseRepository{sharedRepository: shared},
	}
}

// NewServerlessRepositoryFromPool builds a ServerlessRepository on a pool the caller owns, for
// the out-of-process operator binary. The returned cleanup releases the shared repository's
// resources but not the pool.
func NewServerlessRepositoryFromPool(pool *pgxpool.Pool, l *zerolog.Logger) (ServerlessRepository, func() error) {
	v := validator.NewDefaultValidator()

	shared, cleanupShared := newSharedRepository(pool, pool, v, l, PayloadStoreRepositoryOpts{}, limits.LimitConfigFile{}, false, time.Minute)

	return newServerlessRepository(shared), cleanupShared
}

func (r *serverlessRepository) Endpoints() ServerlessEndpointRepository {
	return r.endpoints
}

func (r *serverlessRepository) Tenants() ServerlessTenantRepository {
	return r.tenants
}

func (r *serverlessRepository) Processes() ServerlessProcessRepository {
	return r.processes
}

func (r *serverlessRepository) Leases() ServerlessLeaseRepository {
	return r.leases
}

// unitArrays splits units into the parallel arrays the unnest-based queries take.
func unitArrays(units []ServerlessUnit) ([]uuid.UUID, []int32) {
	tenantIds := make([]uuid.UUID, len(units))
	shards := make([]int32, len(units))

	for i, u := range units {
		tenantIds[i] = u.TenantId
		shards[i] = u.Shard
	}

	return tenantIds, shards
}
