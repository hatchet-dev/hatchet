package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/internal/dagutils"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type workflowAPIRepository struct {
	*sharedRepository
}

func NewWorkflowRepository(shared *sharedRepository) repository.WorkflowAPIRepository {
	return &workflowAPIRepository{
		sharedRepository: shared,
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
		Tenantid: *pgTenantId,
	}

	countParams := dbsqlc.CountWorkflowsParams{
		TenantId: *pgTenantId,
	}

	if opts.Offset != nil {
		queryParams.Offset = *opts.Offset
	}

	if opts.Limit != nil {
		queryParams.Limit = *opts.Limit
	}

	if opts.Name != nil {
		queryParams.Search = pgtype.Text{String: *opts.Name, Valid: true}
	}

	orderByField := "createdAt"
	orderByDirection := "DESC"

	queryParams.Orderby = orderByField + " " + orderByDirection

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(context.Background(), r.l, tx.Rollback)

	workflows, err := r.queries.ListWorkflows(context.Background(), tx, queryParams)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflows: %w", err)
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

	res.Rows = sqlcWorkflows

	return res, nil
}

func (r *workflowAPIRepository) UpdateWorkflow(ctx context.Context, tenantId, workflowId string, opts *repository.UpdateWorkflowOpts) (*dbsqlc.Workflow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	pgWorkflowId := sqlchelpers.UUIDFromStr(workflowId)

	params := dbsqlc.UpdateWorkflowParams{
		ID: pgWorkflowId,
	}

	if opts.IsPaused != nil {
		params.IsPaused = pgtype.Bool{
			Valid: true,
			Bool:  *opts.IsPaused,
		}
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 25000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	workflow, err := r.queries.UpdateWorkflow(ctx, tx, params)

	if err != nil {
		return nil, err
	}

	// if we're setting to an unpaused state, update internal queue items
	if opts.IsPaused != nil && !*opts.IsPaused {
		err = r.queries.HandleWorkflowUnpaused(ctx, tx, dbsqlc.HandleWorkflowUnpausedParams{
			Workflowid: workflowId,
			Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil {
			return nil, err
		}
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return workflow, nil
}

func (r *workflowAPIRepository) GetWorkflowById(ctx context.Context, workflowId string) (*dbsqlc.GetWorkflowByIdRow, error) {
	return r.queries.GetWorkflowById(context.Background(), r.pool, sqlchelpers.UUIDFromStr(workflowId))

}

func (r *workflowAPIRepository) GetWorkflowVersionById(tenantId, workflowVersionId string) (
	*dbsqlc.GetWorkflowVersionByIdRow,
	[]*dbsqlc.WorkflowTriggerCronRef,
	[]*dbsqlc.WorkflowTriggerEventRef,
	[]*dbsqlc.WorkflowTriggerScheduledRef,
	error,
) {
	pgWorkflowVersionId := sqlchelpers.UUIDFromStr(workflowVersionId)

	row, err := r.queries.GetWorkflowVersionById(
		context.Background(),
		r.pool,
		pgWorkflowVersionId,
	)

	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to fetch workflow version: %w", err)
	}

	crons, err := r.queries.GetWorkflowVersionCronTriggerRefs(
		context.Background(),
		r.pool,
		pgWorkflowVersionId,
	)

	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to fetch cron triggers: %w", err)
	}

	events, err := r.queries.GetWorkflowVersionEventTriggerRefs(
		context.Background(),
		r.pool,
		pgWorkflowVersionId,
	)

	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to fetch event triggers: %w", err)
	}

	scheduled, err := r.queries.GetWorkflowVersionScheduleTriggerRefs(
		context.Background(),
		r.pool,
		pgWorkflowVersionId,
	)

	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to fetch scheduled triggers: %w", err)
	}

	return row, crons, events, scheduled, nil
}

