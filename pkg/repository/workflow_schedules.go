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
	WorkflowId *string `validate:"omitempty,uuid"`

	// (optional) the parent workflow run id
	ParentWorkflowRunId *string `validate:"omitempty,uuid"`

	// (optional) the parent step run id
	ParentStepRunId *string `validate:"omitempty,uuid"`

	// (optional) statuses to filter by
	Statuses *[]sqlcv1.WorkflowRunStatus

	// (optional) include scheduled runs that are in the future
	IncludeFuture *bool

	// (optional) additional metadata for the workflow run
	AdditionalMetadata map[string]interface{} `validate:"omitempty"`
}

type CreateScheduledWorkflowRunForWorkflowOpts struct {
	ScheduledTrigger   time.Time
	Priority           *int32 `validate:"omitempty,min=1,max=3"`
	WorkflowId         string `validate:"required,uuid"`
	Input              []byte
	AdditionalMetadata []byte
}

type ScheduledWorkflowMeta struct {
	Id              string
	Method          sqlcv1.WorkflowTriggerScheduledRefMethods
	HasTriggeredRun bool
}

type ScheduledWorkflowUpdate struct {
	TriggerAt time.Time
	Id        string
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
	WorkflowId *string `validate:"omitempty,uuid"`

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
	Priority           *int32 `validate:"omitempty,min=1,max=3"`
	WorkflowId         string `validate:"required,uuid"`
	Name               string `validate:"required"`
	Cron               string `validate:"required,cron"`
}

type WorkflowScheduleRepository interface {
	// List ScheduledWorkflows lists workflows by scheduled trigger
	ListScheduledWorkflows(ctx context.Context, tenantId string, opts *ListScheduledWorkflowsOpts) ([]*sqlcv1.ListScheduledWorkflowsRow, int64, error)

	// DeleteScheduledWorkflow deletes a scheduled workflow run
	DeleteScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string) error

	// GetScheduledWorkflow gets a scheduled workflow run
	GetScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string) (*sqlcv1.ListScheduledWorkflowsRow, error)

	// UpdateScheduledWorkflow updates a scheduled workflow run
	UpdateScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string, triggerAt time.Time) error

	// ScheduledWorkflowMetaByIds returns minimal metadata for scheduled workflows by id.
	// Intended for bulk operations to avoid N+1 DB calls.
	ScheduledWorkflowMetaByIds(ctx context.Context, tenantId string, scheduledWorkflowIds []string) (map[string]ScheduledWorkflowMeta, error)

	// BulkDeleteScheduledWorkflows deletes scheduled workflows in bulk and returns deleted ids.
	BulkDeleteScheduledWorkflows(ctx context.Context, tenantId string, scheduledWorkflowIds []string) ([]string, error)

	// BulkUpdateScheduledWorkflows updates scheduled workflows in bulk and returns updated ids.
	BulkUpdateScheduledWorkflows(ctx context.Context, tenantId string, updates []ScheduledWorkflowUpdate) ([]string, error)

	CreateScheduledWorkflow(ctx context.Context, tenantId string, opts *CreateScheduledWorkflowRunForWorkflowOpts) (*sqlcv1.ListScheduledWorkflowsRow, error)

	// CreateCronWorkflow creates a cron trigger
	CreateCronWorkflow(ctx context.Context, tenantId string, opts *CreateCronWorkflowTriggerOpts) (*sqlcv1.ListCronWorkflowsRow, error)

	// List ScheduledWorkflows lists workflows by scheduled trigger
	ListCronWorkflows(ctx context.Context, tenantId string, opts *ListCronWorkflowsOpts) ([]*sqlcv1.ListCronWorkflowsRow, int64, error)

	// GetCronWorkflow gets a cron workflow run
	GetCronWorkflow(ctx context.Context, tenantId, cronWorkflowId string) (*sqlcv1.ListCronWorkflowsRow, error)

	// DeleteCronWorkflow deletes a cron workflow run
	DeleteCronWorkflow(ctx context.Context, tenantId, id string) error

	// UpdateCronWorkflow updates a cron workflow
	UpdateCronWorkflow(ctx context.Context, tenantId, id string, opts *UpdateCronOpts) error

	DeleteInvalidCron(ctx context.Context, id uuid.UUID) error
}

type workflowScheduleRepository struct {
	*sharedRepository

	createCallbacks []TenantScopedCallback[*sqlcv1.WorkflowRun]
}

