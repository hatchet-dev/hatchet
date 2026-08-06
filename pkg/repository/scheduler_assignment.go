package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

// A worker's action set is immutable after registration (the actionHash is a
// sha256 of the registered action names), so cache entries never go stale.
// The TTL only bounds memory for tenants whose workers have churned away.
const workerActionCacheTTL = 30 * time.Minute

type workerActionCacheKey struct {
	actionHash string
	tenantId   uuid.UUID
}

type assignmentRepository struct {
	*sharedRepository

	workerActionCache *expirable.LRU[workerActionCacheKey, []string]
}

func newAssignmentRepository(shared *sharedRepository) *assignmentRepository {
	return &assignmentRepository{
		sharedRepository:  shared,
		workerActionCache: expirable.NewLRU[workerActionCacheKey, []string](10000, nil, workerActionCacheTTL),
	}
}

func (d *assignmentRepository) ListActionsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-actions-for-workers")
	defer span.End()

	liveWorkers, err := d.queries.ListLiveWorkerActionHashes(ctx, d.pool, sqlcv1.ListLiveWorkerActionHashesParams{
		Tenantid:  tenantId,
		Workerids: workerIds,
	})

	if err != nil {
		return nil, err
	}

	hashToActions := make(map[workerActionCacheKey][]string)
	// one live representative worker per uncached hash; all workers sharing a
	// hash have identical action sets by construction
	missedHashRepresentatives := make(map[string]uuid.UUID)
	workersWithoutHash := make([]uuid.UUID, 0)

	for _, w := range liveWorkers {
		if len(w.ActionHash) == 0 {
			workersWithoutHash = append(workersWithoutHash, w.ID)
			continue
		}

		key := workerActionCacheKey{tenantId: tenantId, actionHash: string(w.ActionHash)}

		if _, ok := hashToActions[key]; ok {
			continue
		}

		if _, ok := missedHashRepresentatives[string(w.ActionHash)]; ok {
			continue
		}

		if actions, ok := d.workerActionCache.Get(key); ok {
			hashToActions[key] = actions
		} else {
			missedHashRepresentatives[string(w.ActionHash)] = w.ID
		}
	}

	if len(missedHashRepresentatives) > 0 {
		representativeIds := make([]uuid.UUID, 0, len(missedHashRepresentatives))

		for _, workerId := range missedHashRepresentatives {
			representativeIds = append(representativeIds, workerId)
		}

		records, err := d.queries.ListWorkerActionSets(ctx, d.pool, sqlcv1.ListWorkerActionSetsParams{
			Tenantid:  tenantId,
			Workerids: representativeIds,
		})

		if err != nil {
			return nil, err
		}

		fetched := make(map[workerActionCacheKey][]string, len(missedHashRepresentatives))

		// pre-seed with empty slices so hashes with zero resolved actions
		// still produce a NULL-action row below
		for hash := range missedHashRepresentatives {
			fetched[workerActionCacheKey{tenantId: tenantId, actionHash: hash}] = make([]string, 0)
		}

		for _, record := range records {
			if !record.ActionId.Valid {
				continue
			}

			key := workerActionCacheKey{tenantId: tenantId, actionHash: string(record.ActionHash)}
			fetched[key] = append(fetched[key], record.ActionId.String)
		}

		for key, actions := range fetched {
			// never cache empty sets: an empty result usually means the
			// representative worker was deleted between the liveness lookup
			// and the action set query, and caching it would pin every worker
			// sharing this hash to zero actions until the TTL expires.
			// Re-resolving next cycle is cheap and self-heals.
			if len(actions) > 0 {
				d.workerActionCache.Add(key, actions)
			}

			hashToActions[key] = actions
		}
	}

	rows := make([]*sqlcv1.ListActionsForWorkersRow, 0, len(liveWorkers))

	for _, w := range liveWorkers {
		if len(w.ActionHash) == 0 {
			continue
		}

		key := workerActionCacheKey{tenantId: tenantId, actionHash: string(w.ActionHash)}
		actions := hashToActions[key]

		if len(actions) == 0 {
			rows = append(rows, &sqlcv1.ListActionsForWorkersRow{
				WorkerId: w.ID,
				ActionId: pgtype.Text{},
			})

			continue
		}

		for _, actionId := range actions {
			rows = append(rows, &sqlcv1.ListActionsForWorkersRow{
				WorkerId: w.ID,
				ActionId: pgtype.Text{String: actionId, Valid: true},
			})
		}
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "list_actions.live_workers", Value: len(liveWorkers)},
		telemetry.AttributeKV{Key: "list_actions.hash_cache_misses", Value: len(missedHashRepresentatives)},
		telemetry.AttributeKV{Key: "list_actions.workers_without_hash", Value: len(workersWithoutHash)},
	)

	// fallback for workers registered before actionHash existed
	if len(workersWithoutHash) > 0 {
		fallbackRows, err := d.queries.ListActionsForWorkersLegacyFallback(ctx, d.pool, sqlcv1.ListActionsForWorkersLegacyFallbackParams{
			Tenantid:  tenantId,
			Workerids: workersWithoutHash,
		})

		if err != nil {
			return nil, err
		}

		rows = append(rows, fallbackRows...)
	}

	return rows, nil
}

func (d *assignmentRepository) ListAvailableSlotsForWorkers(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-available-slots-for-workers")
	defer span.End()

	return d.queries.ListAvailableSlotsForWorkers(ctx, d.pool, params)
}

func (d *assignmentRepository) ListAvailableSlotsForWorkersAndTypes(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersAndTypesParams) ([]*sqlcv1.ListAvailableSlotsForWorkersAndTypesRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-available-slots-for-workers-and-types")
	defer span.End()

	return d.queries.ListAvailableSlotsForWorkersAndTypes(ctx, d.pool, params)
}

func (d *assignmentRepository) ListWorkerSlotConfigs(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListWorkerSlotConfigsRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-worker-slot-configs")
	defer span.End()

	return d.queries.ListWorkerSlotConfigs(ctx, d.pool, sqlcv1.ListWorkerSlotConfigsParams{
		Tenantid:  tenantId,
		Workerids: workerIds,
	})
}