func (r *workflowAPIRepository) DeleteWorkflow(ctx context.Context, tenantId, workflowId string) (*dbsqlc.Workflow, error) {
	return r.queries.SoftDeleteWorkflow(ctx, r.pool, sqlchelpers.UUIDFromStr(workflowId))
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

func (w *workflowAPIRepository) ListCronWorkflows(ctx context.Context, tenantId string, opts *repository.ListCronWorkflowsOpts) ([]*dbsqlc.ListCronWorkflowsRow, int64, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	listOpts := dbsqlc.ListCronWorkflowsParams{
		Tenantid: pgTenantId,
	}

	countOpts := dbsqlc.CountCronWorkflowsParams{
		Tenantid: pgTenantId,
	}

	if opts.Limit != nil {
		listOpts.Limit = pgtype.Int4{
			Int32: int32(*opts.Limit), // nolint: gosec
			Valid: true,
		}
	}

	if opts.Offset != nil {
		listOpts.Offset = pgtype.Int4{
			Int32: int32(*opts.Offset), // nolint: gosec
			Valid: true,
		}
	}

	orderByField := "createdAt"

	if opts.OrderBy != nil {
		orderByField = *opts.OrderBy
	}

	orderByDirection := "DESC"

	if opts.OrderDirection != nil {
		orderByDirection = *opts.OrderDirection
	}

	listOpts.Orderby = orderByField + " " + orderByDirection

	if opts.AdditionalMetadata != nil {
		additionalMetadataBytes, err := json.Marshal(opts.AdditionalMetadata)
		if err != nil {
			return nil, 0, err
		}

		listOpts.AdditionalMetadata = additionalMetadataBytes
		countOpts.AdditionalMetadata = additionalMetadataBytes
	}

	if opts.WorkflowId != nil {
		listOpts.Workflowid = sqlchelpers.UUIDFromStr(*opts.WorkflowId)
		countOpts.Workflowid = sqlchelpers.UUIDFromStr(*opts.WorkflowId)
	}

	cronWorkflows, err := w.queries.ListCronWorkflows(ctx, w.pool, listOpts)
	if err != nil {
		return nil, 0, err
	}

	count, err := w.queries.CountCronWorkflows(ctx, w.pool, countOpts)

	if err != nil {
		return nil, count, err
	}

	return cronWorkflows, count, nil
}

func (w *workflowAPIRepository) GetCronWorkflow(ctx context.Context, tenantId, cronWorkflowId string) (*dbsqlc.GetCronWorkflowByIdRow, error) {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)
	cronWorkflow, err := w.queries.GetCronWorkflowById(ctx, w.pool, dbsqlc.GetCronWorkflowByIdParams{
		Tenantid:      pgTenantId,
		Crontriggerid: sqlchelpers.UUIDFromStr(cronWorkflowId),
	})

	if err != nil {
		return nil, err
	}

	return cronWorkflow, nil
}

func (w *workflowAPIRepository) DeleteCronWorkflow(ctx context.Context, tenantId, id string) error {
	return w.queries.DeleteWorkflowTriggerCronRef(ctx, w.pool, sqlchelpers.UUIDFromStr(id))
}

