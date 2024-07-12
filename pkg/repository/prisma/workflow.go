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

	"github.com/hatchet-dev/hatchet/internal/dagutils"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlctoprisma"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type workflowAPIRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewWorkflowRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.WorkflowAPIRepository {
	queries := dbsqlc.New()

	return &workflowAPIRepository{
		client:  client,
		v:       v,
		queries: queries,
		pool:    pool,
		l:       l,
	}
}

func (r *workflowAPIRepository) ListWorkflows(tenantId string, opts *repository.ListWorkflowsOpts) (*repository.ListWorkflowsResult, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	res := &repository.ListWorkflowsResult{}

	pgTenantId := &pgtype.UUID{}

	if err := pgTenantId.Scan(tenantId); err != nil {
		return nil, err
	}

	queryParams := dbsqlc.ListWorkflowsParams{
		TenantId: *pgTenantId,
	}

	latestRunParams := dbsqlc.ListWorkflowsLatestRunsParams{
		TenantId: *pgTenantId,
	}

	countParams := dbsqlc.CountWorkflowsParams{
		TenantId: *pgTenantId,
	}

	if opts.EventKey != nil {
		pgEventKey := &pgtype.Text{}

		if err := pgEventKey.Scan(*opts.EventKey); err != nil {
			return nil, err
		}

		queryParams.EventKey = *pgEventKey
		countParams.EventKey = *pgEventKey
		latestRunParams.EventKey = *pgEventKey
	}

	if opts.Offset != nil {
		queryParams.Offset = *opts.Offset
	}

	if opts.Limit != nil {
		queryParams.Limit = *opts.Limit
	}

	orderByField := "createdAt"
	orderByDirection := "DESC"

	queryParams.Orderby = orderByField + " " + orderByDirection

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), r.l, tx.Rollback)

	workflows, err := r.queries.ListWorkflows(context.Background(), tx, queryParams)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflows: %w", err)
	}

	latestRuns, err := r.queries.ListWorkflowsLatestRuns(context.Background(), tx, latestRunParams)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest runs: %w", err)
	}

	latestRunsMap := map[string]*dbsqlc.WorkflowRun{}

	for i := range latestRuns {
		uuid := sqlchelpers.UUIDToStr(latestRuns[i].WorkflowId)
		latestRunsMap[uuid] = &latestRuns[i].WorkflowRun
	}

	count, err := r.queries.CountWorkflows(context.Background(), tx, countParams)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	res.Count = int(count)

	sqlcWorkflows := make([]*dbsqlc.Workflow, len(workflows))

	for i := range workflows {
		sqlcWorkflows[i] = &workflows[i].Workflow
	}

	prismaWorkflows := sqlctoprisma.NewConverter[dbsqlc.Workflow, db.WorkflowModel]().ToPrismaList(sqlcWorkflows)

	rows := make([]*repository.ListWorkflowsRow, 0)

	for i, workflow := range prismaWorkflows {

		var prismaRun *db.WorkflowRunModel

		if latestRun, exists := latestRunsMap[workflow.ID]; exists {
			prismaRun = sqlctoprisma.NewConverter[dbsqlc.WorkflowRun, db.WorkflowRunModel]().ToPrisma(latestRun)
		}

		rows = append(rows, &repository.ListWorkflowsRow{
			WorkflowModel: prismaWorkflows[i],
			LatestRun:     prismaRun,
		})
	}

	res.Rows = rows

	return res, nil
}

func (r *workflowAPIRepository) GetWorkflowById(workflowId string) (*db.WorkflowModel, error) {
	return r.client.Workflow.FindFirst(
		db.Workflow.ID.Equals(workflowId),
		db.Workflow.DeletedAt.IsNull(),
	).With(
		defaultWorkflowPopulator()...,
	).Exec(context.Background())
}

func (r *workflowAPIRepository) GetWorkflowByName(tenantId, workflowName string) (*db.WorkflowModel, error) {
	return r.client.Workflow.FindFirst(
		db.Workflow.TenantIDName(
			db.Workflow.TenantID.Equals(tenantId),
			db.Workflow.Name.Equals(workflowName),
		),
		db.Workflow.DeletedAt.IsNull(),
	).With(
		defaultWorkflowPopulator()...,
	).Exec(context.Background())
}

