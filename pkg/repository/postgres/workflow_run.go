package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/defaults"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type workflowRunAPIRepository struct {
	*sharedRepository

	m  *metered.Metered
	cf *server.ConfigFileRuntime

	createCallbacks []repository.TenantScopedCallback[*dbsqlc.WorkflowRun]
}

func NewWorkflowRunRepository(shared *sharedRepository, m *metered.Metered, cf *server.ConfigFileRuntime) repository.WorkflowRunAPIRepository {
	return &workflowRunAPIRepository{
		sharedRepository: shared,
		m:                m,
		cf:               cf,
	}
}

func (w *workflowRunAPIRepository) RegisterCreateCallback(callback repository.TenantScopedCallback[*dbsqlc.WorkflowRun]) {
	if w.createCallbacks == nil {
		w.createCallbacks = make([]repository.TenantScopedCallback[*dbsqlc.WorkflowRun], 0)
	}

	w.createCallbacks = append(w.createCallbacks, callback)
}

func (w *workflowRunAPIRepository) ListWorkflowRuns(ctx context.Context, tenantId string, opts *repository.ListWorkflowRunsOpts) (*repository.ListWorkflowRunsResult, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	return w.listWorkflowRuns(ctx, w.pool, tenantId, opts)
}

func (w *workflowRunAPIRepository) WorkflowRunMetricsCount(ctx context.Context, tenantId string, opts *repository.WorkflowRunsMetricsOpts) (*dbsqlc.WorkflowRunsMetricsCountRow, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	return workflowRunMetricsCount(ctx, w.pool, w.queries, tenantId, opts)
}

func (w *workflowRunAPIRepository) ListScheduledWorkflows(ctx context.Context, tenantId string, opts *repository.ListScheduledWorkflowsOpts) ([]*dbsqlc.ListScheduledWorkflowsRow, int64, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	listOpts := dbsqlc.ListScheduledWorkflowsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	}

	countParams := dbsqlc.CountScheduledWorkflowsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	}

	if opts.WorkflowId != nil {
		pgWorkflowId := sqlchelpers.UUIDFromStr(*opts.WorkflowId)

		listOpts.Workflowid = pgWorkflowId
		countParams.Workflowid = pgWorkflowId
	}

	if opts.AdditionalMetadata != nil {
		additionalMetadataBytes, err := json.Marshal(opts.AdditionalMetadata)
		if err != nil {
			return nil, 0, err
		}

		listOpts.AdditionalMetadata = additionalMetadataBytes
		countParams.AdditionalMetadata = additionalMetadataBytes
	}

	if opts.ParentWorkflowRunId != nil {
		pgParentId := sqlchelpers.UUIDFromStr(*opts.ParentWorkflowRunId)

		listOpts.Parentworkflowrunid = pgParentId
		countParams.Parentworkflowrunid = pgParentId
	}

	if opts.ParentStepRunId != nil {
		pgParentStepRunId := sqlchelpers.UUIDFromStr(*opts.ParentStepRunId)

		listOpts.Parentsteprunid = pgParentStepRunId
		countParams.Parentsteprunid = pgParentStepRunId
	}

	if opts.Statuses != nil {
		statuses := make([]string, 0)

		for _, status := range *opts.Statuses {
			if status == "SCHEDULED" {
				listOpts.Includescheduled = true
				countParams.Includescheduled = true
				continue
			}

			statuses = append(statuses, string(status))
		}

		listOpts.Statuses = statuses
		countParams.Statuses = statuses
	}

	count, err := w.queries.CountScheduledWorkflows(ctx, w.pool, countParams)

	if err != nil {
		return nil, 0, err
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

	orderByField := "triggerAt"

	if opts.OrderBy != nil {
		orderByField = *opts.OrderBy
	}

	orderByDirection := "DESC"

	if opts.OrderDirection != nil {
		orderByDirection = *opts.OrderDirection
	}

	listOpts.Orderby = orderByField + " " + orderByDirection

	scheduledWorkflows, err := w.queries.ListScheduledWorkflows(ctx, w.pool, listOpts)
	if err != nil {
		return nil, 0, err
	}

	return scheduledWorkflows, count, nil
}

func (w *workflowRunAPIRepository) DeleteScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string) error {
	return w.queries.DeleteScheduledWorkflow(ctx, w.pool, sqlchelpers.UUIDFromStr(scheduledWorkflowId))
}

func (w *workflowRunAPIRepository) GetScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string) (*dbsqlc.ListScheduledWorkflowsRow, error) {

	listOpts := dbsqlc.ListScheduledWorkflowsParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Scheduleid: sqlchelpers.UUIDFromStr(scheduledWorkflowId),
	}

	scheduledWorkflows, err := w.queries.ListScheduledWorkflows(ctx, w.pool, listOpts)
	if err != nil {
		return nil, err
	}

	if len(scheduledWorkflows) == 0 {
		return nil, nil
	}

	return scheduledWorkflows[0], nil

}

func (w *workflowRunAPIRepository) UpdateScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string, triggerAt time.Time) error {
	return w.queries.UpdateScheduledWorkflow(ctx, w.pool, dbsqlc.UpdateScheduledWorkflowParams{
		Scheduleid: sqlchelpers.UUIDFromStr(scheduledWorkflowId),
		Triggerat:  sqlchelpers.TimestampFromTime(triggerAt),
	})
}

func (w *workflowRunAPIRepository) GetWorkflowRunShape(ctx context.Context, workflowVersionId uuid.UUID) ([]*dbsqlc.GetWorkflowRunShapeRow, error) {
	return w.queries.GetWorkflowRunShape(ctx, w.pool, sqlchelpers.UUIDFromStr(workflowVersionId.String()))
}

func (w *workflowRunEngineRepository) GetWorkflowRunInputData(tenantId, workflowRunId string) (map[string]interface{}, error) {
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
		opts.TenantId = tenantId

		if err := w.v.Validate(opts); err != nil {
			return nil, nil, err
		}
		var wfr *dbsqlc.WorkflowRun
		var err error

		if w.cf.BufferCreateWorkflowRuns {
			wfr, err = w.bulkWorkflowRunBuffer.FireAndWait(ctx, tenantId, opts)

			if err != nil {
				return nil, nil, err
			}
		} else {
			workflowRuns, err := createNewWorkflowRuns(ctx, w.pool, w.queries, w.l, []*repository.CreateWorkflowRunOpts{opts})

			if err != nil {
				return nil, nil, err
			}
			wfr = workflowRuns[0]
		}

		id := sqlchelpers.UUIDToStr(wfr.ID)

		for _, cb := range w.createCallbacks {
			cb.Do(w.l, tenantId, wfr)
		}

		return &id, wfr, nil
	})

}

type updateWorkflowRunQueueData struct {
	WorkflowRunId string `json:"workflow_run_id"`

	Event *repository.CreateStepRunEventOpts `json:"event,omitempty"`
}

func (w *workflowRunEngineRepository) QueuePausedWorkflowRun(ctx context.Context, tenantId, workflowId, workflowRunId string) error {
	return insertPausedWorkflowRunQueueItem(
		ctx,
		w.pool,
		w.queries,
		sqlchelpers.UUIDFromStr(tenantId),
		unpauseWorkflowRunQueueData{
			WorkflowId:    workflowId,
			WorkflowRunId: workflowRunId,
		},
	)
}

func (w *workflowRunEngineRepository) queuePausedWorkflowRunWithTx(ctx context.Context, tx dbsqlc.DBTX, tenantId, workflowId, workflowRunId string) error {
	return insertPausedWorkflowRunQueueItem(
		ctx,
		tx,
		w.queries,
		sqlchelpers.UUIDFromStr(tenantId),
		unpauseWorkflowRunQueueData{
			WorkflowId:    workflowId,
			WorkflowRunId: workflowRunId,
		},
	)
}

