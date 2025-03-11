package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type workerAPIRepository struct {
	*sharedRepository

	m *metered.Metered
}

func NewWorkerAPIRepository(shared *sharedRepository, m *metered.Metered) repository.WorkerAPIRepository {
	return &workerAPIRepository{
		sharedRepository: shared,
		m:                m,
	}
}

func (w *workerAPIRepository) GetWorkerById(workerId string) (*dbsqlc.GetWorkerByIdRow, error) {
	return w.queries.GetWorkerById(context.Background(), w.pool, sqlchelpers.UUIDFromStr(workerId))
}

func (w *workerAPIRepository) GetWorkerActionsByWorkerId(tenantid, workerId string) ([]pgtype.Text, error) {
	return w.queries.GetWorkerActionsByWorkerId(context.Background(), w.pool, dbsqlc.GetWorkerActionsByWorkerIdParams{
		Workerid: sqlchelpers.UUIDFromStr(workerId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantid),
	})
}

func (w *workerAPIRepository) ListWorkerState(tenantId, workerId string, maxRuns int) ([]*dbsqlc.ListSemaphoreSlotsWithStateForWorkerRow, []*dbsqlc.GetStepRunForEngineRow, error) {
	slots, err := w.queries.ListSemaphoreSlotsWithStateForWorker(context.Background(), w.pool, dbsqlc.ListSemaphoreSlotsWithStateForWorkerParams{
		Workerid: sqlchelpers.UUIDFromStr(workerId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Limit: pgtype.Int4{
			Int32: int32(maxRuns), // nolint: gosec
			Valid: true,
		},
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not list worker slot state: %w", err)
	}

	// get recent assignment events
	assignedEvents, err := w.queries.ListRecentAssignedEventsForWorker(context.Background(), w.pool, dbsqlc.ListRecentAssignedEventsForWorkerParams{
		Workerid: sqlchelpers.UUIDFromStr(workerId),
		Limit: pgtype.Int4{
			Int32: int32(maxRuns), // nolint: gosec
			Valid: true,
		},
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not list worker recent assigned events: %w", err)
	}

	// construct unique array of recent step run ids
	uniqueStepRunIds := make(map[string]bool)

	for _, event := range assignedEvents {
		// unmarshal to string array
		var stepRunIds []string

		if err := json.Unmarshal(event.AssignedStepRuns, &stepRunIds); err != nil {
			return nil, nil, fmt.Errorf("could not unmarshal assigned step runs: %w", err)
		}

		for _, stepRunId := range stepRunIds {
			if _, ok := uniqueStepRunIds[stepRunId]; ok {
				continue
			}

			// just do 20 for now
			if len(uniqueStepRunIds) > 20 {
				break
			}

			uniqueStepRunIds[stepRunId] = true
		}
	}

	stepRunIds := make([]pgtype.UUID, 0, len(uniqueStepRunIds))

	for stepRunId := range uniqueStepRunIds {
		stepRunIds = append(stepRunIds, sqlchelpers.UUIDFromStr(stepRunId))
	}

	recent, err := w.queries.GetStepRunForEngine(context.Background(), w.pool, dbsqlc.GetStepRunForEngineParams{
		Ids:      stepRunIds,
		TenantId: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not list worker recent step runs: %w", err)
	}

	return slots, recent, nil
}

func (r *workerAPIRepository) ListWorkers(tenantId string, opts *repository.ListWorkersOpts) ([]*dbsqlc.ListWorkersWithSlotCountRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	queryParams := dbsqlc.ListWorkersWithSlotCountParams{
		Tenantid: pgTenantId,
	}

	if opts.Action != nil {
		queryParams.ActionId = sqlchelpers.TextFromStr(*opts.Action)
	}

	if opts.LastHeartbeatAfter != nil {
		queryParams.LastHeartbeatAfter = sqlchelpers.TimestampFromTime(opts.LastHeartbeatAfter.UTC())
	}

	if opts.Assignable != nil {
		queryParams.Assignable = pgtype.Bool{
			Bool:  *opts.Assignable,
			Valid: true,
		}
	}

	workers, err := r.queries.ListWorkersWithSlotCount(context.Background(), r.pool, queryParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			workers = make([]*dbsqlc.ListWorkersWithSlotCountRow, 0)
		} else {
			return nil, fmt.Errorf("could not list workers: %w", err)
		}
	}

	return workers, nil
}

func (w *workerAPIRepository) ListWorkerLabels(tenantId, workerId string) ([]*dbsqlc.ListWorkerLabelsRow, error) {
	return w.queries.ListWorkerLabels(context.Background(), w.pool, sqlchelpers.UUIDFromStr(workerId))
}

func (w *workerAPIRepository) UpdateWorker(tenantId, workerId string, opts repository.ApiUpdateWorkerOpts) (*dbsqlc.Worker, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	updateParams := dbsqlc.UpdateWorkerParams{
		ID: sqlchelpers.UUIDFromStr(workerId),
	}

	if opts.IsPaused != nil {
		updateParams.IsPaused = pgtype.Bool{
			Bool:  *opts.IsPaused,
			Valid: true,
		}
	}

	worker, err := w.queries.UpdateWorker(context.Background(), w.pool, updateParams)

	if err != nil {
		return nil, fmt.Errorf("could not update worker: %w", err)
	}

	return worker, nil
}

type workerEngineRepository struct {
	pool          *pgxpool.Pool
	essentialPool *pgxpool.Pool
	v             validator.Validator
	queries       *dbsqlc.Queries
	l             *zerolog.Logger
	m             *metered.Metered
}

func NewWorkerEngineRepository(pool *pgxpool.Pool, essentialPool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, m *metered.Metered) repository.WorkerEngineRepository {
	queries := dbsqlc.New()

	return &workerEngineRepository{
		pool:          pool,
		essentialPool: essentialPool,
		v:             v,
		queries:       queries,
		l:             l,
		m:             m,
	}
}

func (w *workerEngineRepository) GetWorkerForEngine(ctx context.Context, tenantId, workerId string) (*dbsqlc.GetWorkerForEngineRow, error) {
	return w.queries.GetWorkerForEngine(ctx, w.pool, dbsqlc.GetWorkerForEngineParams{
		ID:       sqlchelpers.UUIDFromStr(workerId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})
}

func (w *workerEngineRepository) CreateNewWorker(ctx context.Context, tenantId string, opts *repository.CreateWorkerOpts) (*dbsqlc.Worker, error) {
	return metered.MakeMetered(ctx, w.m, dbsqlc.LimitResourceWORKER, tenantId, 1, func() (*string, *dbsqlc.Worker, error) {
		if err := w.v.Validate(opts); err != nil {
			return nil, nil, err
		}

		tx, err := w.pool.Begin(ctx)

		if err != nil {
			return nil, nil, err
		}

		defer sqlchelpers.DeferRollback(ctx, w.l, tx.Rollback)

		pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

		createParams := dbsqlc.CreateWorkerParams{
			Tenantid:     pgTenantId,
			Dispatcherid: sqlchelpers.UUIDFromStr(opts.DispatcherId),
			Name:         opts.Name,
		}

		// Default to self hosted
		createParams.Type = dbsqlc.NullWorkerType{
			WorkerType: dbsqlc.WorkerTypeSELFHOSTED,
			Valid:      true,
		}

		if opts.WebhookId != nil {
			createParams.WebhookId = sqlchelpers.UUIDFromStr(*opts.WebhookId)
			createParams.Type = dbsqlc.NullWorkerType{
				WorkerType: dbsqlc.WorkerTypeWEBHOOK,
				Valid:      true,
			}
		}

		if opts.MaxRuns != nil {
			createParams.MaxRuns = pgtype.Int4{
				Int32: int32(*opts.MaxRuns), // nolint: gosec
				Valid: true,
			}
		} else {
			createParams.MaxRuns = pgtype.Int4{
				Int32: 100,
				Valid: true,
			}
		}

		var worker *dbsqlc.Worker

		// HACK upsert webhook worker
		if opts.WebhookId != nil {
			worker, err = w.queries.GetWorkerByWebhookId(ctx, tx, dbsqlc.GetWorkerByWebhookIdParams{
				Webhookid: createParams.WebhookId,
				Tenantid:  pgTenantId,
			})

			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, nil, fmt.Errorf("could not get worker: %w", err)
			}

			if errors.Is(err, pgx.ErrNoRows) {
				worker = nil
			}
		}

		if opts.RuntimeInfo != nil {
			if opts.RuntimeInfo.SdkVersion != nil {
				createParams.SdkVersion = sqlchelpers.TextFromStr(*opts.RuntimeInfo.SdkVersion)
			}
			if opts.RuntimeInfo.Language != nil {
				switch *opts.RuntimeInfo.Language {
				case contracts.SDKS_GO:
					createParams.Language = dbsqlc.NullWorkerSDKS{
						WorkerSDKS: dbsqlc.WorkerSDKSGO,
						Valid:      true,
					}
				case contracts.SDKS_PYTHON:
					createParams.Language = dbsqlc.NullWorkerSDKS{
						WorkerSDKS: dbsqlc.WorkerSDKSPYTHON,
						Valid:      true,
					}
				case contracts.SDKS_TYPESCRIPT:
					createParams.Language = dbsqlc.NullWorkerSDKS{
						WorkerSDKS: dbsqlc.WorkerSDKSTYPESCRIPT,
						Valid:      true,
					}
				default:
					return nil, nil, fmt.Errorf("invalid sdk: %s", *opts.RuntimeInfo.Language)
				}
			}
			if opts.RuntimeInfo.LanguageVersion != nil {
				createParams.LanguageVersion = sqlchelpers.TextFromStr(*opts.RuntimeInfo.LanguageVersion)
			}
			if opts.RuntimeInfo.Os != nil {
				createParams.Os = sqlchelpers.TextFromStr(*opts.RuntimeInfo.Os)
			}
			if opts.RuntimeInfo.Extra != nil {
				createParams.RuntimeExtra = sqlchelpers.TextFromStr(*opts.RuntimeInfo.Extra)
			}
		}

		if worker == nil {
			worker, err = w.queries.CreateWorker(ctx, tx, createParams)

			if err != nil {
				return nil, nil, fmt.Errorf("could not create worker: %w", err)
			}
		}

		svcUUIDs := make([]pgtype.UUID, len(opts.Services))

		for i, svc := range opts.Services {
			dbSvc, err := w.queries.UpsertService(ctx, tx, dbsqlc.UpsertServiceParams{
				Name:     svc,
				Tenantid: pgTenantId,
			})

			if err != nil {
				return nil, nil, fmt.Errorf("could not upsert service: %w", err)
			}

			svcUUIDs[i] = dbSvc.ID
		}

		err = w.queries.LinkServicesToWorker(ctx, tx, dbsqlc.LinkServicesToWorkerParams{
			Services: svcUUIDs,
			Workerid: worker.ID,
		})

		if err != nil {
			return nil, nil, fmt.Errorf("could not link services to worker: %w", err)
		}

		actionUUIDs := make([]pgtype.UUID, len(opts.Actions))

		for i, action := range opts.Actions {
			dbAction, err := w.queries.UpsertAction(ctx, tx, dbsqlc.UpsertActionParams{
				Action:   action,
				Tenantid: pgTenantId,
			})

			if err != nil {
				return nil, nil, fmt.Errorf("could not upsert action: %w", err)
			}

			actionUUIDs[i] = dbAction.ID
		}

		err = w.queries.LinkActionsToWorker(ctx, tx, dbsqlc.LinkActionsToWorkerParams{
			Actionids: actionUUIDs,
			Workerid:  worker.ID,
		})

		if err != nil {
			return nil, nil, fmt.Errorf("could not link actions to worker: %w", err)
		}

		err = tx.Commit(ctx)

		if err != nil {
			return nil, nil, fmt.Errorf("could not commit transaction: %w", err)
		}

		id := sqlchelpers.UUIDToStr(worker.ID)

		return &id, worker, nil
	})
}

// UpdateWorker updates a worker in the repository.
// It will only update the worker if there is no lock on the worker, else it will skip.
func (w *workerEngineRepository) UpdateWorker(ctx context.Context, tenantId, workerId string, opts *repository.UpdateWorkerOpts) (*dbsqlc.Worker, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, err := w.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, w.l, tx.Rollback)

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	updateParams := dbsqlc.UpdateWorkerParams{
		ID: sqlchelpers.UUIDFromStr(workerId),
	}

	if opts.LastHeartbeatAt != nil {
		updateParams.LastHeartbeatAt = sqlchelpers.TimestampFromTime(*opts.LastHeartbeatAt)
	}

	if opts.DispatcherId != nil {
		updateParams.DispatcherId = sqlchelpers.UUIDFromStr(*opts.DispatcherId)
	}

	if opts.IsActive != nil {
		updateParams.IsActive = pgtype.Bool{
			Bool:  *opts.IsActive,
			Valid: true,
		}
	}

	worker, err := w.queries.UpdateWorker(ctx, tx, updateParams)

	if err != nil {
		return nil, fmt.Errorf("could not update worker: %w", err)
	}

	if len(opts.Actions) > 0 {
		actionUUIDs := make([]pgtype.UUID, len(opts.Actions))

		for i, action := range opts.Actions {
			dbAction, err := w.queries.UpsertAction(ctx, tx, dbsqlc.UpsertActionParams{
				Action:   action,
				Tenantid: pgTenantId,
			})

			if err != nil {
				return nil, fmt.Errorf("could not upsert action: %w", err)
			}

			actionUUIDs[i] = dbAction.ID
		}

		err = w.queries.LinkActionsToWorker(ctx, tx, dbsqlc.LinkActionsToWorkerParams{
			Actionids: actionUUIDs,
			Workerid:  sqlchelpers.UUIDFromStr(workerId),
		})

		if err != nil {
			return nil, fmt.Errorf("could not link actions to worker: %w", err)
		}
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return worker, nil
}

func (w *workerEngineRepository) UpdateWorkerHeartbeat(ctx context.Context, tenantId, workerId string, lastHeartbeat time.Time) error {

	_, err := w.queries.UpdateWorkerHeartbeat(ctx, w.essentialPool, dbsqlc.UpdateWorkerHeartbeatParams{
		ID:              sqlchelpers.UUIDFromStr(workerId),
		LastHeartbeatAt: sqlchelpers.TimestampFromTime(lastHeartbeat),
	})

	if err != nil {
		return fmt.Errorf("could not update worker heartbeat: %w", err)
	}

	return nil
}

func (w *workerEngineRepository) DeleteWorker(ctx context.Context, tenantId, workerId string) error {
	_, err := w.queries.DeleteWorker(ctx, w.pool, sqlchelpers.UUIDFromStr(workerId))

	return err
}

func (w *workerEngineRepository) UpdateWorkersByWebhookId(ctx context.Context, params dbsqlc.UpdateWorkersByWebhookIdParams) error {
	_, err := w.queries.UpdateWorkersByWebhookId(ctx, w.pool, params)
	return err
}

func (w *workerEngineRepository) UpdateWorkerActiveStatus(ctx context.Context, tenantId, workerId string, isActive bool, timestamp time.Time) (*dbsqlc.Worker, error) {
	worker, err := w.queries.UpdateWorkerActiveStatus(ctx, w.pool, dbsqlc.UpdateWorkerActiveStatusParams{
		ID:                      sqlchelpers.UUIDFromStr(workerId),
		Isactive:                isActive,
		LastListenerEstablished: sqlchelpers.TimestampFromTime(timestamp),
	})

	if err != nil {
		return nil, fmt.Errorf("could not update worker active status: %w", err)
	}

	return worker, nil
}

func (w *workerEngineRepository) UpsertWorkerLabels(ctx context.Context, workerId pgtype.UUID, opts []repository.UpsertWorkerLabelOpts) ([]*dbsqlc.WorkerLabel, error) {
	if len(opts) == 0 {
		return nil, nil
	}

	affinities := make([]*dbsqlc.WorkerLabel, 0, len(opts))

	for _, opt := range opts {

		intValue := pgtype.Int4{Valid: false}
		if opt.IntValue != nil {
			intValue = pgtype.Int4{
				Int32: *opt.IntValue,
				Valid: true,
			}
		}

		strValue := pgtype.Text{Valid: false}
		if opt.StrValue != nil {
			strValue = pgtype.Text{
				String: *opt.StrValue,
				Valid:  true,
			}
		}

		dbsqlcOpts := dbsqlc.UpsertWorkerLabelParams{
			Workerid: workerId,
			Key:      opt.Key,
			IntValue: intValue,
			StrValue: strValue,
		}

		affinity, err := w.queries.UpsertWorkerLabel(ctx, w.pool, dbsqlcOpts)
		if err != nil {
			return nil, fmt.Errorf("could not update worker affinity state: %w", err)
		}

		affinities = append(affinities, affinity)
	}

	return affinities, nil
}

func (r *workerEngineRepository) DeleteOldWorkers(ctx context.Context, tenantId string, lastHeartbeatBefore time.Time) (bool, error) {
	hasMore, err := r.queries.DeleteOldWorkers(ctx, r.pool, dbsqlc.DeleteOldWorkersParams{
		Tenantid:            sqlchelpers.UUIDFromStr(tenantId),
		Lastheartbeatbefore: sqlchelpers.TimestampFromTime(lastHeartbeatBefore),
		Limit:               20,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return hasMore, nil
}

func (r *workerEngineRepository) DeleteOldWorkerEvents(ctx context.Context, tenantId string, lastHeartbeatAfter time.Time) error {
	// list workers
	workers, err := r.queries.ListWorkersWithSlotCount(ctx, r.pool, dbsqlc.ListWorkersWithSlotCountParams{
		Tenantid:           sqlchelpers.UUIDFromStr(tenantId),
		LastHeartbeatAfter: sqlchelpers.TimestampFromTime(lastHeartbeatAfter),
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		return err
	}

	for _, worker := range workers {
		hasMore := true

		for hasMore {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// delete worker events
			hasMore, err = r.queries.DeleteOldWorkerAssignEvents(ctx, r.pool, dbsqlc.DeleteOldWorkerAssignEventsParams{
				Workerid: worker.Worker.ID,
				MaxRuns:  worker.Worker.MaxRuns,
				Limit:    100,
			})

			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					break
				}

				return fmt.Errorf("could not delete old worker events: %w", err)
			}
		}
	}

	return nil
}

func (r *workerEngineRepository) GetDispatcherIdsForWorkers(ctx context.Context, tenantId string, workerIds []string) (map[string][]string, error) {
	pgWorkerIds := make([]pgtype.UUID, len(workerIds))

	for i, workerId := range workerIds {
		pgWorkerIds[i] = sqlchelpers.UUIDFromStr(workerId)
	}

	rows, err := r.queries.ListDispatcherIdsForWorkers(ctx, r.pool, dbsqlc.ListDispatcherIdsForWorkersParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Workerids: sqlchelpers.UniqueSet(pgWorkerIds),
	})

	if err != nil {
		return nil, fmt.Errorf("could not get dispatcher ids for workers: %w", err)
	}

	dispatcherIdsToWorkers := make(map[string][]string)

	for _, row := range rows {
		dispatcherId := sqlchelpers.UUIDToStr(row.DispatcherId)
		workerId := sqlchelpers.UUIDToStr(row.WorkerId)

		if _, ok := dispatcherIdsToWorkers[dispatcherId]; !ok {
			dispatcherIdsToWorkers[dispatcherId] = make([]string, 0)
		}

		dispatcherIdsToWorkers[dispatcherId] = append(dispatcherIdsToWorkers[dispatcherId], workerId)
	}

	return dispatcherIdsToWorkers, nil
}