func (r *workflowAPIRepository) GetWorkflowVersionById(tenantId, workflowVersionId string) (*db.WorkflowVersionModel, error) {
	return r.client.WorkflowVersion.FindFirst(
		db.WorkflowVersion.ID.Equals(workflowVersionId),
		db.WorkflowVersion.DeletedAt.IsNull(),
	).With(
		defaultWorkflowVersionPopulator()...,
	).Exec(context.Background())
}

func (r *workflowAPIRepository) DeleteWorkflow(tenantId, workflowId string) (*dbsqlc.Workflow, error) {
	return r.queries.SoftDeleteWorkflow(context.Background(), r.pool, sqlchelpers.UUIDFromStr(workflowId))
}

func (r *workflowAPIRepository) GetWorkflowMetrics(tenantId, workflowId string, opts *repository.GetWorkflowMetricsOpts) (*repository.WorkflowMetrics, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)
	pgWorkflowId := sqlchelpers.UUIDFromStr(workflowId)

	countRunsParams := dbsqlc.CountWorkflowRunsRoundRobinParams{
		Tenantid:   pgTenantId,
		Workflowid: pgWorkflowId,
	}

	countGroupKeysParams := dbsqlc.CountRoundRobinGroupKeysParams{
		Tenantid:   pgTenantId,
		Workflowid: pgWorkflowId,
	}

	if opts.Status != nil {
		status := dbsqlc.NullWorkflowRunStatus{
			Valid:             true,
			WorkflowRunStatus: dbsqlc.WorkflowRunStatus(*opts.Status),
		}

		countRunsParams.Status = status
		countGroupKeysParams.Status = status
	}

	if opts.GroupKey != nil {
		countRunsParams.GroupKey = sqlchelpers.TextFromStr(*opts.GroupKey)
	}

	runsCount, err := r.queries.CountWorkflowRunsRoundRobin(context.Background(), r.pool, countRunsParams)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow run counts: %w", err)
	}

	groupKeysCount, err := r.queries.CountRoundRobinGroupKeys(context.Background(), r.pool, countGroupKeysParams)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch group key counts: %w", err)
	}

	return &repository.WorkflowMetrics{
		GroupKeyRunsCount: int(runsCount),
		GroupKeyCount:     int(groupKeysCount),
	}, nil
}

type workflowEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
	m       *metered.Metered
}

func NewWorkflowEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, m *metered.Metered) repository.WorkflowEngineRepository {
	queries := dbsqlc.New()

	return &workflowEngineRepository{
		v:       v,
		queries: queries,
		pool:    pool,
		l:       l,
		m:       m,
	}
}

func (r *workflowEngineRepository) CreateNewWorkflow(ctx context.Context, tenantId string, opts *repository.CreateWorkflowVersionOpts) (*dbsqlc.GetWorkflowVersionForEngineRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	// ensure no cycles
	for _, job := range opts.Jobs {
		if dagutils.HasCycle(job.Steps) {
			return nil, &repository.JobRunHasCycleError{
				JobName: job.Name,
			}
		}
	}

	// preflight check to ensure the workflow doesn't already exist
	workflow, err := r.queries.GetWorkflowByName(ctx, r.pool, dbsqlc.GetWorkflowByNameParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Name:     opts.Name,
	})

	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	} else if workflow != nil {
		return nil, fmt.Errorf(
			"workflow with name '%s' already exists",
			opts.Name,
		)
	}

	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, r.l, tx.Rollback)

	workflowId := sqlchelpers.UUIDFromStr(uuid.New().String())
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	// create a workflow
	_, err = r.queries.CreateWorkflow(
		ctx,
		tx,
		dbsqlc.CreateWorkflowParams{
			ID:          workflowId,
			Tenantid:    pgTenantId,
			Name:        opts.Name,
			Description: *opts.Description,
		},
	)

	if err != nil {
		return nil, err
	}

	// create any tags
	if len(opts.Tags) > 0 {
		for _, tag := range opts.Tags {
			var tagColor pgtype.Text

			if tag.Color != nil {
				tagColor = sqlchelpers.TextFromStr(*tag.Color)
			}

			err = r.queries.UpsertWorkflowTag(
				ctx,
				tx,
				dbsqlc.UpsertWorkflowTagParams{
					Tenantid: pgTenantId,
					Tagname:  tag.Name,
					TagColor: tagColor,
				},
			)

			if err != nil {
				return nil, err
			}
		}
	}

	workflowVersionId, err := r.createWorkflowVersionTxs(ctx, tx, pgTenantId, workflowId, opts)

	if err != nil {
		return nil, err
	}

	workflowVersion, err := r.queries.GetWorkflowVersionForEngine(ctx, tx, dbsqlc.GetWorkflowVersionForEngineParams{
		Tenantid: pgTenantId,
		Ids:      []pgtype.UUID{sqlchelpers.UUIDFromStr(workflowVersionId)},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow version: %w", err)
	}

	if len(workflowVersion) != 1 {
		return nil, fmt.Errorf("expected 1 workflow version when creating new, got %d", len(workflowVersion))
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, err
	}

	return workflowVersion[0], nil
}