func (w *workflowRunEngineRepository) ProcessWorkflowRunUpdates(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-workflow-run-updates-database")
	defer span.End()

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	limit := 100

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, w.pool, w.l, 25000)

	if err != nil {
		return false, err
	}

	defer rollback()

	// list queues
	queueItems, err := w.queries.ListInternalQueueItems(ctx, tx, dbsqlc.ListInternalQueueItemsParams{
		Tenantid: pgTenantId,
		Queue:    dbsqlc.InternalQueueWORKFLOWRUNUPDATE,
		Limit: pgtype.Int4{
			Int32: int32(limit),
			Valid: true,
		},
	})

	if err != nil {
		return false, fmt.Errorf("could not list internal queue items: %w", err)
	}

	data, err := toQueueItemData[updateWorkflowRunQueueData](queueItems)

	if err != nil {
		return false, fmt.Errorf("could not convert internal queue item data to worker semaphore queue data: %w", err)
	}

	eventTimeSeen := make([]pgtype.Timestamp, 0, len(data))
	eventReasons := make([]dbsqlc.StepRunEventReason, 0, len(data))
	eventWorkflowRunIds := make([]pgtype.UUID, 0, len(data))
	eventSeverities := make([]dbsqlc.StepRunEventSeverity, 0, len(data))
	eventMessages := make([]string, 0, len(data))
	eventData := make([]map[string]interface{}, 0, len(data))
	dedupe := make(map[string]bool)

	for _, item := range data {
		workflowRunId := sqlchelpers.UUIDFromStr(item.WorkflowRunId)

		if item.Event.EventMessage == nil || item.Event.EventReason == nil {
			continue
		}

		dedupeKey := fmt.Sprintf("EVENT-%s-%s", item.WorkflowRunId, *item.Event.EventReason)

		if _, ok := dedupe[dedupeKey]; ok {
			continue
		}

		dedupe[dedupeKey] = true

		eventWorkflowRunIds = append(eventWorkflowRunIds, workflowRunId)
		eventMessages = append(eventMessages, *item.Event.EventMessage)
		eventReasons = append(eventReasons, *item.Event.EventReason)

		if item.Event.EventSeverity != nil {
			eventSeverities = append(eventSeverities, *item.Event.EventSeverity)
		} else {
			eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityINFO)
		}

		if item.Event.EventData != nil {
			eventData = append(eventData, item.Event.EventData)
		} else {
			eventData = append(eventData, map[string]interface{}{})
		}

		if item.Event.Timestamp != nil {
			eventTimeSeen = append(eventTimeSeen, sqlchelpers.TimestampFromTime(*item.Event.Timestamp))
		} else {
			eventTimeSeen = append(eventTimeSeen, sqlchelpers.TimestampFromTime(time.Now().UTC()))
		}
	}

	qiIds := make([]int64, 0, len(data))

	for _, item := range queueItems {
		qiIds = append(qiIds, item.ID)
	}

	// update the processed semaphore queue items
	err = w.queries.MarkInternalQueueItemsProcessed(ctx, tx, qiIds)

	if err != nil {
		return false, fmt.Errorf("could not mark worker semaphore queue items processed: %w", err)
	}

	// NOTE: actually not deferred
	bulkWorkflowRunEvents(ctx, w.l, tx, w.queries, eventWorkflowRunIds, eventTimeSeen, eventReasons, eventSeverities, eventMessages, eventData)

	err = commit(ctx)

	if err != nil {
		return false, fmt.Errorf("could not commit transaction: %w", err)
	}

	return len(queueItems) == limit, nil
}

type unpauseWorkflowRunQueueData struct {
	// NOTE: do not change this workflow_id without also changing HandleWorkflowUnpaused,
	// as we've written a query which selects on this field
	WorkflowId    string `json:"workflow_id"`
	WorkflowRunId string `json:"workflow_run_id"`
}

func (w *workflowRunEngineRepository) ProcessUnpausedWorkflowRuns(ctx context.Context, tenantId string) ([]*dbsqlc.GetWorkflowRunRow, bool, error) {

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, w.pool, w.l, 25000)

	if err != nil {
		return nil, false, err
	}

	defer rollback()

	toQueue, res, err := w.processUnpausedWorkflowRunsWithTx(ctx, tx, tenantId)

	if err != nil {
		return nil, false, err
	}

	err = commit(ctx)

	if err != nil {
		return nil, false, fmt.Errorf("could not commit transaction: %w", err)
	}

	return toQueue, res, nil

}
func (w *workflowRunEngineRepository) processUnpausedWorkflowRunsWithTx(ctx context.Context, tx dbsqlc.DBTX, tenantId string) ([]*dbsqlc.GetWorkflowRunRow, bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-workflow-run-updates-database")
	defer span.End()

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	limit := 1000

	// list queues
	queueItems, err := w.queries.ListInternalQueueItems(ctx, tx, dbsqlc.ListInternalQueueItemsParams{
		Tenantid: pgTenantId,
		Queue:    dbsqlc.InternalQueueWORKFLOWRUNPAUSED,
		Limit: pgtype.Int4{
			Int32: int32(limit),
			Valid: true,
		},
	})

	if err != nil {
		return nil, false, fmt.Errorf("could not list internal queue items for paused workflow runs: %w", err)
	}

	if len(queueItems) == 0 {
		return nil, false, nil
	}

	data, err := toQueueItemData[unpauseWorkflowRunQueueData](queueItems)

	if err != nil {
		return nil, false, fmt.Errorf("could not convert internal queue item data to worker semaphore queue data: %w", err)
	}

	// construct a map of workflow IDs
	candidateUnpausedWorkflows := make(map[string]bool)

	for _, item := range data {
		candidateUnpausedWorkflows[item.WorkflowId] = true
	}

	// list paused workflows
	pausedWorkflowIds, err := w.queries.ListPausedWorkflows(ctx, tx, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		return nil, false, fmt.Errorf("could not list paused workflows: %w", err)
	}

	// for each workflow ID, check whether it is paused
	for _, pausedWorkflowId := range pausedWorkflowIds {
		delete(candidateUnpausedWorkflows, sqlchelpers.UUIDToStr(pausedWorkflowId))
	}

	// if there are no paused workflows to unpause, return
	if len(candidateUnpausedWorkflows) == 0 {
		return nil, false, nil
	}

	// if there are paused workflows to unpause, queue them
	workflowRunsToQueue := make([]pgtype.UUID, 0)
	qiIds := make([]int64, 0)

	for i, item := range data {
		if _, ok := candidateUnpausedWorkflows[item.WorkflowId]; ok {
			workflowRunsToQueue = append(workflowRunsToQueue, sqlchelpers.UUIDFromStr(item.WorkflowRunId))
			qiIds = append(qiIds, queueItems[i].ID)
		}
	}

	// update the processed semaphore queue items for the workflow runs which were unpaused
	err = w.queries.MarkInternalQueueItemsProcessed(ctx, tx, qiIds)

	if err != nil {
		return nil, false, fmt.Errorf("could not mark worker semaphore queue items processed: %w", err)
	}

	// get the workflow runs by id
	workflowRuns, err := w.queries.GetWorkflowRun(ctx, tx, dbsqlc.GetWorkflowRunParams{
		Ids:      workflowRunsToQueue,
		Tenantid: pgTenantId,
	})

	if err != nil {
		return nil, false, fmt.Errorf("could not get workflow runs by id: %w", err)
	}

	// if we reached this point, it means that some of the workflows in the queue were unpaused, so
	// we should continue until this is no longer true
	return workflowRuns, true, nil
}

func (w *workflowRunAPIRepository) GetWorkflowRunById(ctx context.Context, tenantId, id string) (*dbsqlc.GetWorkflowRunByIdRow, error) {
	return w.queries.GetWorkflowRunById(ctx, w.pool, dbsqlc.GetWorkflowRunByIdParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Workflowrunid: sqlchelpers.UUIDFromStr(id),
	})
}

