package prisma

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/steebchen/prisma-client-go/runtime/transaction"
)

type workflowRunRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
}

func NewWorkflowRunRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator) repository.WorkflowRunRepository {
	queries := dbsqlc.New()

	return &workflowRunRepository{
		client:  client,
		v:       v,
		pool:    pool,
		queries: queries,
	}
}

func (w *workflowRunRepository) ListWorkflowRuns(tenantId string, opts *repository.ListWorkflowRunsOpts) (*repository.ListWorkflowRunsResult, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	res := &repository.ListWorkflowRunsResult{}

	pgTenantId := &pgtype.UUID{}

	if err := pgTenantId.Scan(tenantId); err != nil {
		return nil, err
	}

	queryParams := dbsqlc.ListWorkflowRunsParams{
		TenantId: *pgTenantId,
	}

	countParams := dbsqlc.CountWorkflowRunsParams{
		TenantId: *pgTenantId,
	}

	if opts.Offset != nil {
		queryParams.Offset = *opts.Offset
	}

	if opts.Limit != nil {
		queryParams.Limit = *opts.Limit
	}

	if opts.WorkflowId != nil {
		pgWorkflowId := sqlchelpers.UUIDFromStr(*opts.WorkflowId)

		queryParams.WorkflowId = pgWorkflowId
		countParams.WorkflowId = pgWorkflowId
	}

	if opts.EventId != nil {
		pgEventId := sqlchelpers.UUIDFromStr(*opts.EventId)

		queryParams.EventId = pgEventId
		countParams.EventId = pgEventId
	}

	orderByField := "createdAt"
	orderByDirection := "DESC"

	queryParams.Orderby = orderByField + " " + orderByDirection

	tx, err := w.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer tx.Rollback(context.Background())

	workflowRuns, err := w.queries.ListWorkflowRuns(context.Background(), tx, queryParams)

	if err != nil {
		return nil, err
	}

	count, err := w.queries.CountWorkflowRuns(context.Background(), tx, countParams)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	res.Rows = workflowRuns
	res.Count = int(count)

	return res, nil
}

