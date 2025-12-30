package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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

func (w *sharedRepository) DeleteScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string) error {
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

func (w *workflowRunAPIRepository) ScheduledWorkflowMetaByIds(ctx context.Context, tenantId string, scheduledWorkflowIds []string) (map[string]repository.ScheduledWorkflowMeta, error) {
	if len(scheduledWorkflowIds) == 0 {
		return map[string]repository.ScheduledWorkflowMeta{}, nil
	}

	ids := make([]pgtype.UUID, 0, len(scheduledWorkflowIds))
	for _, id := range scheduledWorkflowIds {
		ids = append(ids, sqlchelpers.UUIDFromStr(id))
	}

	rows, err := w.queries.GetScheduledWorkflowMetaByIds(ctx, w.pool, dbsqlc.GetScheduledWorkflowMetaByIdsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Ids:      ids,
	})
	if err != nil {
		return nil, err
	}

	out := make(map[string]repository.ScheduledWorkflowMeta, len(scheduledWorkflowIds))
	for _, row := range rows {
		idStr := sqlchelpers.UUIDToStr(row.ID)
		out[idStr] = repository.ScheduledWorkflowMeta{
			Id:              idStr,
			Method:          row.Method,
			HasTriggeredRun: row.HasTriggeredRun,
		}
	}

	return out, nil
}

func (w *workflowRunAPIRepository) BulkDeleteScheduledWorkflows(ctx context.Context, tenantId string, scheduledWorkflowIds []string) ([]string, error) {
	if len(scheduledWorkflowIds) == 0 {
		return []string{}, nil
	}

	ids := make([]pgtype.UUID, 0, len(scheduledWorkflowIds))
	for _, id := range scheduledWorkflowIds {
		ids = append(ids, sqlchelpers.UUIDFromStr(id))
	}

	deletedIds, err := w.queries.BulkDeleteScheduledWorkflows(ctx, w.pool, dbsqlc.BulkDeleteScheduledWorkflowsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Ids:      ids,
	})
	if err != nil {
		return nil, err
	}

	deleted := make([]string, 0, len(scheduledWorkflowIds))
	for _, id := range deletedIds {
		deleted = append(deleted, sqlchelpers.UUIDToStr(id))
	}

	return deleted, nil
}

func (w *workflowRunAPIRepository) BulkUpdateScheduledWorkflows(ctx context.Context, tenantId string, updates []repository.ScheduledWorkflowUpdate) ([]string, error) {
	if len(updates) == 0 {
		return []string{}, nil
	}

	ids := make([]pgtype.UUID, 0, len(updates))
	triggerAts := make([]pgtype.Timestamp, 0, len(updates))
	for _, u := range updates {
		ids = append(ids, sqlchelpers.UUIDFromStr(u.Id))
		triggerAts = append(triggerAts, sqlchelpers.TimestampFromTime(u.TriggerAt))
	}

	updatedIds, err := w.queries.BulkUpdateScheduledWorkflows(ctx, w.pool, dbsqlc.BulkUpdateScheduledWorkflowsParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Ids:        ids,
		Triggerats: triggerAts,
	})
	if err != nil {
		return nil, err
	}

	updated := make([]string, 0, len(updates))
	for _, id := range updatedIds {
		updated = append(updated, sqlchelpers.UUIDToStr(id))
	}

	return updated, nil
}

func (w *workflowRunAPIRepository) GetWorkflowRunShape(ctx context.Context, workflowVersionId uuid.UUID) ([]*dbsqlc.GetWorkflowRunShapeRow, error) {
	return w.queries.GetWorkflowRunShape(ctx, w.pool, sqlchelpers.UUIDFromStr(workflowVersionId.String()))
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
