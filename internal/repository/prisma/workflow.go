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
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlctoprisma"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/internal/validator"
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
	return r.client.Workflow.FindUnique(
		db.Workflow.ID.Equals(workflowId),
	).With(
		defaultWorkflowPopulator()...,
	).Exec(context.Background())
}

func (r *workflowAPIRepository) GetWorkflowByName(tenantId, workflowName string) (*db.WorkflowModel, error) {
	return r.client.Workflow.FindUnique(
		db.Workflow.TenantIDName(
			db.Workflow.TenantID.Equals(tenantId),
			db.Workflow.Name.Equals(workflowName),
		),
	).With(
		defaultWorkflowPopulator()...,
	).Exec(context.Background())
}

func (r *workflowAPIRepository) GetWorkflowVersionById(tenantId, workflowVersionId string) (*db.WorkflowVersionModel, error) {
	return r.client.WorkflowVersion.FindUnique(
		db.WorkflowVersion.ID.Equals(workflowVersionId),
	).With(
		defaultWorkflowVersionPopulator()...,
	).Exec(context.Background())
}

func (r *workflowAPIRepository) DeleteWorkflow(tenantId, workflowId string) (*db.WorkflowModel, error) {
	return r.client.Workflow.FindUnique(
		db.Workflow.ID.Equals(workflowId),
	).With(
		defaultWorkflowPopulator()...,
	).Delete().Exec(context.Background())
}

func (r *workflowAPIRepository) UpsertWorkflowDeploymentConfig(workflowId string, opts *repository.UpsertWorkflowDeploymentConfigOpts) (*db.WorkflowDeploymentConfigModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	// upsert the deployment config
	deploymentConfig, err := r.client.WorkflowDeploymentConfig.UpsertOne(
		db.WorkflowDeploymentConfig.WorkflowID.Equals(workflowId),
	).Create(
		db.WorkflowDeploymentConfig.Workflow.Link(
			db.Workflow.ID.Equals(workflowId),
		),
		db.WorkflowDeploymentConfig.GitRepoName.Set(opts.GitRepoName),
		db.WorkflowDeploymentConfig.GitRepoOwner.Set(opts.GitRepoOwner),
		db.WorkflowDeploymentConfig.GitRepoBranch.Set(opts.GitRepoBranch),
		db.WorkflowDeploymentConfig.GithubAppInstallation.Link(
			db.GithubAppInstallation.ID.Equals(opts.GithubAppInstallationId),
		),
	).Update(
		db.WorkflowDeploymentConfig.GitRepoName.Set(opts.GitRepoName),
		db.WorkflowDeploymentConfig.GitRepoOwner.Set(opts.GitRepoOwner),
		db.WorkflowDeploymentConfig.GitRepoBranch.Set(opts.GitRepoBranch),
		db.WorkflowDeploymentConfig.GithubAppInstallation.Link(
			db.GithubAppInstallation.ID.Equals(opts.GithubAppInstallationId),
		),
	).Exec(context.Background())

	if err != nil {
		return nil, err
	}

	return deploymentConfig, nil
}

type workflowEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewWorkflowEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.WorkflowEngineRepository {
	queries := dbsqlc.New()

	return &workflowEngineRepository{
		v:       v,
		queries: queries,
		pool:    pool,
		l:       l,
	}
}

