package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/internal/statusutils"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type CreateTaskOpts struct {
	// (required) the external id
	ExternalId uuid.UUID `validate:"required"`

	// (required) the workflow run id. note this may be the same as the external id if this is a
	// single-task workflow, otherwise it represents the external id of the DAG.
	WorkflowRunId uuid.UUID `validate:"required"`

	// (required) the step id
	StepId uuid.UUID `validate:"required"`

	// (required) the input bytes to the task
	Input *TaskInput

	FilterPayload []byte

	// (required) the step index for the task
	StepIndex int

	// (optional) the additional metadata for the task
	AdditionalMetadata []byte

	// (optional) the desired worker id
	DesiredWorkerId *uuid.UUID

	// (optional) the DAG id for the task
	DagId *int64

	// (optional) the DAG inserted at for the task
	DagInsertedAt pgtype.Timestamptz

	// (required) the initial state for the task
	InitialState sqlcv1.V1TaskInitialState

	// (optional) the parent task external id
	ParentTaskExternalId *uuid.UUID

	// (optional) the parent task id
	ParentTaskId *int64

	// (optional) the parent task inserted at
	ParentTaskInsertedAt *time.Time

	// (optional) The priority of a task, between 1 and 3
	Priority *int32

	// (optional) the child index for the task
	ChildIndex *int64

	// (optional) the child key for the task
	ChildKey *string
}

type ReplayTasksResult struct {
	ReplayedTasks []TaskIdInsertedAtRetryCount

	UpsertedTasks []*V1TaskWithPayload

	InternalEventResults *EventMatchResults
}

type ReplayTaskOpts struct {
	// (required) the task id
	TaskId int64

	// (required) the inserted at time
	InsertedAt pgtype.Timestamptz

	// (required) the external id
	ExternalId uuid.UUID

	// (required) the step id
	StepId uuid.UUID

	// (optional) the input bytes to the task, uses the existing input if not set
	Input *TaskInput

	// (required) the initial state for the task
	InitialState sqlcv1.V1TaskInitialState

	// (optional) the additional metadata for the task
	AdditionalMetadata []byte
}

type TaskIdInsertedAtRetryCount struct {
	// (required) the id
	Id int64 `validate:"required"`

	// (required) the inserted at time
	InsertedAt pgtype.Timestamptz

	// (required) the retry count
	RetryCount int32
}

type TaskIdInsertedAtSignalKey struct {
	// (required) the external id
	Id int64 `validate:"required"`

	// (required) the inserted at time
	InsertedAt pgtype.Timestamptz

	// (required) the signal key for the event
	SignalKey string
}

type CompleteTaskOpts struct {
	*TaskIdInsertedAtRetryCount

	// (required) the output bytes for the task
	Output []byte
}

type FailTaskOpts struct {
	*TaskIdInsertedAtRetryCount

	// (required) whether this is an application-level error or an internal error on the Hatchet side
	IsAppError bool

	// (optional) the error message for the task
	ErrorMessage string

	// (optional) A boolean flag to indicate whether the error is non-retryable, meaning it should _not_ be retried. Defaults to false.
	IsNonRetryable bool
}

type TaskIdEventKeyTuple struct {
	Id int64 `validate:"required"`

	EventKey string `validate:"required"`
}

// InternalTaskEvent resembles sqlcv1.V1TaskEvent, but doesn't include the id field as we
// use COPY FROM to write the events to the database.
type InternalTaskEvent struct {
	TenantID       uuid.UUID              `json:"tenant_id"`
	TaskID         int64                  `json:"task_id"`
	TaskExternalID uuid.UUID              `json:"task_external_id"`
	RetryCount     int32                  `json:"retry_count"`
	EventType      sqlcv1.V1TaskEventType `json:"event_type"`
	EventKey       string                 `json:"event_key"`
	Data           []byte                 `json:"data"`
}

type FinalizedTaskResponse struct {
	ReleasedTasks []*sqlcv1.ReleaseTasksRow

	InternalEvents []InternalTaskEvent
}

type RetriedTask struct {
	*TaskIdInsertedAtRetryCount

	IsAppError bool

	AppRetryCount int32

	RetryBackoffFactor pgtype.Float8

	RetryMaxBackoff pgtype.Int4
}

type FailTasksResponse struct {
	*FinalizedTaskResponse

	RetriedTasks []RetriedTask
}

type TimeoutTasksResponse struct {
	*FailTasksResponse

	TimeoutTasks []*sqlcv1.ListTasksToTimeoutRow
}

type ListFinalizedWorkflowRunsResponse struct {
	WorkflowRunId uuid.UUID

	OutputEvents []*TaskOutputEvent
}

type RefreshTimeoutBy struct {
	TaskExternalId uuid.UUID `validate:"required"`

	IncrementTimeoutBy string `validate:"required,duration"`
}

type TaskRepository interface {
	EnsureTablePartitionsExist(ctx context.Context) (bool, error)
	UpdateTablePartitions(ctx context.Context) error

	// GetTaskByExternalId is a heavily cached method to return task metadata by its external id
	GetTaskByExternalId(ctx context.Context, tenantId, taskExternalId uuid.UUID, skipCache bool) (*sqlcv1.FlattenExternalIdsRow, error)

	// FlattenExternalIds is a non-cached method to look up all tasks in a workflow run by their external ids.
	// This is non-cacheable because tasks can be added to a workflow run as it executes.
	FlattenExternalIds(ctx context.Context, tenantId uuid.UUID, externalIds []uuid.UUID) ([]*sqlcv1.FlattenExternalIdsRow, error)

	CompleteTasks(ctx context.Context, tenantId uuid.UUID, tasks []CompleteTaskOpts) (*FinalizedTaskResponse, error)

	FailTasks(ctx context.Context, tenantId uuid.UUID, tasks []FailTaskOpts) (*FailTasksResponse, error)

	CancelTasks(ctx context.Context, tenantId uuid.UUID, tasks []TaskIdInsertedAtRetryCount) (*FinalizedTaskResponse, error)

	ListTasks(ctx context.Context, tenantId uuid.UUID, tasks []int64) ([]*sqlcv1.V1Task, error)

	ListTaskMetas(ctx context.Context, tenantId uuid.UUID, tasks []int64) ([]*sqlcv1.ListTaskMetasRow, error)

	ListFinalizedWorkflowRuns(ctx context.Context, tenantId uuid.UUID, rootExternalIds []uuid.UUID) ([]*ListFinalizedWorkflowRunsResponse, error)

	// ListTaskParentOutputs is a method to return the output of a task's parent and grandparent tasks. This is for v0 compatibility
	// with the v1 engine, and shouldn't be called from new v1 endpoints.
	ListTaskParentOutputs(ctx context.Context, tenantId uuid.UUID, tasks []*sqlcv1.V1Task) (map[int64][]*TaskOutputEvent, error)

	DefaultTaskActivityGauge(ctx context.Context, tenantId string) (int, error)

	ProcessTaskTimeouts(ctx context.Context, tenantId uuid.UUID) (*TimeoutTasksResponse, bool, error)

	ProcessTaskReassignments(ctx context.Context, tenantId uuid.UUID) (*FailTasksResponse, bool, error)

	ProcessTaskRetryQueueItems(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.V1RetryQueueItem, bool, error)

	ProcessDurableSleeps(ctx context.Context, tenantId uuid.UUID) (*EventMatchResults, bool, error)

	GetQueueCounts(ctx context.Context, tenantId uuid.UUID) (map[string]interface{}, error)

	ReplayTasks(ctx context.Context, tenantId uuid.UUID, tasks []TaskIdInsertedAtRetryCount) (*ReplayTasksResult, error)

	RefreshTimeoutBy(ctx context.Context, tenantId uuid.UUID, opt RefreshTimeoutBy) (*sqlcv1.V1TaskRuntime, error)

	ReleaseSlot(ctx context.Context, tenantId, externalId uuid.UUID) (*sqlcv1.V1TaskRuntime, error)

	ListSignalCompletedEvents(ctx context.Context, tenantId uuid.UUID, tasks []TaskIdInsertedAtSignalKey) ([]*V1TaskEventWithPayload, error)

	// AnalyzeTaskTables runs ANALYZE on the task tables
	AnalyzeTaskTables(ctx context.Context) error

	// Cleanup makes sure to get rid of invalid old entries
	// Returns (shouldContinue, error) where shouldContinue indicates if there's more work
	Cleanup(ctx context.Context) (bool, error)

	GetTaskStats(ctx context.Context, tenantId uuid.UUID) (map[string]TaskStat, error)

	FindOldestRunningTaskInsertedAt(ctx context.Context) (*time.Time, error)

	FindOldestTaskInsertedAt(ctx context.Context) (*time.Time, error)

	// run "details" getter, used for retrieving payloads and status of a run for external consumption without going through the REST API
	GetWorkflowRunResultDetails(ctx context.Context, tenantId uuid.UUID, externalId uuid.UUID) (*WorkflowRunDetails, error)

	FilterValidTasks(ctx context.Context, tenantId uuid.UUID, opts []TaskIdInsertedAtRetryCount) (map[int64]struct{}, error)
}

type TaskRepositoryImpl struct {
	*sharedRepository

	taskRetentionPeriod   time.Duration
	maxInternalRetryCount int32
	timeoutLimit          int
	reassignLimit         int
	retryQueueLimit       int
	durableSleepLimit     int
}

func newTaskRepository(s *sharedRepository, taskRetentionPeriod time.Duration, maxInternalRetryCount int32, timeoutLimit, reassignLimit, retryQueueLimit, durableSleepLimit int) TaskRepository {
	return &TaskRepositoryImpl{
		sharedRepository:      s,
		taskRetentionPeriod:   taskRetentionPeriod,
		maxInternalRetryCount: maxInternalRetryCount,
		timeoutLimit:          timeoutLimit,
		reassignLimit:         reassignLimit,
		retryQueueLimit:       retryQueueLimit,
		durableSleepLimit:     durableSleepLimit,
	}
}

func (r *TaskRepositoryImpl) EnsureTablePartitionsExist(ctx context.Context) (bool, error) {
	return r.queries.EnsureTablePartitionsExist(ctx, r.pool)
}

func (r *TaskRepositoryImpl) UpdateTablePartitions(ctx context.Context) error {
	const PARTITION_LOCK_OFFSET = 9000000000000000000
	const partitionLockKey = PARTITION_LOCK_OFFSET + 1

	tx, commit, rollback, err := sqlchelpers.PrepareTxWithStatementTimeout(ctx, r.pool, r.l, 600000) // 10 minutes
	if err != nil {
		return fmt.Errorf("failed to prepare transaction: %w", err)
	}
	defer rollback()

	acquired, err := r.queries.TryAdvisoryLock(ctx, tx, partitionLockKey)
	if err != nil {
		return fmt.Errorf("failed to try advisory lock for partition operations: %w", err)
	}

	if !acquired {
		r.l.Debug().Msg("partition operations already running on another controller instance, skipping")
		return nil
	}

	r.l.Debug().Msg("acquired advisory lock for partition operations")

	today := time.Now().UTC()
	tomorrow := today.AddDate(0, 0, 1)
	removeBefore := today.Add(-1 * r.taskRetentionPeriod)

	err = r.queries.CreatePartitions(ctx, r.pool, pgtype.Date{
		Time:  today,
		Valid: true,
	})

	if err != nil {
		return err
	}

	err = r.queries.CreatePartitions(ctx, r.pool, pgtype.Date{
		Time:  tomorrow,
		Valid: true,
	})

	if err != nil {
		return err
	}

	partitions, err := r.queries.ListPartitionsBeforeDate(ctx, r.pool, pgtype.Date{
		Time:  removeBefore,
		Valid: true,
	})

	if err != nil {
		return err
	}

	if len(partitions) > 0 {
		r.l.Warn().Msgf("removing partitions before %s using retention period of %s", removeBefore.Format(time.RFC3339), r.taskRetentionPeriod)
	}

	for _, partition := range partitions {
		r.l.Debug().Msgf("detaching partition %s", partition.PartitionName)

		conn, release, err := sqlchelpers.AcquireConnectionWithStatementTimeout(ctx, r.pool, r.l, 30*60*1000) // 30 minutes

		if err != nil {
			return err
		}

		_, err = conn.Exec(
			ctx,
			fmt.Sprintf("ALTER TABLE %s DETACH PARTITION %s CONCURRENTLY", partition.ParentTable, partition.PartitionName),
		)

		if err != nil {
			release()
			return err
		}

		_, err = conn.Exec(
			ctx,
			fmt.Sprintf("DROP TABLE %s", partition.PartitionName),
		)

		if err != nil {
			release()
			return err
		}

		release()
	}

	err = commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *sharedRepository) GetTaskByExternalId(ctx context.Context, tenantId, taskExternalId uuid.UUID, skipCache bool) (*sqlcv1.FlattenExternalIdsRow, error) {

	ctx, span := telemetry.NewSpan(ctx, "TaskRepositoryImpl.GetTaskByExternalId")
	defer span.End()

	if !skipCache {
		// check the cache first
		key := taskExternalIdTenantIdTuple{
			externalId: taskExternalId,
			tenantId:   tenantId,
		}

		if val, ok := r.taskLookupCache.Get(key); ok {
			span.SetAttributes(attribute.Bool("cache_hit", true))
			return val, nil
		}
	}

	span.SetAttributes(attribute.Bool("cache_hit", false))

	// lookup the task
	dbTasks, err := r.queries.FlattenExternalIds(ctx, r.pool, sqlcv1.FlattenExternalIdsParams{
		Tenantid:    tenantId,
		Externalids: []uuid.UUID{taskExternalId},
	})

	if err != nil {
		return nil, err
	}

	if len(dbTasks) == 0 {
		return nil, pgx.ErrNoRows
	}

	if len(dbTasks) > 1 {
		return nil, fmt.Errorf("found more than one task for %s", taskExternalId)
	}

	// set the cache
	res := dbTasks[0]

	key := taskExternalIdTenantIdTuple{
		externalId: taskExternalId,
		tenantId:   tenantId,
	}

	r.taskLookupCache.Add(key, res)

	return res, nil
}

func (r *TaskRepositoryImpl) FlattenExternalIds(ctx context.Context, tenantId uuid.UUID, externalIds []uuid.UUID) ([]*sqlcv1.FlattenExternalIdsRow, error) {
	return r.lookupExternalIds(ctx, r.pool, tenantId, externalIds)
}

func (r *sharedRepository) lookupExternalIds(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, externalIds []uuid.UUID) ([]*sqlcv1.FlattenExternalIdsRow, error) {
	externalIdsToLookup := make([]uuid.UUID, 0, len(externalIds))
	res := make([]*sqlcv1.FlattenExternalIdsRow, 0, len(externalIds))

	for _, externalId := range externalIds {
		if externalId == uuid.Nil {
			r.l.Error().Msgf("passed in empty external id")
			continue
		}

		externalIdsToLookup = append(externalIdsToLookup, externalId)
	}

	// lookup the task
	dbTasks, err := r.queries.FlattenExternalIds(ctx, tx, sqlcv1.FlattenExternalIdsParams{
		Tenantid:    tenantId,
		Externalids: externalIdsToLookup,
	})

	if err != nil {
		return nil, err
	}

	// set the cache
	groupedExternalIds := make(map[string][]*sqlcv1.FlattenExternalIdsRow)

	for _, task := range dbTasks {
		rootExternalId := task.WorkflowRunExternalID.String()

		groupedExternalIds[rootExternalId] = append(groupedExternalIds[rootExternalId], task)
	}

	for _, tasks := range groupedExternalIds {
		res = append(res, tasks...)
	}

	return res, nil
}