func newWorkflowScheduleRepository(shared *sharedRepository) WorkflowScheduleRepository {
	return &workflowScheduleRepository{
		sharedRepository: shared,
	}
}

func (w *workflowScheduleRepository) RegisterCreateCallback(callback TenantScopedCallback[*sqlcv1.WorkflowRun]) {
	if w.createCallbacks == nil {
		w.createCallbacks = make([]TenantScopedCallback[*sqlcv1.WorkflowRun], 0)
	}

	w.createCallbacks = append(w.createCallbacks, callback)
}

func (w *workflowScheduleRepository) CreateScheduledWorkflow(ctx context.Context, tenantId string, opts *CreateScheduledWorkflowRunForWorkflowOpts) (*sqlcv1.ListScheduledWorkflowsRow, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	var err error

	if err != nil {
		return nil, err
	}

	var priority int32 = 1

	if opts.Priority != nil {
		priority = *opts.Priority
	}

	createParams := sqlcv1.CreateWorkflowTriggerScheduledRefForWorkflowParams{
		Workflowid:         sqlchelpers.UUIDFromStr(opts.WorkflowId),
		Scheduledtrigger:   sqlchelpers.TimestampFromTime(opts.ScheduledTrigger),
		Input:              opts.Input,
		Additionalmetadata: opts.AdditionalMetadata,
		Method: sqlcv1.NullWorkflowTriggerScheduledRefMethods{
			Valid:                              true,
			WorkflowTriggerScheduledRefMethods: sqlcv1.WorkflowTriggerScheduledRefMethodsAPI,
		},
		Priority: sqlchelpers.ToInt(priority),
	}

	created, err := w.queries.CreateWorkflowTriggerScheduledRefForWorkflow(ctx, w.pool, createParams)

	if err != nil {
		return nil, err
	}

	scheduled, err := w.queries.ListScheduledWorkflows(ctx, w.pool, sqlcv1.ListScheduledWorkflowsParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Scheduleid: created.ID,
	})

	if err != nil {
		return nil, err
	}

	return scheduled[0], nil
}

func (w *workflowScheduleRepository) ListScheduledWorkflows(ctx context.Context, tenantId string, opts *ListScheduledWorkflowsOpts) ([]*sqlcv1.ListScheduledWorkflowsRow, int64, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	listOpts := sqlcv1.ListScheduledWorkflowsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	}

	countParams := sqlcv1.CountScheduledWorkflowsParams{
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

func (w *workflowScheduleRepository) DeleteScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string) error {
	return w.queries.DeleteScheduledWorkflow(ctx, w.pool, sqlchelpers.UUIDFromStr(scheduledWorkflowId))
}

func (w *workflowScheduleRepository) GetScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string) (*sqlcv1.ListScheduledWorkflowsRow, error) {

	listOpts := sqlcv1.ListScheduledWorkflowsParams{
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

func (w *workflowScheduleRepository) UpdateScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string, triggerAt time.Time) error {
	return w.queries.UpdateScheduledWorkflow(ctx, w.pool, sqlcv1.UpdateScheduledWorkflowParams{
		Scheduleid: sqlchelpers.UUIDFromStr(scheduledWorkflowId),
		Triggerat:  sqlchelpers.TimestampFromTime(triggerAt),
	})
}

func (w *workflowScheduleRepository) ScheduledWorkflowMetaByIds(ctx context.Context, tenantId string, scheduledWorkflowIds []string) (map[string]ScheduledWorkflowMeta, error) {
	if len(scheduledWorkflowIds) == 0 {
		return map[string]ScheduledWorkflowMeta{}, nil
	}

	ids := make([]uuid.UUID, 0, len(scheduledWorkflowIds))
	for _, id := range scheduledWorkflowIds {
		ids = append(ids, sqlchelpers.UUIDFromStr(id))
	}

	rows, err := w.queries.GetScheduledWorkflowMetaByIds(ctx, w.pool, sqlcv1.GetScheduledWorkflowMetaByIdsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Ids:      ids,
	})
	if err != nil {
		return nil, err
	}

	out := make(map[string]ScheduledWorkflowMeta, len(scheduledWorkflowIds))
	for _, row := range rows {
		idStr := sqlchelpers.UUIDToStr(row.ID)
		out[idStr] = ScheduledWorkflowMeta{
			Id:              idStr,
			Method:          row.Method,
			HasTriggeredRun: row.HasTriggeredRun,
		}
	}

	return out, nil
}