func (w *workflowAPIRepository) CreateCronWorkflow(ctx context.Context, tenantId string, opts *repository.CreateCronWorkflowTriggerOpts) (*dbsqlc.ListCronWorkflowsRow, error) {

	var input, additionalMetadata []byte
	var err error

	if opts.Input != nil {
		input, err = json.Marshal(opts.Input)

		if err != nil {
			return nil, err
		}
	}

	if opts.AdditionalMetadata != nil {
		additionalMetadata, err = json.Marshal(opts.AdditionalMetadata)

		if err != nil {
			return nil, err
		}
	}

	var priority int32 = 1

	if opts.Priority != nil {
		priority = *opts.Priority
	}

	createParams := dbsqlc.CreateWorkflowTriggerCronRefForWorkflowParams{
		Workflowid:         sqlchelpers.UUIDFromStr(opts.WorkflowId),
		Crontrigger:        opts.Cron,
		Name:               sqlchelpers.TextFromStr(opts.Name),
		Input:              input,
		AdditionalMetadata: additionalMetadata,
		Method: dbsqlc.NullWorkflowTriggerCronRefMethods{
			Valid:                         true,
			WorkflowTriggerCronRefMethods: dbsqlc.WorkflowTriggerCronRefMethodsAPI,
		},
		Priority: sqlchelpers.ToInt(priority),
	}

	cronTrigger, err := w.queries.CreateWorkflowTriggerCronRefForWorkflow(ctx, w.pool, createParams)

	if err != nil {
		return nil, err
	}

	row, err := w.queries.ListCronWorkflows(ctx, w.pool, dbsqlc.ListCronWorkflowsParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Crontriggerid: cronTrigger.ID,
		Limit:         1,
	})

	if err != nil {
		return nil, err
	}

	if len(row) == 0 {
		return nil, fmt.Errorf("failed to fetch cron workflow")
	}

	return row[0], nil
}

type workflowEngineRepository struct {
	*sharedRepository
	m *metered.Metered

	cache cache.Cacheable
}

func NewWorkflowEngineRepository(shared *sharedRepository, m *metered.Metered, cache cache.Cacheable) repository.WorkflowEngineRepository {

	return &workflowEngineRepository{
		sharedRepository: shared,
		m:                m,
		cache:            cache,
	}
}

func (r *workflowEngineRepository) CreateNewWorkflow(ctx context.Context, tenantId string, opts *repository.CreateWorkflowVersionOpts) (*dbsqlc.GetWorkflowVersionForEngineRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	// ensure no cycles
	for i, job := range opts.Jobs {
		if dagutils.HasCycle(job.Steps) {
			return nil, &repository.JobRunHasCycleError{
				JobName: job.Name,
			}
		}

		var err error
		opts.Jobs[i].Steps, err = dagutils.OrderWorkflowSteps(job.Steps)

		if err != nil {
			return nil, err
		}
	}

	// preflight check to ensure the workflow doesn't already exist
	_, err := r.queries.GetWorkflowByName(ctx, r.pool, dbsqlc.GetWorkflowByNameParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Name:     opts.Name,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	} else if err == nil {
		return nil, fmt.Errorf(
			"workflow with name '%s' already exists",
			opts.Name,
		)
	}

	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, r.l, tx.Rollback)

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

	workflowVersionId, err := r.createWorkflowVersionTxs(ctx, tx, pgTenantId, workflowId, opts, nil)

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

