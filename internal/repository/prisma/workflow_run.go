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
	"github.com/hatchet-dev/hatchet/internal/services/shared/defaults"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type workflowRunAPIRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewWorkflowRunRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.WorkflowRunAPIRepository {
	queries := dbsqlc.New()

	return &workflowRunAPIRepository{
		client:  client,
		v:       v,
		pool:    pool,
		queries: queries,
		l:       l,
	}
}

func (w *workflowRunAPIRepository) ListWorkflowRuns(tenantId string, opts *repository.ListWorkflowRunsOpts) (*repository.ListWorkflowRunsResult, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	return listWorkflowRuns(context.Background(), w.pool, w.queries, w.l, tenantId, opts)
}

func (w *workflowRunAPIRepository) CreateNewWorkflowRun(ctx context.Context, tenantId string, opts *repository.CreateWorkflowRunOpts) (*db.WorkflowRunModel, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	workflowRunId, err := createNewWorkflowRun(ctx, w.pool, w.queries, w.l, tenantId, opts)

	if err != nil {
		return nil, err
	}

	res, err := w.client.WorkflowRun.FindUnique(
		db.WorkflowRun.ID.Equals(workflowRunId),
	).With(
		defaultWorkflowRunPopulator()...,
	).Exec(context.Background())

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (w *workflowRunAPIRepository) GetWorkflowRunById(tenantId, id string) (*db.WorkflowRunModel, error) {
	return w.client.WorkflowRun.FindUnique(
		db.WorkflowRun.ID.Equals(id),
	).With(
		defaultWorkflowRunPopulator()...,
	).Exec(context.Background())
}

func (s *workflowRunAPIRepository) CreateWorkflowRunPullRequest(tenantId, workflowRunId string, opts *repository.CreateWorkflowRunPullRequestOpts) (*db.GithubPullRequestModel, error) {
	return s.client.GithubPullRequest.CreateOne(
		db.GithubPullRequest.Tenant.Link(
			db.Tenant.ID.Equals(tenantId),
		),
		db.GithubPullRequest.RepositoryOwner.Set(opts.RepositoryOwner),
		db.GithubPullRequest.RepositoryName.Set(opts.RepositoryName),
		db.GithubPullRequest.PullRequestID.Set(opts.PullRequestID),
		db.GithubPullRequest.PullRequestTitle.Set(opts.PullRequestTitle),
		db.GithubPullRequest.PullRequestNumber.Set(opts.PullRequestNumber),
		db.GithubPullRequest.PullRequestHeadBranch.Set(opts.PullRequestHeadBranch),
		db.GithubPullRequest.PullRequestBaseBranch.Set(opts.PullRequestBaseBranch),
		db.GithubPullRequest.PullRequestState.Set(opts.PullRequestState),
		db.GithubPullRequest.WorkflowRuns.Link(
			db.WorkflowRun.ID.Equals(workflowRunId),
		),
	).Exec(context.Background())
}

func (s *workflowRunAPIRepository) ListPullRequestsForWorkflowRun(tenantId, workflowRunId string, opts *repository.ListPullRequestsForWorkflowRunOpts) ([]db.GithubPullRequestModel, error) {
	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	optionals := []db.GithubPullRequestWhereParam{
		db.GithubPullRequest.WorkflowRuns.Some(
			db.WorkflowRun.ID.Equals(workflowRunId),
			db.WorkflowRun.TenantID.Equals(tenantId),
		),
	}

	if opts.State != nil {
		optionals = append(optionals, db.GithubPullRequest.PullRequestState.Equals(*opts.State))
	}

	return s.client.GithubPullRequest.FindMany(
		optionals...,
	).Exec(context.Background())
}

type workflowRunEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewWorkflowRunEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.WorkflowRunEngineRepository {
	queries := dbsqlc.New()

	return &workflowRunEngineRepository{
		v:       v,
		pool:    pool,
		queries: queries,
		l:       l,
	}
}

func (w *workflowRunEngineRepository) GetWorkflowRunById(tenantId, id string) (*dbsqlc.GetWorkflowRunRow, error) {
	runs, err := w.queries.GetWorkflowRun(context.Background(), w.pool, dbsqlc.GetWorkflowRunParams{
		Ids: []pgtype.UUID{
			sqlchelpers.UUIDFromStr(id),
		},
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	if len(runs) != 1 {
		return nil, errors.New("workflow run not found")
	}

	return runs[0], nil
}

func (w *workflowRunEngineRepository) ListWorkflowRuns(tenantId string, opts *repository.ListWorkflowRunsOpts) (*repository.ListWorkflowRunsResult, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	return listWorkflowRuns(context.Background(), w.pool, w.queries, w.l, tenantId, opts)
}

func (w *workflowRunEngineRepository) GetChildWorkflowRun(parentId, parentStepRunId string, childIndex int, childkey *string) (*dbsqlc.WorkflowRun, error) {
	params := dbsqlc.GetChildWorkflowRunParams{
		Parentid:        sqlchelpers.UUIDFromStr(parentId),
		Parentsteprunid: sqlchelpers.UUIDFromStr(parentStepRunId),
		Childindex: pgtype.Int4{
			Int32: int32(childIndex),
			Valid: true,
		},
	}

	if childkey != nil {
		params.ChildKey = sqlchelpers.TextFromStr(*childkey)
	}

	return w.queries.GetChildWorkflowRun(context.Background(), w.pool, params)
}

func (w *workflowRunEngineRepository) GetScheduledChildWorkflowRun(parentId, parentStepRunId string, childIndex int, childkey *string) (*dbsqlc.WorkflowTriggerScheduledRef, error) {
	params := dbsqlc.GetScheduledChildWorkflowRunParams{
		Parentid:        sqlchelpers.UUIDFromStr(parentId),
		Parentsteprunid: sqlchelpers.UUIDFromStr(parentStepRunId),
		Childindex: pgtype.Int4{
			Int32: int32(childIndex),
			Valid: true,
		},
	}

	if childkey != nil {
		params.ChildKey = sqlchelpers.TextFromStr(*childkey)
	}

	return w.queries.GetScheduledChildWorkflowRun(context.Background(), w.pool, params)
}

func (w *workflowRunEngineRepository) PopWorkflowRunsRoundRobin(tenantId, workflowVersionId string, maxRuns int) ([]*dbsqlc.WorkflowRun, error) {
	pgTenantId := &pgtype.UUID{}

	if err := pgTenantId.Scan(tenantId); err != nil {
		return nil, err
	}

	tx, err := w.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), w.l, tx.Rollback)

	res, err := w.queries.PopWorkflowRunsRoundRobin(context.Background(), tx, dbsqlc.PopWorkflowRunsRoundRobinParams{
		Maxruns:  int32(maxRuns),
		TenantId: *pgTenantId,
		ID:       sqlchelpers.UUIDFromStr(workflowVersionId),
	})

	if err != nil {
		return nil, err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (w *workflowRunEngineRepository) CreateNewWorkflowRun(ctx context.Context, tenantId string, opts *repository.CreateWorkflowRunOpts) (string, error) {
	if err := w.v.Validate(opts); err != nil {
		return "", err
	}

	return createNewWorkflowRun(ctx, w.pool, w.queries, w.l, tenantId, opts)
}

func listWorkflowRuns(ctx context.Context, pool *pgxpool.Pool, queries *dbsqlc.Queries, l *zerolog.Logger, tenantId string, opts *repository.ListWorkflowRunsOpts) (*repository.ListWorkflowRunsResult, error) {
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

	if opts.WorkflowVersionId != nil {
		pgWorkflowVersionId := sqlchelpers.UUIDFromStr(*opts.WorkflowVersionId)

		queryParams.WorkflowVersionId = pgWorkflowVersionId
		countParams.WorkflowVersionId = pgWorkflowVersionId
	}

	if opts.ParentId != nil {
		pgParentId := sqlchelpers.UUIDFromStr(*opts.ParentId)

		queryParams.ParentId = pgParentId
		countParams.ParentId = pgParentId
	}

	if opts.ParentStepRunId != nil {
		pgParentStepRunId := sqlchelpers.UUIDFromStr(*opts.ParentStepRunId)

		queryParams.ParentStepRunId = pgParentStepRunId
		countParams.ParentStepRunId = pgParentStepRunId
	}

	if opts.EventId != nil {
		pgEventId := sqlchelpers.UUIDFromStr(*opts.EventId)

		queryParams.EventId = pgEventId
		countParams.EventId = pgEventId
	}

	if opts.GroupKey != nil {
		queryParams.GroupKey = sqlchelpers.TextFromStr(*opts.GroupKey)
		countParams.GroupKey = sqlchelpers.TextFromStr(*opts.GroupKey)
	}

	if opts.Status != nil {
		var status dbsqlc.NullWorkflowRunStatus

		if err := status.Scan(string(*opts.Status)); err != nil {
			return nil, err
		}

		queryParams.Status = status
		countParams.Status = status
	}

	orderByField := "createdAt"

	if opts.OrderBy != nil {
		orderByField = *opts.OrderBy
	}

	orderByDirection := "DESC"

	if opts.OrderDirection != nil {
		orderByDirection = *opts.OrderDirection
	}

	queryParams.Orderby = orderByField + " " + orderByDirection

	tx, err := pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), l, tx.Rollback)

	workflowRuns, err := queries.ListWorkflowRuns(context.Background(), tx, queryParams)

	if err != nil {
		return nil, err
	}

	count, err := queries.CountWorkflowRuns(context.Background(), tx, countParams)

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

func createNewWorkflowRun(ctx context.Context, pool *pgxpool.Pool, queries *dbsqlc.Queries, l *zerolog.Logger, tenantId string, opts *repository.CreateWorkflowRunOpts) (string, error) {
	ctx, span := telemetry.NewSpan(ctx, "db-create-new-workflow-run")
	defer span.End()

	sqlcWorkflowRun, err := func() (*dbsqlc.WorkflowRun, error) {
		tx1Ctx, tx1Span := telemetry.NewSpan(ctx, "db-create-new-workflow-run-tx")
		defer tx1Span.End()

		// begin a transaction
		workflowRunId := uuid.New().String()

		tx, err := pool.Begin(tx1Ctx)

		if err != nil {
			return nil, err
		}

		defer deferRollback(context.Background(), l, tx.Rollback)

		pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

		createParams := dbsqlc.CreateWorkflowRunParams{
			ID:                sqlchelpers.UUIDFromStr(workflowRunId),
			Tenantid:          pgTenantId,
			Workflowversionid: sqlchelpers.UUIDFromStr(opts.WorkflowVersionId),
		}

		if opts.DisplayName != nil {
			createParams.DisplayName = sqlchelpers.TextFromStr(*opts.DisplayName)
		}

		if opts.ChildIndex != nil {
			createParams.ChildIndex = pgtype.Int4{
				Int32: int32(*opts.ChildIndex),
				Valid: true,
			}
		}

		if opts.ChildKey != nil {
			createParams.ChildKey = sqlchelpers.TextFromStr(*opts.ChildKey)
		}

		if opts.ParentId != nil {
			createParams.ParentId = sqlchelpers.UUIDFromStr(*opts.ParentId)
		}

		if opts.ParentStepRunId != nil {
			createParams.ParentStepRunId = sqlchelpers.UUIDFromStr(*opts.ParentStepRunId)
		}

		// create a workflow
		sqlcWorkflowRun, err := queries.CreateWorkflowRun(
			tx1Ctx,
			tx,
			createParams,
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

		_, err = queries.CreateWorkflowRunTriggeredBy(
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

		requeueAfter := time.Now().UTC().Add(5 * time.Second)

		if opts.GetGroupKeyRun != nil {
			scheduleTimeoutAt := time.Now().UTC().Add(defaults.DefaultScheduleTimeout)

			params := dbsqlc.CreateGetGroupKeyRunParams{
				Tenantid:          pgTenantId,
				Workflowrunid:     sqlcWorkflowRun.ID,
				Input:             opts.GetGroupKeyRun.Input,
				Requeueafter:      sqlchelpers.TimestampFromTime(requeueAfter),
				Scheduletimeoutat: sqlchelpers.TimestampFromTime(scheduleTimeoutAt),
			}

			_, err = queries.CreateGetGroupKeyRun(
				tx1Ctx,
				tx,
				params,
			)

			if err != nil {
				return nil, err
			}
		}

		jobRunIds, err := queries.CreateJobRuns(
			tx1Ctx,
			tx,
			dbsqlc.CreateJobRunsParams{
				Tenantid:          pgTenantId,
				Workflowrunid:     sqlcWorkflowRun.ID,
				Workflowversionid: sqlchelpers.UUIDFromStr(opts.WorkflowVersionId),
			},
		)

		if err != nil {
			return nil, err
		}

		// create the child jobs
		for _, jobRunId := range jobRunIds {
			lookupParams := dbsqlc.CreateJobRunLookupDataParams{
				Tenantid:    pgTenantId,
				Jobrunid:    jobRunId,
				Triggeredby: opts.TriggeredBy,
			}

			if opts.InputData != nil {
				lookupParams.Input = opts.InputData
			}

			// create the job run lookup data
			_, err = queries.CreateJobRunLookupData(
				tx1Ctx,
				tx,
				lookupParams,
			)

			if err != nil {
				return nil, err
			}

			err = queries.CreateStepRuns(
				tx1Ctx,
				tx,
				dbsqlc.CreateStepRunsParams{
					Jobrunid: jobRunId,
					Tenantid: pgTenantId,
				},
			)

			if err != nil {
				return nil, err
			}

			// link all step runs with correct parents/children
			err = queries.LinkStepRunParents(
				tx1Ctx,
				tx,
				jobRunId,
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
		return "", err
	}

	return sqlchelpers.UUIDToStr(sqlcWorkflowRun.ID), nil
}

func defaultWorkflowRunPopulator() []db.WorkflowRunRelationWith {
	return []db.WorkflowRunRelationWith{
		db.WorkflowRun.WorkflowVersion.Fetch().With(
			db.WorkflowVersion.Workflow.Fetch(),
			db.WorkflowVersion.Concurrency.Fetch().With(
				db.WorkflowConcurrency.GetConcurrencyGroup.Fetch(),
			),
		),
		db.WorkflowRun.GetGroupKeyRun.Fetch(),
		db.WorkflowRun.TriggeredBy.Fetch().With(
			db.WorkflowRunTriggeredBy.Event.Fetch(),
			db.WorkflowRunTriggeredBy.Cron.Fetch(),
		),
		db.WorkflowRun.JobRuns.Fetch().With(
			db.JobRun.Job.Fetch().With(
				db.Job.Steps.Fetch().With(
					db.Step.Action.Fetch(),
					db.Step.Parents.Fetch(),
				),
			),
			db.JobRun.StepRuns.Fetch().With(
				db.StepRun.ChildWorkflowRuns.Fetch(),
				db.StepRun.Step.Fetch().With(
					db.Step.Action.Fetch(),
					db.Step.Parents.Fetch(),
				),
			),
		),
	}
}
