package prisma

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/steebchen/prisma-client-go/runtime/transaction"
)

type workerRepository struct {
	client *db.PrismaClient
	v      validator.Validator
}

func NewWorkerRepository(client *db.PrismaClient, v validator.Validator) repository.WorkerRepository {
	return &workerRepository{
		client: client,
		v:      v,
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

func (w *workerRepository) ListRecentWorkerStepRuns(tenantId, workerId string) ([]db.StepRunModel, error) {
	return w.client.StepRun.FindMany(
		db.StepRun.WorkerID.Equals(workerId),
		db.StepRun.TenantID.Equals(tenantId),
	).Take(10).OrderBy(
		db.StepRun.CreatedAt.Order(db.SortOrderDesc),
	).With(
		db.StepRun.Next.Fetch(),
		db.StepRun.Prev.Fetch(),
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

func (w *workerRepository) ListWorkers(tenantId string, opts *repository.ListWorkersOpts) ([]repository.WorkerWithStepCount, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	queryParams := []db.WorkerWhereParam{
		db.Worker.TenantID.Equals(tenantId),
	}

	if opts.Action != nil {
		queryParams = append(queryParams, db.Worker.Actions.Some(
			db.Action.TenantID.Equals(tenantId),
			db.Action.ID.Equals(*opts.Action),
		))
	}

	workers, err := w.client.Worker.FindMany(
		queryParams...,
	).With(
		db.Worker.Dispatcher.Fetch(),
	).Exec(context.Background())

	if err != nil {
		return nil, err
	}

	if len(workers) == 0 {
		return []repository.WorkerWithStepCount{}, nil
	}

	workerIds := make([]string, len(workers))

	for i, worker := range workers {
		workerIds[i] = worker.ID
	}

	var rows []struct {
		ID    string `json:"id"`
		Count string `json:"count"`
	}

	workerIdStrs := make([]string, len(workerIds))

	for i, workerId := range workerIds {
		// verify that the worker id is a valid uuid
		if _, err := uuid.Parse(workerId); err != nil {
			return nil, err
		}

		workerIdStrs[i] = fmt.Sprintf("'%s'", workerId)
	}

	workerIdsStr := strings.Join(workerIdStrs, ",")

	// raw query to get the number of active job runs for each worker
	err = w.client.Prisma.QueryRaw(
		fmt.Sprintf(`
		SELECT "Worker"."id" AS id, COUNT("StepRun"."id") AS count
		FROM "Worker"
		LEFT JOIN "StepRun" ON "StepRun"."workerId" = "Worker"."id" AND "StepRun"."status" = 'RUNNING'
		WHERE "Worker"."tenantId"::text = $1 AND "Worker"."id" IN (%s)
		GROUP BY "Worker"."id"
		`, workerIdsStr),
		tenantId, workerIds,
	).Exec(context.Background(), &rows)

	if err != nil {
		return nil, err
	}

	workerMap := make(map[string]int)

	for _, row := range rows {
		stepCount, err := strconv.ParseInt(row.Count, 10, 64)

		if err == nil {
			workerMap[row.ID] = int(stepCount)
		} else {
			workerMap[row.ID] = 0
		}
	}

	res := make([]repository.WorkerWithStepCount, len(workers))

	for i, worker := range workers {
		workerCp := worker
		res[i] = repository.WorkerWithStepCount{
			Worker:       &workerCp,
			StepRunCount: workerMap[worker.ID],
		}
	}

	return res, nil
}

func (w *workerRepository) CreateNewWorker(tenantId string, opts *repository.CreateWorkerOpts) (*db.WorkerModel, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	txs := []transaction.Param{}

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
				db.Action.TenantIDID(
					db.Action.TenantID.Equals(tenantId),
					db.Action.ID.Equals(action),
				),
			).Create(
				db.Action.ID.Set(action),
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
					db.Action.TenantIDID(
						db.Action.TenantID.Equals(tenantId),
						db.Action.ID.Equals(action),
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

	txs := []transaction.Param{}

	optionals := []db.WorkerSetParam{}

	if opts.Status != nil {
		optionals = append(optionals, db.Worker.Status.Set(*opts.Status))
	}

	if opts.LastHeartbeatAt != nil {
		optionals = append(optionals, db.Worker.LastHeartbeatAt.Set(*opts.LastHeartbeatAt))
	}

	if len(opts.Actions) > 0 {
		for _, action := range opts.Actions {
			txs = append(txs, w.client.Action.UpsertOne(
				db.Action.TenantIDID(
					db.Action.TenantID.Equals(tenantId),
					db.Action.ID.Equals(action),
				),
			).Create(
				db.Action.ID.Set(action),
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
					db.Action.TenantIDID(
						db.Action.TenantID.Equals(tenantId),
						db.Action.ID.Equals(action),
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