func (r *workflowEngineRepository) CreateWorkflowVersion(ctx context.Context, tenantId string, opts *repository.CreateWorkflowVersionOpts, oldWorkflowVersion *dbsqlc.GetWorkflowVersionForEngineRow) (*dbsqlc.GetWorkflowVersionForEngineRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	// ensure no cycles
	for i, job := range opts.Jobs {
		if dagutils.HasCycle(job.Steps) {
			return nil, &repository.JobRunHasCycleError{
				JobName: job.Name,
			}
		}

		var err error
		opts.Jobs[i].Steps, err = dagutils.OrderWorkflowSteps(job.Steps)

		if err != nil {
			return nil, err
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

	defer sqlchelpers.DeferRollback(ctx, r.l, tx.Rollback)

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	workflowVersionId, err := r.createWorkflowVersionTxs(ctx, tx, pgTenantId, workflow.ID, opts, oldWorkflowVersion)

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

	var priority int32 = 1

	if opts.Priority != nil {
		priority = *opts.Priority
	}

	createParams := dbsqlc.CreateSchedulesParams{
		Workflowrunid:      sqlchelpers.UUIDFromStr(workflowVersionId),
		Input:              opts.Input,
		Triggertimes:       make([]pgtype.Timestamp, len(opts.ScheduledTriggers)),
		Additionalmetadata: opts.AdditionalMetadata,
		Priority:           sqlchelpers.ToInt(priority),
	}

	for i, scheduledTrigger := range opts.ScheduledTriggers {
		createParams.Triggertimes[i] = sqlchelpers.TimestampFromTime(scheduledTrigger)
	}

	return r.queries.CreateSchedules(ctx, r.pool, createParams)
}

func (r *workflowAPIRepository) CreateScheduledWorkflow(ctx context.Context, tenantId string, opts *repository.CreateScheduledWorkflowRunForWorkflowOpts) (*dbsqlc.ListScheduledWorkflowsRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	var input, additionalMetadata []byte
	var err error

	if opts.Input != nil {
		input, err = json.Marshal(opts.Input)
	}

	if opts.AdditionalMetadata != nil {
		additionalMetadata, err = json.Marshal(opts.AdditionalMetadata)
	}

	if err != nil {
		return nil, err
	}

	var priority int32 = 1

	if opts.Priority != nil {
		priority = *opts.Priority
	}

	createParams := dbsqlc.CreateWorkflowTriggerScheduledRefForWorkflowParams{
		Workflowid:         sqlchelpers.UUIDFromStr(opts.WorkflowId),
		Scheduledtrigger:   sqlchelpers.TimestampFromTime(opts.ScheduledTrigger),
		Input:              input,
		Additionalmetadata: additionalMetadata,
		Method: dbsqlc.NullWorkflowTriggerScheduledRefMethods{
			Valid:                              true,
			WorkflowTriggerScheduledRefMethods: dbsqlc.WorkflowTriggerScheduledRefMethodsAPI,
		},
		Priority: sqlchelpers.ToInt(priority),
	}

	created, err := r.queries.CreateWorkflowTriggerScheduledRefForWorkflow(ctx, r.pool, createParams)

	if err != nil {
		return nil, err
	}

	scheduled, err := r.queries.ListScheduledWorkflows(ctx, r.pool, dbsqlc.ListScheduledWorkflowsParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Scheduleid: created.ID,
	})

	if err != nil {
		return nil, err
	}

	return scheduled[0], nil
}

func (r *workflowEngineRepository) GetLatestWorkflowVersions(ctx context.Context, tenantId string, workflowIds []string) ([]*dbsqlc.GetWorkflowVersionForEngineRow, error) {

	var workflowVersionIds = make([]pgtype.UUID, len(workflowIds))

	for i, id := range workflowIds {
		workflowVersionIds[i] = sqlchelpers.UUIDFromStr(id)
	}

	getLatestWorkflowVersionForWorkflowsParams := dbsqlc.GetLatestWorkflowVersionForWorkflowsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Workflowids: workflowVersionIds,
	}

	versionIds, err := r.queries.GetLatestWorkflowVersionForWorkflows(ctx, r.pool, getLatestWorkflowVersionForWorkflowsParams)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest version: %w", err)
	}

	if len(versionIds) != len(workflowIds) {
		return nil, fmt.Errorf("expected %d workflow version for latest, got %d", len(workflowIds), len(versionIds))
	}

	return r.queries.GetWorkflowVersionForEngine(ctx, r.pool, dbsqlc.GetWorkflowVersionForEngineParams{
		Ids:      versionIds,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})
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