func (w *workflowRunAPIRepository) GetWorkflowRunByIds(ctx context.Context, tenantId string, ids []string) ([]*dbsqlc.GetWorkflowRunByIdsRow, error) {
	uuids := make([]pgtype.UUID, len(ids))

	for i, id := range ids {
		uuids[i] = sqlchelpers.UUIDFromStr(id)
	}

	return w.queries.GetWorkflowRunByIds(ctx, w.pool, dbsqlc.GetWorkflowRunByIdsParams{
		Tenantid:       sqlchelpers.UUIDFromStr(tenantId),
		Workflowrunids: uuids,
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

	stepRuns, err := w.queries.GetStepRunsForJobRunsWithOutput(ctx, w.pool, dbsqlc.GetStepRunsForJobRunsWithOutputParams{
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
			GetStepRunsForJobRunsWithOutputRow: stepRun,
			ChildWorkflowsCount:                childCount,
		}
	}

	return res, nil
}

type workflowRunEngineRepository struct {
	*sharedRepository

	m  *metered.Metered
	cf *server.ConfigFileRuntime

	createCallbacks []repository.TenantScopedCallback[*dbsqlc.WorkflowRun]
	queuedCallbacks []repository.TenantScopedCallback[pgtype.UUID]
}

func NewWorkflowRunEngineRepository(shared *sharedRepository, m *metered.Metered, cf *server.ConfigFileRuntime, cbs ...repository.TenantScopedCallback[*dbsqlc.WorkflowRun]) repository.WorkflowRunEngineRepository {
	return &workflowRunEngineRepository{
		sharedRepository: shared,
		m:                m,
		createCallbacks:  cbs,
		cf:               cf,
	}
}

func (w *workflowRunEngineRepository) RegisterCreateCallback(callback repository.TenantScopedCallback[*dbsqlc.WorkflowRun]) {
	if w.createCallbacks == nil {
		w.createCallbacks = make([]repository.TenantScopedCallback[*dbsqlc.WorkflowRun], 0)
	}

	w.createCallbacks = append(w.createCallbacks, callback)
}

func (w *workflowRunEngineRepository) RegisterQueuedCallback(callback repository.TenantScopedCallback[pgtype.UUID]) {
	if w.queuedCallbacks == nil {
		w.queuedCallbacks = make([]repository.TenantScopedCallback[pgtype.UUID], 0)
	}

	w.queuedCallbacks = append(w.queuedCallbacks, callback)
}
func (w *workflowRunEngineRepository) getWorkflowRunByIdWithTx(ctx context.Context, tx dbsqlc.DBTX, tenantId, id string) (*dbsqlc.GetWorkflowRunRow, error) {
	runs, err := w.queries.GetWorkflowRun(ctx, tx, dbsqlc.GetWorkflowRunParams{
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

func (w *workflowRunEngineRepository) GetWorkflowRunById(ctx context.Context, tenantId, id string) (*dbsqlc.GetWorkflowRunRow, error) {
	return w.getWorkflowRunByIdWithTx(ctx, w.pool, tenantId, id)
}

func (w *workflowRunEngineRepository) GetWorkflowRunByIds(ctx context.Context, tenantId string, ids []string) ([]*dbsqlc.GetWorkflowRunRow, error) {

	// we need to only search for unique ids
	uniqueIds := make(map[string]bool)

	for _, id := range ids {
		uniqueIds[id] = true
	}

	ids = make([]string, 0, len(uniqueIds))

	for id := range uniqueIds {
		ids = append(ids, id)
	}

	uuids := make([]pgtype.UUID, len(ids))

	for i, id := range ids {
		uuids[i] = sqlchelpers.UUIDFromStr(id)
	}

	runs, err := w.queries.GetWorkflowRun(ctx, w.pool, dbsqlc.GetWorkflowRunParams{
		Ids:      uuids,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	if len(runs) != len(uuids) {
		missingIds := make([]string, 0, len(uuids)-len(runs))

		for _, id := range uuids {
			found := false
			for _, run := range runs {
				if run.WorkflowRun.ID == id {
					found = true
					break
				}
			}

			if !found {
				missingIds = append(missingIds, sqlchelpers.UUIDToStr(id))
			}
		}

		return nil, fmt.Errorf("%w: could not find workflow runs with ids: %s", repository.ErrWorkflowRunNotFound, strings.Join(missingIds, ", "))
	}

	return runs, nil
}

func (w *workflowRunEngineRepository) GetWorkflowRunAdditionalMeta(ctx context.Context, tenantId, workflowRunId string) (*dbsqlc.GetWorkflowRunAdditionalMetaRow, error) {
	return w.queries.GetWorkflowRunAdditionalMeta(ctx, w.pool, dbsqlc.GetWorkflowRunAdditionalMetaParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Workflowrunid: sqlchelpers.UUIDFromStr(workflowRunId),
	})
}

func (w *workflowRunEngineRepository) ListWorkflowRuns(ctx context.Context, tenantId string, opts *repository.ListWorkflowRunsOpts) (*repository.ListWorkflowRunsResult, error) {

	res, err := w.listWorkflowRuns(ctx, w.pool, tenantId, opts)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (w *workflowRunEngineRepository) GetChildWorkflowRun(ctx context.Context, parentId, parentStepRunId string, childIndex int, childkey *string) (*dbsqlc.WorkflowRun, error) {
	params := dbsqlc.GetChildWorkflowRunParams{
		Parentid:        sqlchelpers.UUIDFromStr(parentId),
		Parentsteprunid: sqlchelpers.UUIDFromStr(parentStepRunId),
		Childindex: pgtype.Int4{
			Int32: int32(childIndex), // nolint: gosec
			Valid: true,
		},
	}

	if childkey != nil {
		params.ChildKey = sqlchelpers.TextFromStr(*childkey)
	}

	return w.queries.GetChildWorkflowRun(ctx, w.pool, params)
}

func (w *workflowRunEngineRepository) GetChildWorkflowRuns(ctx context.Context, childWorkflowRuns []repository.ChildWorkflowRun) ([]*dbsqlc.WorkflowRun, error) {
	parentIdsWithIndex := make([]pgtype.UUID, 0, len(childWorkflowRuns))
	parentStepRunIdsWithIndex := make([]pgtype.UUID, 0, len(childWorkflowRuns))
	parentIdsWithKey := make([]pgtype.UUID, 0, len(childWorkflowRuns))
	parentStepRunIdsWithKey := make([]pgtype.UUID, 0, len(childWorkflowRuns))
	childIndexes := make([]int32, 0, len(childWorkflowRuns))
	childKeys := make([]string, 0, len(childWorkflowRuns))

	for _, childWorkflowRun := range childWorkflowRuns {
		if childWorkflowRun.ChildIndex <= math.MinInt32 || childWorkflowRun.ChildIndex >= math.MaxInt32 {
			return nil, fmt.Errorf("invalid child index: %d", childWorkflowRun.ChildIndex)
		}

		safeInt32 := int32(childWorkflowRun.ChildIndex) // nolint: gosec

		if childWorkflowRun.Childkey != nil {
			parentIdsWithKey = append(parentIdsWithKey, sqlchelpers.UUIDFromStr(childWorkflowRun.ParentId))
			parentStepRunIdsWithKey = append(parentStepRunIdsWithKey, sqlchelpers.UUIDFromStr(childWorkflowRun.ParentStepRunId))
			childKeys = append(childKeys, *childWorkflowRun.Childkey)
		} else {
			parentIdsWithIndex = append(parentIdsWithIndex, sqlchelpers.UUIDFromStr(childWorkflowRun.ParentId))
			parentStepRunIdsWithIndex = append(parentStepRunIdsWithIndex, sqlchelpers.UUIDFromStr(childWorkflowRun.ParentStepRunId))
			childIndexes = append(childIndexes, safeInt32)
		}
	}

	wrs, err := w.queries.GetChildWorkflowRunsByIndex(ctx, w.pool, dbsqlc.GetChildWorkflowRunsByIndexParams{
		Parentids:        parentIdsWithIndex,
		Parentsteprunids: parentStepRunIdsWithIndex,
		Childindexes:     childIndexes,
	})

	if err != nil {
		return nil, err
	}

	wrs2, err := w.queries.GetChildWorkflowRunsByKey(ctx, w.pool, dbsqlc.GetChildWorkflowRunsByKeyParams{
		Parentids:        parentIdsWithKey,
		Parentsteprunids: parentStepRunIdsWithKey,
		Childkeys:        childKeys,
	})

	if err != nil {
		return nil, err
	}

	return append(wrs, wrs2...), nil
}

func (w *workflowRunEngineRepository) CreateDeDupeKey(ctx context.Context, tenantId, workflowRunId string, workflowVersionId string, key string) error {
	_, err := w.queries.CreateWorkflowRunDedupe(
		ctx,
		w.pool,
		dbsqlc.CreateWorkflowRunDedupeParams{
			Tenantid:          sqlchelpers.UUIDFromStr(tenantId),
			Workflowversionid: sqlchelpers.UUIDFromStr(workflowVersionId),
			Value:             sqlchelpers.TextFromStr(key),
			Workflowrunid:     sqlchelpers.UUIDFromStr(workflowRunId),
		},
	)

	if err != nil {
		if isUniqueViolationOnDedupe(err) {
			return repository.ErrDedupeValueExists{
				DedupeValue: key,
			}
		}
	}
	return err
}

func (w *workflowRunEngineRepository) GetScheduledChildWorkflowRun(ctx context.Context, parentId, parentStepRunId string, childIndex int, childkey *string) (*dbsqlc.WorkflowTriggerScheduledRef, error) {
	childParams := dbsqlc.GetScheduledChildWorkflowRunParams{
		Parentid:        sqlchelpers.UUIDFromStr(parentId),
		Parentsteprunid: sqlchelpers.UUIDFromStr(parentStepRunId),
		Childindex: pgtype.Int4{
			Int32: int32(childIndex), // nolint: gosec
			Valid: true,
		},
	}

	if childkey != nil {
		childParams.ChildKey = sqlchelpers.TextFromStr(*childkey)
	}

	return w.queries.GetScheduledChildWorkflowRun(ctx, w.pool, childParams)
}

func (w *workflowRunEngineRepository) PopWorkflowRunsCancelInProgress(ctx context.Context, tenantId, workflowVersionId string, maxRuns int) ([]*dbsqlc.WorkflowRun, []*dbsqlc.WorkflowRun, error) {
	ctx, span := telemetry.NewSpan(ctx, "queue-by-cancel-in-progress")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, w.pool, w.l, 15000)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	// place a FOR UPDATE lock on queued and running workflow runs to prevent concurrent updates
	allToProcess, err := w.queries.LockWorkflowRunsForQueueing(ctx, tx, dbsqlc.LockWorkflowRunsForQueueingParams{
		Tenantid:          sqlchelpers.UUIDFromStr(tenantId),
		Workflowversionid: sqlchelpers.UUIDFromStr(workflowVersionId),
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not lock workflow runs for queueing: %w", err)
	}

	// group each workflow run by concurrency key
	keyToWorkflowRuns := make(map[string][]*dbsqlc.WorkflowRun)

	for _, row := range allToProcess {
		key := row.ConcurrencyGroupId.String

		if _, ok := keyToWorkflowRuns[key]; !ok {
			keyToWorkflowRuns[key] = make([]*dbsqlc.WorkflowRun, 0)
		}

		keyToWorkflowRuns[key] = append(keyToWorkflowRuns[key], row)
	}

	allToCancel := make([]*dbsqlc.WorkflowRun, 0)
	allToStart := make([]*dbsqlc.WorkflowRun, 0)
	cancelIds := make([]pgtype.UUID, 0)

	for _, toProcess := range keyToWorkflowRuns {
		runningWorkflowRuns := make([]*dbsqlc.WorkflowRun, 0, len(toProcess))
		queuedWorkflowRuns := make([]*dbsqlc.WorkflowRun, 0, len(toProcess))

		for _, row := range toProcess {
			if row.Status == dbsqlc.WorkflowRunStatusRUNNING {
				runningWorkflowRuns = append(runningWorkflowRuns, row)
			} else if row.Status == dbsqlc.WorkflowRunStatusQUEUED {
				queuedWorkflowRuns = append(queuedWorkflowRuns, row)
			}
		}

		// iterate over the running workflow runs and cancel them
		toCancel := max(0, len(runningWorkflowRuns)+len(queuedWorkflowRuns)-maxRuns)

		workflowRunsToCancel := make([]*dbsqlc.WorkflowRun, 0, toCancel)
		workflowRunsToStart := make([]*dbsqlc.WorkflowRun, 0, len(queuedWorkflowRuns))

		for i := 0; i < toCancel && i < len(runningWorkflowRuns); i++ {
			row := runningWorkflowRuns[i]

			workflowRunsToCancel = append(workflowRunsToCancel, row)
		}

		toCancel -= len(workflowRunsToCancel)

		// additionally, cancel any queued workflow runs that aren't running but should be cancelled
		for i := 0; i < toCancel && i < len(queuedWorkflowRuns); i++ {
			row := queuedWorkflowRuns[i]

			workflowRunsToCancel = append(workflowRunsToCancel, row)
		}

		//  start the new runs. anything leftover in the queuedWorkflowRuns slice should be started
		for i := toCancel; i < len(queuedWorkflowRuns); i++ {
			row := queuedWorkflowRuns[i]

			workflowRunsToStart = append(workflowRunsToStart, row)
		}

		for _, row := range workflowRunsToCancel {
			cancelIds = append(cancelIds, row.ID)
		}

		allToCancel = append(allToCancel, workflowRunsToCancel...)
		allToStart = append(allToStart, workflowRunsToStart...)
	}

	// cancel the workflow runs
	err = w.queries.MarkWorkflowRunsCancelling(ctx, tx, dbsqlc.MarkWorkflowRunsCancellingParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Ids:      cancelIds,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not mark workflow runs as cancelling: %w", err)
	}

	// commit the transaction
	err = commit(ctx)

	if err != nil {
		return nil, nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	// return the workflow runs to cancel and the ones to start
	return allToCancel, allToStart, nil
}

func (w *workflowRunEngineRepository) PopWorkflowRunsCancelNewest(ctx context.Context, tenantId, workflowVersionId string, maxRuns int) ([]*dbsqlc.WorkflowRun, []*dbsqlc.WorkflowRun, error) {
	ctx, span := telemetry.NewSpan(ctx, "queue-by-cancel-newest")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, w.pool, w.l, 15000)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	// place a FOR UPDATE lock on queued and running workflow runs to prevent concurrent updates
	allToProcess, err := w.queries.LockWorkflowRunsForQueueing(ctx, tx, dbsqlc.LockWorkflowRunsForQueueingParams{
		Tenantid:          sqlchelpers.UUIDFromStr(tenantId),
		Workflowversionid: sqlchelpers.UUIDFromStr(workflowVersionId),
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not lock workflow runs for queueing: %w", err)
	}

	// group each workflow run by concurrency key
	keyToWorkflowRuns := make(map[string][]*dbsqlc.WorkflowRun)

	for _, row := range allToProcess {
		key := row.ConcurrencyGroupId.String

		if _, ok := keyToWorkflowRuns[key]; !ok {
			keyToWorkflowRuns[key] = make([]*dbsqlc.WorkflowRun, 0)
		}

		keyToWorkflowRuns[key] = append(keyToWorkflowRuns[key], row)
	}

	// reverse the order of the workflow runs
	for _, toProcess := range keyToWorkflowRuns {
		sort.SliceStable(toProcess, func(i, j int) bool {
			return toProcess[i].CreatedAt.Time.After(toProcess[j].CreatedAt.Time)
		})
	}

	allToCancel := make([]*dbsqlc.WorkflowRun, 0)
	allToStart := make([]*dbsqlc.WorkflowRun, 0)
	cancelIds := make([]pgtype.UUID, 0)

	for _, toProcess := range keyToWorkflowRuns {
		runningWorkflowRuns := make([]*dbsqlc.WorkflowRun, 0, len(toProcess))
		queuedWorkflowRuns := make([]*dbsqlc.WorkflowRun, 0, len(toProcess))

		for _, row := range toProcess {
			if row.Status == dbsqlc.WorkflowRunStatusRUNNING {
				runningWorkflowRuns = append(runningWorkflowRuns, row)
			} else if row.Status == dbsqlc.WorkflowRunStatusQUEUED {
				queuedWorkflowRuns = append(queuedWorkflowRuns, row)
			}
		}

		// iterate over the queued workflow runs and cancel them
		toCancel := max(0, len(runningWorkflowRuns)+len(queuedWorkflowRuns)-maxRuns)

		workflowRunsToCancel := make([]*dbsqlc.WorkflowRun, 0, toCancel)
		workflowRunsToStart := make([]*dbsqlc.WorkflowRun, 0, len(queuedWorkflowRuns))

		for i := 0; i < toCancel && i < len(queuedWorkflowRuns); i++ {
			row := queuedWorkflowRuns[i]

			workflowRunsToCancel = append(workflowRunsToCancel, row)
		}

		//  start the new runs. anything leftover in the queuedWorkflowRuns slice should be started
		for i := toCancel; i < len(queuedWorkflowRuns); i++ {
			row := queuedWorkflowRuns[i]

			workflowRunsToStart = append(workflowRunsToStart, row)
		}

		for _, row := range workflowRunsToCancel {
			cancelIds = append(cancelIds, row.ID)
		}

		allToCancel = append(allToCancel, workflowRunsToCancel...)
		allToStart = append(allToStart, workflowRunsToStart...)
	}

	// cancel the workflow runs
	err = w.queries.MarkWorkflowRunsCancelling(ctx, tx, dbsqlc.MarkWorkflowRunsCancellingParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Ids:      cancelIds,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not mark workflow runs as cancelling: %w", err)
	}

	// commit the transaction
	err = commit(ctx)

	if err != nil {
		return nil, nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	// return the workflow runs to cancel and the ones to start
	return allToCancel, allToStart, nil
}

func (w *workflowRunEngineRepository) PopWorkflowRunsRoundRobin(ctx context.Context, tenantId string, workflowVersionId string, maxRuns int) ([]*dbsqlc.WorkflowRun, []*dbsqlc.GetStepRunForEngineRow, error) {

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, w.pool, w.l, 5000)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()
	poppedWorkflowRuns, err := w.queries.PopWorkflowRunsRoundRobin(ctx, tx, dbsqlc.PopWorkflowRunsRoundRobinParams{
		Maxruns:           int32(maxRuns), // nolint: gosec
		Tenantid:          sqlchelpers.UUIDFromStr(tenantId),
		Workflowversionid: sqlchelpers.UUIDFromStr(workflowVersionId),
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not list queued workflow runs: %w", err)
	}
	var startableStepRuns []*dbsqlc.GetStepRunForEngineRow
	for i := range poppedWorkflowRuns {
		row := poppedWorkflowRuns[i]

		workflowRunId := sqlchelpers.UUIDToStr(row.ID)

		w.l.Info().Msgf("popped workflow run %s", workflowRunId)
		workflowRun, err := w.getWorkflowRunByIdWithTx(ctx, tx, tenantId, workflowRunId)

		if err != nil {
			return nil, nil, fmt.Errorf("could not get workflow run: %w", err)
		}

		isPaused := workflowRun.IsPaused.Valid && workflowRun.IsPaused.Bool

		if isPaused {
			continue
		}

		ssr, err := w.queueWorkflowRunJobs(ctx, tx, workflowRun, isPaused)

		if err != nil {
			return nil, nil, fmt.Errorf("could not queue workflow run jobs: %w", err)
		}

		startableStepRuns = append(startableStepRuns, ssr...)
	}

	err = commit(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return poppedWorkflowRuns, startableStepRuns, nil
}
func (w *workflowRunEngineRepository) QueueWorkflowRunJobs(ctx context.Context, tenantId, workflowRunId string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, w.pool, w.l, 15000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	workflowRun, err := w.getWorkflowRunByIdWithTx(ctx, tx, tenantId, workflowRunId)

	if err != nil {
		return nil, fmt.Errorf("could not get workflow run: %w", err)
	}

	isPaused := workflowRun.IsPaused.Valid && workflowRun.IsPaused.Bool

	if isPaused {
		return nil, nil
	}

	ssr, err := w.queueWorkflowRunJobs(ctx, tx, workflowRun, isPaused)

	if err != nil {
		return nil, fmt.Errorf("could not queue workflow run jobs: %w", err)
	}

	err = commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return ssr, nil
}

func (w *workflowRunEngineRepository) queueWorkflowRunJobs(ctx context.Context, tx dbsqlc.DBTX, workflowRun *dbsqlc.GetWorkflowRunRow, isPaused bool) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "queue-workflow-run-jobs") // nolint:ineffassign
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(workflowRun.WorkflowRun.TenantId)
	workflowRunId := sqlchelpers.UUIDToStr(workflowRun.WorkflowRun.ID)
	workflowId := sqlchelpers.UUIDToStr(workflowRun.WorkflowVersion.WorkflowId)

	if isPaused {
		return nil, w.queuePausedWorkflowRunWithTx(ctx, tx, tenantId, workflowId, workflowRunId)
	}

	jobRuns, err := w.listJobRunsForWorkflowRunWithTx(ctx, tx, tenantId, workflowRunId)

	if err != nil {
		return nil, fmt.Errorf("could not list job runs: %w", err)
	}

	jobRunIds := make([]string, 0)

	for i := range jobRuns {
		// don't start job runs that are onFailure
		if workflowRun.WorkflowVersion.OnFailureJobId.Valid && jobRuns[i].JobId == workflowRun.WorkflowVersion.OnFailureJobId {
			continue
		}

		jobRunIds = append(jobRunIds, sqlchelpers.UUIDToStr(jobRuns[i].ID))
	}

	return w.startManyJobRuns(ctx, tx, tenantId, jobRunIds)
}

func (w workflowRunEngineRepository) startManyJobRuns(ctx context.Context, tx dbsqlc.DBTX, tenantId string, jobRunIds []string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	var startableStepRuns []*dbsqlc.GetStepRunForEngineRow

	err := queueutils.BatchConcurrent(50, jobRunIds, func(group []string) error {
		for i := range group {
			ssr, err := w.startJobRun(ctx, tx, tenantId, group[i])

			if err != nil {
				return err
			}
			startableStepRuns = append(startableStepRuns, ssr...)
		}

		return nil
	})

	return startableStepRuns, err
}

func (j *jobRunEngineRepository) StartJobRun(ctx context.Context, tenantId, jobRunId string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, j.pool, j.l, 15000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	startedJobRuns, err := j.startJobRun(ctx, tx, tenantId, jobRunId)

	if err != nil {
		return nil, err
	}

	err = commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return startedJobRuns, nil
}

func (w *sharedRepository) startJobRun(ctx context.Context, tx dbsqlc.DBTX, tenantId, jobRunId string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "handle-start-job-run")
	defer span.End()

	err := w.setJobRunStatusRunningWithTx(ctx, tx, tenantId, jobRunId)

	if err != nil {
		return nil, fmt.Errorf("could not set job run status to running: %w", err)
	}

	// list the step runs which are startable
	startableStepRuns, err := w.listInitialStepRunsForJobRunWithTx(ctx, tx, tenantId, jobRunId)

	if err != nil {
		return nil, fmt.Errorf("could not list startable step runs: %w", err)
	}

	return startableStepRuns, nil

}

func (w *workflowRunAPIRepository) BulkCreateWorkflowRuns(ctx context.Context, opts []*repository.CreateWorkflowRunOpts) ([]*dbsqlc.WorkflowRun, error) {
	if len(opts) == 0 {
		return nil, fmt.Errorf("no workflow runs to create")
	}

	w.l.Debug().Msgf("bulk creating %d workflow runs", len(opts))

	return createNewWorkflowRuns(ctx, w.pool, w.queries, w.l, opts)
}

func (w *workflowRunEngineRepository) GetUpstreamErrorsForOnFailureStep(
	ctx context.Context,
	onFailureStepRunId string,
) ([]*dbsqlc.GetUpstreamErrorsForOnFailureStepRow, error) {
	return w.queries.GetUpstreamErrorsForOnFailureStep(
		ctx,
		w.pool,
		sqlchelpers.UUIDFromStr(onFailureStepRunId),
	)
}

// this is single tenant
func (w *workflowRunEngineRepository) CreateNewWorkflowRuns(ctx context.Context, tenantId string, opts []*repository.CreateWorkflowRunOpts) ([]*dbsqlc.WorkflowRun, error) {

	meteredAmount := len(opts)

	if meteredAmount == 0 {

		return nil, fmt.Errorf("no workflow runs to create")
	}

	if meteredAmount > math.MaxInt32 || meteredAmount < 0 {
		return nil, fmt.Errorf("invalid amount of workflow runs to create: %d", meteredAmount)
	}

	for _, opt := range opts {
		opt.TenantId = tenantId
	}

	wfrs, err := metered.MakeMetered(ctx, w.m, dbsqlc.LimitResourceWORKFLOWRUN, tenantId, int32(meteredAmount), func() (*string, *[]*dbsqlc.WorkflowRun, error) { // nolint: gosec

		wfrs, err := createNewWorkflowRuns(ctx, w.pool, w.queries, w.l, opts)

		if err != nil {
			return nil, nil, err
		}

		for _, cb := range w.createCallbacks {
			for _, wfr := range wfrs {
				cb.Do(w.l, tenantId, wfr) // nolint: errcheck
			}
		}

		ids := make([]string, len(wfrs))

		for i, wfr := range wfrs {
			ids[i] = sqlchelpers.UUIDToStr(wfr.ID)
		}

		str := strings.Join(ids, ",")

		return &str,
			&wfrs, nil
	})

	if err != nil {
		w.l.Error().Err(err).Msg("error creating workflow runs")
		return nil, err
	}

	return *wfrs, err
}

func (w *workflowRunEngineRepository) CreateNewWorkflowRun(ctx context.Context, tenantId string, opts *repository.CreateWorkflowRunOpts) (*dbsqlc.WorkflowRun, error) {
	wfr, err := metered.MakeMetered(ctx, w.m, dbsqlc.LimitResourceWORKFLOWRUN, tenantId, 1, func() (*string, *dbsqlc.WorkflowRun, error) {
		opts.TenantId = tenantId

		if err := w.v.Validate(opts); err != nil {
			return nil, nil, err
		}

		var workflowRun *dbsqlc.WorkflowRun
		var err error

		if w.cf.BufferCreateWorkflowRuns {
			workflowRun, err = w.bulkWorkflowRunBuffer.FireAndWait(ctx, tenantId, opts)

			if err != nil {
				return nil, nil, err
			}
		} else {
			wfrs, err := createNewWorkflowRuns(ctx, w.pool, w.queries, w.l, []*repository.CreateWorkflowRunOpts{opts})
			if err != nil {
				return nil, nil, err
			}
			workflowRun = wfrs[0]
		}

		meterKey := sqlchelpers.UUIDToStr(workflowRun.ID)
		return &meterKey, workflowRun, nil
	})

	if err != nil {
		return nil, err
	}

	return wfr, nil
}

func (w *workflowRunEngineRepository) ListActiveQueuedWorkflowVersions(ctx context.Context, tenantId string) ([]*dbsqlc.ListActiveQueuedWorkflowVersionsRow, error) {
	return w.queries.ListActiveQueuedWorkflowVersions(ctx, w.pool, sqlchelpers.UUIDFromStr(tenantId))
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

	err := sqlchelpers.DeadlockRetry(s.l, func() error {
		tx, err := s.pool.Begin(ctx)

		if err != nil {
			return err
		}

		defer sqlchelpers.DeferRollback(ctx, s.l, tx.Rollback)

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

			defer s.deferredStepRunEvent(
				tenantId,
				repository.CreateStepRunEventOpts{
					StepRunId:     stepRunIdStr,
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

func (s *workflowRunEngineRepository) UpdateWorkflowRunFromGroupKeyEval(ctx context.Context, tenantId, workflowRunId string, opts *repository.UpdateWorkflowRunFromGroupKeyEvalOpts) error {
	if err := s.v.Validate(opts); err != nil {
		return err
	}

	pgWorkflowRunId := sqlchelpers.UUIDFromStr(workflowRunId)

	updateParams := dbsqlc.UpdateWorkflowRunGroupKeyFromExprParams{
		Workflowrunid: pgWorkflowRunId,
	}

	eventParams := repository.CreateStepRunEventOpts{}

	if opts.GroupKey != nil {
		updateParams.ConcurrencyGroupId = sqlchelpers.TextFromStr(*opts.GroupKey)

		now := time.Now().UTC()

		eventParams.EventReason = repository.StepRunEventReasonPtr(dbsqlc.StepRunEventReasonWORKFLOWRUNGROUPKEYSUCCEEDED)
		eventParams.EventSeverity = repository.StepRunEventSeverityPtr(dbsqlc.StepRunEventSeverityINFO)
		eventParams.EventMessage = repository.StringPtr(fmt.Sprintf("Workflow run group key evaluated as %s", *opts.GroupKey))
		eventParams.Timestamp = &now
	}

	if opts.Error != nil {
		updateParams.Error = sqlchelpers.TextFromStr(*opts.Error)

		now := time.Now().UTC()

		eventParams.EventReason = repository.StepRunEventReasonPtr(dbsqlc.StepRunEventReasonWORKFLOWRUNGROUPKEYFAILED)
		eventParams.EventSeverity = repository.StepRunEventSeverityPtr(dbsqlc.StepRunEventSeverityCRITICAL)
		eventParams.EventMessage = repository.StringPtr(fmt.Sprintf("Error evaluating workflow run group key: %s", *opts.Error))
		eventParams.Timestamp = &now
	}

	_, err := s.queries.UpdateWorkflowRunGroupKeyFromExpr(ctx, s.pool, updateParams)

	if err != nil {
		return fmt.Errorf("could not update workflow run group key from expr: %w", err)
	}

	for _, cb := range s.queuedCallbacks {
		cb.Do(s.l, tenantId, pgWorkflowRunId)
	}

	defer insertWorkflowRunQueueItem( // nolint: errcheck
		ctx,
		s.pool,
		s.queries,
		tenantId,
		updateWorkflowRunQueueData{
			WorkflowRunId: workflowRunId,
			Event:         &eventParams,
		},
	)

	return nil
}

func (s *sharedRepository) listWorkflowRuns(ctx context.Context, tx dbsqlc.DBTX, tenantId string, opts *repository.ListWorkflowRunsOpts) (*repository.ListWorkflowRunsResult, error) {
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

	if len(opts.Ids) > 0 {
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

	if opts.FinishedBefore != nil {
		countParams.FinishedBefore = sqlchelpers.TimestampFromTime(*opts.FinishedBefore)
		queryParams.FinishedBefore = sqlchelpers.TimestampFromTime(*opts.FinishedBefore)
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

	workflowRuns, err := s.queries.ListWorkflowRuns(ctx, tx, queryParams)

	if err != nil {
		return nil, err
	}

	count, err := s.queries.CountWorkflowRuns(ctx, tx, countParams)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
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

func createNewWorkflowRuns(ctx context.Context, pool *pgxpool.Pool, queries *dbsqlc.Queries, l *zerolog.Logger, inputOpts []*repository.CreateWorkflowRunOpts) ([]*dbsqlc.WorkflowRun, error) {

	ctx, span := telemetry.NewSpan(ctx, "db-create-new-workflow-runs")
	defer span.End()

	sqlcWorkflowRuns, err := func() ([]*dbsqlc.WorkflowRun, error) {
		tx1Ctx, tx1Span := telemetry.NewSpan(ctx, "db-create-new-workflow-runs-tx")
		defer tx1Span.End()

		// begin a transaction
		tx, commit, rollback, err := sqlchelpers.PrepareTx(tx1Ctx, pool, l, 15000)

		if err != nil {
			return nil, err
		}

		var createRunsParams []dbsqlc.CreateWorkflowRunsParams

		workflowRunOptsMap := make(map[string]*repository.CreateWorkflowRunOpts)

		type stickyInfo struct {
			workflowRunId     pgtype.UUID
			workflowVersionId pgtype.UUID
			desiredWorkerId   pgtype.UUID
			tenantId          pgtype.UUID
		}

		var stickyInfos []stickyInfo
		var triggeredByParams []dbsqlc.CreateWorkflowRunTriggeredBysParams
		var groupKeyParams []dbsqlc.CreateGetGroupKeyRunsParams
		var jobRunParams []dbsqlc.CreateJobRunsParams

		for order, opt := range inputOpts {

			// begin a transaction
			workflowRunId := uuid.New().String()

			workflowRunOptsMap[workflowRunId] = opt

			defer rollback()

			createParams := dbsqlc.CreateWorkflowRunParams{
				ID:                sqlchelpers.UUIDFromStr(workflowRunId),
				Tenantid:          sqlchelpers.UUIDFromStr(opt.TenantId),
				Workflowversionid: sqlchelpers.UUIDFromStr(opt.WorkflowVersionId),
			}

			if opt.DisplayName != nil {
				createParams.DisplayName = sqlchelpers.TextFromStr(*opt.DisplayName)
			}

			if opt.ChildIndex != nil {

				if *opt.ChildIndex < -1 {
					l.Error().Msgf("child index must be greater than or equal to -1 but it is : %d", *opt.ChildIndex)
					return nil, errors.New("child index must be greater than or equal to -1 but it is : " + strconv.Itoa(*opt.ChildIndex))
				}

				if *opt.ChildIndex < math.MinInt32 || *opt.ChildIndex > math.MaxInt32 {
					return nil, errors.New("child index must be within the range of a 32-bit signed integer")
				}
				createParams.ChildIndex = pgtype.Int4{
					Int32: int32(*opt.ChildIndex), // nolint: gosec
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

			}

			if opt.Priority != nil {
				createParams.Priority = pgtype.Int4{
					Int32: *opt.Priority,
					Valid: true,
				}
			}
			if order > math.MaxInt32 || order < math.MinInt32 {
				return nil, errors.New("order must be within the range of a 32-bit signed integer")
			}

			crp := dbsqlc.CreateWorkflowRunsParams{
				ID:                 createParams.ID,
				TenantId:           createParams.Tenantid,
				WorkflowVersionId:  createParams.Workflowversionid,
				DisplayName:        createParams.DisplayName,
				ChildIndex:         createParams.ChildIndex,
				ChildKey:           createParams.ChildKey,
				ParentId:           createParams.ParentId,
				ParentStepRunId:    createParams.ParentStepRunId,
				AdditionalMetadata: createParams.Additionalmetadata,
				Priority:           createParams.Priority,
				Status:             "PENDING",
				InsertOrder:        pgtype.Int4{Int32: int32(order), Valid: true},
			}

			createRunsParams = append(createRunsParams, crp)

			var desiredWorkerId pgtype.UUID

			if opt.DesiredWorkerId != nil {

				desiredWorkerId = sqlchelpers.UUIDFromStr(*opt.DesiredWorkerId)
			}

			stickyInfos = append(stickyInfos, stickyInfo{
				workflowRunId:     sqlchelpers.UUIDFromStr(workflowRunId),
				workflowVersionId: sqlchelpers.UUIDFromStr(opt.WorkflowVersionId),
				tenantId:          sqlchelpers.UUIDFromStr(opt.TenantId),
				desiredWorkerId:   desiredWorkerId,
			})

			var (
				eventId, cronParentId, scheduledWorkflowId pgtype.UUID
				cronSchedule, cronName                     pgtype.Text
			)

			if opt.TriggeringEventId != nil {
				eventId = sqlchelpers.UUIDFromStr(*opt.TriggeringEventId)
			}

			if opt.CronParentId != nil {
				cronParentId = sqlchelpers.UUIDFromStr(*opt.CronParentId)

			}
			if opt.Cron != nil {
				cronSchedule = sqlchelpers.TextFromStr(*opt.Cron)
			}

			if opt.CronName != nil {
				cronName = sqlchelpers.TextFromStr(*opt.CronName)
			}

			if opt.ScheduledWorkflowId != nil {
				scheduledWorkflowId = sqlchelpers.UUIDFromStr(*opt.ScheduledWorkflowId)
			}

			cp := dbsqlc.CreateWorkflowRunTriggeredBysParams{
				ID:           sqlchelpers.UUIDFromStr(uuid.New().String()),
				TenantId:     sqlchelpers.UUIDFromStr(opt.TenantId),
				ParentId:     sqlchelpers.UUIDFromStr(workflowRunId),
				EventId:      eventId,
				CronParentId: cronParentId,
				ScheduledId:  scheduledWorkflowId,
				CronSchedule: cronSchedule,
				CronName:     cronName,
			}

			triggeredByParams = append(triggeredByParams, cp)

			if opt.GetGroupKeyRun != nil {
				groupKeyParams = append(groupKeyParams, dbsqlc.CreateGetGroupKeyRunsParams{
					TenantId:          sqlchelpers.UUIDFromStr(opt.TenantId),
					WorkflowRunId:     sqlchelpers.UUIDFromStr(workflowRunId),
					Input:             opt.GetGroupKeyRun.Input,
					RequeueAfter:      sqlchelpers.TimestampFromTime(time.Now().UTC().Add(5 * time.Second)),
					ScheduleTimeoutAt: sqlchelpers.TimestampFromTime(time.Now().UTC().Add(defaults.DefaultScheduleTimeout)),
					Status:            "PENDING",
					ID:                sqlchelpers.UUIDFromStr(uuid.New().String()),
				})
			}

			jobRunParams = append(jobRunParams, dbsqlc.CreateJobRunsParams{
				Tenantid:          sqlchelpers.UUIDFromStr(opt.TenantId),
				Workflowrunid:     sqlchelpers.UUIDFromStr(workflowRunId),
				Workflowversionid: sqlchelpers.UUIDFromStr(opt.WorkflowVersionId),
			})

		}

		_, err = queries.CreateWorkflowRuns(
			tx1Ctx,
			tx,
			createRunsParams,
		)

		if err != nil {
			l.Error().Err(err).Msg("failed to create workflow runs")
			return nil, err
		}

		workflowRuns, err := queries.GetWorkflowRunsInsertedInThisTxn(tx1Ctx, tx)

		if err != nil {
			l.Error().Err(err).Msg("failed to get inserted workflow runs")
			return nil, err
		}

		if len(workflowRuns) == 0 {
			l.Error().Msg("no new workflow runs created in transaction")
			return nil, errors.New("no new workflow runs created")
		}

		if len(workflowRuns) != len(createRunsParams) {
			l.Error().Msg("number of created workflow runs does not match number of returned workflow runs")
			return nil, errors.New("number of created workflow runs does not match number of returned workflow runs")
		}

		if len(stickyInfos) > 0 {

			stickyWorkflowRunIds := make([]pgtype.UUID, 0)
			workflowVersionIds := make([]pgtype.UUID, 0)
			desiredWorkerIds := make([]pgtype.UUID, 0)
			tenantIds := make([]pgtype.UUID, 0)

			for _, stickyInfo := range stickyInfos {
				stickyWorkflowRunIds = append(stickyWorkflowRunIds, stickyInfo.workflowRunId)

				workflowVersionIds = append(workflowVersionIds, stickyInfo.workflowVersionId)
				desiredWorkerIds = append(desiredWorkerIds, stickyInfo.desiredWorkerId)
				tenantIds = append(tenantIds, stickyInfo.tenantId)
			}

			err = queries.CreateMultipleWorkflowRunStickyStates(tx1Ctx, tx, dbsqlc.CreateMultipleWorkflowRunStickyStatesParams{
				Tenantid:           tenantIds,
				Workflowrunids:     stickyWorkflowRunIds,
				Workflowversionids: workflowVersionIds,
				Desiredworkerids:   desiredWorkerIds,
			})

			if err != nil && !errors.Is(err, pgx.ErrNoRows) {

				return nil, fmt.Errorf("failed to create workflow run sticky state: %w", err)
			}
		}

		if len(triggeredByParams) > 0 {

			_, err = queries.CreateWorkflowRunTriggeredBys(tx1Ctx, tx, triggeredByParams)

			if err != nil {

				l.Info().Msgf("failed to create workflow run triggered by %+v", triggeredByParams)
				l.Error().Err(err).Msg("failed to create workflow run triggered by")
				return nil, err
			}

		}

		if len(groupKeyParams) > 0 {

			_, err = queries.CreateGetGroupKeyRuns(
				tx1Ctx,
				tx,
				groupKeyParams,
			)

			if err != nil {
				l.Error().Err(err).Msg("failed to create get group key runs")
				return nil, err
			}

		}

		if len(jobRunParams) > 0 {
			tenantIds := make([]pgtype.UUID, 0)
			workflowRunIds := make([]pgtype.UUID, 0)
			workflowVersionIds := make([]pgtype.UUID, 0)

			for _, jobRunParam := range jobRunParams {
				tenantIds = append(tenantIds, jobRunParam.Tenantid)
				workflowRunIds = append(workflowRunIds, jobRunParam.Workflowrunid)
				workflowVersionIds = append(workflowVersionIds, jobRunParam.Workflowversionid)
			}
			// update to relate jobrunId to workflowRunId
			createJobRunResults, err := queries.CreateManyJobRuns(
				tx1Ctx,
				tx,
				dbsqlc.CreateManyJobRunsParams{
					Tenantids:          tenantIds,
					Workflowrunids:     workflowRunIds,
					Workflowversionids: workflowVersionIds,
				},
			)

			if err != nil {
				l.Error().Err(err).Msg("failed to create job runs")
				return nil, err
			}

			jobRunLookupDataParams := make([]dbsqlc.CreateJobRunLookupDataParams, 0)
			for _, jobRunResult := range createJobRunResults {

				workflowRunId := jobRunResult.WorkflowRunId
				jobRunId := jobRunResult.ID

				workflowRunOpts := workflowRunOptsMap[sqlchelpers.UUIDToStr(workflowRunId)]

				lookupParams := dbsqlc.CreateJobRunLookupDataParams{
					Tenantid:    jobRunResult.TenantId,
					Triggeredby: workflowRunOpts.TriggeredBy,
					Jobrunid:    jobRunId,
				}

				if workflowRunOpts.InputData != nil {
					lookupParams.Input = workflowRunOpts.InputData
				}

				jobRunLookupDataParams = append(jobRunLookupDataParams, lookupParams)

			}

			ids := make([]pgtype.UUID, 0)

			triggeredByIds := make([]string, 0)
			inputs := make([][]byte, 0)
			jobRunIds := make([]pgtype.UUID, 0)
			tenantIds = make([]pgtype.UUID, 0)

			for j := range jobRunLookupDataParams {

				ids = append(ids, sqlchelpers.UUIDFromStr(uuid.New().String()))
				jobRunIds = append(jobRunIds, jobRunLookupDataParams[j].Jobrunid)
				tenantIds = append(tenantIds, jobRunLookupDataParams[j].Tenantid)
				triggeredByIds = append(triggeredByIds, jobRunLookupDataParams[j].Triggeredby)
				inputs = append(inputs, jobRunLookupDataParams[j].Input)

			}

			_, err = queries.CreateJobRunLookupDatas(
				tx1Ctx,
				tx,
				dbsqlc.CreateJobRunLookupDatasParams{
					Ids:          ids,
					Tenantids:    tenantIds,
					Jobrunids:    jobRunIds,
					Triggeredbys: triggeredByIds,
					Inputs:       inputs,
				},
			)

			if err != nil {
				l.Error().Err(err).Msg("failed to create job run lookup data")
				return nil, err
			}

			stepRunIds, err := queries.CreateStepRunsForJobRunIds(tx1Ctx, tx, dbsqlc.CreateStepRunsForJobRunIdsParams{
				Jobrunids: jobRunIds,
				Priority:  1,
			},
			)

			if err != nil {
				l.Error().Err(err).Msg("failed to create step runs")
				return nil, err
			}

			err = queries.LinkStepRunParents(
				tx1Ctx,
				tx,
				stepRunIds,
			)

			if err != nil {
				l.Err(err).Msg("failed to link step run parents")
				return nil, err
			}

		}

		err = commit(tx1Ctx)

		if err != nil {
			l.Error().Err(err).Msg("failed to commit transaction")

			return nil, err
		}
		return workflowRuns, nil
	}()

	if err != nil {
		return nil, err
	}

	return sqlcWorkflowRuns, nil
}

func isUniqueViolationOnDedupe(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "WorkflowRunDedupe_tenantId_workflowId_value_key") &&
		strings.Contains(err.Error(), "SQLSTATE 23505")
}

func insertWorkflowRunQueueItem(
	ctx context.Context,
	dbtx dbsqlc.DBTX,
	queries *dbsqlc.Queries,
	tenantId string,
	data updateWorkflowRunQueueData,
) error {
	insertData := make([]any, 1)
	insertData[0] = data

	return bulkInsertInternalQueueItem(
		ctx,
		dbtx,
		queries,
		[]pgtype.UUID{sqlchelpers.UUIDFromStr(tenantId)},
		[]dbsqlc.InternalQueue{dbsqlc.InternalQueueWORKFLOWRUNUPDATE},
		insertData,
	)
}

func insertPausedWorkflowRunQueueItem(
	ctx context.Context,
	dbtx dbsqlc.DBTX,
	queries *dbsqlc.Queries,
	tenantId pgtype.UUID,
	data unpauseWorkflowRunQueueData,
) error {
	insertData := make([]any, 1)
	insertData[0] = data

	return bulkInsertInternalQueueItem(
		ctx,
		dbtx,
		queries,
		[]pgtype.UUID{tenantId},
		[]dbsqlc.InternalQueue{dbsqlc.InternalQueueWORKFLOWRUNPAUSED},
		insertData,
	)
}

func bulkWorkflowRunEvents(
	ctx context.Context,
	l *zerolog.Logger,
	dbtx dbsqlc.DBTX,
	queries *dbsqlc.Queries,
	workflowRunIds []pgtype.UUID,
	timeSeen []pgtype.Timestamp,
	reasons []dbsqlc.StepRunEventReason,
	severities []dbsqlc.StepRunEventSeverity,
	messages []string,
	data []map[string]interface{},
) {
	inputData := [][]byte{}
	inputReasons := []string{}
	inputSeverities := []string{}

	for _, d := range data {
		dataBytes, err := json.Marshal(d)

		if err != nil {
			l.Err(err).Msg("could not marshal deferred step run event data")
			return
		}

		inputData = append(inputData, dataBytes)
	}

	for _, r := range reasons {
		inputReasons = append(inputReasons, string(r))
	}

	for _, s := range severities {
		inputSeverities = append(inputSeverities, string(s))
	}

	err := queries.BulkCreateWorkflowRunEvent(ctx, dbtx, dbsqlc.BulkCreateWorkflowRunEventParams{
		Workflowrunids: workflowRunIds,
		Reasons:        inputReasons,
		Severities:     inputSeverities,
		Messages:       messages,
		Data:           inputData,
		Timeseen:       timeSeen,
	})

	if err != nil {
		l.Err(err).Msg("could not create bulk workflow run event")
	}
}
