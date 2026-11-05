package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	// nil/empty tenantIds slice rebuilds the count table from the live rows
	// and returns every non-zero tenant. A non-empty slice reads the table.
	CountAllocatedResourcesByTenant(ctx context.Context, tenantIds []uuid.UUID) ([]*sqlcv1.TenantAllocatedResourceCount, error)

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

func (w *workflowScheduleRepository) CountAllocatedResourcesByTenant(ctx context.Context, tenantIds []uuid.UUID) ([]*sqlcv1.TenantAllocatedResourceCount, error) {
	db := w.pool.ForShared()

	if len(tenantIds) == 0 {
		tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, db, w.l)

		if err != nil {
			return nil, err
		}

		defer rollback()

		if err := w.queries.LockAllocatedResourceCountTable(ctx, tx); err != nil {
			return nil, err
		}

		if err := w.queries.DeleteAllocatedResourceCounts(ctx, tx); err != nil {
			return nil, err
		}

		if err := w.queries.InsertAllocatedResourceCounts(ctx, tx); err != nil {
			return nil, err
		}

		if err := commit(ctx); err != nil {
			return nil, err
		}

		tenantIds = nil
	}

	return w.queries.CountAllocatedResourcesByTenant(ctx, db, tenantIds)
}

// cronCountDelta is the change in the enabled-on-latest cron count for this
// cron. nextEnabled nil means the cron is about to be deleted. A cron that is
// not on the latest live version does not count.
func (w *workflowScheduleRepository) cronCountDelta(ctx context.Context, db sqlcv1.DBTX, tenantId, id uuid.UUID, nextEnabled *bool) (int64, error) {
	state, err := w.queries.GetCronAllocationState(ctx, db, sqlcv1.GetCronAllocationStateParams{
		ID:       id,
		Tenantid: tenantId,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	if !state.OnLatest {
		return 0, nil
	}

	if nextEnabled == nil {
		if state.Enabled {
			return -1, nil
		}

		return 0, nil
	}

	if state.Enabled == *nextEnabled {
		return 0, nil
	}

	if *nextEnabled {
		return 1, nil
	}

	return -1, nil
}

// lockedCronCountDelta locks the workflow version, then reads the cron. The
// lock is the same one version registration holds across its own cron delta.
func (w *workflowScheduleRepository) lockedCronCountDelta(ctx context.Context, db sqlcv1.DBTX, tenantId, id uuid.UUID, nextEnabled *bool) (int64, error) {
	workflowId, err := w.queries.GetCronWorkflowID(ctx, db, sqlcv1.GetCronWorkflowIDParams{
		ID:       id,
		Tenantid: tenantId,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	if _, err := lockLatestWorkflowVersion(ctx, w.queries, db, workflowId); err != nil && !errors.Is(err, errNoLiveWorkflowVersion) {
		return 0, err
	}

	return w.cronCountDelta(ctx, db, tenantId, id, nextEnabled)
}

// countLockedSchedules locks each schedule's workflow version, then the
// schedules that still count, and returns how many of those there are.
func (w *workflowScheduleRepository) countLockedSchedules(ctx context.Context, db sqlcv1.DBTX, tenantId uuid.UUID, ids []uuid.UUID, apiOnly bool) (int64, error) {
	workflowIDs, err := w.queries.ListScheduleWorkflowIDs(ctx, db, sqlcv1.ListScheduleWorkflowIDsParams{
		Ids:      ids,
		Tenantid: tenantId,
	})
	if err != nil {
		return 0, err
	}

	slices.SortFunc(workflowIDs, func(a, b uuid.UUID) int {
		return bytes.Compare(a[:], b[:])
	})

	for _, workflowID := range workflowIDs {
		if _, err := lockLatestWorkflowVersion(ctx, w.queries, db, workflowID); err != nil && !errors.Is(err, errNoLiveWorkflowVersion) {
			return 0, err
		}
	}

	locked, err := w.queries.LockSchedulesCountedTowardAllocation(ctx, db, sqlcv1.LockSchedulesCountedTowardAllocationParams{
		Ids:      ids,
		Tenantid: tenantId,
		Apionly:  apiOnly,
	})
	if err != nil {
		return 0, err
	}

	return int64(len(locked)), nil
}

func (w *workflowScheduleRepository) RegisterCreateCallback(callback TenantScopedCallback[*sqlcv1.WorkflowRun]) {
	if w.createCallbacks == nil {
		w.createCallbacks = make([]TenantScopedCallback[*sqlcv1.WorkflowRun], 0)
	}

	w.createCallbacks = append(w.createCallbacks, callback)
}

func (w *workflowScheduleRepository) CreateScheduledWorkflow(ctx context.Context, tenantId uuid.UUID, opts *CreateScheduledWorkflowRunForWorkflowOpts) (*sqlcv1.ListScheduledWorkflowsRow, error) {
	db := w.pool.ForTenant(tenantId)

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

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, db, w.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	if _, err := lockLatestWorkflowVersion(ctx, w.queries, tx, opts.WorkflowId); err != nil {
		return nil, fmt.Errorf("locking workflow version: %w", err)
	}

	created, err := w.queries.CreateWorkflowTriggerScheduledRefForWorkflow(ctx, tx, createParams)

	if err != nil {
		return nil, err
	}

	scheduled, err := w.queries.ListScheduledWorkflows(ctx, tx, sqlcv1.ListScheduledWorkflowsParams{
		Tenantid:   tenantId,
		ScheduleId: &created.ID,
	})

	if err != nil {
		return nil, err
	}

	if len(scheduled) == 0 {
		return nil, fmt.Errorf("failed to fetch scheduled workflow")
	}

	// The insert lands on the latest live version and has not fired.
	if err := adjustAllocatedResourceCounts(ctx, w.queries, tx, tenantId, allocatedResourceCountDelta{ScheduledRun: 1}); err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	w.notifyAllocatedResourceChange(tenantId)
	return scheduled[0], nil
}

func (w *workflowScheduleRepository) ListScheduledWorkflows(ctx context.Context, tenantId uuid.UUID, opts *ListScheduledWorkflowsOpts) ([]*sqlcv1.ListScheduledWorkflowsRow, int64, error) {
	db := w.pool.ForTenant(tenantId)

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

	count, err := w.queries.CountScheduledWorkflows(ctx, db, countParams)

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

	scheduledWorkflows, err := w.queries.ListScheduledWorkflows(ctx, db, listOpts)
	if err != nil {
		return nil, 0, err
	}

	return scheduledWorkflows, count, nil
}

func (w *workflowScheduleRepository) DeleteScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId uuid.UUID) error {
	db := w.pool.ForTenant(tenantId)

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, db, w.l)

	if err != nil {
		return err
	}

	defer rollback()

	counted, err := w.countLockedSchedules(ctx, tx, tenantId, []uuid.UUID{scheduledWorkflowId}, false)
	if err != nil {
		return err
	}

	if err := w.queries.DeleteScheduledWorkflow(ctx, tx, scheduledWorkflowId); err != nil {
		return err
	}

	if err := adjustAllocatedResourceCounts(ctx, w.queries, tx, tenantId, allocatedResourceCountDelta{ScheduledRun: -counted}); err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	w.notifyAllocatedResourceChange(tenantId)
	return nil
}

func (w *workflowScheduleRepository) GetScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId uuid.UUID) (*sqlcv1.ListScheduledWorkflowsRow, error) {
	db := w.pool.ForTenant(tenantId)

	listOpts := sqlcv1.ListScheduledWorkflowsParams{
		Tenantid:   tenantId,
		ScheduleId: &scheduledWorkflowId,
	}

	scheduledWorkflows, err := w.queries.ListScheduledWorkflows(ctx, db, listOpts)
	if err != nil {
		return nil, err
	}

	if len(scheduledWorkflows) == 0 {
		return nil, nil
	}

	return scheduledWorkflows[0], nil

}

func (w *workflowScheduleRepository) UpdateScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId uuid.UUID, triggerAt time.Time) error {
	db := w.pool.ForTenant(tenantId)

	return w.queries.UpdateScheduledWorkflow(ctx, db, sqlcv1.UpdateScheduledWorkflowParams{
		Scheduleid: scheduledWorkflowId,
		Triggerat:  sqlchelpers.TimestampFromTime(triggerAt),
	})
}

