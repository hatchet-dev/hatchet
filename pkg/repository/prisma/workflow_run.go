package prisma

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/defaults"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type workflowRunAPIRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
	m       *metered.Metered

	callbacks []repository.Callback[*dbsqlc.WorkflowRun]
}

func NewWorkflowRunRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, m *metered.Metered) repository.WorkflowRunAPIRepository {
	queries := dbsqlc.New()

	return &workflowRunAPIRepository{
		client:  client,
		v:       v,
		pool:    pool,
		queries: queries,
		l:       l,
		m:       m,
	}
}

func (w *workflowRunAPIRepository) RegisterCreateCallback(callback repository.Callback[*dbsqlc.WorkflowRun]) {
	if w.callbacks == nil {
		w.callbacks = make([]repository.Callback[*dbsqlc.WorkflowRun], 0)
	}

	w.callbacks = append(w.callbacks, callback)
}

func (w *workflowRunAPIRepository) ListWorkflowRuns(ctx context.Context, tenantId string, opts *repository.ListWorkflowRunsOpts) (*repository.ListWorkflowRunsResult, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	return listWorkflowRuns(ctx, w.pool, w.queries, w.l, tenantId, opts)
}

func (w *workflowRunAPIRepository) WorkflowRunMetricsCount(ctx context.Context, tenantId string, opts *repository.WorkflowRunsMetricsOpts) (*dbsqlc.WorkflowRunsMetricsCountRow, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	return workflowRunMetricsCount(context.Background(), w.pool, w.queries, tenantId, opts)
}

func (w *workflowRunAPIRepository) GetWorkflowRunInputData(tenantId, workflowRunId string) (map[string]interface{}, error) {
	lookupData := datautils.JobRunLookupData{}

	jsonBytes, err := w.queries.GetWorkflowRunInput(
		context.Background(),
		w.pool,
		sqlchelpers.UUIDFromStr(workflowRunId),
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonBytes, &lookupData); err != nil {
		return nil, err
	}

	return lookupData.Input, nil
}

func (w *workflowRunAPIRepository) CreateNewWorkflowRun(ctx context.Context, tenantId string, opts *repository.CreateWorkflowRunOpts) (*dbsqlc.WorkflowRun, error) {
	return metered.MakeMetered(ctx, w.m, dbsqlc.LimitResourceWORKFLOWRUN, tenantId, 1, func() (*string, *dbsqlc.WorkflowRun, error) {
		if err := w.v.Validate(opts); err != nil {
			return nil, nil, err
		}

		workflowRuns, err := createNewWorkflowRuns(ctx, w.pool, w.queries, w.l, tenantId, []*repository.CreateWorkflowRunOpts{opts})

		if err != nil {
			return nil, nil, err
		}
		var id string
		for _, workflowRun := range workflowRuns {

			id = sqlchelpers.UUIDToStr(workflowRun.ID)

			// res, err := w.client.WorkflowRun.FindUnique(
			// 	db.WorkflowRun.ID.Equals(id),
			// ).With(
			// 	defaultWorkflowRunPopulator()...,
			// ).Exec(context.Background())

			// if err != nil {
			// 	return nil, nil, err
			// }

			for _, cb := range w.callbacks {
				cb.Do(workflowRun) // nolint: errcheck
			}

		}
		return &id, workflowRuns[0], nil
	})
}

func (w *workflowRunAPIRepository) GetWorkflowRunById(ctx context.Context, tenantId, id string) (*dbsqlc.GetWorkflowRunByIdRow, error) {
	return w.queries.GetWorkflowRunById(ctx, w.pool, dbsqlc.GetWorkflowRunByIdParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Workflowrunid: sqlchelpers.UUIDFromStr(id),
	})
}

func (w *workflowRunAPIRepository) GetStepsForJobs(ctx context.Context, tenantId string, jobIds []string) ([]*dbsqlc.GetStepsForJobsRow, error) {
	jobIdsPg := make([]pgtype.UUID, len(jobIds))

	for i := range jobIds {
		jobIdsPg[i] = sqlchelpers.UUIDFromStr(jobIds[i])
	}

	return w.queries.GetStepsForJobs(ctx, w.pool, dbsqlc.GetStepsForJobsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Jobids:   jobIdsPg,
	})
}

func (w *workflowRunAPIRepository) GetStepRunsForJobRuns(ctx context.Context, tenantId string, jobRunIds []string) ([]*repository.StepRunForJobRun, error) {
	jobRunIdsPg := make([]pgtype.UUID, len(jobRunIds))

	for i := range jobRunIds {
		jobRunIdsPg[i] = sqlchelpers.UUIDFromStr(jobRunIds[i])
	}

	stepRuns, err := w.queries.GetStepRunsForJobRuns(ctx, w.pool, dbsqlc.GetStepRunsForJobRunsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Jobids:   jobRunIdsPg,
	})

	if err != nil {
		return nil, err
	}

	stepRunIds := make([]pgtype.UUID, len(stepRuns))

	for i, stepRun := range stepRuns {
		stepRunIds[i] = stepRun.ID
	}

	childCounts, err := w.queries.ListChildWorkflowRunCounts(ctx, w.pool, stepRunIds)

	if err != nil {
		return nil, err
	}

	stepRunIdToChildCount := make(map[string]int)

	for _, childCount := range childCounts {
		stepRunIdToChildCount[sqlchelpers.UUIDToStr(childCount.ParentStepRunId)] = int(childCount.Count)
	}

	res := make([]*repository.StepRunForJobRun, len(stepRuns))

	for i, stepRun := range stepRuns {
		childCount := stepRunIdToChildCount[sqlchelpers.UUIDToStr(stepRun.ID)]

		res[i] = &repository.StepRunForJobRun{
			GetStepRunsForJobRunsRow: stepRun,
			ChildWorkflowsCount:      childCount,
		}
	}

	return res, nil
}

type workflowRunEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
	m       *metered.Metered

	callbacks []repository.Callback[*dbsqlc.WorkflowRun]
}

func NewWorkflowRunEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, m *metered.Metered, cbs ...repository.Callback[*dbsqlc.WorkflowRun]) repository.WorkflowRunEngineRepository {
	queries := dbsqlc.New()

	return &workflowRunEngineRepository{
		v:         v,
		pool:      pool,
		queries:   queries,
		l:         l,
		m:         m,
		callbacks: cbs,
	}
}

func (w *workflowRunEngineRepository) RegisterCreateCallback(callback repository.Callback[*dbsqlc.WorkflowRun]) {
	if w.callbacks == nil {
		w.callbacks = make([]repository.Callback[*dbsqlc.WorkflowRun], 0)
	}

	w.callbacks = append(w.callbacks, callback)
}

func (w *workflowRunEngineRepository) GetWorkflowRunById(ctx context.Context, tenantId, id string) (*dbsqlc.GetWorkflowRunRow, error) {
	runs, err := w.queries.GetWorkflowRun(ctx, w.pool, dbsqlc.GetWorkflowRunParams{
		Ids: []pgtype.UUID{
			sqlchelpers.UUIDFromStr(id),
		},
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	if len(runs) != 1 {
		return nil, repository.ErrWorkflowRunNotFound
	}

	return runs[0], nil
}

func (w *workflowRunEngineRepository) GetWorkflowRunAdditionalMeta(ctx context.Context, tenantId, workflowRunId string) (*dbsqlc.GetWorkflowRunAdditionalMetaRow, error) {
	return w.queries.GetWorkflowRunAdditionalMeta(ctx, w.pool, dbsqlc.GetWorkflowRunAdditionalMetaParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Workflowrunid: sqlchelpers.UUIDFromStr(workflowRunId),
	})
}

func (w *workflowRunEngineRepository) ListWorkflowRuns(ctx context.Context, tenantId string, opts *repository.ListWorkflowRunsOpts) (*repository.ListWorkflowRunsResult, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	return listWorkflowRuns(ctx, w.pool, w.queries, w.l, tenantId, opts)
}

func (w *workflowRunEngineRepository) GetChildWorkflowRun(ctx context.Context, parentId, parentStepRunId string, childIndex int, childkey *string) (*dbsqlc.WorkflowRun, error) {
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

	return w.queries.GetChildWorkflowRun(ctx, w.pool, params)
}

func (w *workflowRunEngineRepository) GetScheduledChildWorkflowRun(ctx context.Context, parentId, parentStepRunId string, childIndex int, childkey *string) (*dbsqlc.WorkflowTriggerScheduledRef, error) {
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

	return w.queries.GetScheduledChildWorkflowRun(ctx, w.pool, params)
}

func (w *workflowRunEngineRepository) PopWorkflowRunsRoundRobin(ctx context.Context, tenantId, workflowId string, maxRuns int) ([]*dbsqlc.WorkflowRun, error) {
	return w.queries.PopWorkflowRunsRoundRobin(ctx, w.pool, dbsqlc.PopWorkflowRunsRoundRobinParams{
		Maxruns:    int32(maxRuns),
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Workflowid: sqlchelpers.UUIDFromStr(workflowId),
	})
}

func (w *workflowRunEngineRepository) CreateNewWorkflowRun(ctx context.Context, tenantId string, opts *repository.CreateWorkflowRunOpts) (string, error) {

	wfr, err := metered.MakeMetered(ctx, w.m, dbsqlc.LimitResourceWORKFLOWRUN, tenantId, 1, func() (*string, *dbsqlc.WorkflowRun, error) {

		if err := w.v.Validate(opts); err != nil {
			return nil, nil, err
		}

		wfr, err := createNewWorkflowRun(ctx, w.pool, w.queries, w.l, tenantId, opts)

		if err != nil {
			return nil, nil, err
		}

		id := sqlchelpers.UUIDToStr(wfr.ID)

		return &id, wfr, nil
	})

	if err != nil {
		return "", err
	}

	for _, cb := range w.callbacks {
		cb.Do(wfr) // nolint: errcheck
	}

	id := sqlchelpers.UUIDToStr(wfr.ID)

	return id, nil
}

func (w *workflowRunEngineRepository) ListActiveQueuedWorkflowVersions(ctx context.Context) ([]*dbsqlc.ListActiveQueuedWorkflowVersionsRow, error) {
	return w.queries.ListActiveQueuedWorkflowVersions(ctx, w.pool)
}

func (w *workflowRunEngineRepository) SoftDeleteExpiredWorkflowRuns(ctx context.Context, tenantId string, statuses []dbsqlc.WorkflowRunStatus, before time.Time) (bool, error) {
	paramStatuses := make([]string, 0)

	for _, status := range statuses {
		paramStatuses = append(paramStatuses, string(status))
	}

	hasMore, err := w.queries.SoftDeleteExpiredWorkflowRunsWithDependencies(ctx, w.pool, dbsqlc.SoftDeleteExpiredWorkflowRunsWithDependenciesParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Statuses:      paramStatuses,
		Createdbefore: sqlchelpers.TimestampFromTime(before),
		Limit:         1000,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return hasMore, nil
}

func (s *workflowRunEngineRepository) ReplayWorkflowRun(ctx context.Context, tenantId, workflowRunId string) (*dbsqlc.GetWorkflowRunRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "replay-workflow-run")
	defer span.End()

	err := deadlockRetry(s.l, func() error {
		tx, err := s.pool.Begin(ctx)

		if err != nil {
			return err
		}

		defer deferRollback(ctx, s.l, tx.Rollback)

		pgWorkflowRunId := sqlchelpers.UUIDFromStr(workflowRunId)

		// reset the job run, workflow run and all fields as part of the core tx
		_, err = s.queries.ReplayStepRunResetWorkflowRun(ctx, tx, pgWorkflowRunId)

		if err != nil {
			return fmt.Errorf("error resetting workflow run: %w", err)
		}

		jobRuns, err := s.queries.ListJobRunsForWorkflowRun(ctx, tx, pgWorkflowRunId)

		if err != nil {
			return fmt.Errorf("error listing job runs: %w", err)
		}

		for _, jobRun := range jobRuns {
			_, err = s.queries.ReplayWorkflowRunResetJobRun(ctx, tx, jobRun.ID)

			if err != nil {
				return fmt.Errorf("error resetting job run: %w", err)
			}
		}

		// reset concurrency key
		_, err = s.queries.ReplayWorkflowRunResetGetGroupKeyRun(ctx, tx, pgWorkflowRunId)

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("error resetting get group key run: %w", err)
		}

		// get all step runs for the workflow
		stepRuns, err := s.queries.ListStepRuns(ctx, tx, dbsqlc.ListStepRunsParams{
			TenantId: sqlchelpers.UUIDFromStr(tenantId),
			WorkflowRunIds: []pgtype.UUID{
				sqlchelpers.UUIDFromStr(workflowRunId),
			},
		})

		if err != nil {
			return fmt.Errorf("error listing step runs: %w", err)
		}

		// archive each of the step run results
		for _, stepRunId := range stepRuns {
			stepRunIdStr := sqlchelpers.UUIDToStr(stepRunId)
			err = archiveStepRunResult(ctx, s.queries, tx, tenantId, stepRunIdStr, nil)

			if err != nil {
				return fmt.Errorf("error archiving step run result: %w", err)
			}

			// remove the previous step run result from the job lookup data
			err = s.queries.UpdateJobRunLookupDataWithStepRun(
				ctx,
				tx,
				dbsqlc.UpdateJobRunLookupDataWithStepRunParams{
					Steprunid: stepRunId,
					Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
				},
			)

			if err != nil {
				return fmt.Errorf("error updating job run lookup data: %w", err)
			}

			// create a deferred event for each of these step runs
			sev := dbsqlc.StepRunEventSeverityINFO
			reason := dbsqlc.StepRunEventReasonRETRIEDBYUSER

			defer deferredStepRunEvent(
				s.l,
				s.pool,
				s.queries,
				tenantId,
				stepRunIdStr,
				repository.CreateStepRunEventOpts{
					EventMessage:  repository.StringPtr("Workflow run was replayed, resetting step run result"),
					EventSeverity: &sev,
					EventReason:   &reason,
				},
			)
		}

		// reset all later step runs to a pending state
		_, err = s.queries.ResetStepRunsByIds(ctx, tx, dbsqlc.ResetStepRunsByIdsParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			Ids:      stepRuns,
		})

		if err != nil {
			return fmt.Errorf("error resetting step runs: %w", err)
		}

		err = tx.Commit(ctx)

		return err
	})

	if err != nil {
		return nil, err
	}

	workflowRuns, err := s.queries.GetWorkflowRun(ctx, s.pool, dbsqlc.GetWorkflowRunParams{
		Ids: []pgtype.UUID{
			sqlchelpers.UUIDFromStr(workflowRunId),
		},
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	if len(workflowRuns) != 1 {
		return nil, repository.ErrWorkflowRunNotFound
	}

	return workflowRuns[0], nil
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

	if opts.AdditionalMetadata != nil {
		additionalMetadataBytes, err := json.Marshal(opts.AdditionalMetadata)
		if err != nil {
			return nil, err
		}

		queryParams.AdditionalMetadata = additionalMetadataBytes
		countParams.AdditionalMetadata = additionalMetadataBytes
	}

	if opts.Ids != nil && len(opts.Ids) > 0 {
		pgIds := make([]pgtype.UUID, len(opts.Ids))

		for i, id := range opts.Ids {
			pgIds[i] = sqlchelpers.UUIDFromStr(id)
		}

		queryParams.Ids = pgIds
		countParams.Ids = pgIds
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

	if opts.Statuses != nil {
		statuses := make([]string, 0)

		for _, status := range *opts.Statuses {
			statuses = append(statuses, string(status))
		}

		queryParams.Statuses = statuses
		countParams.Statuses = statuses
	}

	if opts.Kinds != nil {
		kinds := make([]string, 0)

		for _, kind := range *opts.Kinds {
			kinds = append(kinds, string(kind))
		}

		queryParams.Kinds = kinds
		countParams.Kinds = kinds
	}

	if opts.CreatedAfter != nil {
		countParams.CreatedAfter = sqlchelpers.TimestampFromTime(*opts.CreatedAfter)
		queryParams.CreatedAfter = sqlchelpers.TimestampFromTime(*opts.CreatedAfter)
	}

	if opts.CreatedBefore != nil {
		countParams.CreatedBefore = sqlchelpers.TimestampFromTime(*opts.CreatedBefore)
		queryParams.CreatedBefore = sqlchelpers.TimestampFromTime(*opts.CreatedBefore)
	}

	if opts.FinishedAfter != nil {
		countParams.FinishedAfter = sqlchelpers.TimestampFromTime(*opts.FinishedAfter)
		queryParams.FinishedAfter = sqlchelpers.TimestampFromTime(*opts.FinishedAfter)
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
	countParams.Orderby = orderByField + " " + orderByDirection

	tx, err := pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, l, tx.Rollback)

	workflowRuns, err := queries.ListWorkflowRuns(ctx, tx, queryParams)

	if err != nil {
		return nil, err
	}

	count, err := queries.CountWorkflowRuns(ctx, tx, countParams)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, err
	}

	res.Rows = workflowRuns
	res.Count = int(count)

	return res, nil
}

func workflowRunMetricsCount(ctx context.Context, pool *pgxpool.Pool, queries *dbsqlc.Queries, tenantId string, opts *repository.WorkflowRunsMetricsOpts) (*dbsqlc.WorkflowRunsMetricsCountRow, error) {

	pgTenantId := &pgtype.UUID{}

	if err := pgTenantId.Scan(tenantId); err != nil {
		return nil, err
	}

	queryParams := dbsqlc.WorkflowRunsMetricsCountParams{
		Tenantid: *pgTenantId,
	}

	if opts.WorkflowId != nil {
		pgWorkflowId := sqlchelpers.UUIDFromStr(*opts.WorkflowId)

		queryParams.WorkflowId = pgWorkflowId
	}

	if opts.ParentId != nil {
		pgParentId := sqlchelpers.UUIDFromStr(*opts.ParentId)

		queryParams.ParentId = pgParentId
	}

	if opts.ParentStepRunId != nil {
		pgParentStepRunId := sqlchelpers.UUIDFromStr(*opts.ParentStepRunId)

		queryParams.ParentStepRunId = pgParentStepRunId
	}

	if opts.EventId != nil {
		pgEventId := sqlchelpers.UUIDFromStr(*opts.EventId)

		queryParams.EventId = pgEventId
	}

	if opts.CreatedAfter != nil {
		queryParams.CreatedAfter = sqlchelpers.TimestampFromTime(*opts.CreatedAfter)
	}

	if opts.CreatedBefore != nil {
		queryParams.CreatedBefore = sqlchelpers.TimestampFromTime(*opts.CreatedBefore)
	}

	if opts.AdditionalMetadata != nil {
		additionalMetadataBytes, err := json.Marshal(opts.AdditionalMetadata)
		if err != nil {
			return nil, err
		}
		queryParams.AdditionalMetadata = additionalMetadataBytes
	}

	workflowRunsCount, err := queries.WorkflowRunsMetricsCount(ctx, pool, queryParams)

	if err != nil {
		return nil, err
	}

	return workflowRunsCount, nil
}

func createNewWorkflowRun(ctx context.Context, pool *pgxpool.Pool, queries *dbsqlc.Queries, l *zerolog.Logger, tenantId string, opts *repository.CreateWorkflowRunOpts) (*dbsqlc.WorkflowRun, error) {
	ctx, span := telemetry.NewSpan(ctx, "db-create-new-workflow-run")
	defer span.End()

	sqlcWorkflowRun, err := func() (*dbsqlc.WorkflowRun, error) {
		tx1Ctx, tx1Span := telemetry.NewSpan(ctx, "db-create-new-workflow-run-tx")
		defer tx1Span.End()

		// begin a transaction
		workflowRunId := uuid.New().String()

		tx, commit, rollback, err := prepareTx(tx1Ctx, pool, l, 15000)

		if err != nil {
			return nil, err
		}

		defer rollback()

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

		if opts.AdditionalMetadata != nil {
			additionalMetadataBytes, err := json.Marshal(opts.AdditionalMetadata)
			if err != nil {
				return nil, err
			}
			createParams.Additionalmetadata = additionalMetadataBytes

			// if additional metadata contains a "dedupe" key, use it as the dedupe value
			if dedupeValue, ok := opts.AdditionalMetadata["dedupe"]; ok {
				if dedupeStr, ok := dedupeValue.(string); ok {
					opts.DedupeValue = &dedupeStr
				}

				if dedupeInt, ok := dedupeValue.(int); ok {
					dedupeStr := fmt.Sprintf("%d", dedupeInt)
					opts.DedupeValue = &dedupeStr
				}
			}
		}

		if opts.Priority != nil {
			createParams.Priority = pgtype.Int4{
				Int32: *opts.Priority,
				Valid: true,
			}
		}

		// // create the dedupe value
		// if opts.DedupeValue != nil {
		// 	_, err = queries.CreateWorkflowRunDedupe(
		// 		tx1Ctx,
		// 		tx,
		// 		dbsqlc.CreateWorkflowRunDedupeParams{
		// 			Tenantid:          pgTenantId,
		// 			Workflowversionid: sqlchelpers.UUIDFromStr(opts.WorkflowVersionId),
		// 			Value:             sqlchelpers.TextFromStr(*opts.DedupeValue),
		// 			Workflowrunid:     sqlchelpers.UUIDFromStr(workflowRunId),
		// 		},
		// 	)

		// 	if err != nil {
		// 		// if this is a unique violation, return stable error
		// 		if isUniqueViolationOnDedupe(err) {
		// 			return nil, repository.ErrDedupeValueExists{
		// 				DedupeValue: *opts.DedupeValue,
		// 			}
		// 		}

		// 		return nil, err
		// 	}
		// }

		if opts.DedupeValue != nil {
			_, err = queries.CreateMultipleWorkflowRunDedupes(
				tx1Ctx,
				tx,
				dbsqlc.CreateMultipleWorkflowRunDedupesParams{
					Tenantid:           pgTenantId,
					Workflowversionids: []pgtype.UUID{sqlchelpers.UUIDFromStr(opts.WorkflowVersionId)},
					Values:             []string{*opts.DedupeValue},
					Workflowrunids:     []pgtype.UUID{sqlchelpers.UUIDFromStr(workflowRunId)},
				},
			)
			if err != nil {
				// if this is a unique violation, return stable error
				if isUniqueViolationOnDedupe(err) {
					return nil, repository.ErrDedupeValueExists{
						DedupeValue: *opts.DedupeValue,
					}
				}

				return nil, err
			}

		}

		// create a workflow
		// sqlcWorkflowRun, err := queries.CreateWorkflowRun(
		// 	tx1Ctx,
		// 	tx,
		// 	createParams,
		// )

		createRunsParams := dbsqlc.CreateWorkflowRunsParams{
			ID:                 createParams.ID,
			TenantId:           pgTenantId,
			WorkflowVersionId:  createParams.Workflowversionid,
			DisplayName:        createParams.DisplayName,
			ChildIndex:         createParams.ChildIndex,
			ChildKey:           createParams.ChildKey,
			ParentId:           createParams.ParentId,
			ParentStepRunId:    createParams.ParentStepRunId,
			AdditionalMetadata: createParams.Additionalmetadata,
			Priority:           createParams.Priority,
			Status:             "PENDING",
		}

		_, err = queries.CreateWorkflowRuns(
			tx1Ctx,
			tx,
			[]dbsqlc.CreateWorkflowRunsParams{createRunsParams},
		)

		if err != nil {
			return nil, err
		}

		workflowRuns, err := queries.GetInsertedWorkflowRuns(tx1Ctx, tx)

		if err != nil {
			return nil, err
		}

		if len(workflowRuns) == 0 {
			return nil, errors.New("no workflow runs created")
		}

		sqlcWorkflowRun := workflowRuns[0]

		desiredWorkerId := pgtype.UUID{
			Valid: false,
		}

		if opts.DesiredWorkerId != nil {
			desiredWorkerId = sqlchelpers.UUIDFromStr(*opts.DesiredWorkerId)
		}

		// _, err = queries.CreateWorkflowRunStickyState(
		// 	tx1Ctx,
		// 	tx,
		// 	dbsqlc.CreateWorkflowRunStickyStateParams{
		// 		Workflowrunid:     sqlcWorkflowRun.ID,
		// 		Tenantid:          pgTenantId,
		// 		Workflowversionid: createParams.Workflowversionid,
		// 		DesiredWorkerId:   desiredWorkerId,
		// 	},
		// )

		// CreateWorkflowRunStickyState

		// create the sticky state for the workflow run
		// we need a tuple for each stcky state
		// workflow run id, workflow version id, desired worker id
		_, err = queries.CreateMultipleWorkflowRunStickyStates(tx1Ctx, tx, dbsqlc.CreateMultipleWorkflowRunStickyStatesParams{
			Tenantid:           pgTenantId,
			Workflowrunids:     []pgtype.UUID{sqlcWorkflowRun.ID},
			Workflowversionids: []pgtype.UUID{createParams.Workflowversionid},
			Desiredworkerids:   []pgtype.UUID{desiredWorkerId},
		})

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to create workflow run sticky state: %w", err)
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

			// list steps for the job
			steps, err := queries.ListStepsForJob(
				tx1Ctx,
				tx,
				jobRunId,
			)

			if err != nil {
				return nil, err
			}

			stepRunIds := make([]pgtype.UUID, 0)

			for _, step := range steps {
				err = queries.UpsertQueue(
					tx1Ctx,
					tx,
					dbsqlc.UpsertQueueParams{
						Tenantid: pgTenantId,
						Name:     step.ActionId,
					},
				)

				if err != nil {
					return nil, err
				}

				stepRunId, err := queries.CreateStepRun(
					tx1Ctx,
					tx,
					dbsqlc.CreateStepRunParams{
						Tenantid: createParams.Tenantid,
						Jobrunid: jobRunId,
						Stepid:   step.ID,
						Queue:    sqlchelpers.TextFromStr(step.ActionId),
						Priority: createParams.Priority,
					},
				)

				if err != nil {
					return nil, err
				}

				stepRunIds = append(stepRunIds, stepRunId)
			}

			// link all step runs with correct parents/children
			err = queries.LinkStepRunParents(
				tx1Ctx,
				tx,
				stepRunIds,
			)

			if err != nil {
				return nil, err
			}
		}

		err = commit(tx1Ctx)

		if err != nil {
			// check unique violation again on commit, to account for inserts which were uncommitted
			// at the time of the first check
			if isUniqueViolationOnDedupe(err) {
				return nil, repository.ErrDedupeValueExists{
					DedupeValue: *opts.DedupeValue,
				}
			}

			return nil, err
		}

		return sqlcWorkflowRun, nil
	}()

	if err != nil {
		return nil, err
	}

	return sqlcWorkflowRun, nil
}