func (r *TaskRepositoryImpl) verifyAllTasksFinalized(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, flattenedTasks []*sqlcv1.FlattenExternalIdsRow) ([]uuid.UUID, map[uuid.UUID]int64, error) {
	taskIdsToCheck := make([]int64, len(flattenedTasks))
	taskInsertedAtsToCheck := make([]pgtype.Timestamptz, len(flattenedTasks))
	taskIdsToTasks := make(map[int64]*sqlcv1.FlattenExternalIdsRow)
	minInsertedAt := sqlchelpers.TimestamptzFromTime(time.Now()) // current time as a placeholder - will be overwritten

	for i, task := range flattenedTasks {
		taskIdsToCheck[i] = task.ID
		taskInsertedAtsToCheck[i] = task.InsertedAt
		taskIdsToTasks[task.ID] = task

		if task.InsertedAt.Time.Before(minInsertedAt.Time) {
			minInsertedAt = task.InsertedAt
		}
	}

	// run preflight check on tasks
	notFinalized, err := r.queries.PreflightCheckTasksForReplay(ctx, tx, sqlcv1.PreflightCheckTasksForReplayParams{
		Tenantid:        tenantId,
		Taskids:         taskIdsToCheck,
		Taskinsertedats: taskInsertedAtsToCheck,
		Mininsertedat:   minInsertedAt,
	})

	if err != nil {
		return nil, nil, err
	}

	notFinalizedMap := make(map[int64]bool)

	for _, task := range notFinalized {
		notFinalizedMap[task.ID] = true
	}

	dagsToCheck := make([]int64, 0)
	dagsToTasks := make(map[int64][]*sqlcv1.FlattenExternalIdsRow)

	for _, task := range flattenedTasks {
		if !notFinalizedMap[task.ID] && task.DagID.Valid {
			dagsToCheck = append(dagsToCheck, task.DagID.Int64)
			dagsToTasks[task.DagID.Int64] = append(dagsToTasks[task.DagID.Int64], task)
		}
	}

	// check DAGs
	notFinalizedDags, err := r.queries.PreflightCheckDAGsForReplay(ctx, tx, sqlcv1.PreflightCheckDAGsForReplayParams{
		Dagids:   dagsToCheck,
		Tenantid: tenantId,
	})

	if err != nil {
		return nil, nil, err
	}

	notFinalizedDAGsMap := make(map[int64]bool)
	finalizedDAGToStepCount := make(map[uuid.UUID]int64)

	for _, dag := range notFinalizedDags {
		if dag.StepCount != dag.TaskCount {
			notFinalizedDAGsMap[dag.ID] = true
		} else {
			rootId := dag.ExternalID
			finalizedDAGToStepCount[rootId] = dag.StepCount
		}
	}

	candidateFinalizedRootExternalIds := make(map[uuid.UUID]bool, 0)

	for _, task := range flattenedTasks {
		candidateFinalizedRootExternalIds[task.WorkflowRunExternalID] = true
	}

	// iterate through tasks one last time
	for _, task := range flattenedTasks {
		rootId := task.WorkflowRunExternalID

		// if root is already non-finalized, skip
		if !candidateFinalizedRootExternalIds[rootId] {
			continue
		}

		if notFinalizedMap[task.ID] {
			candidateFinalizedRootExternalIds[rootId] = false
			continue
		}

		if task.DagID.Valid && notFinalizedDAGsMap[task.DagID.Int64] {
			candidateFinalizedRootExternalIds[rootId] = false
			continue
		}
	}

	finalizedRootExternalIds := make([]uuid.UUID, 0)

	for rootId, finalized := range candidateFinalizedRootExternalIds {
		if finalized {
			finalizedRootExternalIds = append(finalizedRootExternalIds, rootId)
		}
	}

	return finalizedRootExternalIds, finalizedDAGToStepCount, nil
}

func (r *TaskRepositoryImpl) CompleteTasks(ctx context.Context, tenantId uuid.UUID, tasks []CompleteTaskOpts) (*FinalizedTaskResponse, error) {
	ctx, span := telemetry.NewSpan(ctx, "TaskRepositoryImpl.CompleteTasks")
	defer span.End()

	// TODO: ADD BACK VALIDATION
	// if err := r.v.Validate(tasks); err != nil {
	// 	fmt.Println("FAILED VALIDATION HERE!!!")

	// 	return err
	// }

	taskIdRetryCounts := make([]TaskIdInsertedAtRetryCount, len(tasks))

	for i, task := range tasks {
		taskIdRetryCounts[i] = *task.TaskIdInsertedAtRetryCount
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		err = fmt.Errorf("failed to prepare tx: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to prepare tx")
		return nil, err
	}

	defer rollback()

	taskIdRetryCounts = uniqueSet(taskIdRetryCounts)

	// release queue items
	releasedTasks, err := r.releaseTasks(ctx, tx, tenantId, taskIdRetryCounts)

	if err != nil {
		err = fmt.Errorf("failed to release tasks: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to release tasks")
		return nil, err
	}

	if len(taskIdRetryCounts) != len(releasedTasks) {
		return nil, fmt.Errorf("failed to release all tasks")
	}

	outputs := make([][]byte, len(releasedTasks))

	for i, releasedTask := range releasedTasks {
		out := NewCompletedTaskOutputEvent(releasedTask, tasks[i].Output).Bytes()

		outputs[i] = out
	}

	internalEvents, err := r.createTaskEventsAfterRelease(
		ctx,
		tx,
		tenantId,
		taskIdRetryCounts,
		outputs,
		releasedTasks,
		sqlcv1.V1TaskEventTypeCOMPLETED,
	)

	if err != nil {
		err = fmt.Errorf("failed to create task events after release: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create task events after release")
		return nil, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		err = fmt.Errorf("failed to commit transaction: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to commit transaction")
		return nil, err
	}

	return &FinalizedTaskResponse{
		ReleasedTasks:  releasedTasks,
		InternalEvents: internalEvents,
	}, nil
}

func (r *TaskRepositoryImpl) FailTasks(ctx context.Context, tenantId uuid.UUID, failureOpts []FailTaskOpts) (*FailTasksResponse, error) {
	ctx, span := telemetry.NewSpan(ctx, "TaskRepositoryImpl.FailTasks")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		err = fmt.Errorf("failed to prepare tx: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to prepare tx")
		return nil, err
	}

	defer rollback()

	res, err := r.failTasksTx(ctx, tx, tenantId, failureOpts)

	if err != nil {
		err = fmt.Errorf("failed to fail tasks: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fail tasks")
		return nil, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		err = fmt.Errorf("failed to commit transaction: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to commit transaction")
		return nil, err
	}

	return res, nil
}

func (r *TaskRepositoryImpl) failTasksTx(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, failureOpts []FailTaskOpts) (*FailTasksResponse, error) {
	// TODO: ADD BACK VALIDATION
	// if err := r.v.Validate(tasks); err != nil {
	// 	fmt.Println("FAILED VALIDATION HERE!!!")

	// 	return err
	// }

	ctx, span := telemetry.NewSpan(ctx, "tasks_repository_impl.fail_tasks_tx")
	defer span.End()

	tasks := make([]TaskIdInsertedAtRetryCount, len(failureOpts))
	appFailureTaskIds := make([]int64, 0)
	appFailureTaskInsertedAts := make([]pgtype.Timestamptz, 0)
	appFailureTaskRetryCounts := make([]int32, 0)
	appFailureIsNonRetryableStatuses := make([]bool, 0)

	internalFailureTaskIds := make([]int64, 0)
	internalFailureInsertedAts := make([]pgtype.Timestamptz, 0)
	internalFailureTaskRetryCounts := make([]int32, 0)

	for i, failureOpt := range failureOpts {
		tasks[i] = *failureOpt.TaskIdInsertedAtRetryCount

		if failureOpt.IsAppError {
			appFailureTaskIds = append(appFailureTaskIds, failureOpt.Id)
			appFailureTaskInsertedAts = append(appFailureTaskInsertedAts, failureOpt.InsertedAt)
			appFailureTaskRetryCounts = append(appFailureTaskRetryCounts, failureOpt.RetryCount)
			appFailureIsNonRetryableStatuses = append(appFailureIsNonRetryableStatuses, failureOpt.IsNonRetryable)
		} else {
			internalFailureTaskIds = append(internalFailureTaskIds, failureOpt.Id)
			internalFailureInsertedAts = append(internalFailureInsertedAts, failureOpt.InsertedAt)
			internalFailureTaskRetryCounts = append(internalFailureTaskRetryCounts, failureOpt.RetryCount)
		}
	}

	tasks = uniqueSet(tasks)

	retriedTasks := make([]RetriedTask, 0)

	// write app failures
	if len(appFailureTaskIds) > 0 {
		span.SetAttributes(
			attribute.KeyValue{
				Key:   "tasks_repository_impl.fail_tasks_tx.fail_task_app_failure.batch_size",
				Value: attribute.IntValue(len(appFailureTaskIds)),
			},
		)
		appFailureRetries, err := r.queries.FailTaskAppFailure(ctx, tx, sqlcv1.FailTaskAppFailureParams{
			Tenantid:        tenantId,
			Taskids:         appFailureTaskIds,
			Taskinsertedats: appFailureTaskInsertedAts,
			Taskretrycounts: appFailureTaskRetryCounts,
			Isnonretryables: appFailureIsNonRetryableStatuses,
		})

		if err != nil {
			return nil, err
		}

		for _, task := range appFailureRetries {
			retriedTasks = append(retriedTasks, RetriedTask{
				TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
					Id:         task.ID,
					InsertedAt: task.InsertedAt,
					RetryCount: task.RetryCount,
				},
				IsAppError:         true,
				AppRetryCount:      task.AppRetryCount,
				RetryBackoffFactor: task.RetryBackoffFactor,
				RetryMaxBackoff:    task.RetryMaxBackoff,
			},
			)
		}
	}

	// write internal failures
	if len(internalFailureTaskIds) > 0 {
		span.SetAttributes(
			attribute.KeyValue{
				Key:   "tasks_repository_impl.fail_tasks_tx.fail_task_internal_failure.batch_size",
				Value: attribute.IntValue(len(internalFailureTaskIds)),
			},
		)
		internalFailureRetries, err := r.queries.FailTaskInternalFailure(ctx, tx, sqlcv1.FailTaskInternalFailureParams{
			Tenantid:           tenantId,
			Taskids:            internalFailureTaskIds,
			Taskinsertedats:    internalFailureInsertedAts,
			Taskretrycounts:    internalFailureTaskRetryCounts,
			Maxinternalretries: r.maxInternalRetryCount,
		})

		if err != nil {
			return nil, err
		}

		for _, task := range internalFailureRetries {
			retriedTasks = append(retriedTasks, RetriedTask{
				TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
					Id:         task.ID,
					InsertedAt: task.InsertedAt,
					RetryCount: task.RetryCount,
				},
			})
		}
	}

	// release queue items
	// NOTE: it's important that we do this after we've written the retries, as some of the triggers for concurrency
	// slots case on the retry queue item's existence.
	releasedTasks, err := r.releaseTasks(ctx, tx, tenantId, tasks)

	if err != nil {
		return nil, err
	}

	outputs := make([][]byte, len(releasedTasks))

	for i, releasedTask := range releasedTasks {
		out := NewFailedTaskOutputEvent(releasedTask, failureOpts[i].ErrorMessage).Bytes()

		outputs[i] = out
	}

	internalEvents, err := r.createTaskEventsAfterRelease(
		ctx,
		tx,
		tenantId,
		tasks,
		outputs,
		releasedTasks,
		sqlcv1.V1TaskEventTypeFAILED,
	)

	if err != nil {
		return nil, err
	}

	return &FailTasksResponse{
		FinalizedTaskResponse: &FinalizedTaskResponse{
			ReleasedTasks:  releasedTasks,
			InternalEvents: internalEvents,
		},
		RetriedTasks: retriedTasks,
	}, nil
}

func (r *TaskRepositoryImpl) ListFinalizedWorkflowRuns(ctx context.Context, tenantId uuid.UUID, rootExternalIds []uuid.UUID) ([]*ListFinalizedWorkflowRunsResponse, error) {
	start := time.Now()
	checkpoint := time.Now()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	externalIdsToEvents := make(map[uuid.UUID][]*TaskOutputEvent)

	tasks, err := r.lookupExternalIds(ctx, tx, tenantId, rootExternalIds)

	if err != nil {
		return nil, err
	}

	durLookup := time.Since(checkpoint)
	checkpoint = time.Now()

	finalizedRootIds, rootIdToStepCounts, err := r.verifyAllTasksFinalized(ctx, tx, tenantId, tasks)

	if err != nil {
		return nil, err
	}

	durVerify := time.Since(checkpoint)
	checkpoint = time.Now()

	finalizedRootIdsSet := make(map[uuid.UUID]bool)
	for _, rootId := range finalizedRootIds {
		finalizedRootIdsSet[rootId] = true
	}

	taskExternalIds := make([]uuid.UUID, 0, len(tasks))
	taskExternalIdsToRootIds := make(map[uuid.UUID]uuid.UUID)

	for _, task := range tasks {
		rootId := task.WorkflowRunExternalID
		taskExternalId := task.ExternalID

		if finalizedRootIdsSet[rootId] {
			taskExternalIds = append(taskExternalIds, taskExternalId)
			taskExternalIdsToRootIds[taskExternalId] = rootId
		}
	}

	if len(taskExternalIds) == 0 {
		if commitErr := commit(ctx); commitErr != nil {
			return nil, commitErr
		}
		return []*ListFinalizedWorkflowRunsResponse{}, nil
	}

	outputEvents, err := r.listTaskOutputEvents(ctx, tx, tenantId, taskExternalIds)

	if err != nil {
		return nil, err
	}

	durOutputEvents := time.Since(checkpoint)

	if err := commit(ctx); err != nil {
		return nil, err
	}

	taskExternalIdsHasOutputEvent := make(map[uuid.UUID]bool)

	// group the output events by their parent id
	for _, outputEvent := range outputEvents {
		rootId, ok := taskExternalIdsToRootIds[outputEvent.TaskExternalId]

		if !ok {
			r.l.Warn().Msgf("could not find root id for task %s", outputEvent.TaskExternalId)
			continue
		}

		externalIdsToEvents[rootId] = append(externalIdsToEvents[rootId], outputEvent)
		taskExternalIdsHasOutputEvent[outputEvent.TaskExternalId] = true
	}

	// if tasks that we read originally don't have a TaskOutputEvent, they're not finalized, so set their root
	// ids in finalizedRootIdsSet to false (safety check for race conditions)
	for _, taskExternalId := range taskExternalIds {
		if !taskExternalIdsHasOutputEvent[taskExternalId] {
			rootId, ok := taskExternalIdsToRootIds[taskExternalId]

			if !ok {
				r.l.Warn().Msgf("could not find root id for task %s", taskExternalId)
				continue
			}

			finalizedRootIdsSet[rootId] = false
		}
	}

	// look for finalized events...
	eventsForFinalizedRootIds := make(map[uuid.UUID][]*TaskOutputEvent)

	for _, rootId := range finalizedRootIds {
		if !finalizedRootIdsSet[rootId] {
			continue
		}

		events := externalIdsToEvents[rootId]

		// if the length of the rootId -> stepCount is less than the number of events, it's not finalized
		if _, ok := rootIdToStepCounts[rootId]; ok && rootIdToStepCounts[rootId] > int64(len(events)) {
			continue
		}

		eventsForFinalizedRootIds[rootId] = events
	}

	// put together response
	res := make([]*ListFinalizedWorkflowRunsResponse, 0, len(eventsForFinalizedRootIds))

	for rootId, events := range eventsForFinalizedRootIds {
		res = append(res, &ListFinalizedWorkflowRunsResponse{
			WorkflowRunId: rootId,
			OutputEvents:  events,
		})
	}

	if time.Since(start) > 100*time.Millisecond {
		r.l.Warn().Dur(
			"lookup_duration",
			durLookup,
		).Dur(
			"verify_duration",
			durVerify,
		).Dur(
			"output_events_duration",
			durOutputEvents,
		).Dur("total_duration", time.Since(start)).Msgf("slow finalized workflow runs lookup")
	}

	return res, nil
}

func (r *TaskRepositoryImpl) CancelTasks(ctx context.Context, tenantId uuid.UUID, tasks []TaskIdInsertedAtRetryCount) (*FinalizedTaskResponse, error) {
	ctx, span := telemetry.NewSpan(ctx, "TaskRepositoryImpl.CancelTasks")
	defer span.End()

	// TODO: ADD BACK VALIDATION
	// if err := r.v.Validate(tasks); err != nil {
	// 	fmt.Println("FAILED VALIDATION HERE!!!")

	// 	return err
	// }

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		err = fmt.Errorf("failed to prepare tx: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to prepare tx")
		return nil, err
	}

	defer rollback()

	// release queue items
	res, err := r.cancelTasks(ctx, tx, tenantId, tasks)

	if err != nil {
		err = fmt.Errorf("failed to cancel tasks: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to cancel tasks")
		return nil, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		err = fmt.Errorf("failed to commit transaction: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to commit transaction")
		return nil, err
	}

	return res, nil
}

func (r *sharedRepository) cancelTasks(ctx context.Context, dbtx sqlcv1.DBTX, tenantId uuid.UUID, tasks []TaskIdInsertedAtRetryCount) (*FinalizedTaskResponse, error) {
	// get a unique set of task ids and retry counts
	tasks = uniqueSet(tasks)

	// release queue items
	releasedTasks, err := r.releaseTasks(ctx, dbtx, tenantId, tasks)

	if err != nil {
		return nil, err
	}

	outputs := make([][]byte, len(releasedTasks))

	for i, releasedTask := range releasedTasks {
		out := NewCancelledTaskOutputEvent(releasedTask).Bytes()

		outputs[i] = out
	}

	internalEvents, err := r.createTaskEventsAfterRelease(
		ctx,
		dbtx,
		tenantId,
		tasks,
		outputs,
		releasedTasks,
		sqlcv1.V1TaskEventTypeCANCELLED,
	)

	if err != nil {
		return nil, err
	}

	return &FinalizedTaskResponse{
		ReleasedTasks:  releasedTasks,
		InternalEvents: internalEvents,
	}, nil
}

func (r *TaskRepositoryImpl) ListTasks(ctx context.Context, tenantId uuid.UUID, tasks []int64) ([]*sqlcv1.V1Task, error) {
	return r.listTasks(ctx, r.pool, tenantId, tasks)
}

func (r *sharedRepository) listTasks(ctx context.Context, dbtx sqlcv1.DBTX, tenantId uuid.UUID, tasks []int64) ([]*sqlcv1.V1Task, error) {
	return r.queries.ListTasks(ctx, dbtx, sqlcv1.ListTasksParams{
		TenantID: tenantId,
		Ids:      tasks,
	})
}