func (r *workflowEngineRepository) CreateWorkflowVersion(ctx context.Context, tenantId string, opts *repository.CreateWorkflowVersionOpts) (*dbsqlc.GetWorkflowVersionForEngineRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	// ensure no cycles
	for _, job := range opts.Jobs {
		if dagutils.HasCycle(job.Steps) {
			return nil, &repository.JobRunHasCycleError{
				JobName: job.Name,
			}
		}
	}

	// preflight check to ensure the workflow already exists
	workflow, err := r.queries.GetWorkflowByName(ctx, r.pool, dbsqlc.GetWorkflowByNameParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Name:     opts.Name,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow: %w", err)
	}

	if workflow == nil {
		return nil, fmt.Errorf(
			"workflow with name '%s' does not exist",
			opts.Name,
		)
	}

	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, r.l, tx.Rollback)

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	workflowVersionId, err := r.createWorkflowVersionTxs(ctx, tx, pgTenantId, workflow.ID, opts)

	if err != nil {
		return nil, err
	}

	workflowVersion, err := r.queries.GetWorkflowVersionForEngine(ctx, tx, dbsqlc.GetWorkflowVersionForEngineParams{
		Tenantid: pgTenantId,
		Ids:      []pgtype.UUID{sqlchelpers.UUIDFromStr(workflowVersionId)},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow version: %w", err)
	}

	if len(workflowVersion) != 1 {
		return nil, fmt.Errorf("expected 1 workflow version when creating version, got %d", len(workflowVersion))
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, err
	}

	return workflowVersion[0], nil
}

func (r *workflowEngineRepository) CreateSchedules(
	ctx context.Context,
	tenantId, workflowVersionId string,
	opts *repository.CreateWorkflowSchedulesOpts,
) ([]*dbsqlc.WorkflowTriggerScheduledRef, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	createParams := dbsqlc.CreateSchedulesParams{
		Workflowrunid: sqlchelpers.UUIDFromStr(workflowVersionId),
		Input:         opts.Input,
		Triggertimes:  make([]pgtype.Timestamp, len(opts.ScheduledTriggers)),
	}

	for i, scheduledTrigger := range opts.ScheduledTriggers {
		createParams.Triggertimes[i] = sqlchelpers.TimestampFromTime(scheduledTrigger)
	}

	return r.queries.CreateSchedules(ctx, r.pool, createParams)
}

func (r *workflowEngineRepository) GetLatestWorkflowVersion(ctx context.Context, tenantId, workflowId string) (*dbsqlc.GetWorkflowVersionForEngineRow, error) {
	versionId, err := r.queries.GetWorkflowLatestVersion(ctx, r.pool, sqlchelpers.UUIDFromStr(workflowId))

	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest version: %w", err)
	}

	versions, err := r.queries.GetWorkflowVersionForEngine(ctx, r.pool, dbsqlc.GetWorkflowVersionForEngineParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Ids:      []pgtype.UUID{versionId},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow version: %w", err)
	}

	if len(versions) != 1 {
		return nil, fmt.Errorf("expected 1 workflow version for latest, got %d", len(versions))
	}

	return versions[0], nil
}