func createNewWorkflowRuns(ctx context.Context, pool *pgxpool.Pool, queries *dbsqlc.Queries, l *zerolog.Logger, tenantId string, inputOpts []*repository.CreateWorkflowRunOpts) ([]*dbsqlc.WorkflowRun, error) {
	ctx, span := telemetry.NewSpan(ctx, "db-create-new-workflow-run")
	defer span.End()

	sqlcWorkflowRuns, err := func() ([]*dbsqlc.WorkflowRun, error) {
		tx1Ctx, tx1Span := telemetry.NewSpan(ctx, "db-create-new-workflow-run-tx")
		defer tx1Span.End()
		tx, commit, rollback, err := prepareTx(tx1Ctx, pool, l, 15000)
		pgTenantId := sqlchelpers.UUIDFromStr(tenantId)
		var createRunsParams []dbsqlc.CreateWorkflowRunsParams

		type stickyInfo struct {
			workflowRunId     pgtype.UUID
			workflowVersionId pgtype.UUID
			desiredWorkerId   pgtype.UUID
		}

		var stickyInfos []stickyInfo
		var triggeredByParams []dbsqlc.CreateWorkflowRunTriggeredByParams
		var groupKeyParams []dbsqlc.CreateGetGroupKeyRunParams
		var jobRunParams []dbsqlc.CreateJobRunsParams
		var jobRunLookupDataParams []dbsqlc.CreateJobRunLookupDataParams
		var stepRunParams []dbsqlc.CreateStepRunParams

		for _, opt := range inputOpts {

			// begin a transaction
			workflowRunId := uuid.New().String()

			if err != nil {
				return nil, err
			}

			defer rollback()

			createParams := dbsqlc.CreateWorkflowRunParams{
				ID:                sqlchelpers.UUIDFromStr(workflowRunId),
				Tenantid:          pgTenantId,
				Workflowversionid: sqlchelpers.UUIDFromStr(opt.WorkflowVersionId),
			}

			if opt.DisplayName != nil {
				createParams.DisplayName = sqlchelpers.TextFromStr(*opt.DisplayName)
			}

			if opt.ChildIndex != nil {
				// make sure the child index won't overflow
				if *opt.ChildIndex < 0 {
					return nil, errors.New("child index must be greater than or equal to 0")
				}
				if *opt.ChildIndex < math.MinInt32 || *opt.ChildIndex > math.MaxInt32 {
					return nil, errors.New("child index must be within the range of a 32-bit signed integer")
				}
				createParams.ChildIndex = pgtype.Int4{
					Int32: int32(*opt.ChildIndex), // #nosec G115 (not sure why the linter is complaining about this)
					Valid: true,
				}
			}

			if opt.ChildKey != nil {
				createParams.ChildKey = sqlchelpers.TextFromStr(*opt.ChildKey)
			}

			if opt.ParentId != nil {
				createParams.ParentId = sqlchelpers.UUIDFromStr(*opt.ParentId)
			}

			if opt.ParentStepRunId != nil {
				createParams.ParentStepRunId = sqlchelpers.UUIDFromStr(*opt.ParentStepRunId)
			}

			if opt.AdditionalMetadata != nil {
				additionalMetadataBytes, err := json.Marshal(opt.AdditionalMetadata)
				if err != nil {
					return nil, err
				}
				createParams.Additionalmetadata = additionalMetadataBytes

				// if additional metadata contains a "dedupe" key, use it as the dedupe value
				if dedupeValue, ok := opt.AdditionalMetadata["dedupe"]; ok {
					if dedupeStr, ok := dedupeValue.(string); ok {
						opt.DedupeValue = &dedupeStr
					}

					if dedupeInt, ok := dedupeValue.(int); ok {
						dedupeStr := fmt.Sprintf("%d", dedupeInt)
						opt.DedupeValue = &dedupeStr
					}
				}
			}

			if opt.Priority != nil {
				createParams.Priority = pgtype.Int4{
					Int32: *opt.Priority,
					Valid: true,
				}
			}

			if opt.DedupeValue != nil {
				_, err = queries.CreateMultipleWorkflowRunDedupes(
					tx1Ctx,
					tx,
					dbsqlc.CreateMultipleWorkflowRunDedupesParams{
						Tenantid:           pgTenantId,
						Workflowversionids: []pgtype.UUID{sqlchelpers.UUIDFromStr(opt.WorkflowVersionId)},
						Values:             []string{*opt.DedupeValue},
						Workflowrunids:     []pgtype.UUID{sqlchelpers.UUIDFromStr(workflowRunId)},
					},
				)
				if err != nil {
					// if this is a unique violation, return stable error
					if isUniqueViolationOnDedupe(err) {
						return nil, repository.ErrDedupeValueExists{
							DedupeValue: *opt.DedupeValue,
						}
					}

					return nil, err
				}

			}

			crp := dbsqlc.CreateWorkflowRunsParams{
				ID:                 createParams.ID,
				TenantId:           pgTenantId,
				WorkflowVersionId:  createParams.Workflowversionid,
				DisplayName:        createParams.DisplayName,
				ChildIndex:         createParams.ChildIndex,
				ChildKey:           createParams.ChildKey,
				ParentId:           createParams.ParentId,
				ParentStepRunId:    createParams.ParentStepRunId,
				AdditionalMetadata: createParams.Additionalmetadata,
				Priority:           createParams.Priority,
				Status:             "PENDING",
			}

			createRunsParams = append(createRunsParams, crp)
			if opt.DesiredWorkerId != nil {
				stickyInfos = append(stickyInfos, stickyInfo{
					workflowRunId:     sqlchelpers.UUIDFromStr(workflowRunId),
					workflowVersionId: sqlchelpers.UUIDFromStr(opt.WorkflowVersionId),
					desiredWorkerId:   sqlchelpers.UUIDFromStr(*opt.DesiredWorkerId),
				})
			}

			var (
				eventId, cronParentId, scheduledWorkflowId pgtype.UUID
				cronId                                     pgtype.Text
			)

			if opt.TriggeringEventId != nil {
				eventId = sqlchelpers.UUIDFromStr(*opt.TriggeringEventId)
			}

			if opt.CronParentId != nil {
				cronParentId = sqlchelpers.UUIDFromStr(*opt.CronParentId)

			}
			if opt.Cron != nil {
				cronId = sqlchelpers.TextFromStr(*opt.Cron)
			}

			if opt.ScheduledWorkflowId != nil {
				scheduledWorkflowId = sqlchelpers.UUIDFromStr(*opt.ScheduledWorkflowId)
			}

			triggeredByParams = append(triggeredByParams, dbsqlc.CreateWorkflowRunTriggeredByParams{
				Tenantid:      pgTenantId,
				Workflowrunid: sqlchelpers.UUIDFromStr(workflowRunId),
				EventId:       eventId,
				CronParentId:  cronParentId,
				Cron:          cronId,
				ScheduledId:   scheduledWorkflowId,
			})

			if opt.GetGroupKeyRun != nil {
				groupKeyParams = append(groupKeyParams, dbsqlc.CreateGetGroupKeyRunParams{
					Tenantid:          pgTenantId,
					Workflowrunid:     sqlchelpers.UUIDFromStr(workflowRunId),
					Input:             opt.GetGroupKeyRun.Input,
					Requeueafter:      sqlchelpers.TimestampFromTime(time.Now().UTC().Add(5 * time.Second)),
					Scheduletimeoutat: sqlchelpers.TimestampFromTime(time.Now().UTC().Add(defaults.DefaultScheduleTimeout)),
				})
			}

			jobRunParams = append(jobRunParams, dbsqlc.CreateJobRunsParams{
				Tenantid:          pgTenantId,
				Workflowrunid:     sqlchelpers.UUIDFromStr(workflowRunId),
				Workflowversionid: sqlchelpers.UUIDFromStr(opt.WorkflowVersionId),
			})

			lookupParams := dbsqlc.CreateJobRunLookupDataParams{
				Tenantid:    pgTenantId,
				Triggeredby: opt.TriggeredBy,
			}

			if opt.InputData != nil {
				lookupParams.Input = opt.InputData
			}

			jobRunLookupDataParams = append(jobRunLookupDataParams, lookupParams)

			stepRunParams = append(stepRunParams, dbsqlc.CreateStepRunParams{
				Tenantid: pgTenantId,
				Priority: createParams.Priority,
			})

		}
		_, err = queries.CreateWorkflowRuns(
			tx1Ctx,
			tx,
			createRunsParams,
		)

		if err != nil {
			return nil, err
		}

		workflowRuns, err := queries.GetInsertedWorkflowRuns(tx1Ctx, tx)

		if err != nil {
			return nil, err
		}

		if len(workflowRuns) == 0 {
			return nil, errors.New("no workflow runs created")
		}

		if len(stickyInfos) > 0 {

			workflowRunIds := make([]pgtype.UUID, 0)
			workflowVersionIds := make([]pgtype.UUID, 0)
			desiredWorkerIds := make([]pgtype.UUID, 0)

			for _, stickyInfo := range stickyInfos {
				workflowRunIds = append(workflowRunIds, stickyInfo.workflowRunId)
				workflowVersionIds = append(workflowVersionIds, stickyInfo.workflowVersionId)
				desiredWorkerIds = append(desiredWorkerIds, stickyInfo.desiredWorkerId)
			}

			_, err = queries.CreateMultipleWorkflowRunStickyStates(tx1Ctx, tx, dbsqlc.CreateMultipleWorkflowRunStickyStatesParams{
				Tenantid:           pgTenantId,
				Workflowrunids:     workflowRunIds,
				Workflowversionids: workflowVersionIds,
				Desiredworkerids:   desiredWorkerIds,
			})

			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("failed to create workflow run sticky state: %w", err)
			}
		}

		if len(triggeredByParams) > 0 {
			for _, triggeredByParam := range triggeredByParams {

				_, err = queries.CreateWorkflowRunTriggeredBy(
					tx1Ctx,
					tx,
					triggeredByParam,
				)

				if err != nil {
					return nil, err
				}
			}
		}

		if len(groupKeyParams) > 0 {

			for _, params := range groupKeyParams {
				_, err = queries.CreateGetGroupKeyRun(
					tx1Ctx,
					tx,
					params,
				)

				if err != nil {
					return nil, err
				}
			}
		}

		if len(jobRunParams) > 0 {

			for _, jobRunParam := range jobRunParams {
				jobRunIds, err := queries.CreateJobRuns(
					tx1Ctx,
					tx,
					jobRunParam,
				)

				if err != nil {
					return nil, err
				}

				// create the child jobs
				for i, jobRunId := range jobRunIds {

					jobRunLookupDataParams[i].Jobrunid = jobRunId

					// create the job run lookup data
					_, err = queries.CreateJobRunLookupData(
						tx1Ctx,
						tx,
						jobRunLookupDataParams[i],
					)

					if err != nil {
						return nil, err
					}

					// list steps for the job
					steps, err := queries.ListStepsForJob(
						tx1Ctx,
						tx,
						jobRunId,
					)

					if err != nil {
						return nil, err
					}

					stepRunIds := make([]pgtype.UUID, 0)

					for _, step := range steps {
						err = queries.UpsertQueue(
							tx1Ctx,
							tx,
							dbsqlc.UpsertQueueParams{
								Tenantid: pgTenantId,
								Name:     step.ActionId,
							},
						)

						if err != nil {
							return nil, err
						}

						stepRunParams[i].Jobrunid = jobRunId
						stepRunParams[i].Stepid = step.ID
						stepRunParams[i].Queue = sqlchelpers.TextFromStr(step.ActionId)

						stepRunId, err := queries.CreateStepRun(
							tx1Ctx,
							tx,
							stepRunParams[i],
						)

						if err != nil {
							return nil, err
						}

						stepRunIds = append(stepRunIds, stepRunId)
					}

					// link all step runs with correct parents/children
					err = queries.LinkStepRunParents(
						tx1Ctx,
						tx,
						stepRunIds,
					)

					if err != nil {
						return nil, err
					}
				}
			}
		}

		createdWorkflows, err := queries.GetInsertedWorkflowRuns(tx1Ctx, tx)
		if err != nil {
			return nil, err
		}

		err = commit(tx1Ctx)

		if err != nil {
			// check unique violation again on commit, to account for inserts which were uncommitted
			// at the time of the first check

			// TODO verify that this is the correct response - giving a dedupe value but no idea which one
			if isUniqueViolationOnDedupe(err) {
				return nil, repository.ErrDedupeValueExists{
					DedupeValue: *inputOpts[0].DedupeValue,
				}
			}

			return nil, err
		}
		return createdWorkflows, nil
	}()

	if err != nil {
		return nil, err
	}

	return sqlcWorkflowRuns, nil
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

func isUniqueViolationOnDedupe(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "WorkflowRunDedupe_tenantId_workflowId_value_key") &&
		strings.Contains(err.Error(), "SQLSTATE 23505")
}