func (r *TaskRepositoryImpl) listTaskOutputEvents(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, taskExternalIds []uuid.UUID) ([]*TaskOutputEvent, error) {
	eventTypes := make([][]string, 0)

	for range taskExternalIds {
		eventTypes = append(eventTypes, []string{
			string(sqlcv1.V1TaskEventTypeCOMPLETED),
			string(sqlcv1.V1TaskEventTypeFAILED),
			string(sqlcv1.V1TaskEventTypeCANCELLED),
		})
	}

	matchedEvents, err := r.queries.ListMatchingTaskEvents(ctx, tx, sqlcv1.ListMatchingTaskEventsParams{
		Tenantid:        tenantId,
		Taskexternalids: taskExternalIds,
		Eventtypes:      eventTypes,
	})

	if err != nil {
		return nil, err
	}

	retrieveOpts := make([]RetrievePayloadOpts, len(matchedEvents))
	retrieveOptsToEventData := make(map[RetrievePayloadOpts][]byte)
	matchedEventToRetrieveOpts := make(map[*sqlcv1.ListMatchingTaskEventsRow]RetrievePayloadOpts)

	for i, event := range matchedEvents {
		opt := RetrievePayloadOpts{
			Id:         event.ID,
			InsertedAt: event.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeTASKEVENTDATA,
			TenantId:   tenantId,
		}

		retrieveOpts[i] = opt
		retrieveOptsToEventData[opt] = event.Data
		matchedEventToRetrieveOpts[event] = opt
	}

	payloads, err := r.payloadStore.Retrieve(ctx, tx, retrieveOpts...)

	if err != nil {
		return nil, err
	}

	res := make([]*TaskOutputEvent, 0, len(matchedEvents))

	for _, event := range matchedEvents {
		retrieveOpts := matchedEventToRetrieveOpts[event]
		payload, ok := payloads[retrieveOpts]

		if !ok {
			payload = retrieveOptsToEventData[retrieveOpts]
		}

		o, err := newTaskEventFromBytes(payload)

		if err != nil {
			return nil, err
		}

		res = append(res, o)
	}

	return res, nil
}

func (r *TaskRepositoryImpl) ListTaskMetas(ctx context.Context, tenantId uuid.UUID, tasks []int64) ([]*sqlcv1.ListTaskMetasRow, error) {
	return r.queries.ListTaskMetas(ctx, r.pool, sqlcv1.ListTaskMetasParams{
		TenantID: tenantId,
		Ids:      tasks,
	})
}

// DefaultTaskActivityGauge is a heavily cached method that returns the number of queues that have had activity since
// the task retention period.
func (r *TaskRepositoryImpl) DefaultTaskActivityGauge(ctx context.Context, tenantId string) (int, error) {
	today := time.Now().UTC()
	notBefore := today.Add(-1 * r.taskRetentionPeriod)
	tenantIdUuid, err := uuid.Parse(tenantId)
	if err != nil {
		return 0, err
	}

	res, err := r.queries.DefaultTaskActivityGauge(ctx, r.pool, sqlcv1.DefaultTaskActivityGaugeParams{
		Tenantid: tenantIdUuid,
		Activesince: pgtype.Timestamptz{
			Time:  notBefore,
			Valid: true,
		},
	})

	return int(res), err
}

func (r *TaskRepositoryImpl) ProcessTaskTimeouts(ctx context.Context, tenantId uuid.UUID) (*TimeoutTasksResponse, bool, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, false, err
	}

	defer rollback()

	limit := r.timeoutLimit

	// get task timeouts
	toTimeout, err := r.queries.ListTasksToTimeout(ctx, tx, sqlcv1.ListTasksToTimeoutParams{
		Tenantid: tenantId,
		Limit: pgtype.Int4{
			Int32: int32(limit),
			Valid: true,
		},
	})

	if err != nil {
		return nil, false, err
	}

	if len(toTimeout) == 0 {
		return &TimeoutTasksResponse{
			FailTasksResponse: &FailTasksResponse{
				FinalizedTaskResponse: &FinalizedTaskResponse{
					ReleasedTasks:  make([]*sqlcv1.ReleaseTasksRow, 0),
					InternalEvents: make([]InternalTaskEvent, 0),
				},
				RetriedTasks: make([]RetriedTask, 0),
			},
			TimeoutTasks: make([]*sqlcv1.ListTasksToTimeoutRow, 0),
		}, false, nil
	}

	// parse into FailTaskOpts
	failOpts := make([]FailTaskOpts, 0, len(toTimeout))

	for _, task := range toTimeout {
		failOpts = append(failOpts, FailTaskOpts{
			TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
				Id:         task.ID,
				InsertedAt: task.InsertedAt,
				RetryCount: task.RetryCount,
			},
			IsAppError:     true,
			ErrorMessage:   fmt.Sprintf("Task exceeded timeout of %s", task.StepTimeout.String),
			IsNonRetryable: false,
		})
	}

	// fail the tasks
	failResp, err := r.failTasksTx(ctx, tx, tenantId, failOpts)

	if err != nil {
		return nil, false, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return nil, false, err
	}

	return &TimeoutTasksResponse{
		FailTasksResponse: failResp,
		TimeoutTasks:      toTimeout,
	}, len(toTimeout) == limit, nil
}

func (r *TaskRepositoryImpl) ProcessTaskReassignments(ctx context.Context, tenantId uuid.UUID) (*FailTasksResponse, bool, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, false, err
	}

	defer rollback()

	limit := r.reassignLimit

	toReassign, err := r.queries.ListTasksToReassign(ctx, tx, sqlcv1.ListTasksToReassignParams{
		Tenantid: tenantId,
		Limit: pgtype.Int4{
			Int32: int32(limit),
			Valid: true,
		},
	})

	if err != nil {
		return nil, false, err
	}

	if len(toReassign) == 0 {
		return &FailTasksResponse{
			FinalizedTaskResponse: &FinalizedTaskResponse{
				ReleasedTasks:  make([]*sqlcv1.ReleaseTasksRow, 0),
				InternalEvents: make([]InternalTaskEvent, 0),
			},
			RetriedTasks: make([]RetriedTask, 0),
		}, false, nil
	}

	// parse into FailTaskOpts
	failOpts := make([]FailTaskOpts, 0, len(toReassign))

	for _, task := range toReassign {
		failOpts = append(failOpts, FailTaskOpts{
			TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
				Id:         task.ID,
				InsertedAt: task.InsertedAt,
				RetryCount: task.RetryCount,
			},
			IsAppError:     false,
			IsNonRetryable: false,
		})
	}

	// fail the tasks
	res, err := r.failTasksTx(ctx, tx, tenantId, failOpts)

	if err != nil {
		return nil, false, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return nil, false, err
	}

	return res, len(toReassign) == limit, nil
}

func (r *TaskRepositoryImpl) ProcessTaskRetryQueueItems(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.V1RetryQueueItem, bool, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, false, err
	}

	defer rollback()

	limit := r.retryQueueLimit

	// get task reassignments
	res, err := r.queries.ProcessRetryQueueItems(ctx, tx, sqlcv1.ProcessRetryQueueItemsParams{
		Tenantid: tenantId,
		Limit: pgtype.Int4{
			Int32: int32(limit),
			Valid: true,
		},
	})

	if err != nil {
		return nil, false, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return nil, false, err
	}

	return res, len(res) == limit, nil
}

type durableSleepEventData struct {
	SleepDuration string `json:"sleep_duration"`
}

func (r *TaskRepositoryImpl) ProcessDurableSleeps(ctx context.Context, tenantId uuid.UUID) (*EventMatchResults, bool, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, false, err
	}

	defer rollback()

	limit := r.durableSleepLimit

	emitted, err := r.queries.PopDurableSleep(ctx, tx, sqlcv1.PopDurableSleepParams{
		TenantID: tenantId,
		Limit:    pgtype.Int4{Int32: int32(limit), Valid: true},
	})

	if err != nil {
		return nil, false, err
	}

	// each emitted item becomes a candidate event match for internal events
	events := make([]CandidateEventMatch, 0, len(emitted))

	for _, sleep := range emitted {
		data, err := json.Marshal(durableSleepEventData{
			SleepDuration: sleep.SleepDuration,
		})

		if err != nil {
			return nil, false, err
		}

		events = append(events, CandidateEventMatch{
			ID:             uuid.New(),
			EventTimestamp: time.Now(),
			Key:            getDurableSleepEventKey(sleep.ID),
			Data:           data,
		})
	}

	results, err := r.processEventMatches(ctx, tx, tenantId, events, sqlcv1.V1EventTypeINTERNAL)

	if err != nil {
		return nil, false, err
	}

	storePayloadOpts := make([]StorePayloadOpts, len(results.CreatedTasks))
	for i, task := range results.CreatedTasks {
		storePayloadOpts[i] = StorePayloadOpts{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeTASKINPUT,
			ExternalId: task.ExternalID,
			Payload:    task.Payload,
			TenantId:   task.TenantID,
		}
	}

	if len(storePayloadOpts) > 0 {
		err = r.payloadStore.Store(ctx, tx, storePayloadOpts...)

		if err != nil {
			return nil, false, fmt.Errorf("failed to store payloads for created tasks for durable sleep matches: %w", err)
		}
	}

	if err := commit(ctx); err != nil {
		return nil, false, err
	}

	return results, len(emitted) == limit, nil
}

func (r *TaskRepositoryImpl) GetQueueCounts(ctx context.Context, tenantId uuid.UUID) (map[string]interface{}, error) {
	counts, err := r.getFIFOQueuedCounts(ctx, tenantId)

	if err != nil {
		return nil, err
	}

	concurrencyCounts, err := r.getConcurrencyQueuedCounts(ctx, tenantId)

	if err != nil {
		return nil, err
	}

	res := make(map[string]interface{})

	for k, v := range counts {
		res[k] = v
	}

	for k, v := range concurrencyCounts {
		res[k] = v
	}

	return res, nil
}

func (r *TaskRepositoryImpl) getFIFOQueuedCounts(ctx context.Context, tenantId uuid.UUID) (map[string]interface{}, error) {
	counts, err := r.queries.GetQueuedCounts(ctx, r.pool, tenantId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return map[string]interface{}{}, nil
		}

		return nil, err
	}

	res := make(map[string]interface{})

	for _, count := range counts {
		res[count.Queue] = int(count.Count)
	}

	return res, nil
}

func (r *TaskRepositoryImpl) getConcurrencyQueuedCounts(ctx context.Context, tenantId uuid.UUID) (map[string]interface{}, error) {
	concurrencyCounts, err := r.queries.GetWorkflowConcurrencyQueueCounts(ctx, r.pool, tenantId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return map[string]interface{}{}, nil
		}

		return nil, err
	}

	res := make(map[string]interface{})

	for _, count := range concurrencyCounts {
		if _, ok := res[count.WorkflowName]; !ok {
			res[count.WorkflowName] = map[string]int{}
		}

		v := res[count.WorkflowName].(map[string]int)

		v[count.Key] = int(count.Count)

		res[count.WorkflowName] = v
	}

	return res, nil
}