func (w *workflowRunRepository) CreateNewWorkflowRun(tenantId string, opts *repository.CreateWorkflowRunOpts) (*db.WorkflowRunModel, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	// begin a transaction
	workflowRunId := uuid.New().String()

	txs := []transaction.Param{
		w.client.WorkflowRun.CreateOne(
			db.WorkflowRun.Tenant.Link(
				db.Tenant.ID.Equals(tenantId),
			),
			db.WorkflowRun.WorkflowVersion.Link(
				db.WorkflowVersion.ID.Equals(opts.WorkflowVersionId),
			),
			db.WorkflowRun.ID.Set(workflowRunId),
		).Tx(),
	}

	triggerOptionals := []db.WorkflowRunTriggeredBySetParam{}

	if opts.TriggeringEventId != nil {
		triggerOptionals = append(triggerOptionals, db.WorkflowRunTriggeredBy.Event.Link(
			db.Event.ID.Equals(*opts.TriggeringEventId),
		),
		)
	}

	if opts.Cron != nil && opts.CronParentId != nil {
		triggerOptionals = append(triggerOptionals, db.WorkflowRunTriggeredBy.Cron.Link(
			db.WorkflowTriggerCronRef.ParentIDCron(
				db.WorkflowTriggerCronRef.ParentID.Equals(*opts.CronParentId),
				db.WorkflowTriggerCronRef.Cron.Equals(*opts.Cron),
			),
		))
	}

	if opts.ScheduledWorkflowId != nil {
		triggerOptionals = append(triggerOptionals, db.WorkflowRunTriggeredBy.Scheduled.Link(
			db.WorkflowTriggerScheduledRef.ID.Equals(*opts.ScheduledWorkflowId),
		))
	}

	// create the trigger definition
	txs = append(txs, w.client.WorkflowRunTriggeredBy.CreateOne(
		db.WorkflowRunTriggeredBy.Tenant.Link(
			db.Tenant.ID.Equals(tenantId),
		),
		db.WorkflowRunTriggeredBy.Parent.Link(
			db.WorkflowRun.ID.Equals(workflowRunId),
		),
		triggerOptionals...,
	).Tx())

	// create the child jobs
	for _, jobOpts := range opts.JobRuns {
		jobRunId := uuid.New().String()

		requeueAfter := time.Now().UTC().Add(5 * time.Second)

		if jobOpts.RequeueAfter != nil {
			requeueAfter = *jobOpts.RequeueAfter
		}

		txs = append(txs, w.client.JobRun.CreateOne(
			db.JobRun.Tenant.Link(
				db.Tenant.ID.Equals(tenantId),
			),
			db.JobRun.WorkflowRun.Link(
				db.WorkflowRun.ID.Equals(workflowRunId),
			),
			db.JobRun.Job.Link(
				db.Job.ID.Equals(jobOpts.JobId),
			),
			db.JobRun.ID.Set(jobRunId),
		).Tx())

		txs = append(txs, w.client.JobRunLookupData.CreateOne(
			db.JobRunLookupData.JobRun.Link(
				db.JobRun.ID.Equals(jobRunId),
			),

			db.JobRunLookupData.Tenant.Link(
				db.Tenant.ID.Equals(tenantId),
			),
			db.JobRunLookupData.Data.SetIfPresent(jobOpts.Input),
		).Tx())

		// create the workflow job step runs
		var prev *string

		for _, stepOpts := range jobOpts.StepRuns {
			stepRunId := uuid.New().String()

			optionals := []db.StepRunSetParam{
				db.StepRun.ID.Set(stepRunId),
				db.StepRun.RequeueAfter.Set(requeueAfter),
			}

			if prev != nil {
				optionals = append(optionals, db.StepRun.Prev.Link(
					db.StepRun.ID.Equals(*prev),
				))
			}

			txs = append(txs, w.client.StepRun.CreateOne(
				db.StepRun.Tenant.Link(
					db.Tenant.ID.Equals(tenantId),
				),
				db.StepRun.JobRun.Link(
					db.JobRun.ID.Equals(jobRunId),
				),
				db.StepRun.Step.Link(
					db.Step.ID.Equals(stepOpts.StepId),
				),
				optionals...,
			).Tx())

			prev = &stepRunId
		}
	}

	err := w.client.Prisma.Transaction(txs...).Exec(context.Background())

	if err != nil {
		return nil, err
	}

	return w.client.WorkflowRun.FindUnique(
		db.WorkflowRun.ID.Equals(workflowRunId),
	).With(
		defaultWorkflowRunPopulator()...,
	).Exec(context.Background())
}

func (w *workflowRunRepository) GetWorkflowRunById(tenantId, id string) (*db.WorkflowRunModel, error) {
	return w.client.WorkflowRun.FindUnique(
		db.WorkflowRun.TenantIDID(
			db.WorkflowRun.TenantID.Equals(tenantId),
			db.WorkflowRun.ID.Equals(id),
		),
	).With(
		defaultWorkflowRunPopulator()...,
	).Exec(context.Background())
}

func defaultWorkflowRunPopulator() []db.WorkflowRunRelationWith {
	return []db.WorkflowRunRelationWith{
		db.WorkflowRun.WorkflowVersion.Fetch().With(
			db.WorkflowVersion.Workflow.Fetch(),
		),
		db.WorkflowRun.TriggeredBy.Fetch().With(
			db.WorkflowRunTriggeredBy.Event.Fetch(),
			db.WorkflowRunTriggeredBy.Cron.Fetch(),
		),
		db.WorkflowRun.JobRuns.Fetch().With(
			db.JobRun.Job.Fetch().With(
				db.Job.Steps.Fetch(),
			),
			db.JobRun.StepRuns.Fetch().With(
				db.StepRun.Step.Fetch().With(
					db.Step.Action.Fetch(),
				),
			),
		),
	}
}