func (r *workflowEngineRepository) GetWorkflowsByNames(ctx context.Context, tenantId string, workflowNames []string) ([]*dbsqlc.Workflow, error) {

	// we need to error if we don't have a workflow for a name

	var distinctNamesMap = make(map[string]string)

	for _, name := range workflowNames {
		distinctNamesMap[name] = name
	}

	var distinctNames []string

	for _, value := range distinctNamesMap {
		distinctNames = append(distinctNames, value)
	}
	results, err := r.queries.GetWorkflowsByNames(ctx, r.pool, dbsqlc.GetWorkflowsByNamesParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Names:    distinctNames,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflows: %w", err)
	}
	if len(results) != len(distinctNames) {
		if len(results) > len(distinctNames) {
			return nil, fmt.Errorf("expected %d workflows, got %d ", len(distinctNames), len(results))

		}
		mismatched := make(map[string]bool)

		for _, result := range results {
			mismatched[result.Name] = true
		}

		var missingNames []string

		for _, name := range distinctNames {
			if _, ok := mismatched[name]; !ok {
				missingNames = append(missingNames, name)
			}

		}

		return nil, fmt.Errorf("expected %d workflows, got %d  - missing '%s'", len(distinctNames), len(results), strings.Join(missingNames, ","))

	}

	return results, nil

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
	cachedArr, err := cache.MakeCacheable(r.cache, fmt.Sprintf("%s-%s", tenantId, eventKey), func() (*[]*dbsqlc.GetWorkflowVersionForEngineRow, error) {
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
				return &[]*dbsqlc.GetWorkflowVersionForEngineRow{}, nil
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

		return &workflows, nil
	})

	if err != nil {
		return nil, err
	}

	return *cachedArr, nil
}

func (r *workflowAPIRepository) GetWorkflowWorkerCount(tenantId, workflowId string) (int, int, error) {
	params := dbsqlc.GetWorkflowWorkerCountParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Workflowid: sqlchelpers.UUIDFromStr(workflowId),
	}

	results, err := r.queries.GetWorkflowWorkerCount(context.Background(), r.pool, params)

	if err != nil {
		return 0, 0, err
	}

	return int(results.Freeslotcount), int(results.Totalslotcount), nil

}

