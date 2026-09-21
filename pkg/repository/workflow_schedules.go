package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type ListScheduledWorkflowsOpts struct {
	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int

	// (optional) the order by field
	OrderBy *string `validate:"omitempty,oneof=createdAt triggerAt"`

	// (optional) the order direction
	OrderDirection *string `validate:"omitempty,oneof=ASC DESC"`

	// (optional) the workflow id
	WorkflowId *uuid.UUID `validate:"omitempty"`

	// (optional) the parent workflow run id
	ParentWorkflowRunId *uuid.UUID `validate:"omitempty"`

	// (optional) the parent step run id
	ParentStepRunId *uuid.UUID `validate:"omitempty"`

	// (optional) statuses to filter by
	Statuses *[]sqlcv1.WorkflowRunStatus

	// (optional) include scheduled runs that are in the future
	IncludeFuture *bool

	// (optional) additional metadata for the workflow run
	AdditionalMetadata map[string]interface{} `validate:"omitempty"`
}

type CreateScheduledWorkflowRunForWorkflowOpts struct {
	ScheduledTrigger   time.Time
	Priority           *int32    `validate:"omitempty,min=1,max=3"`
	WorkflowId         uuid.UUID `validate:"required"`
	Input              []byte
	AdditionalMetadata []byte
}

type ScheduledWorkflowMeta struct {
	Id              uuid.UUID
	Method          sqlcv1.WorkflowTriggerScheduledRefMethods
	HasTriggeredRun bool
}

type ScheduledWorkflowUpdate struct {
	TriggerAt time.Time
	Id        uuid.UUID
}

type UpdateCronOpts struct {
	// (optional) a flag indicating whether or not the cron is enabled
	Enabled *bool
}

type ListCronWorkflowsOpts struct {
	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int

	// (optional) the order by field
	OrderBy *string `validate:"omitempty,oneof=createdAt name"`

	// (optional) the order direction
	OrderDirection *string `validate:"omitempty,oneof=ASC DESC"`

	// (optional) the workflow id
	WorkflowId *uuid.UUID `validate:"omitempty"`

	// (optional) additional metadata for the workflow run
	AdditionalMetadata map[string]interface{} `validate:"omitempty"`

	// (optional) the name of the cron to filter by
	CronName *string `validate:"omitempty"`

	// (optional) the name of the workflow to filter by
	WorkflowName *string `validate:"omitempty"`
}

type CreateCronWorkflowTriggerOpts struct {
	Input              map[string]interface{}
	AdditionalMetadata map[string]interface{}
	Priority           *int32    `validate:"omitempty,min=1,max=3"`
	WorkflowId         uuid.UUID `validate:"required"`
	Name               string    `validate:"required"`
	Cron               string    `validate:"required,cron"`
}

type WorkflowScheduleRepository interface {
	// List ScheduledWorkflows lists workflows by scheduled trigger
	ListScheduledWorkflows(ctx context.Context, tenantId uuid.UUID, opts *ListScheduledWorkflowsOpts) ([]*sqlcv1.ListScheduledWorkflowsRow, int64, error)

	// DeleteScheduledWorkflow deletes a scheduled workflow run
	DeleteScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId uuid.UUID) error

	// GetScheduledWorkflow gets a scheduled workflow run
	GetScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId uuid.UUID) (*sqlcv1.ListScheduledWorkflowsRow, error)

	// UpdateScheduledWorkflow updates a scheduled workflow run
	UpdateScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId uuid.UUID, triggerAt time.Time) error

	// ScheduledWorkflowMetaByIds returns minimal metadata for scheduled workflows by id.
	// Intended for bulk operations to avoid N+1 DB calls.
	ScheduledWorkflowMetaByIds(ctx context.Context, tenantId uuid.UUID, scheduledWorkflowIds []uuid.UUID) (map[uuid.UUID]ScheduledWorkflowMeta, error)

	// BulkDeleteScheduledWorkflows deletes scheduled workflows in bulk and returns deleted ids.
	BulkDeleteScheduledWorkflows(ctx context.Context, tenantId uuid.UUID, scheduledWorkflowIds []uuid.UUID) ([]uuid.UUID, error)

	// BulkUpdateScheduledWorkflows updates scheduled workflows in bulk and returns updated ids.
	BulkUpdateScheduledWorkflows(ctx context.Context, tenantId uuid.UUID, updates []ScheduledWorkflowUpdate) ([]uuid.UUID, error)

	CreateScheduledWorkflow(ctx context.Context, tenantId uuid.UUID, opts *CreateScheduledWorkflowRunForWorkflowOpts) (*sqlcv1.ListScheduledWorkflowsRow, error)

	// CreateCronWorkflow creates a cron trigger
	CreateCronWorkflow(ctx context.Context, tenantId uuid.UUID, opts *CreateCronWorkflowTriggerOpts) (*sqlcv1.ListCronWorkflowsRow, error)

	// List ScheduledWorkflows lists workflows by scheduled trigger
	ListCronWorkflows(ctx context.Context, tenantId uuid.UUID, opts *ListCronWorkflowsOpts) ([]*sqlcv1.ListCronWorkflowsRow, int64, error)

	// GetCronWorkflow gets a cron workflow run
	GetCronWorkflow(ctx context.Context, tenantId, cronWorkflowId uuid.UUID) (*sqlcv1.ListCronWorkflowsRow, error)

	// DeleteCronWorkflow deletes a cron workflow run
	DeleteCronWorkflow(ctx context.Context, tenantId, id uuid.UUID) error

	// UpdateCronWorkflow updates a cron workflow
	UpdateCronWorkflow(ctx context.Context, tenantId, id uuid.UUID, opts *UpdateCronOpts) error

	DeleteInvalidCron(ctx context.Context, id uuid.UUID) error

	// CountAllocatedResourcesByTenant returns live cron, pending schedule, and
	// incoming-webhook counts grouped by tenant (one row per tenant). A
	// nil/empty tenantIds slice is an unfiltered shard scan for the hourly
	// collect; it does not sum tenants together.
	CountAllocatedResourcesByTenant(ctx context.Context, tenantIds []uuid.UUID) ([]*sqlcv1.CountAllocatedResourcesByTenantRow, error)

	// RegisterAllocatedResourceChangeCallback runs after a successful create,
	// delete, or enable/disable of a cron or scheduled run. The callback
	// receives the tenant id.
	RegisterAllocatedResourceChangeCallback(callback TenantScopedCallback[struct{}])
}