func (r *workflowEngineRepository) CreateNewWorkflow(tenantId string, opts *repository.CreateWorkflowVersionOpts) (*dbsqlc.GetWorkflowVersionForEngineRow, error) {
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
	workflow, err := r.queries.GetWorkflowByName(context.Background(), r.pool, dbsqlc.GetWorkflowByNameParams{
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

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), r.l, tx.Rollback)

	workflowId := sqlchelpers.UUIDFromStr(uuid.New().String())
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	// create a workflow
	_, err = r.queries.CreateWorkflow(
		context.Background(),
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
				context.Background(),
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

	workflowVersionId, err := r.createWorkflowVersionTxs(context.Background(), tx, pgTenantId, workflowId, opts)

	if err != nil {
		return nil, err
	}

	workflowVersion, err := r.queries.GetWorkflowVersionForEngine(context.Background(), tx, dbsqlc.GetWorkflowVersionForEngineParams{
		Tenantid: pgTenantId,
		Ids:      []pgtype.UUID{sqlchelpers.UUIDFromStr(workflowVersionId)},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow version: %w", err)
	}

	if len(workflowVersion) != 1 {
		return nil, fmt.Errorf("expected 1 workflow version when creating new, got %d", len(workflowVersion))
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	return workflowVersion[0], nil
}

func (r *workflowEngineRepository) CreateWorkflowVersion(tenantId string, opts *repository.CreateWorkflowVersionOpts) (*dbsqlc.GetWorkflowVersionForEngineRow, error) {
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
	workflow, err := r.queries.GetWorkflowByName(context.Background(), r.pool, dbsqlc.GetWorkflowByNameParams{
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

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), r.l, tx.Rollback)

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	workflowVersionId, err := r.createWorkflowVersionTxs(context.Background(), tx, pgTenantId, workflow.ID, opts)

	if err != nil {
		return nil, err
	}

	workflowVersion, err := r.queries.GetWorkflowVersionForEngine(context.Background(), tx, dbsqlc.GetWorkflowVersionForEngineParams{
		Tenantid: pgTenantId,
		Ids:      []pgtype.UUID{sqlchelpers.UUIDFromStr(workflowVersionId)},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow version: %w", err)
	}

	if len(workflowVersion) != 1 {
		return nil, fmt.Errorf("expected 1 workflow version when creating version, got %d", len(workflowVersion))
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	return workflowVersion[0], nil
}

func (r *workflowEngineRepository) CreateSchedules(
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

	return r.queries.CreateSchedules(context.Background(), r.pool, createParams)
}

func (r *workflowEngineRepository) GetLatestWorkflowVersion(tenantId, workflowId string) (*dbsqlc.GetWorkflowVersionForEngineRow, error) {
	versionId, err := r.queries.GetWorkflowLatestVersion(context.Background(), r.pool, sqlchelpers.UUIDFromStr(workflowId))

	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest version: %w", err)
	}

	versions, err := r.queries.GetWorkflowVersionForEngine(context.Background(), r.pool, dbsqlc.GetWorkflowVersionForEngineParams{
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

func (r *workflowEngineRepository) GetWorkflowByName(tenantId, workflowName string) (*dbsqlc.Workflow, error) {
	return r.queries.GetWorkflowByName(context.Background(), r.pool, dbsqlc.GetWorkflowByNameParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Name:     workflowName,
	})
}

func (r *workflowEngineRepository) GetWorkflowVersionById(tenantId, workflowId string) (*dbsqlc.GetWorkflowVersionForEngineRow, error) {
	versions, err := r.queries.GetWorkflowVersionForEngine(context.Background(), r.pool, dbsqlc.GetWorkflowVersionForEngineParams{
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

	workflows, err := r.queries.GetWorkflowVersionForEngine(context.Background(), r.pool, dbsqlc.GetWorkflowVersionForEngineParams{
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
		context.Background(),
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
			context.Background(),
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
			context.Background(),
			tx,
			params,
		)

		if err != nil {
			return "", fmt.Errorf("could not create concurrency group: %w", err)
		}
	}

	// create the workflow jobs
	for _, jobOpts := range opts.Jobs {
		jobId := uuid.New().String()

		var (
			description, timeout string
		)

		if jobOpts.Description != nil {
			description = *jobOpts.Description
		}

		if jobOpts.Timeout != nil {
			timeout = *jobOpts.Timeout
		}

		sqlcJob, err := r.queries.CreateJob(
			context.Background(),
			tx,
			dbsqlc.CreateJobParams{
				ID:                sqlchelpers.UUIDFromStr(jobId),
				Tenantid:          tenantId,
				Workflowversionid: sqlcWorkflowVersion.ID,
				Name:              jobOpts.Name,
				Description:       description,
				Timeout:           timeout,
			},
		)

		if err != nil {
			return "", err
		}

		for _, stepOpts := range jobOpts.Steps {
			stepId := uuid.New().String()

			var (
				timeout        string
				customUserData []byte
				retries        pgtype.Int4
			)

			if stepOpts.Timeout != nil {
				timeout = *stepOpts.Timeout
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
				context.Background(),
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
				context.Background(),
				tx,
				createStepParams,
			)

			if err != nil {
				return "", err
			}

			if len(stepOpts.Parents) > 0 {
				err := r.queries.AddStepParents(
					context.Background(),
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
		}
	}

	// create the workflow triggers
	workflowTriggersId := uuid.New().String()

	sqlcWorkflowTriggers, err := r.queries.CreateWorkflowTriggers(
		context.Background(),
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
			context.Background(),
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
			context.Background(),
			tx,
			dbsqlc.CreateWorkflowTriggerCronRefParams{
				Workflowtriggersid: sqlcWorkflowTriggers.ID,
				Crontrigger:        cronTrigger,
			},
		)

		if err != nil {
			return "", err
		}
	}

	for _, scheduledTrigger := range opts.ScheduledTriggers {
		_, err := r.queries.CreateWorkflowTriggerScheduledRef(
			context.Background(),
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

func defaultWorkflowPopulator() []db.WorkflowRelationWith {
	return []db.WorkflowRelationWith{
		db.Workflow.Tags.Fetch(),
		db.Workflow.DeploymentConfig.Fetch(),
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