func (r *TaskRepositoryImpl) RefreshTimeoutBy(ctx context.Context, tenantId uuid.UUID, opt RefreshTimeoutBy) (*sqlcv1.V1TaskRuntime, error) {
	if err := r.v.Validate(opt); err != nil {
		return nil, err
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	res, err := r.queries.RefreshTimeoutBy(ctx, tx, sqlcv1.RefreshTimeoutByParams{
		Tenantid:           tenantId,
		Externalid:         opt.TaskExternalId,
		IncrementTimeoutBy: sqlchelpers.TextFromStr(opt.IncrementTimeoutBy),
	})

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return res, nil
}

func (r *TaskRepositoryImpl) ReleaseSlot(ctx context.Context, tenantId, externalId uuid.UUID) (*sqlcv1.V1TaskRuntime, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	resp, err := r.queries.ManualSlotRelease(
		ctx,
		tx,
		sqlcv1.ManualSlotReleaseParams{
			Tenantid:   tenantId,
			Externalid: externalId,
		},
	)

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return resp, nil
}

func (r *sharedRepository) releaseTasks(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, tasks []TaskIdInsertedAtRetryCount) ([]*sqlcv1.ReleaseTasksRow, error) {
	taskIds := make([]int64, len(tasks))
	taskInsertedAts := make([]pgtype.Timestamptz, len(tasks))
	retryCounts := make([]int32, len(tasks))
	orderedMap := make(map[string]int)

	for i, task := range tasks {
		taskIds[i] = task.Id
		taskInsertedAts[i] = task.InsertedAt
		retryCounts[i] = task.RetryCount

		orderedMap[fmt.Sprintf("%d:%d", task.Id, task.RetryCount)] = i
	}

	releasedTasks, err := r.queries.ReleaseTasks(ctx, tx, sqlcv1.ReleaseTasksParams{
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Retrycounts:     retryCounts,
	})

	if err != nil {
		return nil, err
	}

	if len(releasedTasks) != len(tasks) {
		size := min(10, len(tasks))

		taskIds := make([]int64, size)
		for i := range size {
			taskIds[i] = tasks[i].Id
		}

		return nil, fmt.Errorf("failed to release all tasks for tenant %s: %d/%d. Relevant task IDs: %v", tenantId, len(releasedTasks), size, taskIds)
	}

	res := make([]*sqlcv1.ReleaseTasksRow, len(tasks))

	for _, task := range releasedTasks {
		idx := orderedMap[fmt.Sprintf("%d:%d", task.ID, task.RetryCount)]
		res[idx] = task
	}

	return res, nil
}

func (r *sharedRepository) upsertQueues(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, queues []string) (func(), error) {
	queuesToInsert := make(map[string]struct{}, 0)

	for _, queue := range queues {
		if _, ok := queuesToInsert[queue]; ok {
			continue
		}

		key := getQueueCacheKey(tenantId, queue)

		if hasSetQueue, ok := r.queueCache.Get(key); ok && hasSetQueue.(bool) {
			continue
		}

		queuesToInsert[queue] = struct{}{}
	}

	uniqueQueues := make([]string, 0, len(queuesToInsert))

	for queue := range queuesToInsert {
		uniqueQueues = append(uniqueQueues, queue)
	}

	err := r.queries.UpsertQueues(ctx, tx, sqlcv1.UpsertQueuesParams{
		TenantID: tenantId,
		Names:    uniqueQueues,
	})

	if err != nil {
		return nil, err
	}

	// set all the queues to true in the cache
	save := func() {
		for _, queue := range uniqueQueues {
			key := getQueueCacheKey(tenantId, queue)
			r.queueCache.Set(key, true)
		}
	}

	return save, nil
}

func getQueueCacheKey(tenantId uuid.UUID, queue string) string {
	return fmt.Sprintf("%s:%s", tenantId, queue)
}

func (r *sharedRepository) createTasks(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId uuid.UUID,
	tasks []CreateTaskOpts,
) ([]*V1TaskWithPayload, error) {
	// list the steps for the tasks
	uniqueStepIds := make(map[uuid.UUID]struct{})
	stepIds := make([]uuid.UUID, 0)
	externalIdToPayload := make(map[uuid.UUID][]byte, len(tasks))

	for _, task := range tasks {
		if _, ok := uniqueStepIds[task.StepId]; !ok {
			uniqueStepIds[task.StepId] = struct{}{}
			stepIds = append(stepIds, task.StepId)
			externalIdToPayload[task.ExternalId] = task.Input.Bytes()
		}
	}

	steps, err := r.queries.ListStepsByIds(ctx, tx, sqlcv1.ListStepsByIdsParams{
		Ids:      stepIds,
		Tenantid: tenantId,
	})

	if err != nil {
		return nil, err
	}

	stepIdsToConfig := make(map[uuid.UUID]*sqlcv1.ListStepsByIdsRow)

	for _, step := range steps {
		stepIdsToConfig[step.ID] = step
	}

	return r.insertTasks(ctx, tx, tenantId, tasks, stepIdsToConfig)
}

// insertTasks inserts new tasks into the database. note that we're using Postgres rules to automatically insert the created
// tasks into the queue_items table.
func (r *sharedRepository) insertTasks(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId uuid.UUID,
	tasks []CreateTaskOpts,
	stepIdsToConfig map[uuid.UUID]*sqlcv1.ListStepsByIdsRow,
) ([]*V1TaskWithPayload, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

	expressions, err := r.getStepExpressions(ctx, tx, stepIdsToConfig)

	if err != nil {
		return nil, fmt.Errorf("failed to get step expressions: %w", err)
	}

	concurrencyStrats, err := r.getConcurrencyExpressions(ctx, tx, tenantId, stepIdsToConfig)

	if err != nil {
		return nil, fmt.Errorf("failed to get concurrency expressions: %w", err)
	}

	tenantIds := make([]uuid.UUID, len(tasks))
	queues := make([]string, len(tasks))
	actionIds := make([]string, len(tasks))
	stepIds := make([]uuid.UUID, len(tasks))
	stepReadableIds := make([]string, len(tasks))
	workflowIds := make([]uuid.UUID, len(tasks))
	scheduleTimeouts := make([]string, len(tasks))
	stepTimeouts := make([]string, len(tasks))
	priorities := make([]int32, len(tasks))
	stickies := make([]string, len(tasks))
	desiredWorkerIds := make([]*uuid.UUID, len(tasks))
	externalIds := make([]uuid.UUID, len(tasks))
	displayNames := make([]string, len(tasks))
	retryCounts := make([]int32, len(tasks))
	additionalMetadatas := make([][]byte, len(tasks))
	initialStates := make([]string, len(tasks))
	initialStateReasons := make([]pgtype.Text, len(tasks))
	dagIds := make([]pgtype.Int8, len(tasks))
	dagInsertedAts := make([]pgtype.Timestamptz, len(tasks))
	parentStrategyIds := make([][]pgtype.Int8, len(tasks))
	strategyIds := make([][]int64, len(tasks))
	concurrencyKeys := make([][]string, len(tasks))
	parentTaskExternalIds := make([]*uuid.UUID, len(tasks))
	parentTaskIds := make([]pgtype.Int8, len(tasks))
	parentTaskInsertedAts := make([]pgtype.Timestamptz, len(tasks))
	childIndices := make([]pgtype.Int8, len(tasks))
	childKeys := make([]pgtype.Text, len(tasks))
	stepIndices := make([]int64, len(tasks))
	retryBackoffFactors := make([]pgtype.Float8, len(tasks))
	retryMaxBackoffs := make([]pgtype.Int4, len(tasks))
	createExpressionOpts := make(map[uuid.UUID][]createTaskExpressionEvalOpt, 0)
	workflowVersionIds := make([]uuid.UUID, len(tasks))
	workflowRunIds := make([]uuid.UUID, len(tasks))

	externalIdToInput := make(map[uuid.UUID][]byte, len(tasks))

	unix := time.Now().UnixMilli()

	cleanupParentStrategyIds := make([]int64, 0)
	cleanupWorkflowVersionIds := make([]uuid.UUID, 0)
	cleanupWorkflowRunIds := make([]uuid.UUID, 0)

	for i, task := range tasks {
		stepConfig := stepIdsToConfig[task.StepId]
		tenantIds[i] = tenantId
		queues[i] = stepConfig.ActionId // FIXME: make the queue name dynamic
		actionIds[i] = stepConfig.ActionId
		stepIds[i] = task.StepId
		stepReadableIds[i] = stepConfig.ReadableId.String
		workflowIds[i] = stepConfig.WorkflowId
		workflowVersionIds[i] = stepConfig.WorkflowVersionId
		scheduleTimeouts[i] = stepConfig.ScheduleTimeout
		stepTimeouts[i] = stepConfig.Timeout.String
		externalIds[i] = task.ExternalId
		displayNames[i] = fmt.Sprintf("%s-%d", stepConfig.ReadableId.String, unix)
		stepIndices[i] = int64(task.StepIndex)
		retryBackoffFactors[i] = stepConfig.RetryBackoffFactor
		retryMaxBackoffs[i] = stepConfig.RetryMaxBackoff
		workflowRunIds[i] = task.WorkflowRunId

		// TODO: case on whether this is a v1 or v2 task by looking at the step data. for now,
		// we're assuming a v1 task.
		externalIdToInput[task.ExternalId] = r.ToV1StepRunData(task.Input).Bytes()

		retryCounts[i] = 0

		defaultPriority := stepConfig.DefaultPriority
		priority := defaultPriority

		if task.Priority != nil {
			priority = *task.Priority
		}

		priorities[i] = priority

		stickies[i] = string(sqlcv1.V1StickyStrategyNONE)

		if stepConfig.WorkflowVersionSticky.Valid {
			stickies[i] = string(stepConfig.WorkflowVersionSticky.StickyStrategy)
		}

		desiredWorkerIds[i] = task.DesiredWorkerId

		initialStates[i] = string(task.InitialState)
		if initialStates[i] == "" {
			initialStates[i] = string(sqlcv1.V1TaskInitialStateQUEUED)
		}

		if len(task.AdditionalMetadata) > 0 {
			additionalMetadatas[i] = task.AdditionalMetadata
		}

		if task.DagId != nil && task.DagInsertedAt.Valid {
			dagIds[i] = pgtype.Int8{
				Int64: *task.DagId,
				Valid: true,
			}

			dagInsertedAts[i] = task.DagInsertedAt
		}

		parentTaskExternalIds[i] = task.ParentTaskExternalId

		if task.ParentTaskId != nil {
			parentTaskIds[i] = pgtype.Int8{
				Int64: *task.ParentTaskId,
				Valid: true,
			}
		}

		if task.ParentTaskInsertedAt != nil {
			parentTaskInsertedAts[i] = sqlchelpers.TimestamptzFromTime(*task.ParentTaskInsertedAt)
		}

		if task.ChildIndex != nil {
			childIndices[i] = pgtype.Int8{
				Int64: *task.ChildIndex,
				Valid: true,
			}
		}

		if task.ChildKey != nil {
			childKeys[i] = pgtype.Text{
				String: *task.ChildKey,
				Valid:  true,
			}
		}

		concurrencyKeys[i] = make([]string, 0)

		// we write any parent strategy ids to the task regardless of initial state, as we need to know
		// when to release the parent concurrency slot
		taskParentStrategyIds := make([]pgtype.Int8, 0)
		taskStrategyIds := make([]int64, 0)
		emptyConcurrencyKeys := make([]string, 0)

		if strats, ok := concurrencyStrats[task.StepId]; ok {
			for _, strat := range strats {
				taskStrategyIds = append(taskStrategyIds, strat.ID)
				taskParentStrategyIds = append(taskParentStrategyIds, strat.ParentStrategyID)
				emptyConcurrencyKeys = append(emptyConcurrencyKeys, "")

				// we only need to cleanup parent strategy ids if the task is not in a QUEUED state, because
				// this skips the creation of a concurrency slot and means we might want to cleanup the workflow slot
				if strat.ParentStrategyID.Valid && task.InitialState != sqlcv1.V1TaskInitialStateQUEUED {
					cleanupParentStrategyIds = append(cleanupParentStrategyIds, strat.ParentStrategyID.Int64)
					cleanupWorkflowRunIds = append(cleanupWorkflowRunIds, task.WorkflowRunId)
					cleanupWorkflowVersionIds = append(cleanupWorkflowVersionIds, stepConfig.WorkflowVersionId)
				}
			}
		}

		parentStrategyIds[i] = taskParentStrategyIds
		strategyIds[i] = taskStrategyIds
		concurrencyKeys[i] = emptyConcurrencyKeys

		// only check for concurrency if the task is in a queued state, otherwise we don't need to
		// evaluate the expression (and it will likely fail if we do)
		if task.InitialState == sqlcv1.V1TaskInitialStateQUEUED {
			// if we have a step expression, evaluate the expression
			if strats, ok := concurrencyStrats[task.StepId]; ok {
				taskConcurrencyKeys := make([]string, 0)
				var failTaskError error

				for _, strat := range strats {
					var additionalMeta map[string]interface{}

					if len(additionalMetadatas[i]) > 0 {
						if err := json.Unmarshal(additionalMetadatas[i], &additionalMeta); err != nil {
							failTaskError = fmt.Errorf("failed to process additional metadata: not a json object")
							break
						}
					}

					if task.Input == nil {
						failTaskError = fmt.Errorf("failed to parse step expression (%s): input is nil", strat.Expression)
						break
					}

					// Make sure to fail the task with a user-friendly error if we can't parse the CEL for priority
					// Can set fail task error which will insert with an initial state of failed
					res, err := r.celParser.ParseAndEvalStepRun(strat.Expression, cel.NewInput(
						cel.WithInput(task.Input.Input),
						cel.WithAdditionalMetadata(additionalMeta),
						cel.WithWorkflowRunID(task.ExternalId),
						cel.WithParents(task.Input.TriggerData),
					))

					if err != nil {
						failTaskError = fmt.Errorf("failed to parse step expression (%s): %w", strat.Expression, err)
						break
					}

					if res.String == nil {
						prefix := "expected string output for concurrency key"

						if res.Int != nil {
							failTaskError = fmt.Errorf("failed to parse step expression (%s): %s, got int", strat.Expression, prefix)
							break
						}

						failTaskError = fmt.Errorf("failed to parse step expression (%s): %s, got unknown type", strat.Expression, prefix)
						break
					}

					taskConcurrencyKeys = append(taskConcurrencyKeys, *res.String)
				}

				if failTaskError != nil {
					// place the task into a failed state
					initialStates[i] = string(sqlcv1.V1TaskInitialStateFAILED)

					initialStateReasons[i] = pgtype.Text{
						String: failTaskError.Error(),
						Valid:  true,
					}

					// set to "FAILED" for each strategy to maintain cardinality in multi-dimensional array
					failedKeys := make([]string, len(strats))
					for j := range failedKeys {
						failedKeys[j] = "FAILED"
					}
					concurrencyKeys[i] = failedKeys
				} else {
					concurrencyKeys[i] = taskConcurrencyKeys
				}
			}
		}

		// next, check for step expressions to evaluate
		if task.InitialState == sqlcv1.V1TaskInitialStateQUEUED && stepConfig.ExprCount > 0 {
			expressions, ok := expressions[task.StepId]

			if ok {
				var failTaskError error

				opts := make([]createTaskExpressionEvalOpt, 0)

				for _, expr := range expressions {
					var additionalMeta map[string]interface{}

					if len(additionalMetadatas[i]) > 0 {
						if err := json.Unmarshal(additionalMetadatas[i], &additionalMeta); err != nil {
							failTaskError = fmt.Errorf("failed to process additional metadata: not a json object")
							break
						}
					}

					if task.Input == nil {
						failTaskError = fmt.Errorf("failed to parse step expression (%s): input is nil", expr.Expression)
						break
					}

					res, err := r.celParser.ParseAndEvalStepRun(expr.Expression, cel.NewInput(
						cel.WithInput(task.Input.Input),
						cel.WithAdditionalMetadata(additionalMeta),
						cel.WithWorkflowRunID(task.ExternalId),
						cel.WithParents(task.Input.TriggerData),
					))

					if err != nil {
						failTaskError = fmt.Errorf("failed to parse step expression (%s): %w", expr.Expression, err)
						break
					}

					if err := r.celParser.CheckStepRunOutAgainstKnownV1(res, expr.Kind); err != nil {
						failTaskError = fmt.Errorf("failed to parse step expression (%s): %w", expr.Expression, err)
						break
					}

					opts = append(opts, createTaskExpressionEvalOpt{
						Key:      expr.Key,
						Kind:     expr.Kind,
						ValueStr: res.String,
						ValueInt: res.Int,
					})
				}

				if failTaskError != nil {
					// place the task into a failed state
					initialStates[i] = string(sqlcv1.V1TaskInitialStateFAILED)

					initialStateReasons[i] = pgtype.Text{
						String: failTaskError.Error(),
						Valid:  true,
					}
				} else {
					createExpressionOpts[task.ExternalId] = opts
				}
			} else {
				r.l.Warn().Msgf("no expressions found for step %s", task.StepId)
			}
		}
	}

	saveQueueCache, err := r.upsertQueues(ctx, tx, tenantId, queues)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert queues: %w", err)
	}

	// group by step_id
	stepIdsToParams := make(map[uuid.UUID]sqlcv1.CreateTasksParams, 0)

	for i, task := range tasks {
		params, ok := stepIdsToParams[task.StepId]

		if !ok {
			params = sqlcv1.CreateTasksParams{
				Tenantids:                    make([]uuid.UUID, 0),
				Queues:                       make([]string, 0),
				Actionids:                    make([]string, 0),
				Stepids:                      make([]uuid.UUID, 0),
				Stepreadableids:              make([]string, 0),
				Workflowids:                  make([]uuid.UUID, 0),
				Scheduletimeouts:             make([]string, 0),
				Steptimeouts:                 make([]string, 0),
				Priorities:                   make([]int32, 0),
				Stickies:                     make([]string, 0),
				Desiredworkerids:             make([]*uuid.UUID, 0),
				Externalids:                  make([]uuid.UUID, 0),
				Displaynames:                 make([]string, 0),
				Retrycounts:                  make([]int32, 0),
				Additionalmetadatas:          make([][]byte, 0),
				InitialStates:                make([]string, 0),
				InitialStateReasons:          make([]pgtype.Text, 0),
				Dagids:                       make([]pgtype.Int8, 0),
				Daginsertedats:               make([]pgtype.Timestamptz, 0),
				Concurrencyparentstrategyids: make([][]pgtype.Int8, 0),
				ConcurrencyStrategyIds:       make([][]int64, 0),
				ConcurrencyKeys:              make([][]string, 0),
				ParentTaskExternalIds:        make([]*uuid.UUID, 0),
				ParentTaskIds:                make([]pgtype.Int8, 0),
				ParentTaskInsertedAts:        make([]pgtype.Timestamptz, 0),
				ChildIndex:                   make([]pgtype.Int8, 0),
				ChildKey:                     make([]pgtype.Text, 0),
				StepIndex:                    make([]int64, 0),
				RetryBackoffFactor:           make([]pgtype.Float8, 0),
				RetryMaxBackoff:              make([]pgtype.Int4, 0),
				WorkflowVersionIds:           make([]uuid.UUID, 0),
				WorkflowRunIds:               make([]uuid.UUID, 0),
				Inputs:                       make([][]byte, 0),
			}
		}

		params.Tenantids = append(params.Tenantids, tenantIds[i])
		params.Queues = append(params.Queues, queues[i])
		params.Actionids = append(params.Actionids, actionIds[i])
		params.Stepids = append(params.Stepids, stepIds[i])
		params.Stepreadableids = append(params.Stepreadableids, stepReadableIds[i])
		params.Workflowids = append(params.Workflowids, workflowIds[i])
		params.Scheduletimeouts = append(params.Scheduletimeouts, scheduleTimeouts[i])
		params.Steptimeouts = append(params.Steptimeouts, stepTimeouts[i])
		params.Priorities = append(params.Priorities, priorities[i])
		params.Stickies = append(params.Stickies, stickies[i])
		params.Desiredworkerids = append(params.Desiredworkerids, desiredWorkerIds[i])
		params.Externalids = append(params.Externalids, externalIds[i])
		params.Displaynames = append(params.Displaynames, displayNames[i])
		params.Retrycounts = append(params.Retrycounts, retryCounts[i])
		params.Additionalmetadatas = append(params.Additionalmetadatas, additionalMetadatas[i])
		params.InitialStates = append(params.InitialStates, initialStates[i])
		params.InitialStateReasons = append(params.InitialStateReasons, initialStateReasons[i])
		params.Dagids = append(params.Dagids, dagIds[i])
		params.Daginsertedats = append(params.Daginsertedats, dagInsertedAts[i])
		params.Concurrencyparentstrategyids = append(params.Concurrencyparentstrategyids, parentStrategyIds[i])
		params.ConcurrencyStrategyIds = append(params.ConcurrencyStrategyIds, strategyIds[i])
		params.ConcurrencyKeys = append(params.ConcurrencyKeys, concurrencyKeys[i])
		params.ParentTaskExternalIds = append(params.ParentTaskExternalIds, parentTaskExternalIds[i])
		params.ParentTaskIds = append(params.ParentTaskIds, parentTaskIds[i])
		params.ParentTaskInsertedAts = append(params.ParentTaskInsertedAts, parentTaskInsertedAts[i])
		params.ChildIndex = append(params.ChildIndex, childIndices[i])
		params.ChildKey = append(params.ChildKey, childKeys[i])
		params.StepIndex = append(params.StepIndex, stepIndices[i])
		params.RetryBackoffFactor = append(params.RetryBackoffFactor, retryBackoffFactors[i])
		params.RetryMaxBackoff = append(params.RetryMaxBackoff, retryMaxBackoffs[i])
		params.WorkflowVersionIds = append(params.WorkflowVersionIds, workflowVersionIds[i])
		params.WorkflowRunIds = append(params.WorkflowRunIds, workflowRunIds[i])

		if r.payloadStore.DualWritesEnabled() {
			// if dual writes are enabled, write the inputs to the tasks table
			params.Inputs = append(params.Inputs, externalIdToInput[task.ExternalId])
		} else {
			// otherwise, write an empty json object to the inputs column
			params.Inputs = append(params.Inputs, []byte("{}"))
		}

		stepIdsToParams[task.StepId] = params
	}

	res := make([]*V1TaskWithPayload, 0)

	// for any initial states which are not queued, create a finalizing task event
	eventTaskIdRetryCounts := make([]TaskIdInsertedAtRetryCount, 0)
	eventTaskExternalIds := make([]uuid.UUID, 0)
	eventDatas := make([][]byte, 0)
	eventTypes := make([]sqlcv1.V1TaskEventType, 0)

	for stepId, params := range stepIdsToParams {
		createdTasks, err := r.queries.CreateTasks(ctx, tx, params)

		if err != nil {
			return nil, fmt.Errorf("failed to create tasks for step id %s: %w", stepId, err)
		}

		createdTasksWithPayloads := make([]*V1TaskWithPayload, len(createdTasks))

		for i, task := range createdTasks {
			input := externalIdToInput[task.ExternalID]
			withPayload := V1TaskWithPayload{
				V1Task:  task,
				Payload: input,
			}

			res = append(res, &withPayload)
			createdTasksWithPayloads[i] = &withPayload
		}

		for _, createdTask := range createdTasksWithPayloads {
			idRetryCount := TaskIdInsertedAtRetryCount{
				Id:         createdTask.ID,
				InsertedAt: createdTask.InsertedAt,
				RetryCount: createdTask.RetryCount,
			}

			switch createdTask.InitialState {
			case sqlcv1.V1TaskInitialStateFAILED:
				eventTaskIdRetryCounts = append(eventTaskIdRetryCounts, idRetryCount)
				eventTaskExternalIds = append(eventTaskExternalIds, createdTask.ExternalID)
				eventDatas = append(eventDatas, NewFailedTaskOutputEventFromTask(createdTask).Bytes())
				eventTypes = append(eventTypes, sqlcv1.V1TaskEventTypeFAILED)
			case sqlcv1.V1TaskInitialStateCANCELLED:
				eventTaskIdRetryCounts = append(eventTaskIdRetryCounts, idRetryCount)
				eventTaskExternalIds = append(eventTaskExternalIds, createdTask.ExternalID)
				eventDatas = append(eventDatas, NewCancelledTaskOutputEventFromTask(createdTask).Bytes())
				eventTypes = append(eventTypes, sqlcv1.V1TaskEventTypeCANCELLED)
			case sqlcv1.V1TaskInitialStateSKIPPED:
				eventTaskIdRetryCounts = append(eventTaskIdRetryCounts, idRetryCount)
				eventTaskExternalIds = append(eventTaskExternalIds, createdTask.ExternalID)
				eventDatas = append(eventDatas, NewSkippedTaskOutputEventFromTask(createdTask).Bytes())
				eventTypes = append(eventTypes, sqlcv1.V1TaskEventTypeCOMPLETED)
			}
		}
	}

	_, err = r.createTaskEvents(
		ctx,
		tx,
		tenantId,
		eventTaskIdRetryCounts,
		eventTaskExternalIds,
		eventDatas,
		eventTypes,
		make([]string, len(eventTaskIdRetryCounts)),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create task events: %w", err)
	}

	if len(createExpressionOpts) > 0 {
		err = r.createExpressionEvals(ctx, tx, res, createExpressionOpts)

		if err != nil {
			return nil, fmt.Errorf("failed to create expression evals: %w", err)
		}
	}

	if len(cleanupParentStrategyIds) > 0 {
		err = r.queries.CleanupWorkflowConcurrencySlotsAfterInsert(
			ctx,
			tx,
			sqlcv1.CleanupWorkflowConcurrencySlotsAfterInsertParams{
				Concurrencyparentstrategyids: cleanupParentStrategyIds,
				Workflowrunids:               cleanupWorkflowRunIds,
				Workflowversionids:           cleanupWorkflowVersionIds,
			},
		)

		if err != nil {
			return nil, fmt.Errorf("failed to cleanup workflow concurrency slots after insert: %w", err)
		}
	}

	// TODO: this should be moved to after the transaction commits
	saveQueueCache()

	return res, nil
}