func (r *workflowEngineRepository) GetWorkflowByName(ctx context.Context, tenantId, workflowName string) (*dbsqlc.Workflow, error) {
	return r.queries.GetWorkflowByName(ctx, r.pool, dbsqlc.GetWorkflowByNameParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Name:     workflowName,
	})
}

func (r *workflowEngineRepository) GetWorkflowVersionById(ctx context.Context, tenantId, workflowId string) (*dbsqlc.GetWorkflowVersionForEngineRow, error) {
	versions, err := r.queries.GetWorkflowVersionForEngine(ctx, r.pool, dbsqlc.GetWorkflowVersionForEngineParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Ids:      []pgtype.UUID{sqlchelpers.UUIDFromStr(workflowId)},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow version: %w", err)
	}

	if len(versions) != 1 {
		return nil, fmt.Errorf("expected 1 workflow version when getting by id, got %d", len(versions))
	}

	return versions[0], nil
}

func (r *workflowEngineRepository) ListWorkflowsForEvent(ctx context.Context, tenantId, eventKey string) ([]*dbsqlc.GetWorkflowVersionForEngineRow, error) {
	ctx, span1 := telemetry.NewSpan(ctx, "db-list-workflows-for-event")
	defer span1.End()

	ctx, span2 := telemetry.NewSpan(ctx, "db-list-workflows-for-event-query")
	defer span2.End()

	workflowVersionIds, err := r.queries.ListWorkflowsForEvent(ctx, r.pool, dbsqlc.ListWorkflowsForEventParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Eventkey: eventKey,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*dbsqlc.GetWorkflowVersionForEngineRow{}, nil
		}

		return nil, fmt.Errorf("failed to fetch workflows: %w", err)
	}

	span2.End()

	ctx, span3 := telemetry.NewSpan(ctx, "db-get-workflow-versions-for-engine") // nolint: ineffassign
	defer span3.End()

	workflows, err := r.queries.GetWorkflowVersionForEngine(ctx, r.pool, dbsqlc.GetWorkflowVersionForEngineParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Ids:      workflowVersionIds,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow versions: %w", err)
	}

	return workflows, nil
}

