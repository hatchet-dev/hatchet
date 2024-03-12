package prisma

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
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

type workerRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewWorkerRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.WorkerRepository {
	queries := dbsqlc.New()

	return &workerRepository{
		client:  client,
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}

func (w *workerRepository) GetWorkerById(workerId string) (*db.WorkerModel, error) {
	return w.client.Worker.FindUnique(
		db.Worker.ID.Equals(workerId),
	).With(
		db.Worker.Dispatcher.Fetch(),
		db.Worker.Actions.Fetch(),
	).Exec(context.Background())
}

func (w *workerRepository) GetWorkerForEngine(tenantId, workerId string) (*dbsqlc.GetWorkerForEngineRow, error) {
	return w.queries.GetWorkerForEngine(context.Background(), w.pool, dbsqlc.GetWorkerForEngineParams{
		ID:       sqlchelpers.UUIDFromStr(workerId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})
}

func (w *workerRepository) ListRecentWorkerStepRuns(tenantId, workerId string) ([]db.StepRunModel, error) {
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

func (r *workerRepository) ListWorkers(tenantId string, opts *repository.ListWorkersOpts) ([]*dbsqlc.ListWorkersWithStepCountRow, error) {
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

func (w *workerRepository) CreateNewWorker(tenantId string, opts *repository.CreateWorkerOpts) (*db.WorkerModel, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	txs := []db.PrismaTransaction{}

	workerId := uuid.New().String()

	createTx := w.client.Worker.CreateOne(
		db.Worker.Tenant.Link(
			db.Tenant.ID.Equals(tenantId),
		),
		db.Worker.Name.Set(opts.Name),
		db.Worker.Dispatcher.Link(
			db.Dispatcher.ID.Equals(opts.DispatcherId),
		),
		db.Worker.ID.Set(workerId),
		db.Worker.MaxRuns.SetIfPresent(opts.MaxRuns),
	).Tx()

	txs = append(txs, createTx)

	for _, svc := range opts.Services {
		upsertServiceTx := w.client.Service.UpsertOne(
			db.Service.TenantIDName(
				db.Service.TenantID.Equals(tenantId),
				db.Service.Name.Equals(svc),
			),
		).Create(
			db.Service.Name.Set(svc),
			db.Service.Tenant.Link(
				db.Tenant.ID.Equals(tenantId),
			),
			db.Service.Workers.Link(
				db.Worker.ID.Equals(workerId),
			),
		).Update(
			db.Service.Workers.Link(
				db.Worker.ID.Equals(workerId),
			),
		).Tx()

		txs = append(txs, upsertServiceTx)
	}

	if len(opts.Actions) > 0 {
		for _, action := range opts.Actions {
			txs = append(txs, w.client.Action.UpsertOne(
				db.Action.TenantIDActionID(
					db.Action.TenantID.Equals(tenantId),
					db.Action.ActionID.Equals(action),
				),
			).Create(
				db.Action.ActionID.Set(action),
				db.Action.Tenant.Link(
					db.Tenant.ID.Equals(tenantId),
				),
			).Update().Tx())

			// This is unfortunate but due to https://github.com/steebchen/prisma-client-go/issues/1095,
			// we cannot set db.Worker.Actions.Link multiple times, and since Link required a unique action
			// where clause, we have to do these in separate transactions
			txs = append(txs, w.client.Worker.FindUnique(
				db.Worker.ID.Equals(workerId),
			).Update(
				db.Worker.Actions.Link(
					db.Action.TenantIDActionID(
						db.Action.TenantID.Equals(tenantId),
						db.Action.ActionID.Equals(action),
					),
				),
			).Tx())
		}
	}

	err := w.client.Prisma.Transaction(txs...).Exec(context.Background())

	if err != nil {
		return nil, err
	}

	return createTx.Result(), nil
}

func (w *workerRepository) UpdateWorker(tenantId, workerId string, opts *repository.UpdateWorkerOpts) (*db.WorkerModel, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	txs := []db.PrismaTransaction{}

	optionals := []db.WorkerSetParam{}

	if opts.Status != nil {
		optionals = append(optionals, db.Worker.Status.Set(*opts.Status))
	}

	if opts.LastHeartbeatAt != nil {
		optionals = append(optionals, db.Worker.LastHeartbeatAt.Set(*opts.LastHeartbeatAt))
	}

	if opts.DispatcherId != nil {
		optionals = append(optionals, db.Worker.Dispatcher.Link(
			db.Dispatcher.ID.Equals(*opts.DispatcherId),
		))
	}

	if len(opts.Actions) > 0 {
		for _, action := range opts.Actions {
			txs = append(txs, w.client.Action.UpsertOne(
				db.Action.TenantIDActionID(
					db.Action.TenantID.Equals(tenantId),
					db.Action.ActionID.Equals(action),
				),
			).Create(
				db.Action.ActionID.Set(action),
				db.Action.Tenant.Link(
					db.Tenant.ID.Equals(tenantId),
				),
			).Update().Tx())

			// This is unfortunate but due to https://github.com/steebchen/prisma-client-go/issues/1095,
			// we cannot set db.Worker.Actions.Link multiple times, and since Link required a unique action
			// where clause, we have to do these in separate transactions
			txs = append(txs, w.client.Worker.FindUnique(
				db.Worker.ID.Equals(workerId),
			).Update(
				db.Worker.Actions.Link(
					db.Action.TenantIDActionID(
						db.Action.TenantID.Equals(tenantId),
						db.Action.ActionID.Equals(action),
					),
				),
			).Tx())
		}
	}

	updateTx := w.client.Worker.FindUnique(
		db.Worker.ID.Equals(workerId),
	).Update(
		optionals...,
	).Tx()

	txs = append(txs, updateTx)

	err := w.client.Prisma.Transaction(txs...).Exec(context.Background())

	if err != nil {
		return nil, err
	}

	return updateTx.Result(), nil
}

func (w *workerRepository) DeleteWorker(tenantId, workerId string) error {
	_, err := w.client.Worker.FindUnique(
		db.Worker.ID.Equals(workerId),
	).Delete().Exec(context.Background())

	return err
}

func (w *workerRepository) AddStepRun(tenantId, workerId, stepRunId string) error {
	tx1 := w.client.Worker.FindUnique(
		db.Worker.ID.Equals(workerId),
	).Update(
		db.Worker.StepRuns.Link(
			db.StepRun.ID.Equals(stepRunId),
		),
	).Tx()

	tx2 := w.client.StepRun.FindUnique(
		db.StepRun.ID.Equals(stepRunId),
	).Update(
		db.StepRun.Status.Set(db.StepRunStatusAssigned),
	).Tx()

	err := w.client.Prisma.Transaction(tx1, tx2).Exec(context.Background())

	return err
}

func (w *workerRepository) AddGetGroupKeyRun(tenantId, workerId, getGroupKeyRunId string) error {
	tx1 := w.client.Worker.FindUnique(
		db.Worker.ID.Equals(workerId),
	).Update(
		db.Worker.GroupKeyRuns.Link(
			db.GetGroupKeyRun.ID.Equals(getGroupKeyRunId),
		),
	).Tx()

	tx2 := w.client.GetGroupKeyRun.FindUnique(
		db.GetGroupKeyRun.ID.Equals(getGroupKeyRunId),
	).Update(
		db.GetGroupKeyRun.Status.Set(db.StepRunStatusAssigned),
	).Tx()

	err := w.client.Prisma.Transaction(tx1, tx2).Exec(context.Background())

	return err
}