// replayTasks updates tasks into the database. note that we're using Postgres rules to automatically insert the created
// tasks into the queue_items table.
func (r *sharedRepository) replayTasks(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId uuid.UUID,
	tasks []ReplayTaskOpts,
) ([]*V1TaskWithPayload, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

	uniqueStepIds := make(map[uuid.UUID]struct{})
	stepIds := make([]uuid.UUID, 0)

	for _, task := range tasks {
		if _, ok := uniqueStepIds[task.StepId]; !ok {
			uniqueStepIds[task.StepId] = struct{}{}
			stepIds = append(stepIds, task.StepId)
		}
	}

	steps, err := r.queries.ListStepsByIds(ctx, tx, sqlcv1.ListStepsByIdsParams{
		Ids:      stepIds,
		Tenantid: tenantId,
	})

	if err != nil {
		return nil, err
	}

	stepIdsToConfig := make(map[uuid.UUID]*sqlcv1.ListStepsByIdsRow)

	for _, step := range steps {
		stepIdsToConfig[step.ID] = step
	}

	concurrencyStrats, err := r.getConcurrencyExpressions(ctx, tx, tenantId, stepIdsToConfig)

	if err != nil {
		return nil, fmt.Errorf("failed to get step expressions: %w", err)
	}

	taskIds := make([]int64, len(tasks))
	taskInsertedAts := make([]pgtype.Timestamptz, len(tasks))
	initialStates := make([]string, len(tasks))
	initialStateReasons := make([]pgtype.Text, len(tasks))
	concurrencyKeys := make([][]string, len(tasks))
	additionalMetadatas := make([][]byte, len(tasks))
	queues := make([]string, len(tasks))

	externalIdToInput := make(map[uuid.UUID][]byte, len(tasks))

	for i, task := range tasks {
		stepConfig := stepIdsToConfig[task.StepId]
		queues[i] = stepConfig.ActionId // FIXME: make the queue name dynamic

		taskIds[i] = task.TaskId
		taskInsertedAts[i] = task.InsertedAt

		// TODO: case on whether this is a v1 or v2 task by looking at the step data. for now,
		// we're assuming a v1 task.
		if task.Input != nil {
			externalIdToInput[task.ExternalId] = r.ToV1StepRunData(task.Input).Bytes()
		}

		initialStates[i] = string(task.InitialState)
		if initialStates[i] == "" {
			initialStates[i] = string(sqlcv1.V1TaskInitialStateQUEUED)
		}

		if len(task.AdditionalMetadata) > 0 {
			additionalMetadatas[i] = task.AdditionalMetadata
		}

		if strats, ok := concurrencyStrats[task.StepId]; ok {
			emptyConcurrencyKeys := make([]string, 0)

			for range strats {
				emptyConcurrencyKeys = append(emptyConcurrencyKeys, "")
			}

			concurrencyKeys[i] = emptyConcurrencyKeys
		} else {
			concurrencyKeys[i] = make([]string, 0)
		}

		// only check for concurrency if the task is in a queued state, otherwise we don't need to
		// evaluate the expression (and it will likely fail if we do)
		if task.InitialState == sqlcv1.V1TaskInitialStateQUEUED {
			// if we have a step expression, evaluate the expression
			if strats, ok := concurrencyStrats[task.StepId]; ok {
				taskConcurrencyKeys := make([]string, 0)

				var failTaskError error

				for _, strat := range strats {
					var additionalMeta map[string]interface{}

					if len(additionalMetadatas[i]) > 0 {
						if err := json.Unmarshal(additionalMetadatas[i], &additionalMeta); err != nil {
							failTaskError = fmt.Errorf("failed to process additional metadata: not a json object")
							break
						}
					}

					if task.Input == nil {
						failTaskError = fmt.Errorf("failed to parse step expression (%s): input is nil", strat.Expression)
						break
					}

					res, err := r.celParser.ParseAndEvalStepRun(strat.Expression, cel.NewInput(
						cel.WithInput(task.Input.Input),
						cel.WithAdditionalMetadata(additionalMeta),
						cel.WithWorkflowRunID(task.ExternalId),
						cel.WithParents(task.Input.TriggerData),
					))

					if err != nil {
						failTaskError = fmt.Errorf("failed to parse step expression (%s): %w", strat.Expression, err)
						break
					}

					if res.String == nil {
						prefix := "expected string output for concurrency key"

						if res.Int != nil {
							failTaskError = fmt.Errorf("failed to parse step expression (%s): %s, got int", strat.Expression, prefix)
							break
						}

						failTaskError = fmt.Errorf("failed to parse step expression (%s): %s, got unknown type", strat.Expression, prefix)
						break
					}

					taskConcurrencyKeys = append(taskConcurrencyKeys, *res.String)

				}

				if failTaskError != nil {
					// place the task into a failed state
					initialStates[i] = string(sqlcv1.V1TaskInitialStateFAILED)

					initialStateReasons[i] = pgtype.Text{
						String: failTaskError.Error(),
						Valid:  true,
					}

					// set to "FAILED" for each strategy to maintain cardinality in multi-dimensional array
					failedKeys := make([]string, len(strats))
					for j := range failedKeys {
						failedKeys[j] = "FAILED"
					}
					concurrencyKeys[i] = failedKeys
				} else {
					concurrencyKeys[i] = taskConcurrencyKeys
				}
			}
		}
	}

	saveQueueCache, err := r.upsertQueues(ctx, tx, tenantId, queues)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert queues: %w", err)
	}

	stepIdsToParams := make(map[uuid.UUID]sqlcv1.ReplayTasksParams, 0)
	stepIdsToStorePayloadOpts := make(map[uuid.UUID][]StorePayloadOpts, 0)

	for i, task := range tasks {
		params, ok := stepIdsToParams[task.StepId]

		if !ok {
			params = sqlcv1.ReplayTasksParams{
				Taskids:             make([]int64, 0),
				Taskinsertedats:     make([]pgtype.Timestamptz, 0),
				Inputs:              make([][]byte, 0),
				InitialStates:       make([]string, 0),
				InitialStateReasons: make([]pgtype.Text, 0),
				Concurrencykeys:     make([][]string, 0),
			}
		}

		input := externalIdToInput[task.ExternalId]

		params.Taskids = append(params.Taskids, taskIds[i])
		params.Taskinsertedats = append(params.Taskinsertedats, taskInsertedAts[i])
		params.Inputs = append(params.Inputs, input)
		params.InitialStates = append(params.InitialStates, initialStates[i])
		params.InitialStateReasons = append(params.InitialStateReasons, initialStateReasons[i])
		params.Concurrencykeys = append(params.Concurrencykeys, concurrencyKeys[i])

		stepIdsToParams[task.StepId] = params

		storePayloadOpts := StorePayloadOpts{
			Id:         taskIds[i],
			InsertedAt: taskInsertedAts[i],
			Type:       sqlcv1.V1PayloadTypeTASKINPUT,
			ExternalId: task.ExternalId,
			Payload:    input,
			TenantId:   tenantId,
		}

		stepIdsToStorePayloadOpts[task.StepId] = append(stepIdsToStorePayloadOpts[task.StepId], storePayloadOpts)
	}

	res := make([]*V1TaskWithPayload, 0)

	// for any initial states which are not queued, create a finalizing task event
	eventTaskIdRetryCounts := make([]TaskIdInsertedAtRetryCount, 0)
	eventTaskExternalIds := make([]uuid.UUID, 0)
	eventDatas := make([][]byte, 0)
	eventTypes := make([]sqlcv1.V1TaskEventType, 0)

	for stepId, params := range stepIdsToParams {
		replayRes, err := r.queries.ReplayTasks(ctx, tx, params)

		if err != nil {
			return nil, fmt.Errorf("failed to replay tasks for step id %s: %w", stepId, err)
		}

		storePayloadOpts, ok := stepIdsToStorePayloadOpts[stepId]

		if !ok {
			return nil, fmt.Errorf("missing payload store opts for step id %s", stepId)
		}

		err = r.payloadStore.Store(ctx, tx, storePayloadOpts...)

		if err != nil {
			return nil, fmt.Errorf("failed to store payloads for step id %s: %w", stepId, err)
		}

		replayResWithPayloads := make([]*V1TaskWithPayload, len(replayRes))
		for i, task := range replayRes {
			input := externalIdToInput[task.ExternalID]
			withPayload := V1TaskWithPayload{
				V1Task:  task,
				Payload: input,
			}
			replayResWithPayloads[i] = &withPayload
			res = append(res, &withPayload)
		}

		for _, replayedTask := range replayResWithPayloads {
			idRetryCount := TaskIdInsertedAtRetryCount{
				Id:         replayedTask.ID,
				InsertedAt: replayedTask.InsertedAt,
				RetryCount: replayedTask.RetryCount,
			}

			switch replayedTask.InitialState {
			case sqlcv1.V1TaskInitialStateFAILED:
				eventTaskIdRetryCounts = append(eventTaskIdRetryCounts, idRetryCount)
				eventTaskExternalIds = append(eventTaskExternalIds, replayedTask.ExternalID)
				eventDatas = append(eventDatas, NewFailedTaskOutputEventFromTask(replayedTask).Bytes())
				eventTypes = append(eventTypes, sqlcv1.V1TaskEventTypeFAILED)
			case sqlcv1.V1TaskInitialStateCANCELLED:
				eventTaskIdRetryCounts = append(eventTaskIdRetryCounts, idRetryCount)
				eventTaskExternalIds = append(eventTaskExternalIds, replayedTask.ExternalID)
				eventDatas = append(eventDatas, NewCancelledTaskOutputEventFromTask(replayedTask).Bytes())
				eventTypes = append(eventTypes, sqlcv1.V1TaskEventTypeCANCELLED)
			case sqlcv1.V1TaskInitialStateSKIPPED:
				eventTaskIdRetryCounts = append(eventTaskIdRetryCounts, idRetryCount)
				eventTaskExternalIds = append(eventTaskExternalIds, replayedTask.ExternalID)
				eventDatas = append(eventDatas, NewSkippedTaskOutputEventFromTask(replayedTask).Bytes())
				eventTypes = append(eventTypes, sqlcv1.V1TaskEventTypeCOMPLETED)
			}
		}
	}

	_, err = r.createTaskEvents(
		ctx,
		tx,
		tenantId,
		eventTaskIdRetryCounts,
		eventTaskExternalIds,
		eventDatas,
		eventTypes,
		make([]string, len(eventTaskIdRetryCounts)),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create task events: %w", err)
	}

	// TODO: this should be moved to after the transaction commits
	saveQueueCache()

	return res, nil
}

func (r *sharedRepository) getConcurrencyExpressions(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId uuid.UUID,
	stepIdsToConfig map[uuid.UUID]*sqlcv1.ListStepsByIdsRow,
) (map[uuid.UUID][]*sqlcv1.V1StepConcurrency, error) {
	stepIdsWithExpressions := make(map[uuid.UUID]struct{})

	for _, step := range stepIdsToConfig {
		if step.ConcurrencyCount > 0 {
			stepIdsWithExpressions[step.ID] = struct{}{}
		}
	}

	if len(stepIdsWithExpressions) == 0 {
		return nil, nil
	}

	cacheKey := func(stepId uuid.UUID) string {
		return fmt.Sprintf("concurrency-strategies:%s:%s", tenantId, stepId)
	}

	sortStrategies := func(strats []*sqlcv1.V1StepConcurrency) {
		sort.SliceStable(strats, func(i, j int) bool {
			iStrat := strats[i]
			jStrat := strats[j]

			if iStrat.ParentStrategyID.Valid && jStrat.ParentStrategyID.Valid && iStrat.ParentStrategyID.Int64 != jStrat.ParentStrategyID.Int64 {
				return iStrat.ParentStrategyID.Int64 < jStrat.ParentStrategyID.Int64
			}

			if iStrat.ParentStrategyID.Valid && !jStrat.ParentStrategyID.Valid {
				return true
			}

			if !iStrat.ParentStrategyID.Valid && jStrat.ParentStrategyID.Valid {
				return false
			}

			return iStrat.ID < jStrat.ID
		})
	}

	stepIdToStrats := make(map[uuid.UUID][]*sqlcv1.V1StepConcurrency, len(stepIdsWithExpressions))

	// Only hit the DB for step IDs that aren't cached.
	missingStepIds := make([]uuid.UUID, 0, len(stepIdsWithExpressions))
	missingStepIdStrs := make([]uuid.UUID, 0, len(stepIdsWithExpressions))

	for stepId := range stepIdsWithExpressions {
		if cached, ok := r.concurrencyStrategyCache.Get(cacheKey(stepId)); ok {
			stepIdToStrats[stepId] = cached.([]*sqlcv1.V1StepConcurrency)
			continue
		}

		missingStepIds = append(missingStepIds, stepId)
		missingStepIdStrs = append(missingStepIdStrs, stepId)
	}

	if len(missingStepIds) == 0 {
		return stepIdToStrats, nil
	}

	strats, err := r.queries.ListConcurrencyStrategiesByStepId(ctx, tx, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
		Tenantid: tenantId,
		Stepids:  missingStepIds,
	})
	if err != nil {
		return nil, err
	}

	fetchedByStepId := make(map[uuid.UUID][]*sqlcv1.V1StepConcurrency)

	for _, strat := range strats {
		stepId := strat.StepID
		fetchedByStepId[stepId] = append(fetchedByStepId[stepId], strat)
	}

	// Populate cache for all missing step IDs, including negative-caching empty results.
	for _, stepId := range missingStepIdStrs {
		stepStrats := fetchedByStepId[stepId]
		if stepStrats == nil {
			stepStrats = []*sqlcv1.V1StepConcurrency{}
		} else {
			sortStrategies(stepStrats)
		}

		r.concurrencyStrategyCache.Set(cacheKey(stepId), stepStrats)
		stepIdToStrats[stepId] = stepStrats
	}

	return stepIdToStrats, nil
}

func (r *sharedRepository) getStepExpressions(
	ctx context.Context,
	tx sqlcv1.DBTX,
	stepIdsToConfig map[uuid.UUID]*sqlcv1.ListStepsByIdsRow,
) (map[uuid.UUID][]*sqlcv1.StepExpression, error) {
	stepIdsWithExpressions := make(map[uuid.UUID]struct{})

	for _, step := range stepIdsToConfig {
		if step.ExprCount > 0 {
			stepIdsWithExpressions[step.ID] = struct{}{}
		}
	}

	if len(stepIdsWithExpressions) == 0 {
		return map[uuid.UUID][]*sqlcv1.StepExpression{}, nil
	}

	stepIds := make([]uuid.UUID, 0, len(stepIdsWithExpressions))

	for stepId := range stepIdsWithExpressions {
		stepIds = append(stepIds, stepId)
	}

	expressions, err := r.queries.ListStepExpressions(ctx, tx, stepIds)

	if err != nil {
		return nil, err
	}

	stepIdToExpressions := make(map[uuid.UUID][]*sqlcv1.StepExpression)

	for _, expression := range expressions {
		stepId := expression.StepId

		if _, ok := stepIdToExpressions[stepId]; !ok {
			stepIdToExpressions[stepId] = make([]*sqlcv1.StepExpression, 0)
		}

		stepIdToExpressions[stepId] = append(stepIdToExpressions[stepId], expression)
	}

	return stepIdToExpressions, nil
}