type workflowScheduleRepository struct {
	*sharedRepository

	createCallbacks          []TenantScopedCallback[*sqlcv1.WorkflowRun]
	allocatedChangeCallbacks []TenantScopedCallback[struct{}]
}

func newWorkflowScheduleRepository(shared *sharedRepository) WorkflowScheduleRepository {
	return &workflowScheduleRepository{
		sharedRepository: shared,
	}
}

func (w *workflowScheduleRepository) RegisterAllocatedResourceChangeCallback(callback TenantScopedCallback[struct{}]) {
	w.allocatedChangeCallbacks = append(w.allocatedChangeCallbacks, callback)
}

func (w *workflowScheduleRepository) notifyAllocatedResourceChange(tenantId uuid.UUID) {
	for _, cb := range w.allocatedChangeCallbacks {
		cb.Do(w.l, tenantId, struct{}{})
	}
}

func (w *workflowScheduleRepository) CountAllocatedResourcesByTenant(ctx context.Context, tenantIds []uuid.UUID) ([]*sqlcv1.CountAllocatedResourcesByTenantRow, error) {
	if len(tenantIds) == 0 {
		tenantIds = nil
	}
	return w.queries.CountAllocatedResourcesByTenant(ctx, w.pool, tenantIds)
}

func (w *workflowScheduleRepository) RegisterCreateCallback(callback TenantScopedCallback[*sqlcv1.WorkflowRun]) {
	if w.createCallbacks == nil {
		w.createCallbacks = make([]TenantScopedCallback[*sqlcv1.WorkflowRun], 0)
	}

	w.createCallbacks = append(w.createCallbacks, callback)
}

func (w *workflowScheduleRepository) CreateScheduledWorkflow(ctx context.Context, tenantId uuid.UUID, opts *CreateScheduledWorkflowRunForWorkflowOpts) (*sqlcv1.ListScheduledWorkflowsRow, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	var priority int32 = 1

	if opts.Priority != nil {
		priority = *opts.Priority
	}

	createParams := sqlcv1.CreateWorkflowTriggerScheduledRefForWorkflowParams{
		Workflowid:         opts.WorkflowId,
		Scheduledtrigger:   sqlchelpers.TimestampFromTime(opts.ScheduledTrigger),
		Input:              opts.Input,
		Additionalmetadata: opts.AdditionalMetadata,
		Method: sqlcv1.NullWorkflowTriggerScheduledRefMethods{
			Valid:                              true,
			WorkflowTriggerScheduledRefMethods: sqlcv1.WorkflowTriggerScheduledRefMethodsAPI,
		},
		Priority: sqlchelpers.ToInt(&priority),
	}

	created, err := w.queries.CreateWorkflowTriggerScheduledRefForWorkflow(ctx, w.pool, createParams)

	if err != nil {
		return nil, err
	}

	scheduled, err := w.queries.ListScheduledWorkflows(ctx, w.pool, sqlcv1.ListScheduledWorkflowsParams{
		Tenantid:   tenantId,
		ScheduleId: &created.ID,
	})

	if err != nil {
		return nil, err
	}

	if len(scheduled) == 0 {
		return nil, fmt.Errorf("failed to fetch scheduled workflow")
	}

	w.notifyAllocatedResourceChange(tenantId)
	return scheduled[0], nil
}