func (w *workflowScheduleRepository) BulkDeleteScheduledWorkflows(ctx context.Context, tenantId string, scheduledWorkflowIds []string) ([]string, error) {
	if len(scheduledWorkflowIds) == 0 {
		return []string{}, nil
	}

	ids := make([]uuid.UUID, 0, len(scheduledWorkflowIds))
	for _, id := range scheduledWorkflowIds {
		ids = append(ids, sqlchelpers.UUIDFromStr(id))
	}

	deletedIds, err := w.queries.BulkDeleteScheduledWorkflows(ctx, w.pool, sqlcv1.BulkDeleteScheduledWorkflowsParams{
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

func (w *workflowScheduleRepository) BulkUpdateScheduledWorkflows(ctx context.Context, tenantId string, updates []ScheduledWorkflowUpdate) ([]string, error) {
	if len(updates) == 0 {
		return []string{}, nil
	}

	ids := make([]uuid.UUID, 0, len(updates))
	triggerAts := make([]pgtype.Timestamp, 0, len(updates))
	for _, u := range updates {
		ids = append(ids, sqlchelpers.UUIDFromStr(u.Id))
		triggerAts = append(triggerAts, sqlchelpers.TimestampFromTime(u.TriggerAt))
	}

	updatedIds, err := w.queries.BulkUpdateScheduledWorkflows(ctx, w.pool, sqlcv1.BulkUpdateScheduledWorkflowsParams{
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

func (w *workflowScheduleRepository) ListCronWorkflows(ctx context.Context, tenantId string, opts *ListCronWorkflowsOpts) ([]*sqlcv1.ListCronWorkflowsRow, int64, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	listOpts := sqlcv1.ListCronWorkflowsParams{
		Tenantid: pgTenantId,
	}

	countOpts := sqlcv1.CountCronWorkflowsParams{
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

func (w *workflowScheduleRepository) GetCronWorkflow(ctx context.Context, tenantId, cronWorkflowId string) (*sqlcv1.ListCronWorkflowsRow, error) {
	listOpts := sqlcv1.ListCronWorkflowsParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Crontriggerid: sqlchelpers.UUIDFromStr(cronWorkflowId),
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

func (w *workflowScheduleRepository) DeleteCronWorkflow(ctx context.Context, tenantId, id string) error {
	return w.queries.DeleteWorkflowTriggerCronRef(ctx, w.pool, sqlchelpers.UUIDFromStr(id))
}

func (w *workflowScheduleRepository) UpdateCronWorkflow(ctx context.Context, tenantId, id string, opts *UpdateCronOpts) error {
	params := sqlcv1.UpdateCronTriggerParams{
		Crontriggerid: sqlchelpers.UUIDFromStr(id),
	}

	if opts.Enabled != nil {
		params.Enabled = sqlchelpers.BoolFromBoolean(*opts.Enabled)
	}

	return w.queries.UpdateCronTrigger(ctx, w.pool, params)
}

func (w *workflowScheduleRepository) CreateCronWorkflow(ctx context.Context, tenantId string, opts *CreateCronWorkflowTriggerOpts) (*sqlcv1.ListCronWorkflowsRow, error) {

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
		Workflowid:         sqlchelpers.UUIDFromStr(opts.WorkflowId),
		Crontrigger:        opts.Cron,
		Name:               sqlchelpers.TextFromStr(opts.Name),
		Input:              input,
		AdditionalMetadata: additionalMetadata,
		Method: sqlcv1.NullWorkflowTriggerCronRefMethods{
			Valid:                         true,
			WorkflowTriggerCronRefMethods: sqlcv1.WorkflowTriggerCronRefMethodsAPI,
		},
		Priority: sqlchelpers.ToInt(priority),
	}

	cronTrigger, err := w.queries.CreateWorkflowTriggerCronRefForWorkflow(ctx, w.pool, createParams)

	if err != nil {
		return nil, err
	}

	row, err := w.queries.ListCronWorkflows(ctx, w.pool, sqlcv1.ListCronWorkflowsParams{
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

func (w *workflowScheduleRepository) DeleteInvalidCron(ctx context.Context, id uuid.UUID) error {
	return w.queries.DeleteWorkflowTriggerCronRef(ctx, w.pool, id)
}
