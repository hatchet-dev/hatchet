package prisma

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type workerAPIRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewWorkerAPIRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.WorkerAPIRepository {
	queries := dbsqlc.New()

	return &workerAPIRepository{
		client:  client,
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}

func (w *workerAPIRepository) GetWorkerById(workerId string) (*db.WorkerModel, error) {
	return w.client.Worker.FindUnique(
		db.Worker.ID.Equals(workerId),
	).With(
		db.Worker.Dispatcher.Fetch(),
		db.Worker.Actions.Fetch(),
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
			return nil, fmt.Errorf("could not list events: %w", err)
		}
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return workers, nil
}

type workerEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewWorkerEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.WorkerEngineRepository {
	queries := dbsqlc.New()

	return &workerEngineRepository{
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}

func (w *workerEngineRepository) GetWorkerForEngine(tenantId, workerId string) (*dbsqlc.GetWorkerForEngineRow, error) {
	return w.queries.GetWorkerForEngine(context.Background(), w.pool, dbsqlc.GetWorkerForEngineParams{
		ID:       sqlchelpers.UUIDFromStr(workerId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})
}

func (w *workerEngineRepository) CreateNewWorker(tenantId string, opts *repository.CreateWorkerOpts) (*dbsqlc.Worker, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, err := w.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), w.l, tx.Rollback)

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
	}

	worker, err := w.queries.CreateWorker(context.Background(), tx, createParams)

	if err != nil {
		return nil, fmt.Errorf("could not create worker: %w", err)
	}

	svcUUIDs := make([]pgtype.UUID, len(opts.Services))

	for i, svc := range opts.Services {
		dbSvc, err := w.queries.UpsertService(context.Background(), tx, dbsqlc.UpsertServiceParams{
			Name:     svc,
			Tenantid: pgTenantId,
		})

		if err != nil {
			return nil, fmt.Errorf("could not upsert service: %w", err)
		}

		svcUUIDs[i] = dbSvc.ID
	}

	err = w.queries.LinkServicesToWorker(context.Background(), tx, dbsqlc.LinkServicesToWorkerParams{
		Services: svcUUIDs,
		Workerid: worker.ID,
	})

	if err != nil {
		return nil, fmt.Errorf("could not link services to worker: %w", err)
	}

	actionUUIDs := make([]pgtype.UUID, len(opts.Actions))

	for i, action := range opts.Actions {
		dbAction, err := w.queries.UpsertAction(context.Background(), tx, dbsqlc.UpsertActionParams{
			Action:   action,
			Tenantid: pgTenantId,
		})

		if err != nil {
			return nil, fmt.Errorf("could not upsert action: %w", err)
		}

		actionUUIDs[i] = dbAction.ID
	}

	err = w.queries.LinkActionsToWorker(context.Background(), tx, dbsqlc.LinkActionsToWorkerParams{
		Actionids: actionUUIDs,
		Workerid:  worker.ID,
	})

	if err != nil {
		return nil, fmt.Errorf("could not link actions to worker: %w", err)
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return worker, nil
}

func (w *workerEngineRepository) UpdateWorker(tenantId, workerId string, opts *repository.UpdateWorkerOpts) (*dbsqlc.Worker, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, err := w.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), w.l, tx.Rollback)

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	updateParams := dbsqlc.UpdateWorkerParams{
		ID: sqlchelpers.UUIDFromStr(workerId),
	}

	if opts.Status != nil {
		updateParams.Status = dbsqlc.NullWorkerStatus{
			WorkerStatus: dbsqlc.WorkerStatus(*opts.Status),
			Valid:        true,
		}
	}

	if opts.LastHeartbeatAt != nil {
		updateParams.LastHeartbeatAt = sqlchelpers.TimestampFromTime(*opts.LastHeartbeatAt)
	}

	if opts.DispatcherId != nil {
		updateParams.DispatcherId = sqlchelpers.UUIDFromStr(*opts.DispatcherId)
	}

	worker, err := w.queries.UpdateWorker(context.Background(), tx, updateParams)

	if err != nil {
		return nil, fmt.Errorf("could not update worker: %w", err)
	}

	if len(opts.Actions) > 0 {
		actionUUIDs := make([]pgtype.UUID, len(opts.Actions))

		for i, action := range opts.Actions {
			dbAction, err := w.queries.UpsertAction(context.Background(), tx, dbsqlc.UpsertActionParams{
				Action:   action,
				Tenantid: pgTenantId,
			})

			if err != nil {
				return nil, fmt.Errorf("could not upsert action: %w", err)
			}

			actionUUIDs[i] = dbAction.ID
		}

		err = w.queries.LinkActionsToWorker(context.Background(), tx, dbsqlc.LinkActionsToWorkerParams{
			Actionids: actionUUIDs,
			Workerid:  sqlchelpers.UUIDFromStr(workerId),
		})

		if err != nil {
			return nil, fmt.Errorf("could not link actions to worker: %w", err)
		}
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return worker, nil
}

func (w *workerEngineRepository) DeleteWorker(tenantId, workerId string) error {
	_, err := w.queries.DeleteWorker(context.Background(), w.pool, sqlchelpers.UUIDFromStr(workerId))

	return err
}