func (r *sharedRepository) createTaskEventsAfterRelease(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId uuid.UUID,
	taskIdRetryCounts []TaskIdInsertedAtRetryCount,
	outputs [][]byte,
	releasedTasks []*sqlcv1.ReleaseTasksRow,
	eventType sqlcv1.V1TaskEventType,
) ([]InternalTaskEvent, error) {
	if len(taskIdRetryCounts) != len(releasedTasks) || len(taskIdRetryCounts) != len(outputs) {
		return nil, fmt.Errorf("failed to release all tasks")
	}

	datas := make([][]byte, len(releasedTasks))
	externalIds := make([]uuid.UUID, len(releasedTasks))
	isCurrentRetry := make([]bool, len(releasedTasks))

	for i, releasedTask := range releasedTasks {
		datas[i] = outputs[i]
		externalIds[i] = releasedTask.ExternalID
		isCurrentRetry[i] = releasedTask.IsCurrentRetry
	}

	// filter out any rows which are not the current retry
	filteredTaskIdRetryCounts := make([]TaskIdInsertedAtRetryCount, 0)
	filteredDatas := make([][]byte, 0)
	filteredExternalIds := make([]uuid.UUID, 0)

	for i := range len(datas) {
		if !isCurrentRetry[i] {
			continue
		}

		filteredTaskIdRetryCounts = append(filteredTaskIdRetryCounts, taskIdRetryCounts[i])
		filteredDatas = append(filteredDatas, datas[i])
		filteredExternalIds = append(filteredExternalIds, externalIds[i])
	}

	return r.createTaskEvents(
		ctx,
		tx,
		tenantId,
		filteredTaskIdRetryCounts,
		filteredExternalIds,
		filteredDatas,
		makeEventTypeArr(eventType, len(filteredExternalIds)),
		make([]string, len(filteredExternalIds)),
	)
}

func (r *sharedRepository) createTaskEvents(
	ctx context.Context,
	dbtx sqlcv1.DBTX,
	tenantId uuid.UUID,
	tasks []TaskIdInsertedAtRetryCount,
	taskExternalIds []uuid.UUID,
	eventDatas [][]byte,
	eventTypes []sqlcv1.V1TaskEventType,
	eventKeys []string,
) ([]InternalTaskEvent, error) {
	if len(tasks) != len(eventDatas) {
		return nil, fmt.Errorf("mismatched task and event data lengths")
	}

	taskIds := make([]int64, len(tasks))
	taskInsertedAts := make([]pgtype.Timestamptz, len(tasks))
	retryCounts := make([]int32, len(tasks))
	eventTypesStrs := make([]string, len(tasks))
	paramDatas := make([][]byte, len(tasks))
	paramKeys := make([]pgtype.Text, len(tasks))
	externalIds := make([]uuid.UUID, len(tasks))

	internalTaskEvents := make([]InternalTaskEvent, len(tasks))

	externalIdToData := make(map[uuid.UUID][]byte, len(tasks))

	for i, task := range tasks {
		taskIds[i] = task.Id
		taskInsertedAts[i] = task.InsertedAt
		retryCounts[i] = task.RetryCount
		eventTypesStrs[i] = string(eventTypes[i])

		externalId := uuid.New()
		externalIds[i] = externalId

		// important: if we don't set this to `eventDatas[i]` and instead allow it to be nil optionally
		// we'll get errors downstream when we try to read the payload back and parse it in `registerChildWorkflows`
		// because it'll try to unmarshal the `nil` value.
		externalIdToData[externalId] = eventDatas[i]

		if len(eventDatas[i]) == 0 || !r.payloadStore.TaskEventDualWritesEnabled() {
			paramDatas[i] = nil
		} else {
			paramDatas[i] = eventDatas[i]
		}

		if eventKeys[i] != "" {
			paramKeys[i] = pgtype.Text{
				String: eventKeys[i],
				Valid:  true,
			}
		}

		internalTaskEvents[i] = InternalTaskEvent{
			TaskID:         task.Id,
			TaskExternalID: taskExternalIds[i],
			TenantID:       tenantId,
			RetryCount:     task.RetryCount,
			EventType:      eventTypes[i],
			EventKey:       eventKeys[i],
			Data:           eventDatas[i],
		}
	}

	taskEvents, err := r.queries.CreateTaskEvents(ctx, dbtx, sqlcv1.CreateTaskEventsParams{
		Tenantid:        tenantId,
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Retrycounts:     retryCounts,
		Eventtypes:      eventTypesStrs,
		Datas:           paramDatas,
		Eventkeys:       paramKeys,
		Externalids:     externalIds,
	})

	if err != nil {
		return nil, err
	}

	storePayloadOpts := make([]StorePayloadOpts, len(taskEvents))

	for i, taskEvent := range taskEvents {
		taskEventExternalId := uuid.Nil
		if taskEvent.ExternalID != nil {
			taskEventExternalId = *taskEvent.ExternalID
		}

		data := externalIdToData[taskEventExternalId]

		storePayloadOpts[i] = StorePayloadOpts{
			Id:         taskEvent.ID,
			InsertedAt: taskEvent.InsertedAt,
			ExternalId: taskEventExternalId,
			Type:       sqlcv1.V1PayloadTypeTASKEVENTDATA,
			Payload:    data,
			TenantId:   tenantId,
		}
	}

	err = r.payloadStore.Store(ctx, dbtx, storePayloadOpts...)

	if err != nil {
		return nil, fmt.Errorf("failed to store task event payloads: %w", err)
	}

	return internalTaskEvents, nil
}

func makeEventTypeArr(status sqlcv1.V1TaskEventType, n int) []sqlcv1.V1TaskEventType {
	a := make([]sqlcv1.V1TaskEventType, n)

	for i := range a {
		a[i] = status
	}

	return a
}

func hash(s string) int64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return int64(h.Sum64())
}