func (r *workflowEngineRepository) createWorkflowVersionTxs(ctx context.Context, tx pgx.Tx, tenantId, workflowId pgtype.UUID, opts *repository.CreateWorkflowVersionOpts) (string, error) {
	workflowVersionId := uuid.New().String()

	var version pgtype.Text

	if opts.Version != nil {
		version = sqlchelpers.TextFromStr(*opts.Version)
	}

	cs, err := opts.Checksum()

	if err != nil {
		return "", err
	}

	createParams := dbsqlc.CreateWorkflowVersionParams{
		ID:         sqlchelpers.UUIDFromStr(workflowVersionId),
		Checksum:   cs,
		Version:    version,
		Workflowid: workflowId,
	}

	if opts.ScheduleTimeout != nil {
		createParams.ScheduleTimeout = sqlchelpers.TextFromStr(*opts.ScheduleTimeout)
	}

	sqlcWorkflowVersion, err := r.queries.CreateWorkflowVersion(
		ctx,
		tx,
		createParams,
	)

	if err != nil {
		return "", err
	}

	// create concurrency group
	if opts.Concurrency != nil {
		// upsert the action
		action, err := r.queries.UpsertAction(
			ctx,
			tx,
			dbsqlc.UpsertActionParams{
				Action:   opts.Concurrency.Action,
				Tenantid: tenantId,
			},
		)

		if err != nil {
			return "", fmt.Errorf("could not upsert action: %w", err)
		}

		params := dbsqlc.CreateWorkflowConcurrencyParams{
			ID:                    sqlchelpers.UUIDFromStr(uuid.New().String()),
			Workflowversionid:     sqlcWorkflowVersion.ID,
			Getconcurrencygroupid: action.ID,
		}

		if opts.Concurrency.MaxRuns != nil {
			params.MaxRuns = sqlchelpers.ToInt(*opts.Concurrency.MaxRuns)
		}

		var ls dbsqlc.ConcurrencyLimitStrategy

		if opts.Concurrency.LimitStrategy != nil && *opts.Concurrency.LimitStrategy != "" {
			ls = dbsqlc.ConcurrencyLimitStrategy(*opts.Concurrency.LimitStrategy)
		} else {
			ls = dbsqlc.ConcurrencyLimitStrategyCANCELINPROGRESS
		}

		params.LimitStrategy = dbsqlc.NullConcurrencyLimitStrategy{
			Valid:                    true,
			ConcurrencyLimitStrategy: ls,
		}

		_, err = r.queries.CreateWorkflowConcurrency(
			ctx,
			tx,
			params,
		)

		if err != nil {
			return "", fmt.Errorf("could not create concurrency group: %w", err)
		}
	}

	// create the workflow jobs
	for _, jobOpts := range opts.Jobs {
		jobCp := jobOpts

		_, err := r.createJobTx(ctx, tx, tenantId, sqlcWorkflowVersion.ID, opts, &jobCp)

		if err != nil {
			return "", err
		}
	}

	// create the onFailure job if exists
	if opts.OnFailureJob != nil {
		onFailureJobCp := *opts.OnFailureJob

		jobId, err := r.createJobTx(ctx, tx, tenantId, sqlcWorkflowVersion.ID, opts, &onFailureJobCp)

		if err != nil {
			return "", err
		}

		_, err = r.queries.LinkOnFailureJob(ctx, tx, dbsqlc.LinkOnFailureJobParams{
			Workflowversionid: sqlcWorkflowVersion.ID,
			Jobid:             sqlchelpers.UUIDFromStr(jobId),
		})

		if err != nil {
			return "", err
		}
	}

	// create the workflow triggers
	workflowTriggersId := uuid.New().String()

	sqlcWorkflowTriggers, err := r.queries.CreateWorkflowTriggers(
		ctx,
		tx,
		dbsqlc.CreateWorkflowTriggersParams{
			ID:                sqlchelpers.UUIDFromStr(workflowTriggersId),
			Workflowversionid: sqlcWorkflowVersion.ID,
			Tenantid:          tenantId,
		},
	)

	if err != nil {
		return "", err
	}

	for _, eventTrigger := range opts.EventTriggers {
		_, err := r.queries.CreateWorkflowTriggerEventRef(
			ctx,
			tx,
			dbsqlc.CreateWorkflowTriggerEventRefParams{
				Workflowtriggersid: sqlcWorkflowTriggers.ID,
				Eventtrigger:       eventTrigger,
			},
		)

		if err != nil {
			return "", err
		}
	}

	for _, cronTrigger := range opts.CronTriggers {

		_, err := r.queries.CreateWorkflowTriggerCronRef(
			ctx,
			tx,
			dbsqlc.CreateWorkflowTriggerCronRefParams{
				Workflowtriggersid: sqlcWorkflowTriggers.ID,
				Crontrigger:        cronTrigger,
				Input:              opts.CronInput,
			},
		)

		if err != nil {
			return "", err
		}

	}

	for _, scheduledTrigger := range opts.ScheduledTriggers {
		_, err := r.queries.CreateWorkflowTriggerScheduledRef(
			ctx,
			tx,
			dbsqlc.CreateWorkflowTriggerScheduledRefParams{
				Workflowversionid: sqlcWorkflowVersion.ID,
				Scheduledtrigger:  sqlchelpers.TimestampFromTime(scheduledTrigger),
			},
		)

		if err != nil {
			return "", err
		}
	}

	return workflowVersionId, nil
}

