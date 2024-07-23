package prisma

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type workerAPIRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
	m       *metered.Metered
}

func NewWorkerAPIRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, m *metered.Metered) repository.WorkerAPIRepository {
	queries := dbsqlc.New()

	return &workerAPIRepository{
		client:  client,
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
		m:       m,
	}
}

func (w *workerAPIRepository) GetWorkerById(workerId string) (*db.WorkerModel, error) {
	return w.client.Worker.FindUnique(
		db.Worker.ID.Equals(workerId),
	).With(
		db.Worker.Dispatcher.Fetch(),
		db.Worker.Actions.Fetch(),
		db.Worker.Slots.Fetch(),
	).Exec(context.Background())
}

func (w *workerAPIRepository) ListRecentWorkerStepRuns(tenantId, workerId string) ([]db.StepRunModel, error) {
	return w.client.StepRun.FindMany(
		db.StepRun.WorkerID.Equals(workerId),
		db.StepRun.TenantID.Equals(tenantId),
	).Take(10).OrderBy(
		db.StepRun.CreatedAt.Order(db.SortOrderDesc),
	).With(
		db.StepRun.Children.Fetch(),
		db.StepRun.Parents.Fetch(),
		db.StepRun.JobRun.Fetch().With(
			db.JobRun.WorkflowRun.Fetch(),
		),
		db.StepRun.Step.Fetch().With(
			db.Step.Job.Fetch().With(
				db.Job.Workflow.Fetch(),
			),
			db.Step.Action.Fetch(),
		),
	).Exec(context.Background())
}

func (r *workerAPIRepository) ListWorkers(tenantId string, opts *repository.ListWorkersOpts) ([]*dbsqlc.ListWorkersWithStepCountRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	queryParams := dbsqlc.ListWorkersWithStepCountParams{
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

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), r.l, tx.Rollback)

	workers, err := r.queries.ListWorkersWithStepCount(context.Background(), tx, queryParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			workers = make([]*dbsqlc.ListWorkersWithStepCountRow, 0)
		} else {
			return nil, fmt.Errorf("could not list workers: %w", err)
		}
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
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
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
	m       *metered.Metered
}

func NewWorkerEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, m *metered.Metered) repository.WorkerEngineRepository {
	queries := dbsqlc.New()

	return &workerEngineRepository{
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
		m:       m,
	}
}

func (w *workerEngineRepository) GetWorkerForEngine(ctx context.Context, tenantId, workerId string) (*dbsqlc.GetWorkerForEngineRow, error) {
	return w.queries.GetWorkerForEngine(ctx, w.pool, dbsqlc.GetWorkerForEngineParams{
		ID:       sqlchelpers.UUIDFromStr(workerId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})
}

func (w *workerEngineRepository) CreateNewWorker(ctx context.Context, tenantId string, opts *repository.CreateWorkerOpts) (*dbsqlc.Worker, error) {
	return metered.MakeMetered(ctx, w.m, dbsqlc.LimitResourceWORKER, tenantId, func() (*string, *dbsqlc.Worker, error) {
		if err := w.v.Validate(opts); err != nil {
			return nil, nil, err
		}

		tx, err := w.pool.Begin(ctx)

		if err != nil {
			return nil, nil, err
		}

		defer deferRollback(ctx, w.l, tx.Rollback)

		pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

		createParams := dbsqlc.CreateWorkerParams{
			Tenantid:     pgTenantId,
			Dispatcherid: sqlchelpers.UUIDFromStr(opts.DispatcherId),
			Name:         opts.Name,
		}

		if opts.MaxRuns != nil {
			createParams.MaxRuns = pgtype.Int4{
				Int32: int32(*opts.MaxRuns),
				Valid: true,
			}
		} else {
			createParams.MaxRuns = pgtype.Int4{
				Int32: 100,
				Valid: true,
			}
		}

		worker, err := w.queries.CreateWorker(ctx, tx, createParams)

		if err != nil {
			return nil, nil, fmt.Errorf("could not create worker: %w", err)
		}

		err = w.queries.StubWorkerSemaphoreSlots(ctx, tx, dbsqlc.StubWorkerSemaphoreSlotsParams{
			Workerid: worker.ID,
			MaxRuns: pgtype.Int4{
				Int32: worker.MaxRuns,
				Valid: true,
			},
		})

		if err != nil {
			return nil, nil, fmt.Errorf("could not stub worker semaphore slots: %w", err)
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

func (w *workerEngineRepository) UpdateWorker(ctx context.Context, tenantId, workerId string, opts *repository.UpdateWorkerOpts) (*dbsqlc.Worker, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, err := w.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, w.l, tx.Rollback)

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

func (w *workerEngineRepository) DeleteWorker(ctx context.Context, tenantId, workerId string) error {
	_, err := w.queries.DeleteWorker(ctx, w.pool, sqlchelpers.UUIDFromStr(workerId))

	return err
}

func (w *workerEngineRepository) UpdateWorkersByName(ctx context.Context, params dbsqlc.UpdateWorkersByNameParams) error {
	_, err := w.queries.UpdateWorkersByName(ctx, w.pool, params)
	return err
}

func (w *workerEngineRepository) ResolveWorkerSemaphoreSlots(ctx context.Context, tenantId pgtype.UUID) (*dbsqlc.ResolveWorkerSemaphoreSlotsRow, error) {
	return w.queries.ResolveWorkerSemaphoreSlots(ctx, w.pool, tenantId)
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