func (r *TaskRepositoryImpl) ReplayTasks(ctx context.Context, tenantId uuid.UUID, tasks []TaskIdInsertedAtRetryCount) (*ReplayTasksResult, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	acquired, err := r.queries.TryAdvisoryLock(ctx, tx, hash("replay_"+tenantId.String()))

	if err != nil {
		return nil, fmt.Errorf("failed to try advisory lock for replaying tasks: %w", err)
	}

	if !acquired {
		return nil, fmt.Errorf("could not acquire advisory lock for replaying tasks")
	}

	taskIds := make([]int64, len(tasks))
	taskInsertedAts := make([]pgtype.Timestamptz, len(tasks))

	for i, task := range tasks {
		taskIds[i] = task.Id
		taskInsertedAts[i] = task.InsertedAt
	}

	// list tasks (and augment with task descendants) and locks them for update
	lockedTasks, err := r.queries.ListTasksForReplay(ctx, tx, sqlcv1.ListTasksForReplayParams{
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Tenantid:        tenantId,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list tasks for replay: %w", err)
	}

	lockedTaskIds := make([]int64, len(lockedTasks))
	lockedTaskInsertedAts := make([]pgtype.Timestamptz, len(lockedTasks))
	subtreeStepIds := make(map[int64]map[uuid.UUID]bool) // dag id -> step id -> true
	subtreeExternalIds := make(map[uuid.UUID]struct{})
	dagIdsToLockMap := make(map[int64]struct{})
	minInsertedAt := sqlchelpers.TimestamptzFromTime(time.Now()) // current time as a placeholder - will be overwritten

	for i, task := range lockedTasks {
		lockedTaskIds[i] = task.ID
		lockedTaskInsertedAts[i] = task.InsertedAt

		if task.DagID.Valid {
			if _, ok := subtreeStepIds[task.DagID.Int64]; !ok {
				subtreeStepIds[task.DagID.Int64] = make(map[uuid.UUID]bool)
			}

			dagIdsToLockMap[task.DagID.Int64] = struct{}{}
			subtreeStepIds[task.DagID.Int64][task.StepID] = true
			subtreeExternalIds[task.ExternalID] = struct{}{}
		}

		if task.InsertedAt.Time.Before(minInsertedAt.Time) {
			minInsertedAt = task.InsertedAt
		}
	}

	// lock all tasks in the DAGs
	dagIdsToLock := make([]int64, 0, len(dagIdsToLockMap))

	for dagId := range dagIdsToLockMap {
		dagIdsToLock = append(dagIdsToLock, dagId)
	}

	successfullyLockedDAGIds, err := r.queries.LockDAGsForReplay(ctx, tx, sqlcv1.LockDAGsForReplayParams{
		Dagids:   dagIdsToLock,
		Tenantid: tenantId,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to lock DAGs for replay: %w", err)
	}

	successfullyLockedDAGsMap := make(map[int64]bool)

	for _, dagId := range successfullyLockedDAGIds {
		successfullyLockedDAGsMap[dagId] = true
	}

	// Discard tasks which can't be replayed. Discard rules are as follows:
	// 1. If a task is in a running state, discard it.
	// 2. If a task is in a running state and has a DAG id, discard all tasks in the DAG.
	// 3. If a task has a DAG id but it is not present in the successfully locked DAGs, discard it.
	dagIdsFailedPreflight := make(map[int64]bool)

	preflightDAGs, err := r.queries.PreflightCheckDAGsForReplay(ctx, tx, sqlcv1.PreflightCheckDAGsForReplayParams{
		Dagids:   successfullyLockedDAGIds,
		Tenantid: tenantId,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to preflight check DAGs for replay: %w", err)
	}

	for _, dag := range preflightDAGs {
		if dag.StepCount != dag.TaskCount {
			dagIdsFailedPreflight[dag.ID] = true
		}
	}

	tasksFailedPreflight := make(map[int64]bool)

	failedPreflightChecks, err := r.queries.PreflightCheckTasksForReplay(ctx, tx, sqlcv1.PreflightCheckTasksForReplayParams{
		Taskids:         lockedTaskIds,
		Taskinsertedats: lockedTaskInsertedAts,
		Tenantid:        tenantId,
		Mininsertedat:   minInsertedAt,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to preflight check tasks for replay: %w", err)
	}

	for _, task := range failedPreflightChecks {
		tasksFailedPreflight[task.ID] = true
	}

	// group tasks by their dag_id, if it exists
	dagIdsToChildTasks := make(map[int64][]*sqlcv1.ListTasksForReplayRow)
	dagIds := make(map[int64]struct{}, 0)

	// figure out which tasks to replay immediately
	replayOpts := make([]ReplayTaskOpts, 0)
	replayedTasks := make([]TaskIdInsertedAtRetryCount, 0)

	retrieveOpts := make([]RetrievePayloadOpts, len(lockedTasks))

	for i, task := range lockedTasks {
		retrieveOpts[i] = RetrievePayloadOpts{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeTASKINPUT,
			TenantId:   tenantId,
		}
	}

	payloads, err := r.payloadStore.Retrieve(ctx, tx, retrieveOpts...)

	if err != nil {
		return nil, fmt.Errorf("failed to bulk retrieve task inputs: %w", err)
	}

	for _, task := range lockedTasks {
		// check whether to discard the task
		if task.DagID.Valid && !successfullyLockedDAGsMap[task.DagID.Int64] {
			r.l.Warn().Int64("task_id", task.ID).Msg("discarding task, could not lock DAG")
			continue
		}

		if task.DagID.Valid && dagIdsFailedPreflight[task.DagID.Int64] {
			r.l.Warn().Int64("task_id", task.ID).Msg("discarding task, failed preflight check for DAG")
			continue
		}

		if tasksFailedPreflight[task.ID] {
			r.l.Warn().Int64("task_id", task.ID).Msg("discarding task, failed preflight check")
			continue
		}

		replayedTasks = append(replayedTasks, TaskIdInsertedAtRetryCount{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			RetryCount: task.RetryCount,
		})

		if task.DagID.Valid {
			dagIds[task.DagID.Int64] = struct{}{}
		}

		if task.DagID.Valid && len(task.Parents) > 0 {
			isParentBeingReplayed := false
			for _, parent := range task.Parents {
				if subtreeStepIds[task.DagID.Int64][parent] {
					isParentBeingReplayed = true
					break
				}
			}

			if isParentBeingReplayed {
				if _, ok := dagIdsToChildTasks[task.DagID.Int64]; !ok {
					dagIdsToChildTasks[task.DagID.Int64] = make([]*sqlcv1.ListTasksForReplayRow, 0)
				}

				dagIdsToChildTasks[task.DagID.Int64] = append(dagIdsToChildTasks[task.DagID.Int64], task)

				continue
			}
		}

		if task.DagID.Valid && task.JobKind == sqlcv1.JobKindONFAILURE {
			// we need to check if there are other steps in the subtree
			doesOnFailureHaveOtherSteps := false

			for stepId := range subtreeStepIds[task.DagID.Int64] {
				if stepId == task.StepID {
					continue
				}

				doesOnFailureHaveOtherSteps = true
				break
			}

			if doesOnFailureHaveOtherSteps {
				if _, ok := dagIdsToChildTasks[task.DagID.Int64]; !ok {
					dagIdsToChildTasks[task.DagID.Int64] = make([]*sqlcv1.ListTasksForReplayRow, 0)
				}

				dagIdsToChildTasks[task.DagID.Int64] = append(dagIdsToChildTasks[task.DagID.Int64], task)

				continue
			}
		}

		retrieveOpt := RetrievePayloadOpts{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeTASKINPUT,
			TenantId:   tenantId,
		}

		input, ok := payloads[retrieveOpt]

		if !ok {
			// If the input wasn't found in the payload store,
			// fall back to the input stored on the task itself.

			input = task.Input
		}

		replayOpts = append(replayOpts, ReplayTaskOpts{
			TaskId:             task.ID,
			InsertedAt:         task.InsertedAt,
			StepId:             task.StepID,
			ExternalId:         task.ExternalID,
			InitialState:       sqlcv1.V1TaskInitialStateQUEUED,
			AdditionalMetadata: task.AdditionalMetadata,
			// NOTE: we require the input to be passed in to the replay method so we can re-evaluate the concurrency keys
			// Ideally we could preserve the same concurrency keys, but the replay tasks method is currently unaware of existing concurrency
			// keys because they may change between retries.
			Input: r.newTaskInputFromExistingBytes(input),
		})
	}

	dagIdsArr := make([]int64, 0, len(dagIds))

	for dagId := range dagIds {
		dagIdsArr = append(dagIdsArr, dagId)
	}

	allTasksInDAGs, err := r.queries.ListAllTasksInDags(ctx, tx, sqlcv1.ListAllTasksInDagsParams{
		Dagids:   dagIdsArr,
		Tenantid: tenantId,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list all tasks in DAGs: %w", err)
	}

	dagIdsToAllTasks := make(map[int64][]*sqlcv1.ListAllTasksInDagsRow)
	stepIdsInDAGs := make([]uuid.UUID, 0)

	for _, task := range allTasksInDAGs {
		if _, ok := dagIdsToAllTasks[task.DagID.Int64]; !ok {
			dagIdsToAllTasks[task.DagID.Int64] = make([]*sqlcv1.ListAllTasksInDagsRow, 0)
		}

		stepIdsInDAGs = append(stepIdsInDAGs, task.StepID)

		dagIdsToAllTasks[task.DagID.Int64] = append(dagIdsToAllTasks[task.DagID.Int64], task)
	}

	upsertedTasks := make([]*V1TaskWithPayload, 0)

	// NOTE: the tasks which are passed in represent a *subtree* of the DAG.
	if len(replayOpts) > 0 {
		upsertedTasks, err = r.replayTasks(ctx, tx, tenantId, replayOpts)

		if err != nil {
			return nil, fmt.Errorf("failed to replay existing tasks: %w", err)
		}
	}

	// for any tasks which are child tasks, we need to reset the match signals for the parent tasks
	spawnedChildTasks := make([]*sqlcv1.ListTasksForReplayRow, 0)

	for _, task := range lockedTasks {
		if task.ParentTaskID.Valid {
			spawnedChildTasks = append(spawnedChildTasks, task)
		}
	}

	eventMatches := make([]CreateMatchOpts, 0)

	if len(spawnedChildTasks) > 0 {
		// construct a list of signals to reset
		signalEventKeys := make([]string, 0)
		parentTaskIds := make([]int64, 0)
		parentTaskInsertedAts := make([]pgtype.Timestamptz, 0)

		for _, task := range spawnedChildTasks {
			if !task.ChildIndex.Valid {
				// TODO: handle error better/check with validation that this won't happen
				r.l.Error().Msg("could not find child key or index for child workflow")
				continue
			}

			var childKey *string

			if task.ChildKey.Valid {
				childKey = &task.ChildKey.String
			}

			var parentExternalId uuid.UUID

			if task.ParentTaskExternalID != nil {
				parentExternalId = *task.ParentTaskExternalID
			}
			k := getChildSignalEventKey(parentExternalId, task.StepIndex, task.ChildIndex.Int64, childKey)

			signalEventKeys = append(signalEventKeys, k)
			parentTaskIds = append(parentTaskIds, task.ParentTaskID.Int64)
			parentTaskInsertedAts = append(parentTaskInsertedAts, task.ParentTaskInsertedAt)

			eventMatches = append(eventMatches, CreateMatchOpts{
				Kind:                 sqlcv1.V1MatchKindSIGNAL,
				Conditions:           getChildWorkflowGroupMatches(task.ExternalID, task.StepReadableID),
				SignalExternalId:     task.ParentTaskExternalID,
				SignalTaskId:         &task.ParentTaskID.Int64,
				SignalTaskInsertedAt: task.ParentTaskInsertedAt,
				SignalKey:            &k,
			})
		}

		err = r.queries.DeleteMatchingSignalEvents(ctx, tx, sqlcv1.DeleteMatchingSignalEventsParams{
			Tenantid:        tenantId,
			Eventkeys:       signalEventKeys,
			Taskids:         parentTaskIds,
			Taskinsertedats: parentTaskInsertedAts,
			Eventtype:       sqlcv1.V1TaskEventTypeSIGNALCOMPLETED,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to delete matching signal events: %w", err)
		}
	}

	// For any DAGs, reset all match conditions which refer to internal events within the subtree of the DAG.
	// we do not reset other match conditions (for example, ones which refer to completed events for tasks
	// which are outside of this subtree). otherwise, we would end up in a state where these events would
	// never be matched.
	// if any steps have additional match conditions, query for the additional matches
	stepsToAdditionalMatches := make(map[uuid.UUID][]*sqlcv1.V1StepMatchCondition)

	if len(stepIdsInDAGs) > 0 {
		additionalMatches, err := r.queries.ListStepMatchConditions(ctx, r.pool, sqlcv1.ListStepMatchConditionsParams{
			Stepids:  sqlchelpers.UniqueSet(stepIdsInDAGs),
			Tenantid: tenantId,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to list step match conditions: %w", err)
		}

		for _, match := range additionalMatches {
			stepId := match.StepID

			stepsToAdditionalMatches[stepId] = append(stepsToAdditionalMatches[stepId], match)
		}
	}

	for dagId, tasks := range dagIdsToChildTasks {
		allTasks := dagIdsToAllTasks[dagId]

		for _, task := range tasks {
			taskExternalId := task.ExternalID
			stepId := task.StepID
			switch {
			case task.JobKind == sqlcv1.JobKindONFAILURE:
				conditions := make([]GroupMatchCondition, 0)
				groupId := uuid.New()

				for _, otherTask := range allTasks {
					if otherTask.StepID == stepId {
						continue
					}

					otherExternalId := otherTask.ExternalID
					readableId := otherTask.StepReadableID

					conditions = append(conditions, getParentOnFailureGroupMatches(groupId, otherExternalId, readableId)...)
				}

				eventMatches = append(eventMatches, CreateMatchOpts{
					Kind:              sqlcv1.V1MatchKindTRIGGER,
					Conditions:        conditions,
					TriggerExternalId: &taskExternalId,
					TriggerStepId:     &stepId,
					TriggerStepIndex: pgtype.Int8{
						Int64: task.StepIndex,
						Valid: true,
					},
					TriggerDAGId:         &task.DagID.Int64,
					TriggerDAGInsertedAt: task.DagInsertedAt,
					// NOTE: we don't need to set parent task id/child index/child key because
					// the task already exists
					TriggerExistingTaskId:         &task.ID,
					TriggerExistingTaskInsertedAt: task.InsertedAt,
				})
			default:
				conditions := make([]GroupMatchCondition, 0)

				cancelGroupId := uuid.New()

				additionalMatches, ok := stepsToAdditionalMatches[stepId]

				if !ok {
					additionalMatches = make([]*sqlcv1.V1StepMatchCondition, 0)
				}

				for _, parent := range task.Parents {
					// FIXME: n^2 complexity here, fix it.
					for _, otherTask := range allTasks {
						if otherTask.StepID == parent {
							parentExternalId := otherTask.ExternalID
							readableId := otherTask.StepReadableID

							hasUserEventOrSleepMatches := false
							hasAnySkippingParentOverrides := false

							parentOverrideMatches := make([]*sqlcv1.V1StepMatchCondition, 0)

							for _, match := range additionalMatches {
								if match.Kind == sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE {
									if match.ParentReadableID.String == readableId {
										parentOverrideMatches = append(parentOverrideMatches, match)
									}

									if match.Action == sqlcv1.V1MatchConditionActionSKIP {
										hasAnySkippingParentOverrides = true
									}
								} else {
									hasUserEventOrSleepMatches = true
								}
							}

							conditions = append(conditions, getParentInDAGGroupMatch(cancelGroupId, parentExternalId, readableId, parentOverrideMatches, hasUserEventOrSleepMatches, hasAnySkippingParentOverrides)...)
						}
					}
				}

				// create an event match
				eventMatches = append(eventMatches, CreateMatchOpts{
					Kind:              sqlcv1.V1MatchKindTRIGGER,
					Conditions:        conditions,
					TriggerExternalId: &taskExternalId,
					TriggerStepId:     &stepId,
					TriggerStepIndex: pgtype.Int8{
						Int64: task.StepIndex,
						Valid: true,
					},
					TriggerDAGId:         &task.DagID.Int64,
					TriggerDAGInsertedAt: task.DagInsertedAt,
					// NOTE: we don't need to set parent task id/child index/child key because
					// the task already exists
					TriggerExistingTaskId:         &task.ID,
					TriggerExistingTaskInsertedAt: task.InsertedAt,
				})
			}
		}
	}

	// reconstruct group conditions
	reconstructedMatches, candidateEvents, err := r.reconstructGroupConditions(ctx, tx, tenantId, subtreeExternalIds, eventMatches)

	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct group conditions: %w", err)
	}

	// create the event matches
	err = r.createEventMatches(ctx, tx, tenantId, reconstructedMatches)

	if err != nil {
		return nil, fmt.Errorf("failed to create event matches: %w", err)
	}

	// process event matches
	// TODO: signal the event matches to the caller
	internalMatchResults, err := r.processEventMatches(ctx, tx, tenantId, candidateEvents, sqlcv1.V1EventTypeINTERNAL)

	if err != nil {
		return nil, fmt.Errorf("failed to process internal event matches: %w", err)
	}

	if err := commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &ReplayTasksResult{
		ReplayedTasks:        replayedTasks,
		UpsertedTasks:        upsertedTasks,
		InternalEventResults: internalMatchResults,
	}, nil
}

func (r *TaskRepositoryImpl) reconstructGroupConditions(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId uuid.UUID,
	subtreeExternalIds map[uuid.UUID]struct{},
	eventMatches []CreateMatchOpts,
) ([]CreateMatchOpts, []CandidateEventMatch, error) {
	// track down completed tasks and failed tasks which represent parents that aren't in the subtree
	// of the DAG. for these tasks, we need to write the match conditions which refer to these tasks
	// as satisfied match conditions.
	// in other words, if the group match condition is an INTERNAL event and refers to a parentExternalId
	// which is NOT in the subtree of what we're replaying, it represent a group condition where we'd like
	// to query the task_events table to ensure the event has already occurred. if it has, we can mark the
	// group condition as satisfied.
	externalIds := make([]uuid.UUID, 0)
	eventTypes := make([][]string, 0)

	for _, match := range eventMatches {
		if match.TriggerDAGId == nil {
			continue
		}

		for _, groupCondition := range match.Conditions {
			if groupCondition.EventType == sqlcv1.V1EventTypeINTERNAL && groupCondition.EventResourceHint != nil {
				externalId, err := uuid.Parse(*groupCondition.EventResourceHint)

				if err != nil {
					return nil, nil, fmt.Errorf("failed to parse external id from group condition: %w", err)
				}

				// if the parent task is not in the subtree, we need to query the task_events table
				// to ensure the event has already occurred
				if _, ok := subtreeExternalIds[externalId]; !ok {
					externalIdUuid, err := uuid.Parse(*groupCondition.EventResourceHint)

					if err != nil {
						return nil, nil, fmt.Errorf("failed to parse external id from group condition: %w", err)
					}

					externalIds = append(externalIds, externalIdUuid)
					eventTypes = append(eventTypes, []string{groupCondition.EventKey})
				}
			}
		}
	}

	// for candidate group matches, track down the task events which satisfy the group match conditions.
	// we do this by constructing arrays for dag ids, external ids and event types, and then querying
	// by the dag_id -> v1_task (on external_id) -> v1_task_event (on event type)
	//
	// NOTE: at this point, we have already deleted the tasks and events that are in the subtree, so we
	// don't have to worry about collisions with the tasks we're replaying.
	matchedEvents, err := r.queries.ListMatchingTaskEvents(ctx, tx, sqlcv1.ListMatchingTaskEventsParams{
		Tenantid:        tenantId,
		Taskexternalids: externalIds,
		Eventtypes:      eventTypes,
	})

	if err != nil {
		return nil, nil, err
	}

	foundMatchKeys := make(map[string]*sqlcv1.ListMatchingTaskEventsRow)

	for _, eventMatch := range matchedEvents {
		key := fmt.Sprintf("%s:%s", eventMatch.ExternalID.String(), string(eventMatch.EventType))

		foundMatchKeys[key] = eventMatch
	}

	resMatches := make([]CreateMatchOpts, 0)
	resCandidateEvents := make([]CandidateEventMatch, 0)

	// for each group condition, if we have a match, mark the group condition as satisfied and use
	// the data from the match to update the group condition.
	for _, match := range eventMatches {
		if match.TriggerDAGId == nil {
			resMatches = append(resMatches, match)
			continue
		}

		conditions := make([]GroupMatchCondition, 0)

		for _, groupCondition := range match.Conditions {
			cond := groupCondition

			if groupCondition.EventType == sqlcv1.V1EventTypeINTERNAL && groupCondition.EventResourceHint != nil {
				key := fmt.Sprintf("%s:%s", *groupCondition.EventResourceHint, string(groupCondition.EventKey))

				if match, ok := foundMatchKeys[key]; ok {
					cond.Data = match.Data

					taskExternalId := match.ExternalID.String()

					resCandidateEvents = append(resCandidateEvents, CandidateEventMatch{
						ID:             uuid.New(),
						EventTimestamp: match.CreatedAt.Time,
						Key:            string(match.EventType),
						ResourceHint:   &taskExternalId,
						Data:           match.Data,
					})
				}
			}

			conditions = append(conditions, cond)
		}

		match.Conditions = conditions

		resMatches = append(resMatches, match)
	}

	return resMatches, resCandidateEvents, nil
}

type createTaskExpressionEvalOpt struct {
	Key      string
	ValueStr *string
	ValueInt *int
	Kind     sqlcv1.StepExpressionKind
}

func (r *sharedRepository) createExpressionEvals(ctx context.Context, dbtx sqlcv1.DBTX, createdTasks []*V1TaskWithPayload, opts map[uuid.UUID][]createTaskExpressionEvalOpt) error {
	if len(opts) == 0 {
		return nil
	}

	// map tasks using their external id
	taskExternalIds := make(map[uuid.UUID]*V1TaskWithPayload)

	for _, task := range createdTasks {
		taskExternalIds[task.ExternalID] = task
	}

	taskIds := make([]int64, 0)
	taskInsertedAts := make([]pgtype.Timestamptz, 0)
	keys := make([]string, 0)
	valuesStr := make([]pgtype.Text, 0)
	valuesInt := make([]pgtype.Int4, 0)
	kinds := make([]string, 0)

	for externalId, optList := range opts {
		task, ok := taskExternalIds[externalId]

		if !ok {
			r.l.Warn().Str("external_id", externalId.String()).Msg("could not find task for expression eval")
			continue
		}

		for _, opt := range optList {
			taskIds = append(taskIds, task.ID)
			taskInsertedAts = append(taskInsertedAts, task.InsertedAt)
			keys = append(keys, opt.Key)

			if opt.ValueStr != nil {
				valuesStr = append(valuesStr, pgtype.Text{
					String: *opt.ValueStr,
					Valid:  true,
				})
			} else {
				valuesStr = append(valuesStr, pgtype.Text{})
			}

			if opt.ValueInt != nil {
				valuesInt = append(valuesInt, pgtype.Int4{
					Int32: int32(*opt.ValueInt),
					Valid: true,
				})
			} else {
				valuesInt = append(valuesInt, pgtype.Int4{})
			}

			kinds = append(kinds, string(opt.Kind))
		}
	}

	return r.queries.CreateTaskExpressionEvals(
		ctx,
		dbtx,
		sqlcv1.CreateTaskExpressionEvalsParams{
			Taskids:         taskIds,
			Taskinsertedats: taskInsertedAts,
			Keys:            keys,
			Valuesstr:       valuesStr,
			Valuesint:       valuesInt,
			Kinds:           kinds,
		},
	)
}

func uniqueSet(taskIdRetryCounts []TaskIdInsertedAtRetryCount) []TaskIdInsertedAtRetryCount {
	unique := make(map[string]struct{})
	res := make([]TaskIdInsertedAtRetryCount, 0)

	for _, task := range taskIdRetryCounts {
		k := fmt.Sprintf("%d:%d", task.Id, task.RetryCount)
		if _, ok := unique[k]; !ok {
			unique[k] = struct{}{}
			res = append(res, task)
		}
	}

	return res
}

func (r *TaskRepositoryImpl) ListTaskParentOutputs(ctx context.Context, tenantId uuid.UUID, tasks []*sqlcv1.V1Task) (map[int64][]*TaskOutputEvent, error) {
	taskIds := make([]int64, 0)
	taskInsertedAts := make([]pgtype.Timestamptz, 0)

	for _, task := range tasks {
		if task.DagID.Valid {
			taskIds = append(taskIds, task.ID)
			taskInsertedAts = append(taskInsertedAts, task.InsertedAt)
		}
	}

	resMap := make(map[int64][]*TaskOutputEvent)

	if len(taskIds) == 0 {
		return resMap, nil
	}

	res, err := r.queries.ListTaskParentOutputs(ctx, r.pool, sqlcv1.ListTaskParentOutputsParams{
		Tenantid:        tenantId,
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
	})

	if err != nil {
		return nil, err
	}

	retrieveOpts := make([]RetrievePayloadOpts, 0, len(res))
	retrieveOptsToWorkflowRunId := make(map[RetrievePayloadOpts]uuid.UUID, len(res))
	retrieveOptToPayload := make(map[RetrievePayloadOpts][]byte)

	for _, outputTask := range res {
		if outputTask.WorkflowRunID == uuid.Nil {
			continue
		}

		opt := RetrievePayloadOpts{
			Id:         outputTask.TaskEventID,
			InsertedAt: outputTask.TaskEventInsertedAt,
			Type:       sqlcv1.V1PayloadTypeTASKEVENTDATA,
			TenantId:   tenantId,
		}

		retrieveOpts = append(retrieveOpts, opt)
		retrieveOptsToWorkflowRunId[opt] = outputTask.WorkflowRunID
		retrieveOptToPayload[opt] = outputTask.Output
	}

	payloads, err := r.payloadStore.Retrieve(ctx, r.pool, retrieveOpts...)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve task output payloads: %w", err)
	}

	workflowRunIdsToOutputs := make(map[string][]*TaskOutputEvent)

	for retrieveOpts, workflowRunId := range retrieveOptsToWorkflowRunId {
		wrId := workflowRunId.String()
		payload, ok := payloads[retrieveOpts]

		if !ok {
			payload = retrieveOptToPayload[retrieveOpts]
		}

		e, err := newTaskEventFromBytes(payload)

		if err != nil {
			r.l.Warn().Msgf("failed to parse task output: %v", err)
			continue
		}

		workflowRunIdsToOutputs[wrId] = append(workflowRunIdsToOutputs[wrId], e)
	}

	for _, task := range tasks {
		if task.WorkflowRunID != uuid.Nil {
			wrId := task.WorkflowRunID.String()

			if events, ok := workflowRunIdsToOutputs[wrId]; ok {
				resMap[task.ID] = events
			}
		}
	}

	return resMap, nil
}

func (r *TaskRepositoryImpl) ListSignalCompletedEvents(ctx context.Context, tenantId uuid.UUID, tasks []TaskIdInsertedAtSignalKey) ([]*V1TaskEventWithPayload, error) {
	taskIds := make([]int64, 0)
	taskInsertedAts := make([]pgtype.Timestamptz, 0)
	eventKeys := make([]string, 0)

	for _, task := range tasks {
		taskIds = append(taskIds, task.Id)
		taskInsertedAts = append(taskInsertedAts, task.InsertedAt)
		eventKeys = append(eventKeys, task.SignalKey)
	}

	signalEvents, err := r.queries.ListMatchingSignalEvents(ctx, r.pool, sqlcv1.ListMatchingSignalEventsParams{
		Tenantid:        tenantId,
		Eventtype:       sqlcv1.V1TaskEventTypeSIGNALCOMPLETED,
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Eventkeys:       eventKeys,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list matching signal events: %w", err)
	}

	retrieveOpts := make([]RetrievePayloadOpts, len(signalEvents))

	for i, event := range signalEvents {
		retrieveOpt := RetrievePayloadOpts{
			Id:         event.ID,
			InsertedAt: event.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeTASKEVENTDATA,
			TenantId:   tenantId,
		}

		retrieveOpts[i] = retrieveOpt
	}

	payloads, err := r.payloadStore.Retrieve(ctx, r.pool, retrieveOpts...)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve task event payloads: %w", err)
	}

	res := make([]*V1TaskEventWithPayload, len(signalEvents))

	for i, event := range signalEvents {
		retrieveOpt := RetrievePayloadOpts{
			Id:         event.ID,
			InsertedAt: event.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeTASKEVENTDATA,
			TenantId:   tenantId,
		}

		payload, ok := payloads[retrieveOpt]

		if !ok {
			payload = event.Data
		}

		res[i] = &V1TaskEventWithPayload{
			V1TaskEvent: event,
			Payload:     payload,
		}
	}

	return res, nil
}

func (r *TaskRepositoryImpl) AnalyzeTaskTables(ctx context.Context) error {
	const timeout = 1000 * 60 * 60 // 60 minute timeout
	tx, commit, rollback, err := sqlchelpers.PrepareTxWithStatementTimeout(ctx, r.pool, r.l, timeout)

	if err != nil {
		return fmt.Errorf("error beginning transaction: %v", err)
	}

	defer rollback()

	acquired, err := r.queries.TryAdvisoryLock(ctx, tx, hash("analyze-task-tables"))

	if err != nil {
		return fmt.Errorf("error acquiring advisory lock: %v", err)
	}

	if !acquired {
		r.l.Info().Msg("advisory lock already held, skipping task table analysis")
		return nil
	}

	err = r.queries.AnalyzeV1Task(ctx, tx)

	if err != nil {
		return fmt.Errorf("error analyzing v1_task: %v", err)
	}

	err = r.queries.AnalyzeV1TaskEvent(ctx, tx)

	if err != nil {
		return fmt.Errorf("error analyzing v1_task_event: %v", err)
	}

	err = r.queries.AnalyzeV1Dag(ctx, tx)

	if err != nil {
		return fmt.Errorf("error analyzing v1_dag: %v", err)
	}

	err = r.queries.AnalyzeV1Payload(ctx, tx)

	if err != nil {
		return fmt.Errorf("error analyzing v1_payload: %v", err)
	}

	if err := commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	return nil
}

func (r *TaskRepositoryImpl) Cleanup(ctx context.Context) (bool, error) {
	const timeout = 1000 * 60 // 1 minute timeout
	const batchSize = 1000

	var (
		mu             sync.Mutex
		shouldContinue bool
	)
	eg, ctx := errgroup.WithContext(ctx)

	// Helper to run a cleanup operation with its own transaction and advisory lock
	runCleanup := func(lockName string, cleanupFn func(ctx context.Context, tx sqlcv1.DBTX) error) func() error {
		return func() error {
			tx, commit, rollback, err := sqlchelpers.PrepareTxWithStatementTimeout(ctx, r.pool, r.l, timeout)
			if err != nil {
				return fmt.Errorf("error beginning transaction for %s: %v", lockName, err)
			}
			defer rollback()

			acquired, err := r.queries.TryAdvisoryLock(ctx, tx, hash(lockName))
			if err != nil {
				return fmt.Errorf("error acquiring advisory lock for %s: %v", lockName, err)
			}
			if !acquired {
				return nil
			}

			if err := cleanupFn(ctx, tx); err != nil {
				return fmt.Errorf("error cleaning up %s: %w", lockName, err)
			}

			if err := commit(ctx); err != nil {
				return fmt.Errorf("error committing transaction for %s: %v", lockName, err)
			}

			return nil
		}
	}

	// CleanupV1QueueItem
	eg.Go(runCleanup("cleanup-v1-queue-item", func(ctx context.Context, tx sqlcv1.DBTX) error {
		result, err := r.queries.CleanupV1QueueItem(ctx, tx, batchSize)
		if err != nil {
			return fmt.Errorf("error cleaning up v1_queue_item: %v", err)
		}
		if result.RowsAffected() == batchSize {
			mu.Lock()
			shouldContinue = true
			mu.Unlock()
		}
		return nil
	}))

	// CleanupV1RetryQueueItem
	eg.Go(runCleanup("cleanup-v1-retry-queue-item", func(ctx context.Context, tx sqlcv1.DBTX) error {
		result, err := r.queries.CleanupV1RetryQueueItem(ctx, tx, batchSize)
		if err != nil {
			return fmt.Errorf("error cleaning up v1_retry_queue_item: %v", err)
		}
		if result.RowsAffected() == batchSize {
			mu.Lock()
			shouldContinue = true
			mu.Unlock()
		}
		return nil
	}))

	// CleanupV1RateLimitedQueueItem
	eg.Go(runCleanup("cleanup-v1-rate-limited-queue-item", func(ctx context.Context, tx sqlcv1.DBTX) error {
		result, err := r.queries.CleanupV1RateLimitedQueueItem(ctx, tx, batchSize)
		if err != nil {
			return fmt.Errorf("error cleaning up v1_rate_limited_queue_items: %v", err)
		}
		if result.RowsAffected() == batchSize {
			mu.Lock()
			shouldContinue = true
			mu.Unlock()
		}
		return nil
	}))

	// CleanupMatchWithMatchConditions
	eg.Go(runCleanup("cleanup-v1-match", func(ctx context.Context, tx sqlcv1.DBTX) error {
		today := time.Now().UTC()
		removeBefore := today.Add(-1 * r.taskRetentionPeriod)

		err := r.queries.CleanupMatchWithMatchConditions(ctx, tx, pgtype.Date{
			Time:  removeBefore,
			Valid: true,
		})
		if err != nil {
			return fmt.Errorf("error cleaning up v1_match and v1_match_condition: %v", err)
		}
		return nil
	}))

	// CleanupV1TaskRuntime
	eg.Go(runCleanup("cleanup-v1-task-runtime", func(ctx context.Context, tx sqlcv1.DBTX) error {
		result, err := r.queries.CleanupV1TaskRuntime(ctx, tx, batchSize)
		if err != nil {
			return fmt.Errorf("error cleaning up v1_task_runtime: %v", err)
		}
		if result.RowsAffected() == batchSize {
			mu.Lock()
			shouldContinue = true
			mu.Unlock()
		}
		return nil
	}))

	// CleanupV1ConcurrencySlot
	eg.Go(runCleanup("cleanup-v1-concurrency-slot", func(ctx context.Context, tx sqlcv1.DBTX) error {
		result, err := r.queries.CleanupV1ConcurrencySlot(ctx, tx, batchSize)
		if err != nil {
			return fmt.Errorf("error cleaning up v1_concurrency_slot: %v", err)
		}
		if result.RowsAffected() == batchSize {
			mu.Lock()
			shouldContinue = true
			mu.Unlock()
		}
		return nil
	}))

	// ReactivateInactiveQueuesWithItems
	eg.Go(runCleanup("cleanup-reactivate-queues", func(ctx context.Context, tx sqlcv1.DBTX) error {
		result, err := r.queries.ReactivateInactiveQueuesWithItems(ctx, tx)
		if err != nil {
			return fmt.Errorf("error reactivating inactive queues: %v", err)
		}
		if result.RowsAffected() > 0 {
			// FIXME: this is an error because there is an underlying bug that needs to be fixed
			r.l.Error().Msgf("reactivated %d inactive queues with pending items", result.RowsAffected())
		}
		return nil
	}))

	if err := eg.Wait(); err != nil {
		return false, err
	}

	return shouldContinue, nil
}

// TaskStat represents the statistics for a single task step
type TaskStat struct {
	Queued  *TaskStatusStat `json:"queued,omitempty"`
	Running *TaskStatusStat `json:"running,omitempty"`
}

// TaskStatusStat represents statistics for a specific task status (queued or running)
type TaskStatusStat struct {
	Total       int64             `json:"total"`
	Oldest      *time.Time        `json:"oldest,omitempty"`
	Queues      map[string]int64  `json:"queues,omitempty"`
	Concurrency []ConcurrencyStat `json:"concurrency,omitempty"`
}

// ConcurrencyStat represents concurrency information for a task
type ConcurrencyStat struct {
	Expression string           `json:"expression"`
	Type       string           `json:"type"`
	Keys       map[string]int64 `json:"keys"`
}

func (r *TaskRepositoryImpl) GetTaskStats(ctx context.Context, tenantId uuid.UUID) (map[string]TaskStat, error) {
	rows, err := r.queries.GetTenantTaskStats(ctx, r.pool, tenantId)

	if err != nil {
		return nil, err
	}

	result := make(map[string]TaskStat)

	for _, row := range rows {
		stepReadableId := row.StepReadableID
		taskStatus := row.TaskStatus
		queue := row.Queue
		expression := row.Expression.String
		strategy := row.Strategy.String
		key := row.Key.String
		count := row.Count
		oldest := row.Oldest

		taskStat, ok := result[stepReadableId]
		if !ok {
			result[stepReadableId] = TaskStat{}
			taskStat = result[stepReadableId]
		}

		var statusStat *TaskStatusStat

		switch taskStatus {
		case "queued":
			if taskStat.Queued == nil {
				taskStat.Queued = &TaskStatusStat{
					Queues: make(map[string]int64),
				}
				result[stepReadableId] = taskStat
			}
			statusStat = result[stepReadableId].Queued

			if oldest.Valid && (statusStat.Oldest == nil || oldest.Time.Before(*statusStat.Oldest)) {
				statusStat.Oldest = &oldest.Time
			}
		case "running":
			if taskStat.Running == nil {
				taskStat.Running = &TaskStatusStat{}
				result[stepReadableId] = taskStat
			}
			statusStat = result[stepReadableId].Running

			if oldest.Valid && (statusStat.Oldest == nil || oldest.Time.Before(*statusStat.Oldest)) {
				statusStat.Oldest = &oldest.Time
			}
		}

		statusStat.Total += count

		if taskStatus == "queued" && queue != "" {
			if statusStat.Queues == nil {
				statusStat.Queues = make(map[string]int64)
			}
			statusStat.Queues[queue] += count
		}

		if expression != "" && key != "" && strategy != "NONE" {
			var concurrencyEntry *ConcurrencyStat
			for i := range statusStat.Concurrency {
				if statusStat.Concurrency[i].Expression == expression && statusStat.Concurrency[i].Type == strategy {
					concurrencyEntry = &statusStat.Concurrency[i]
					break
				}
			}

			if concurrencyEntry == nil {
				newEntry := ConcurrencyStat{
					Expression: expression,
					Type:       strategy,
					Keys:       make(map[string]int64),
				}
				statusStat.Concurrency = append(statusStat.Concurrency, newEntry)
				concurrencyEntry = &statusStat.Concurrency[len(statusStat.Concurrency)-1]
			}

			if concurrencyEntry.Keys == nil {
				concurrencyEntry.Keys = make(map[string]int64)
			}
			concurrencyEntry.Keys[key] += count
		}
	}

	return result, nil
}

func (r *TaskRepositoryImpl) FindOldestRunningTaskInsertedAt(ctx context.Context) (*time.Time, error) {
	t, err := r.queries.FindOldestRunningTask(ctx, r.pool)

	if err != nil {
		return nil, err
	}

	if t == nil {
		return nil, nil
	}

	return &t.TaskInsertedAt.Time, nil
}

func (r *TaskRepositoryImpl) FindOldestTaskInsertedAt(ctx context.Context) (*time.Time, error) {
	t, err := r.queries.FindOldestTask(ctx, r.pool)

	if err != nil {
		return nil, err
	}

	if t == nil {
		return nil, nil
	}

	return &t.InsertedAt.Time, nil
}

type TaskRunDetails struct {
	Error         *string
	Status        statusutils.V1RunStatus
	ExternalId    uuid.UUID
	OutputPayload []byte
}

type StepReadableId string

type WorkflowRunDetails struct {
	ReadableIdToDetails map[StepReadableId]TaskRunDetails
	InputPayload        []byte
	AdditionalMetadata  []byte
}

func (r *TaskRepositoryImpl) GetWorkflowRunResultDetails(ctx context.Context, tenantId uuid.UUID, externalId uuid.UUID) (*WorkflowRunDetails, error) {
	flat, err := r.FlattenExternalIds(ctx, tenantId, []uuid.UUID{externalId})

	if err != nil {
		return nil, fmt.Errorf("failed to flatten external ids: %w", err)
	}

	finalizedWorkflowRuns, err := r.ListFinalizedWorkflowRuns(ctx, tenantId, []uuid.UUID{externalId})

	if err != nil {
		return nil, fmt.Errorf("failed to list finalized workflow runs: %w", err)
	}

	if len(flat) == 0 {
		return nil, nil
	}

	var inputRetrieveOpt RetrievePayloadOpts
	firstTask := flat[0]
	isDag := firstTask.DagID.Valid
	additionalMeta := firstTask.AdditionalMetadata

	if isDag {
		inputRetrieveOpt = RetrievePayloadOpts{
			Id:         firstTask.DagID.Int64,
			InsertedAt: firstTask.DagInsertedAt,
			Type:       sqlcv1.V1PayloadTypeDAGINPUT,
			TenantId:   tenantId,
		}
	} else {
		inputRetrieveOpt = RetrievePayloadOpts{
			Id:         firstTask.ID,
			InsertedAt: firstTask.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeTASKINPUT,
			TenantId:   tenantId,
		}
	}

	payloads, err := r.payloadStore.Retrieve(ctx, r.pool, inputRetrieveOpt)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payloads: %w", err)
	}

	input := payloads[inputRetrieveOpt]

	if !isDag && len(input) > 0 {
		// if it's a standalone task, we need to extract the "input" field from the payload
		stepRunData, err := r.V1StepRunDataFromBytes(input)

		if err != nil {
			return nil, fmt.Errorf("failed to parse step run data: %w", err)
		}

		input = stepRunData.InputBytes()
	}

	taskRunDetails := make(map[StepReadableId]TaskRunDetails)
	taskIds := make([]int64, 0)
	taskInsertedAts := make([]pgtype.Timestamptz, 0)
	taskRetryCounts := make([]int32, 0)

	for _, task := range flat {
		taskIds = append(taskIds, task.ID)
		taskInsertedAts = append(taskInsertedAts, task.InsertedAt)
		taskRetryCounts = append(taskRetryCounts, task.RetryCount)
	}

	taskStats, err := r.queries.ListTaskRunningStatuses(ctx, r.pool, sqlcv1.ListTaskRunningStatusesParams{
		Tenantid:        tenantId,
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Taskretrycounts: taskRetryCounts,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list task running statuses: %w", err)
	}

	externalIdToIsRunning := make(map[string]bool)

	for _, stat := range taskStats {
		externalIdToIsRunning[stat.ExternalID.String()] = stat.IsRunning
	}

	for _, task := range flat {
		isRunning := externalIdToIsRunning[task.ExternalID.String()]
		status := statusutils.V1RunStatusQueued

		if isRunning {
			status = statusutils.V1RunStatusRunning
		}

		// default everything to QUEUED
		// we'll overwrite later with more information if available
		taskRunDetails[StepReadableId(task.StepReadableID)] = TaskRunDetails{
			OutputPayload: nil,
			Status:        status,
			Error:         nil,
			ExternalId:    task.ExternalID,
		}
	}

	if len(flat) > 0 && len(finalizedWorkflowRuns) == 0 {
		return &WorkflowRunDetails{
			InputPayload:        input,
			ReadableIdToDetails: taskRunDetails,
			AdditionalMetadata:  additionalMeta,
		}, nil
	}

	outputs := make(map[string][]byte)
	errors := make(map[string]string)
	finalizedRun := finalizedWorkflowRuns[0]

	for _, event := range finalizedRun.OutputEvents {
		outputs[event.StepReadableID] = event.Output
		errors[event.StepReadableID] = event.ErrorMessage

		status, err := statusutils.V1RunStatusFromEventType(event.EventType)

		if err != nil {
			r.l.Error().Msgf("failed to parse event type %s: %v", event.EventType, err)
			statusPtr := statusutils.V1RunStatusQueued
			status = &statusPtr
		}

		taskRunDetails[StepReadableId(event.StepReadableID)] = TaskRunDetails{
			OutputPayload: event.Output,
			Status:        *status,
			Error:         &event.ErrorMessage,
			ExternalId:    event.TaskExternalId,
		}
	}

	return &WorkflowRunDetails{
		InputPayload:        input,
		ReadableIdToDetails: taskRunDetails,
		AdditionalMetadata:  additionalMeta,
	}, nil
}

func (r *TaskRepositoryImpl) FilterValidTasks(ctx context.Context, tenantId uuid.UUID, opts []TaskIdInsertedAtRetryCount) (map[int64]struct{}, error) {
	res := make(map[int64]struct{})

	taskIds := make([]int64, len(opts))
	taskInsertedAts := make([]pgtype.Timestamptz, len(opts))
	taskRetryCounts := make([]int32, len(opts))

	for i, opt := range opts {
		taskIds[i] = opt.Id
		taskInsertedAts[i] = opt.InsertedAt
		taskRetryCounts[i] = opt.RetryCount
	}

	taskIds, err := r.queries.FilterValidTasks(ctx, r.pool, sqlcv1.FilterValidTasksParams{
		Tenantid:        tenantId,
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Taskretrycounts: taskRetryCounts,
	})
	if err != nil {
		return nil, err
	}

	for _, taskId := range taskIds {
		res[taskId] = struct{}{}
	}

	return res, nil
}