func (w *workflowScheduleRepository) ScheduledWorkflowMetaByIds(ctx context.Context, tenantId uuid.UUID, scheduledWorkflowIds []uuid.UUID) (map[uuid.UUID]ScheduledWorkflowMeta, error) {
	db := w.pool.ForTenant(tenantId)

	if len(scheduledWorkflowIds) == 0 {
		return map[uuid.UUID]ScheduledWorkflowMeta{}, nil
	}

	rows, err := w.queries.GetScheduledWorkflowMetaByIds(ctx, db, sqlcv1.GetScheduledWorkflowMetaByIdsParams{
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
	db := w.pool.ForTenant(tenantId)

	if len(scheduledWorkflowIds) == 0 {
		return []uuid.UUID{}, nil
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, db, w.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	counted, err := w.countLockedSchedules(ctx, tx, tenantId, scheduledWorkflowIds, true)
	if err != nil {
		return nil, err
	}

	deleted, err := w.queries.BulkDeleteScheduledWorkflows(ctx, tx, sqlcv1.BulkDeleteScheduledWorkflowsParams{
		Tenantid: tenantId,
		Ids:      scheduledWorkflowIds,
	})
	if err != nil {
		return nil, err
	}

	if err := adjustAllocatedResourceCounts(ctx, w.queries, tx, tenantId, allocatedResourceCountDelta{ScheduledRun: -counted}); err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	if len(deleted) > 0 {
		w.notifyAllocatedResourceChange(tenantId)
	}
	return deleted, nil
}

func (w *workflowScheduleRepository) BulkUpdateScheduledWorkflows(ctx context.Context, tenantId uuid.UUID, updates []ScheduledWorkflowUpdate) ([]uuid.UUID, error) {
	db := w.pool.ForTenant(tenantId)

	if len(updates) == 0 {
		return []uuid.UUID{}, nil
	}

	ids := make([]uuid.UUID, 0, len(updates))
	triggerAts := make([]pgtype.Timestamp, 0, len(updates))
	for _, u := range updates {
		ids = append(ids, u.Id)
		triggerAts = append(triggerAts, sqlchelpers.TimestampFromTime(u.TriggerAt))
	}

	return w.queries.BulkUpdateScheduledWorkflows(ctx, db, sqlcv1.BulkUpdateScheduledWorkflowsParams{
		Tenantid:   tenantId,
		Ids:        ids,
		Triggerats: triggerAts,
	})
}

func (w *workflowScheduleRepository) ListCronWorkflows(ctx context.Context, tenantId uuid.UUID, opts *ListCronWorkflowsOpts) ([]*sqlcv1.ListCronWorkflowsRow, int64, error) {
	db := w.pool.ForTenant(tenantId)

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

	cronWorkflows, err := w.queries.ListCronWorkflows(ctx, db, listOpts)

	if err != nil {
		return nil, 0, err
	}

	count, err := w.queries.CountCronWorkflows(ctx, db, countOpts)

	if err != nil {
		return nil, count, err
	}

	return cronWorkflows, count, nil
}

func (w *workflowScheduleRepository) GetCronWorkflow(ctx context.Context, tenantId, cronWorkflowId uuid.UUID) (*sqlcv1.ListCronWorkflowsRow, error) {
	db := w.pool.ForTenant(tenantId)

	listOpts := sqlcv1.ListCronWorkflowsParams{
		Tenantid:      tenantId,
		CronTriggerId: &cronWorkflowId,
	}

	cronWorkflows, err := w.queries.ListCronWorkflows(ctx, db, listOpts)

	if err != nil {
		return nil, err
	}

	if len(cronWorkflows) == 0 {
		return nil, fmt.Errorf("cron workflow not found")
	}

	return cronWorkflows[0], nil
}

func (w *workflowScheduleRepository) DeleteCronWorkflow(ctx context.Context, tenantId, id uuid.UUID) error {
	db := w.pool.ForTenant(tenantId)

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, db, w.l)

	if err != nil {
		return err
	}

	defer rollback()

	delta, err := w.lockedCronCountDelta(ctx, tx, tenantId, id, nil)
	if err != nil {
		return err
	}

	if err := w.queries.DeleteWorkflowTriggerCronRef(ctx, tx, id); err != nil {
		return err
	}

	if err := adjustAllocatedResourceCounts(ctx, w.queries, tx, tenantId, allocatedResourceCountDelta{Cron: delta}); err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	w.notifyAllocatedResourceChange(tenantId)
	return nil
}

func (w *workflowScheduleRepository) UpdateCronWorkflow(ctx context.Context, tenantId, id uuid.UUID, opts *UpdateCronOpts) error {
	db := w.pool.ForTenant(tenantId)

	if opts.Enabled == nil {
		return w.queries.UpdateCronTrigger(ctx, db, sqlcv1.UpdateCronTriggerParams{
			Crontriggerid: id,
		})
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, db, w.l)

	if err != nil {
		return err
	}

	defer rollback()

	delta, err := w.lockedCronCountDelta(ctx, tx, tenantId, id, opts.Enabled)
	if err != nil {
		return err
	}

	if err := w.queries.UpdateCronTrigger(ctx, tx, sqlcv1.UpdateCronTriggerParams{
		Crontriggerid: id,
		Enabled:       sqlchelpers.BoolFromBoolean(*opts.Enabled),
	}); err != nil {
		return err
	}

	if err := adjustAllocatedResourceCounts(ctx, w.queries, tx, tenantId, allocatedResourceCountDelta{Cron: delta}); err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	w.notifyAllocatedResourceChange(tenantId)
	return nil
}

func (w *workflowScheduleRepository) CreateCronWorkflow(ctx context.Context, tenantId uuid.UUID, opts *CreateCronWorkflowTriggerOpts) (*sqlcv1.ListCronWorkflowsRow, error) {
	db := w.pool.ForTenant(tenantId)

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

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, db, w.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	if _, err := lockLatestWorkflowVersion(ctx, w.queries, tx, opts.WorkflowId); err != nil {
		return nil, fmt.Errorf("locking workflow version: %w", err)
	}

	cronTrigger, err := w.queries.CreateWorkflowTriggerCronRefForWorkflow(ctx, tx, createParams)

	if err != nil {
		return nil, err
	}

	row, err := w.queries.ListCronWorkflows(ctx, tx, sqlcv1.ListCronWorkflowsParams{
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

	// API crons are created enabled on the latest live version.
	if err := adjustAllocatedResourceCounts(ctx, w.queries, tx, tenantId, allocatedResourceCountDelta{Cron: 1}); err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	w.notifyAllocatedResourceChange(tenantId)
	return row[0], nil
}

func (w *workflowScheduleRepository) DeleteInvalidCron(ctx context.Context, id uuid.UUID) error {
	db := w.pool.ForShared()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, db, w.l)
	if err != nil {
		return err
	}
	defer rollback()

	target, err := w.queries.GetCronAllocationTarget(ctx, tx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	if _, err := lockLatestWorkflowVersion(ctx, w.queries, tx, target.WorkflowID); err != nil && !errors.Is(err, errNoLiveWorkflowVersion) {
		return err
	}

	delta, err := w.cronCountDelta(ctx, tx, target.TenantID, id, nil)
	if err != nil {
		return err
	}

	if err := w.queries.DeleteWorkflowTriggerCronRef(ctx, tx, id); err != nil {
		return err
	}

	if err := adjustAllocatedResourceCounts(ctx, w.queries, tx, target.TenantID, allocatedResourceCountDelta{Cron: delta}); err != nil {
		return err
	}

	return commit(ctx)
}