func (w *workflowScheduleRepository) ListScheduledWorkflows(ctx context.Context, tenantId uuid.UUID, opts *ListScheduledWorkflowsOpts) ([]*sqlcv1.ListScheduledWorkflowsRow, int64, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	listOpts := sqlcv1.ListScheduledWorkflowsParams{
		Tenantid: tenantId,
	}

	countParams := sqlcv1.CountScheduledWorkflowsParams{
		Tenantid: tenantId,
	}

	if opts.WorkflowId != nil {
		listOpts.WorkflowId = opts.WorkflowId
		countParams.WorkflowId = opts.WorkflowId
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
		listOpts.ParentWorkflowRunId = opts.ParentWorkflowRunId
		countParams.ParentWorkflowRunId = opts.ParentWorkflowRunId
	}

	if opts.ParentStepRunId != nil {
		listOpts.ParentStepRunId = opts.ParentStepRunId
		countParams.ParentStepRunId = opts.ParentStepRunId
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

func (w *workflowScheduleRepository) DeleteScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId uuid.UUID) error {
	if err := w.queries.DeleteScheduledWorkflow(ctx, w.pool, scheduledWorkflowId); err != nil {
		return err
	}
	w.notifyAllocatedResourceChange(tenantId)
	return nil
}

func (w *workflowScheduleRepository) GetScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId uuid.UUID) (*sqlcv1.ListScheduledWorkflowsRow, error) {

	listOpts := sqlcv1.ListScheduledWorkflowsParams{
		Tenantid:   tenantId,
		ScheduleId: &scheduledWorkflowId,
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

func (w *workflowScheduleRepository) UpdateScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId uuid.UUID, triggerAt time.Time) error {
	return w.queries.UpdateScheduledWorkflow(ctx, w.pool, sqlcv1.UpdateScheduledWorkflowParams{
		Scheduleid: scheduledWorkflowId,
		Triggerat:  sqlchelpers.TimestampFromTime(triggerAt),
	})
}

func (w *workflowScheduleRepository) ScheduledWorkflowMetaByIds(ctx context.Context, tenantId uuid.UUID, scheduledWorkflowIds []uuid.UUID) (map[uuid.UUID]ScheduledWorkflowMeta, error) {
	if len(scheduledWorkflowIds) == 0 {
		return map[uuid.UUID]ScheduledWorkflowMeta{}, nil
	}

	rows, err := w.queries.GetScheduledWorkflowMetaByIds(ctx, w.pool, sqlcv1.GetScheduledWorkflowMetaByIdsParams{
		Tenantid: tenantId,
		Ids:      scheduledWorkflowIds,
	})
	if err != nil {
		return nil, err
	}

	out := make(map[uuid.UUID]ScheduledWorkflowMeta, len(scheduledWorkflowIds))
	for _, row := range rows {
		out[row.ID] = ScheduledWorkflowMeta{
			Id:              row.ID,
			Method:          row.Method,
			HasTriggeredRun: row.HasTriggeredRun,
		}
	}

	return out, nil
}

func (w *workflowScheduleRepository) BulkDeleteScheduledWorkflows(ctx context.Context, tenantId uuid.UUID, scheduledWorkflowIds []uuid.UUID) ([]uuid.UUID, error) {
	if len(scheduledWorkflowIds) == 0 {
		return []uuid.UUID{}, nil
	}

	deleted, err := w.queries.BulkDeleteScheduledWorkflows(ctx, w.pool, sqlcv1.BulkDeleteScheduledWorkflowsParams{
		Tenantid: tenantId,
		Ids:      scheduledWorkflowIds,
	})
	if err != nil {
		return nil, err
	}
	if len(deleted) > 0 {
		w.notifyAllocatedResourceChange(tenantId)
	}
	return deleted, nil
}

func (w *workflowScheduleRepository) BulkUpdateScheduledWorkflows(ctx context.Context, tenantId uuid.UUID, updates []ScheduledWorkflowUpdate) ([]uuid.UUID, error) {
	if len(updates) == 0 {
		return []uuid.UUID{}, nil
	}

	ids := make([]uuid.UUID, 0, len(updates))
	triggerAts := make([]pgtype.Timestamp, 0, len(updates))
	for _, u := range updates {
		ids = append(ids, u.Id)
		triggerAts = append(triggerAts, sqlchelpers.TimestampFromTime(u.TriggerAt))
	}

	return w.queries.BulkUpdateScheduledWorkflows(ctx, w.pool, sqlcv1.BulkUpdateScheduledWorkflowsParams{
		Tenantid:   tenantId,
		Ids:        ids,
		Triggerats: triggerAts,
	})
}

func (w *workflowScheduleRepository) ListCronWorkflows(ctx context.Context, tenantId uuid.UUID, opts *ListCronWorkflowsOpts) ([]*sqlcv1.ListCronWorkflowsRow, int64, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	listOpts := sqlcv1.ListCronWorkflowsParams{
		Tenantid:   tenantId,
		WorkflowId: opts.WorkflowId,
	}

	countOpts := sqlcv1.CountCronWorkflowsParams{
		Tenantid:   tenantId,
		WorkflowId: opts.WorkflowId,
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

	if opts.CronName != nil {
		listOpts.CronName = sqlchelpers.TextFromStr(*opts.CronName)
		countOpts.CronName = sqlchelpers.TextFromStr(*opts.CronName)
	}

	if opts.WorkflowName != nil {
		listOpts.WorkflowName = sqlchelpers.TextFromStr(*opts.WorkflowName)
		countOpts.WorkflowName = sqlchelpers.TextFromStr(*opts.WorkflowName)
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

func (w *workflowScheduleRepository) GetCronWorkflow(ctx context.Context, tenantId, cronWorkflowId uuid.UUID) (*sqlcv1.ListCronWorkflowsRow, error) {
	listOpts := sqlcv1.ListCronWorkflowsParams{
		Tenantid:      tenantId,
		CronTriggerId: &cronWorkflowId,
	}

	cronWorkflows, err := w.queries.ListCronWorkflows(ctx, w.pool, listOpts)

	if err != nil {
		return nil, err
	}

	if len(cronWorkflows) == 0 {
		return nil, fmt.Errorf("cron workflow not found")
	}

	return cronWorkflows[0], nil
}

func (w *workflowScheduleRepository) DeleteCronWorkflow(ctx context.Context, tenantId, id uuid.UUID) error {
	if err := w.queries.DeleteWorkflowTriggerCronRef(ctx, w.pool, id); err != nil {
		return err
	}
	w.notifyAllocatedResourceChange(tenantId)
	return nil
}

func (w *workflowScheduleRepository) UpdateCronWorkflow(ctx context.Context, tenantId, id uuid.UUID, opts *UpdateCronOpts) error {
	params := sqlcv1.UpdateCronTriggerParams{
		Crontriggerid: id,
	}

	if opts.Enabled != nil {
		params.Enabled = sqlchelpers.BoolFromBoolean(*opts.Enabled)
	}

	if err := w.queries.UpdateCronTrigger(ctx, w.pool, params); err != nil {
		return err
	}

	if opts.Enabled != nil {
		w.notifyAllocatedResourceChange(tenantId)
	}

	return nil
}

func (w *workflowScheduleRepository) CreateCronWorkflow(ctx context.Context, tenantId uuid.UUID, opts *CreateCronWorkflowTriggerOpts) (*sqlcv1.ListCronWorkflowsRow, error) {

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

	createParams := sqlcv1.CreateWorkflowTriggerCronRefForWorkflowParams{
		Workflowid:         opts.WorkflowId,
		Crontrigger:        opts.Cron,
		Name:               sqlchelpers.TextFromStr(opts.Name),
		Input:              input,
		AdditionalMetadata: additionalMetadata,
		Method: sqlcv1.NullWorkflowTriggerCronRefMethods{
			Valid:                         true,
			WorkflowTriggerCronRefMethods: sqlcv1.WorkflowTriggerCronRefMethodsAPI,
		},
		Priority: sqlchelpers.ToInt(&priority),
	}

	cronTrigger, err := w.queries.CreateWorkflowTriggerCronRefForWorkflow(ctx, w.pool, createParams)

	if err != nil {
		return nil, err
	}

	row, err := w.queries.ListCronWorkflows(ctx, w.pool, sqlcv1.ListCronWorkflowsParams{
		Tenantid:      tenantId,
		CronTriggerId: &cronTrigger.ID,
		Limit:         1,
	})

	if err != nil {
		return nil, err
	}

	if len(row) == 0 {
		return nil, fmt.Errorf("failed to fetch cron workflow")
	}

	w.notifyAllocatedResourceChange(tenantId)
	return row[0], nil
}

func (w *workflowScheduleRepository) DeleteInvalidCron(ctx context.Context, id uuid.UUID) error {
	return w.queries.DeleteWorkflowTriggerCronRef(ctx, w.pool, id)
}