func (r *workflowEngineRepository) createJobTx(ctx context.Context, tx pgx.Tx, tenantId, workflowVersionId pgtype.UUID, opts *repository.CreateWorkflowVersionOpts, jobOpts *repository.CreateWorkflowJobOpts) (string, error) {
	jobId := uuid.New().String()

	var (
		description, timeout string
	)

	if jobOpts.Description != nil {
		description = *jobOpts.Description
	}

	sqlcJob, err := r.queries.CreateJob(
		ctx,
		tx,
		dbsqlc.CreateJobParams{
			ID:                sqlchelpers.UUIDFromStr(jobId),
			Tenantid:          tenantId,
			Workflowversionid: workflowVersionId,
			Name:              jobOpts.Name,
			Description:       description,
			Timeout:           timeout,
			Kind: dbsqlc.NullJobKind{
				Valid:   true,
				JobKind: dbsqlc.JobKind(jobOpts.Kind),
			},
		},
	)

	if err != nil {
		return "", err
	}

	for _, stepOpts := range jobOpts.Steps {
		stepId := uuid.New().String()

		var (
			timeout        pgtype.Text
			customUserData []byte
			retries        pgtype.Int4
		)

		if stepOpts.Timeout != nil {
			timeout = sqlchelpers.TextFromStr(*stepOpts.Timeout)
		}

		if stepOpts.UserData != nil {
			customUserData = []byte(*stepOpts.UserData)
		}

		if stepOpts.Retries != nil {
			retries = pgtype.Int4{
				Valid: true,
				Int32: int32(*stepOpts.Retries),
			}
		}

		// upsert the action
		_, err := r.queries.UpsertAction(
			ctx,
			tx,
			dbsqlc.UpsertActionParams{
				Action:   stepOpts.Action,
				Tenantid: tenantId,
			},
		)

		if err != nil {
			return "", err
		}

		createStepParams := dbsqlc.CreateStepParams{
			ID:             sqlchelpers.UUIDFromStr(stepId),
			Tenantid:       tenantId,
			Jobid:          sqlchelpers.UUIDFromStr(jobId),
			Actionid:       stepOpts.Action,
			Timeout:        timeout,
			Readableid:     stepOpts.ReadableId,
			CustomUserData: customUserData,
			Retries:        retries,
		}

		if opts.ScheduleTimeout != nil {
			createStepParams.ScheduleTimeout = sqlchelpers.TextFromStr(*opts.ScheduleTimeout)
		}

		_, err = r.queries.CreateStep(
			ctx,
			tx,
			createStepParams,
		)

		if err != nil {
			return "", err
		}

		if len(stepOpts.Parents) > 0 {
			err := r.queries.AddStepParents(
				ctx,
				tx,
				dbsqlc.AddStepParentsParams{
					ID:      sqlchelpers.UUIDFromStr(stepId),
					Parents: stepOpts.Parents,
					Jobid:   sqlcJob.ID,
				},
			)

			if err != nil {
				return "", err
			}
		}

		if len(stepOpts.RateLimits) > 0 {
			for _, rateLimit := range stepOpts.RateLimits {
				_, err := r.queries.CreateStepRateLimit(
					ctx,
					tx,
					dbsqlc.CreateStepRateLimitParams{
						Stepid:       sqlchelpers.UUIDFromStr(stepId),
						Ratelimitkey: rateLimit.Key,
						Units:        int32(rateLimit.Units),
						Tenantid:     tenantId,
					},
				)

				if err != nil {
					return "", err
				}
			}
		}
	}

	return jobId, nil
}

func defaultWorkflowPopulator() []db.WorkflowRelationWith {
	return []db.WorkflowRelationWith{
		db.Workflow.Tags.Fetch(),
		db.Workflow.Versions.Fetch().OrderBy(
			db.WorkflowVersion.Order.Order(db.SortOrderDesc),
		).With(
			defaultWorkflowVersionPopulator()...,
		),
	}
}

func defaultWorkflowVersionPopulator() []db.WorkflowVersionRelationWith {
	return []db.WorkflowVersionRelationWith{
		db.WorkflowVersion.Workflow.Fetch(),
		db.WorkflowVersion.Triggers.Fetch().With(
			db.WorkflowTriggers.Events.Fetch(),
			db.WorkflowTriggers.Crons.Fetch().With(
				db.WorkflowTriggerCronRef.Ticker.Fetch(),
			),
		),
		db.WorkflowVersion.Concurrency.Fetch().With(
			db.WorkflowConcurrency.GetConcurrencyGroup.Fetch(),
		),
		db.WorkflowVersion.Jobs.Fetch().With(
			db.Job.Steps.Fetch().With(
				db.Step.Action.Fetch(),
				db.Step.Parents.Fetch(),
			),
		),
		db.WorkflowVersion.Scheduled.Fetch().With(
			db.WorkflowTriggerScheduledRef.Ticker.Fetch(),
		),
	}
}