func (r *workflowEngineRepository) createWorkflowVersionTxs(ctx context.Context, tx pgx.Tx, tenantId, workflowId pgtype.UUID, opts *repository.CreateWorkflowVersionOpts, oldWorkflowVersion *dbsqlc.GetWorkflowVersionForEngineRow) (string, error) {
	workflowVersionId := uuid.New().String()

	var version pgtype.Text

	if opts.Version != nil {
		version = sqlchelpers.TextFromStr(*opts.Version)
	}

	cs, err := dagutils.Checksum(opts)

	if err != nil {
		return "", err
	}

	var defaultPriority pgtype.Int4

	if opts.DefaultPriority != nil {
		defaultPriority = pgtype.Int4{
			Valid: true,
			Int32: *opts.DefaultPriority,
		}
	}

	createParams := dbsqlc.CreateWorkflowVersionParams{
		ID:              sqlchelpers.UUIDFromStr(workflowVersionId),
		Checksum:        cs,
		Version:         version,
		Workflowid:      workflowId,
		DefaultPriority: defaultPriority,
	}

	if opts.ScheduleTimeout != nil {
		createParams.ScheduleTimeout = sqlchelpers.TextFromStr(*opts.ScheduleTimeout)
	}

	if opts.Sticky != nil {
		createParams.Sticky = dbsqlc.NullStickyStrategy{
			StickyStrategy: dbsqlc.StickyStrategy(*opts.Sticky),
			Valid:          true,
		}
	}

	if opts.Kind != nil {
		createParams.Kind = dbsqlc.NullWorkflowKind{
			WorkflowKind: dbsqlc.WorkflowKind(*opts.Kind),
			Valid:        true,
		}
	}

	sqlcWorkflowVersion, err := r.queries.CreateWorkflowVersion(
		ctx,
		tx,
		createParams,
	)

	if err != nil {
		return "", err
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

	// create concurrency group
	// NOTE: we do this AFTER the creation of steps/jobs because we have a trigger which depends on the existence
	// of the jobs/steps to create the v1 concurrency groups
	if opts.Concurrency != nil {
		params := dbsqlc.CreateWorkflowConcurrencyParams{
			Workflowversionid: sqlcWorkflowVersion.ID,
		}

		// upsert the action
		if opts.Concurrency.Action != nil {
			action, err := r.queries.UpsertAction(
				ctx,
				tx,
				dbsqlc.UpsertActionParams{
					Action:   *opts.Concurrency.Action,
					Tenantid: tenantId,
				},
			)

			if err != nil {
				return "", fmt.Errorf("could not upsert action: %w", err)
			}

			params.GetConcurrencyGroupId = action.ID
		}

		if opts.Concurrency.Expression != nil {
			params.ConcurrencyGroupExpression = sqlchelpers.TextFromStr(*opts.Concurrency.Expression)
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
		var priority pgtype.Int4

		if opts.DefaultPriority != nil {
			priority = sqlchelpers.ToInt(*opts.DefaultPriority)
		}

		_, err := r.queries.CreateWorkflowTriggerCronRef(
			ctx,
			tx,
			dbsqlc.CreateWorkflowTriggerCronRefParams{
				Workflowtriggersid: sqlcWorkflowTriggers.ID,
				Crontrigger:        cronTrigger,
				Input:              opts.CronInput,
				Name: pgtype.Text{
					String: "",
					Valid:  true,
				},
				Priority: priority,
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

	if oldWorkflowVersion != nil {
		// move existing api crons to the new workflow version
		err = r.queries.MoveCronTriggerToNewWorkflowTriggers(ctx, tx, dbsqlc.MoveCronTriggerToNewWorkflowTriggersParams{
			Oldworkflowversionid: oldWorkflowVersion.WorkflowVersion.ID,
			Newworkflowtriggerid: sqlcWorkflowTriggers.ID,
		})

		if err != nil {
			return "", fmt.Errorf("could not move existing cron triggers to new workflow triggers: %w", err)
		}

		// move existing scheduled triggers to the new workflow version
		err = r.queries.MoveScheduledTriggerToNewWorkflowTriggers(ctx, tx, dbsqlc.MoveScheduledTriggerToNewWorkflowTriggersParams{
			Oldworkflowversionid: oldWorkflowVersion.WorkflowVersion.ID,
			Newworkflowtriggerid: sqlcWorkflowTriggers.ID,
		})

		if err != nil {
			return "", fmt.Errorf("could not move existing scheduled triggers to new workflow triggers: %w", err)
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
				Int32: int32(*stepOpts.Retries), // nolint: gosec
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

		if stepOpts.RetryBackoffFactor != nil {
			createStepParams.RetryBackoffFactor = pgtype.Float8{
				Float64: *stepOpts.RetryBackoffFactor,
				Valid:   true,
			}
		}

		if stepOpts.RetryBackoffMaxSeconds != nil {
			createStepParams.RetryMaxBackoff = pgtype.Int4{
				Int32: int32(*stepOpts.RetryBackoffMaxSeconds), // nolint: gosec
				Valid: true,
			}
		}

		_, err = r.queries.CreateStep(
			ctx,
			tx,
			createStepParams,
		)

		if err != nil {
			return "", err
		}

		if len(stepOpts.DesiredWorkerLabels) > 0 {
			for i := range stepOpts.DesiredWorkerLabels {
				key := (stepOpts.DesiredWorkerLabels)[i].Key
				value := (stepOpts.DesiredWorkerLabels)[i]

				if key == "" {
					continue
				}

				opts := dbsqlc.UpsertDesiredWorkerLabelParams{
					Stepid: sqlchelpers.UUIDFromStr(stepId),
					Key:    key,
				}

				if value.IntValue != nil {
					opts.IntValue = sqlchelpers.ToInt(*value.IntValue)
				}

				if value.StrValue != nil {
					opts.StrValue = sqlchelpers.TextFromStr(*value.StrValue)
				}

				if value.Weight != nil {
					opts.Weight = sqlchelpers.ToInt(*value.Weight)
				}

				if value.Required != nil {
					opts.Required = sqlchelpers.BoolFromBoolean(*value.Required)
				}

				if value.Comparator != nil {
					opts.Comparator = dbsqlc.NullWorkerLabelComparator{
						WorkerLabelComparator: dbsqlc.WorkerLabelComparator(*value.Comparator),
						Valid:                 true,
					}
				}

				_, err = r.queries.UpsertDesiredWorkerLabel(
					ctx,
					tx,
					opts,
				)

				if err != nil {
					return "", err
				}
			}
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
			createStepExprParams := dbsqlc.CreateStepExpressionsParams{
				Stepid: sqlchelpers.UUIDFromStr(stepId),
			}

			for _, rateLimit := range stepOpts.RateLimits {
				// if ANY of the step expressions are not nil, we create ALL options as expressions, but with static
				// keys for any nil expressions.
				if rateLimit.KeyExpr != nil || rateLimit.LimitExpr != nil || rateLimit.UnitsExpr != nil {
					var keyExpr, limitExpr, unitsExpr string

					windowExpr := cel.Str("MINUTE")

					if rateLimit.Duration != nil {
						windowExpr = fmt.Sprintf(`"%s"`, *rateLimit.Duration)
					}

					if rateLimit.KeyExpr != nil {
						keyExpr = *rateLimit.KeyExpr
					} else {
						keyExpr = cel.Str(rateLimit.Key)
					}

					if rateLimit.UnitsExpr != nil {
						unitsExpr = *rateLimit.UnitsExpr
					} else {
						unitsExpr = cel.Int(*rateLimit.Units)
					}

					// create the key expression
					createStepExprParams.Kinds = append(createStepExprParams.Kinds, string(dbsqlc.StepExpressionKindDYNAMICRATELIMITKEY))
					createStepExprParams.Keys = append(createStepExprParams.Keys, rateLimit.Key)
					createStepExprParams.Expressions = append(createStepExprParams.Expressions, keyExpr)

					// create the limit value expression, if it's set
					if rateLimit.LimitExpr != nil {
						limitExpr = *rateLimit.LimitExpr

						createStepExprParams.Kinds = append(createStepExprParams.Kinds, string(dbsqlc.StepExpressionKindDYNAMICRATELIMITVALUE))
						createStepExprParams.Keys = append(createStepExprParams.Keys, rateLimit.Key)
						createStepExprParams.Expressions = append(createStepExprParams.Expressions, limitExpr)
					}

					// create the units value expression
					createStepExprParams.Kinds = append(createStepExprParams.Kinds, string(dbsqlc.StepExpressionKindDYNAMICRATELIMITUNITS))
					createStepExprParams.Keys = append(createStepExprParams.Keys, rateLimit.Key)
					createStepExprParams.Expressions = append(createStepExprParams.Expressions, unitsExpr)

					// create the window expression
					createStepExprParams.Kinds = append(createStepExprParams.Kinds, string(dbsqlc.StepExpressionKindDYNAMICRATELIMITWINDOW))
					createStepExprParams.Keys = append(createStepExprParams.Keys, rateLimit.Key)
					createStepExprParams.Expressions = append(createStepExprParams.Expressions, windowExpr)
				} else {
					_, err := r.queries.CreateStepRateLimit(
						ctx,
						tx,
						dbsqlc.CreateStepRateLimitParams{
							Stepid:       sqlchelpers.UUIDFromStr(stepId),
							Ratelimitkey: rateLimit.Key,
							Units:        int32(*rateLimit.Units), // nolint: gosec
							Tenantid:     tenantId,
							Kind:         dbsqlc.StepRateLimitKindSTATIC,
						},
					)

					if err != nil {
						return "", fmt.Errorf("could not create step rate limit: %w", err)
					}
				}
			}

			if len(createStepExprParams.Kinds) > 0 {
				err := r.queries.CreateStepExpressions(
					ctx,
					tx,
					createStepExprParams,
				)

				if err != nil {
					return "", err
				}
			}
		}
	}

	return jobId, nil
}
