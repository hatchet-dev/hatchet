package prisma

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type workflowRunRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewWorkflowRunRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.WorkflowRunRepository {
	queries := dbsqlc.New()

	return &workflowRunRepository{
		client:  client,
		v:       v,
		pool:    pool,
		queries: queries,
		l:       l,
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

	defer deferRollback(context.Background(), w.l, tx.Rollback)

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

func (w *workflowRunRepository) CreateNewWorkflowRun(ctx context.Context, tenantId string, opts *repository.CreateWorkflowRunOpts) (*db.WorkflowRunModel, error) {
	ctx, span := telemetry.NewSpan(ctx, "db-create-new-workflow-run")
	defer span.End()

	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	sqlcWorkflowRun, err := func() (*dbsqlc.WorkflowRun, error) {
		tx1Ctx, tx1Span := telemetry.NewSpan(ctx, "db-create-new-workflow-run-tx")
		defer tx1Span.End()

		// begin a transaction
		workflowRunId := uuid.New().String()

		tx, err := w.pool.Begin(tx1Ctx)

		if err != nil {
			return nil, err
		}

		defer deferRollback(context.Background(), w.l, tx.Rollback)

		pgWorkflowRunId := sqlchelpers.UUIDFromStr(workflowRunId)
		pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

		// create a workflow
		sqlcWorkflowRun, err := w.queries.CreateWorkflowRun(
			tx1Ctx,
			tx,
			dbsqlc.CreateWorkflowRunParams{
				ID:                pgWorkflowRunId,
				Tenantid:          pgTenantId,
				Workflowversionid: sqlchelpers.UUIDFromStr(opts.WorkflowVersionId),
			},
		)

		if err != nil {
			return nil, err
		}

		var (
			eventId, cronParentId, scheduledWorkflowId pgtype.UUID
			cronId                                     pgtype.Text
		)

		if opts.TriggeringEventId != nil {
			eventId = sqlchelpers.UUIDFromStr(*opts.TriggeringEventId)
		}

		if opts.CronParentId != nil {
			cronParentId = sqlchelpers.UUIDFromStr(*opts.CronParentId)
		}

		if opts.Cron != nil {
			cronId = sqlchelpers.TextFromStr(*opts.Cron)
		}

		if opts.ScheduledWorkflowId != nil {
			scheduledWorkflowId = sqlchelpers.UUIDFromStr(*opts.ScheduledWorkflowId)
		}

		_, err = w.queries.CreateWorkflowRunTriggeredBy(
			tx1Ctx,
			tx,
			dbsqlc.CreateWorkflowRunTriggeredByParams{
				Tenantid:      pgTenantId,
				Workflowrunid: sqlcWorkflowRun.ID,
				EventId:       eventId,
				CronParentId:  cronParentId,
				Cron:          cronId,
				ScheduledId:   scheduledWorkflowId,
			},
		)

		if err != nil {
			return nil, err
		}

		// create the child jobs
		for _, jobOpts := range opts.JobRuns {
			jobRunId := uuid.New().String()

			requeueAfter := time.Now().UTC().Add(5 * time.Second)

			if jobOpts.RequeueAfter != nil {
				requeueAfter = *jobOpts.RequeueAfter
			}

			sqlcJobRun, err := w.queries.CreateJobRun(
				tx1Ctx,
				tx,
				dbsqlc.CreateJobRunParams{
					ID:            sqlchelpers.UUIDFromStr(jobRunId),
					Tenantid:      pgTenantId,
					Workflowrunid: sqlcWorkflowRun.ID,
					Jobid:         sqlchelpers.UUIDFromStr(jobOpts.JobId),
				},
			)

			if err != nil {
				return nil, err
			}

			lookupParams := dbsqlc.CreateJobRunLookupDataParams{
				Tenantid:    pgTenantId,
				Jobrunid:    sqlcJobRun.ID,
				Triggeredby: jobOpts.TriggeredBy,
			}

			if jobOpts.InputData != nil {
				lookupParams.Input = jobOpts.InputData
			}

			// create the job run lookup data
			_, err = w.queries.CreateJobRunLookupData(
				tx1Ctx,
				tx,
				lookupParams,
			)

			if err != nil {
				return nil, err
			}

			// create the workflow job step runs
			for _, stepOpts := range jobOpts.StepRuns {
				stepRunId := uuid.New().String()

				_, err := w.queries.CreateStepRun(
					tx1Ctx,
					tx,
					dbsqlc.CreateStepRunParams{
						ID:           sqlchelpers.UUIDFromStr(stepRunId),
						Tenantid:     pgTenantId,
						Jobrunid:     sqlcJobRun.ID,
						Stepid:       sqlchelpers.UUIDFromStr(stepOpts.StepId),
						Requeueafter: sqlchelpers.TimestampFromTime(requeueAfter),
					},
				)

				if err != nil {
					return nil, err
				}
			}

			// link all step runs with correct parents/children
			err = w.queries.LinkStepRunParents(
				tx1Ctx,
				tx,
				sqlcJobRun.ID,
			)

			if err != nil {
				return nil, err
			}
		}

		err = tx.Commit(tx1Ctx)

		if err != nil {
			return nil, err
		}

		return sqlcWorkflowRun, nil
	}()

	if err != nil {
		return nil, err
	}

	tx2Ctx, tx2Span := telemetry.NewSpan(ctx, "db-create-new-workflow-run-tx2")
	defer tx2Span.End()

	res, err := w.client.WorkflowRun.FindUnique(
		db.WorkflowRun.ID.Equals(sqlcWorkflowRun.ID),
	).With(
		defaultWorkflowRunPopulator()...,
	).Exec(tx2Ctx)

	if err != nil {
		return nil, err
	}

	return res, nil
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
