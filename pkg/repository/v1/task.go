package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

const MAX_INTERNAL_RETRIES = 3

type CreateTaskOpts struct {
	// (required) the external id
	ExternalId string `validate:"required,uuid"`

	// (required) the workflow run id. note this may be the same as the external id if this is a
	// single-task workflow, otherwise it represents the external id of the DAG.
	WorkflowRunId string `validate:"required,uuid"`

	// (required) the step id
	StepId string `validate:"required,uuid"`

	// (required) the input bytes to the task
	Input *TaskInput

	// (required) the step index for the task
	StepIndex int

	// (optional) the additional metadata for the task
	AdditionalMetadata []byte

	// (optional) the desired worker id
	DesiredWorkerId *string

	// (optional) the DAG id for the task
	DagId *int64

	// (optional) the DAG inserted at for the task
	DagInsertedAt pgtype.Timestamptz

	// (required) the initial state for the task
	InitialState sqlcv1.V1TaskInitialState

	// (optional) the parent task external id
	ParentTaskExternalId *string

	// (optional) the parent task id
	ParentTaskId *int64

	// (optional) the parent task inserted at
	ParentTaskInsertedAt *time.Time

	// (optional) the child index for the task
	ChildIndex *int64

	// (optional) the child key for the task
	ChildKey *string
}

type ReplayTasksResult struct {
	ReplayedTasks []TaskIdInsertedAtRetryCount

	QueuedTasks []*sqlcv1.V1Task

	InternalEventResults *InternalEventMatchResults
}

type ReplayTaskOpts struct {
	// (required) the task id
	TaskId int64

	// (required) the inserted at time
	InsertedAt pgtype.Timestamptz

	// (required) the external id
	ExternalId string

	// (required) the step id
	StepId string

	// (optional) the input bytes to the task, uses the existing input if not set
	Input *TaskInput

	// (required) the initial state for the task
	InitialState sqlcv1.V1TaskInitialState

	// (optional) the additional metadata for the task
	AdditionalMetadata []byte
}

type TaskIdInsertedAtRetryCount struct {
	// (required) the external id
	Id int64 `validate:"required"`

	// (required) the inserted at time
	InsertedAt pgtype.Timestamptz

	// (required) the retry count
	RetryCount int32
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
}

type TaskIdEventKeyTuple struct {
	Id int64 `validate:"required"`

	EventKey string `validate:"required"`
}

// InternalTaskEvent resembles sqlcv1.V1TaskEvent, but doesn't include the id field as we
// use COPY FROM to write the events to the database.
type InternalTaskEvent struct {
	TenantID       string                 `json:"tenant_id"`
	TaskID         int64                  `json:"task_id"`
	TaskExternalID string                 `json:"task_external_id"`
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

	AppRetryCount int32

	RetryBackoffFactor pgtype.Float8

	RetryMaxBackoff pgtype.Int4
}

type FailTasksResponse struct {
	*FinalizedTaskResponse

	RetriedTasks []RetriedTask
}

type ListFinalizedWorkflowRunsResponse struct {
	WorkflowRunId string

	OutputEvents []*TaskOutputEvent
}

type RefreshTimeoutBy struct {
	TaskExternalId string `validate:"required,uuid"`

	IncrementTimeoutBy string `validate:"required,duration"`
}

type TaskRepository interface {
	UpdateTablePartitions(ctx context.Context) error

	// GetTaskByExternalId is a heavily cached method to return task metadata by its external id
	GetTaskByExternalId(ctx context.Context, tenantId, taskExternalId string, skipCache bool) (*sqlcv1.FlattenExternalIdsRow, error)

	// FlattenExternalIds is a non-cached method to look up all tasks in a workflow run by their external ids.
	// This is non-cacheable because tasks can be added to a workflow run as it executes.
	FlattenExternalIds(ctx context.Context, tenantId string, externalIds []string) ([]*sqlcv1.FlattenExternalIdsRow, error)

	CompleteTasks(ctx context.Context, tenantId string, tasks []CompleteTaskOpts) (*FinalizedTaskResponse, error)

	FailTasks(ctx context.Context, tenantId string, tasks []FailTaskOpts) (*FailTasksResponse, error)

	CancelTasks(ctx context.Context, tenantId string, tasks []TaskIdInsertedAtRetryCount) (*FinalizedTaskResponse, error)

	ListTasks(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv1.V1Task, error)

	ListTaskMetas(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv1.ListTaskMetasRow, error)

	ListFinalizedWorkflowRuns(ctx context.Context, tenantId string, rootExternalIds []string) ([]*ListFinalizedWorkflowRunsResponse, error)

	ProcessTaskTimeouts(ctx context.Context, tenantId string) ([]*sqlcv1.ProcessTaskTimeoutsRow, bool, error)

	ProcessTaskReassignments(ctx context.Context, tenantId string) ([]*sqlcv1.ProcessTaskReassignmentsRow, bool, error)

	ProcessTaskRetryQueueItems(ctx context.Context, tenantId string) ([]*sqlcv1.V1RetryQueueItem, bool, error)

	GetQueueCounts(ctx context.Context, tenantId string) (map[string]int, error)

	ReplayTasks(ctx context.Context, tenantId string, tasks []TaskIdInsertedAtRetryCount) (*ReplayTasksResult, error)

	RefreshTimeoutBy(ctx context.Context, tenantId string, opt RefreshTimeoutBy) (*sqlcv1.V1TaskRuntime, error)

	ReleaseSlot(ctx context.Context, tenantId string, externalId string) (*sqlcv1.V1TaskRuntime, error)
}

type TaskRepositoryImpl struct {
	*sharedRepository

	taskRetentionPeriod time.Duration
}

func newTaskRepository(s *sharedRepository, taskRetentionPeriod time.Duration) TaskRepository {
	return &TaskRepositoryImpl{
		sharedRepository:    s,
		taskRetentionPeriod: taskRetentionPeriod,
	}
}

func (r *TaskRepositoryImpl) UpdateTablePartitions(ctx context.Context) error {
	today := time.Now().UTC()
	tomorrow := today.AddDate(0, 0, 1)
	removeBefore := today.Add(-1 * r.taskRetentionPeriod)

	err := r.queries.CreatePartitions(ctx, r.pool, pgtype.Date{
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
		r.l.Warn().Msgf("detaching partition %s", partition.PartitionName)

		_, err := r.pool.Exec(
			ctx,
			fmt.Sprintf("ALTER TABLE %s DETACH PARTITION %s CONCURRENTLY", partition.ParentTable, partition.PartitionName),
		)

		if err != nil {
			return err
		}

		_, err = r.pool.Exec(
			ctx,
			fmt.Sprintf("DROP TABLE %s", partition.PartitionName),
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *TaskRepositoryImpl) GetTaskByExternalId(ctx context.Context, tenantId, taskExternalId string, skipCache bool) (*sqlcv1.FlattenExternalIdsRow, error) {
	if !skipCache {
		// check the cache first
		key := taskExternalIdTenantIdTuple{
			externalId: taskExternalId,
			tenantId:   tenantId,
		}

		if val, ok := r.taskLookupCache.Get(key); ok {
			return val, nil
		}
	}

	// lookup the task
	dbTasks, err := r.queries.FlattenExternalIds(ctx, r.pool, sqlcv1.FlattenExternalIdsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Externalids: []pgtype.UUID{sqlchelpers.UUIDFromStr(taskExternalId)},
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

func (r *TaskRepositoryImpl) FlattenExternalIds(ctx context.Context, tenantId string, externalIds []string) ([]*sqlcv1.FlattenExternalIdsRow, error) {
	return r.lookupExternalIds(ctx, r.pool, tenantId, externalIds)
}

func (r *sharedRepository) lookupExternalIds(ctx context.Context, tx sqlcv1.DBTX, tenantId string, externalIds []string) ([]*sqlcv1.FlattenExternalIdsRow, error) {
	externalIdsToLookup := make([]pgtype.UUID, 0, len(externalIds))
	res := make([]*sqlcv1.FlattenExternalIdsRow, 0, len(externalIds))

	for _, externalId := range externalIds {
		if externalId == "" {
			r.l.Error().Msgf("passed in empty external id")
			continue
		}

		externalIdsToLookup = append(externalIdsToLookup, sqlchelpers.UUIDFromStr(externalId))
	}

	// lookup the task
	dbTasks, err := r.queries.FlattenExternalIds(ctx, tx, sqlcv1.FlattenExternalIdsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Externalids: externalIdsToLookup,
	})

	if err != nil {
		return nil, err
	}

	// set the cache
	groupedExternalIds := make(map[string][]*sqlcv1.FlattenExternalIdsRow)

	for _, task := range dbTasks {
		rootExternalId := sqlchelpers.UUIDToStr(task.WorkflowRunExternalID)

		groupedExternalIds[rootExternalId] = append(groupedExternalIds[rootExternalId], task)
	}

	for _, tasks := range groupedExternalIds {
		res = append(res, tasks...)
	}

	return res, nil
}

func (r *TaskRepositoryImpl) verifyAllTasksFinalized(ctx context.Context, tx sqlcv1.DBTX, tenantId string, flattenedTasks []*sqlcv1.FlattenExternalIdsRow) ([]string, map[string]int64, error) {
	taskIdsToCheck := make([]int64, 0, len(flattenedTasks))
	taskIdsToTasks := make(map[int64]*sqlcv1.FlattenExternalIdsRow)

	for _, task := range flattenedTasks {
		taskIdsToCheck = append(taskIdsToCheck, task.ID)
		taskIdsToTasks[task.ID] = task
	}

	// run preflight check on tasks
	notFinalized, err := r.queries.PreflightCheckTasksForReplay(ctx, tx, sqlcv1.PreflightCheckTasksForReplayParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Taskids:  taskIdsToCheck,
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
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, nil, err
	}

	notFinalizedDAGsMap := make(map[int64]bool)
	finalizedDAGToStepCount := make(map[string]int64)

	for _, dag := range notFinalizedDags {
		if dag.StepCount != dag.TaskCount {
			notFinalizedDAGsMap[dag.ID] = true
		} else {
			rootId := sqlchelpers.UUIDToStr(dag.ExternalID)
			finalizedDAGToStepCount[rootId] = dag.StepCount
		}
	}

	candidateFinalizedRootExternalIds := make(map[string]bool, 0)

	for _, task := range flattenedTasks {
		candidateFinalizedRootExternalIds[sqlchelpers.UUIDToStr(task.WorkflowRunExternalID)] = true
	}

	// iterate through tasks one last time
	for _, task := range flattenedTasks {
		rootId := sqlchelpers.UUIDToStr(task.WorkflowRunExternalID)

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

	finalizedRootExternalIds := make([]string, 0)

	for rootId, finalized := range candidateFinalizedRootExternalIds {
		if finalized {
			finalizedRootExternalIds = append(finalizedRootExternalIds, rootId)
		}
	}

	return finalizedRootExternalIds, finalizedDAGToStepCount, nil
}

func (r *TaskRepositoryImpl) CompleteTasks(ctx context.Context, tenantId string, tasks []CompleteTaskOpts) (*FinalizedTaskResponse, error) {
	// TODO: ADD BACK VALIDATION
	// if err := r.v.Validate(tasks); err != nil {
	// 	fmt.Println("FAILED VALIDATION HERE!!!")

	// 	return err
	// }

	taskIdRetryCounts := make([]TaskIdInsertedAtRetryCount, len(tasks))

	for i, task := range tasks {
		taskIdRetryCounts[i] = *task.TaskIdInsertedAtRetryCount
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	taskIdRetryCounts = uniqueSet(taskIdRetryCounts)

	// release queue items
	releasedTasks, err := r.releaseTasks(ctx, tx, tenantId, taskIdRetryCounts)

	if err != nil {
		return nil, err
	}

	if len(taskIdRetryCounts) != len(releasedTasks) {
		return nil, fmt.Errorf("failed to release all tasks")
	}

	datas := make([][]byte, len(releasedTasks))
	insertedAts := make([]pgtype.Timestamptz, len(releasedTasks))
	externalIds := make([]string, len(releasedTasks))

	for i, releasedTask := range releasedTasks {
		out := NewCompletedTaskOutputEvent(releasedTask, tasks[i].Output)

		datas[i] = out.Bytes()
		insertedAts[i] = releasedTask.InsertedAt
		externalIds[i] = sqlchelpers.UUIDToStr(releasedTask.ExternalID)
	}

	internalEvents, err := r.createTaskEvents(
		ctx,
		tx,
		tenantId,
		taskIdRetryCounts,
		externalIds,
		datas,
		sqlcv1.V1TaskEventTypeCOMPLETED,
		make([]string, len(tasks)),
	)

	if err != nil {
		return nil, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return nil, err
	}

	return &FinalizedTaskResponse{
		ReleasedTasks:  releasedTasks,
		InternalEvents: internalEvents,
	}, nil
}

func (r *TaskRepositoryImpl) FailTasks(ctx context.Context, tenantId string, failureOpts []FailTaskOpts) (*FailTasksResponse, error) {
	// TODO: ADD BACK VALIDATION
	// if err := r.v.Validate(tasks); err != nil {
	// 	fmt.Println("FAILED VALIDATION HERE!!!")

	// 	return err
	// }

	tasks := make([]TaskIdInsertedAtRetryCount, len(failureOpts))
	appFailureTaskIds := make([]int64, 0)
	appFailureTaskInsertedAts := make([]pgtype.Timestamptz, 0)
	internalFailureTaskIds := make([]int64, 0)
	internalFailureInsertedAts := make([]pgtype.Timestamptz, 0)

	for i, failureOpt := range failureOpts {
		tasks[i] = *failureOpt.TaskIdInsertedAtRetryCount

		if failureOpt.IsAppError {
			appFailureTaskIds = append(appFailureTaskIds, failureOpt.Id)
			appFailureTaskInsertedAts = append(appFailureTaskInsertedAts, failureOpt.InsertedAt)
		} else {
			internalFailureTaskIds = append(internalFailureTaskIds, failureOpt.Id)
			internalFailureInsertedAts = append(internalFailureInsertedAts, failureOpt.InsertedAt)
		}
	}

	tasks = uniqueSet(tasks)

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	// release queue items
	releasedTasks, err := r.releaseTasks(ctx, tx, tenantId, tasks)

	if err != nil {
		return nil, err
	}

	if len(tasks) != len(releasedTasks) {
		return nil, fmt.Errorf("failed to release all tasks")
	}

	datas := make([][]byte, len(releasedTasks))
	externalIds := make([]string, len(releasedTasks))

	for i, releasedTask := range releasedTasks {
		out := NewFailedTaskOutputEvent(releasedTask, failureOpts[i].ErrorMessage)

		datas[i] = out.Bytes()
		externalIds[i] = sqlchelpers.UUIDToStr(releasedTask.ExternalID)
	}

	retriedTasks := make([]RetriedTask, 0)

	// write app failures
	if len(appFailureTaskIds) > 0 {
		appFailureRetries, err := r.queries.FailTaskAppFailure(ctx, tx, sqlcv1.FailTaskAppFailureParams{
			Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
			Taskids:         appFailureTaskIds,
			Taskinsertedats: appFailureTaskInsertedAts,
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
				AppRetryCount:      task.AppRetryCount,
				RetryBackoffFactor: task.RetryBackoffFactor,
				RetryMaxBackoff:    task.RetryMaxBackoff,
			},
			)
		}
	}

	// write internal failures
	if len(internalFailureTaskIds) > 0 {
		internalFailureRetries, err := r.queries.FailTaskInternalFailure(ctx, tx, sqlcv1.FailTaskInternalFailureParams{
			Tenantid:           sqlchelpers.UUIDFromStr(tenantId),
			Taskids:            internalFailureTaskIds,
			Taskinsertedats:    internalFailureInsertedAts,
			Maxinternalretries: MAX_INTERNAL_RETRIES,
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

	// write task events
	internalEvents, err := r.createTaskEvents(
		ctx,
		tx,
		tenantId,
		tasks,
		externalIds,
		datas,
		sqlcv1.V1TaskEventTypeFAILED,
		make([]string, len(tasks)),
	)

	if err != nil {
		return nil, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
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

func (r *TaskRepositoryImpl) ListFinalizedWorkflowRuns(ctx context.Context, tenantId string, rootExternalIds []string) ([]*ListFinalizedWorkflowRunsResponse, error) {
	start := time.Now()
	checkpoint := time.Now()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 30000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	externalIdsToEvents := make(map[string][]*TaskOutputEvent)

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

	taskExternalIds := make([]string, 0, len(tasks))
	taskExternalIdsToRootIds := make(map[string]string)

	for _, task := range tasks {
		taskExternalIds = append(taskExternalIds, sqlchelpers.UUIDToStr(task.ExternalID))
		taskExternalIdsToRootIds[sqlchelpers.UUIDToStr(task.ExternalID)] = sqlchelpers.UUIDToStr(task.WorkflowRunExternalID)
	}

	outputEvents, err := r.listTaskOutputEvents(ctx, tx, tenantId, taskExternalIds)

	if err != nil {
		return nil, err
	}

	durOutputEvents := time.Since(checkpoint)

	if err := commit(ctx); err != nil {
		return nil, err
	}

	taskExternalIdsHasOutputEvent := make(map[string]bool)

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

	finalizedRootIdsMap := make(map[string]bool)

	for _, rootId := range finalizedRootIds {
		finalizedRootIdsMap[rootId] = true
	}

	// if tasks that we read originally don't have a TaskOutputEvent, they're not finalized, so set their root
	// ids in finalizedRootIdsMap to false
	for _, taskExternalId := range taskExternalIds {
		if !taskExternalIdsHasOutputEvent[taskExternalId] {
			rootId, ok := taskExternalIdsToRootIds[taskExternalId]

			if !ok {
				r.l.Warn().Msgf("could not find root id for task %s", taskExternalId)
				continue
			}

			finalizedRootIdsMap[rootId] = false
		}
	}

	// look for finalized events...
	eventsForFinalizedRootIds := make(map[string][]*TaskOutputEvent)

	for _, rootId := range finalizedRootIds {
		if !finalizedRootIdsMap[rootId] {
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

func (r *TaskRepositoryImpl) CancelTasks(ctx context.Context, tenantId string, tasks []TaskIdInsertedAtRetryCount) (*FinalizedTaskResponse, error) {
	// TODO: ADD BACK VALIDATION
	// if err := r.v.Validate(tasks); err != nil {
	// 	fmt.Println("FAILED VALIDATION HERE!!!")

	// 	return err
	// }

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	// release queue items
	res, err := r.cancelTasks(ctx, tx, tenantId, tasks)

	if err != nil {
		return nil, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return nil, err
	}

	return res, nil
}

func (r *sharedRepository) cancelTasks(ctx context.Context, dbtx sqlcv1.DBTX, tenantId string, tasks []TaskIdInsertedAtRetryCount) (*FinalizedTaskResponse, error) {
	// get a unique set of task ids and retry counts
	tasks = uniqueSet(tasks)

	// release queue items
	releasedTasks, err := r.releaseTasks(ctx, dbtx, tenantId, tasks)

	if err != nil {
		return nil, err
	}

	if len(tasks) != len(releasedTasks) {
		return nil, fmt.Errorf("failed to release all tasks")
	}

	datas := make([][]byte, len(releasedTasks))
	insertedAts := make([]pgtype.Timestamptz, len(releasedTasks))
	externalIds := make([]string, len(releasedTasks))

	for i, releasedTask := range releasedTasks {
		out := NewCancelledTaskOutputEvent(releasedTask)

		datas[i] = out.Bytes()
		insertedAts[i] = releasedTask.InsertedAt
		externalIds[i] = sqlchelpers.UUIDToStr(releasedTask.ExternalID)
	}

	// write task events
	internalEvents, err := r.createTaskEvents(
		ctx,
		dbtx,
		tenantId,
		tasks,
		externalIds,
		datas,
		sqlcv1.V1TaskEventTypeCANCELLED,
		make([]string, len(tasks)),
	)

	if err != nil {
		return nil, err
	}

	return &FinalizedTaskResponse{
		ReleasedTasks:  releasedTasks,
		InternalEvents: internalEvents,
	}, nil
}

func (r *TaskRepositoryImpl) ListTasks(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv1.V1Task, error) {
	return r.listTasks(ctx, r.pool, tenantId, tasks)
}

func (r *sharedRepository) listTasks(ctx context.Context, dbtx sqlcv1.DBTX, tenantId string, tasks []int64) ([]*sqlcv1.V1Task, error) {
	return r.queries.ListTasks(ctx, dbtx, sqlcv1.ListTasksParams{
		TenantID: sqlchelpers.UUIDFromStr(tenantId),
		Ids:      tasks,
	})
}

func (r *TaskRepositoryImpl) listTaskOutputEvents(ctx context.Context, tx sqlcv1.DBTX, tenantId string, taskExternalIds []string) ([]*TaskOutputEvent, error) {
	externalIds := make([]pgtype.UUID, 0)
	eventTypes := make([][]string, 0)

	for _, externalId := range taskExternalIds {
		externalIds = append(externalIds, sqlchelpers.UUIDFromStr(externalId))
		eventTypes = append(eventTypes, []string{
			string(sqlcv1.V1TaskEventTypeCOMPLETED),
			string(sqlcv1.V1TaskEventTypeFAILED),
			string(sqlcv1.V1TaskEventTypeCANCELLED),
		})
	}

	matchedEvents, err := r.queries.ListMatchingTaskEvents(ctx, tx, sqlcv1.ListMatchingTaskEventsParams{
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
		Taskexternalids: externalIds,
		Eventtypes:      eventTypes,
	})

	if err != nil {
		return nil, err
	}

	res := make([]*TaskOutputEvent, 0, len(matchedEvents))

	for _, event := range matchedEvents {
		o, err := newTaskEventFromBytes(event.Data)

		if err != nil {
			return nil, err
		}

		res = append(res, o)
	}

	return res, nil
}

func (r *TaskRepositoryImpl) ListTaskMetas(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv1.ListTaskMetasRow, error) {
	return r.queries.ListTaskMetas(ctx, r.pool, sqlcv1.ListTaskMetasParams{
		TenantID: sqlchelpers.UUIDFromStr(tenantId),
		Ids:      tasks,
	})
}

func (r *TaskRepositoryImpl) ProcessTaskTimeouts(ctx context.Context, tenantId string) ([]*sqlcv1.ProcessTaskTimeoutsRow, bool, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, false, err
	}

	defer rollback()

	// TODO: make limit configurable
	limit := 1000

	// get task timeouts
	res, err := r.queries.ProcessTaskTimeouts(ctx, tx, sqlcv1.ProcessTaskTimeoutsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
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

func (r *TaskRepositoryImpl) ProcessTaskReassignments(ctx context.Context, tenantId string) ([]*sqlcv1.ProcessTaskReassignmentsRow, bool, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, false, err
	}

	defer rollback()

	// TODO: make limit configurable
	limit := 1000

	// get task reassignments
	res, err := r.queries.ProcessTaskReassignments(ctx, tx, sqlcv1.ProcessTaskReassignmentsParams{
		Tenantid:           sqlchelpers.UUIDFromStr(tenantId),
		Maxinternalretries: MAX_INTERNAL_RETRIES,
		Limit: pgtype.Int4{
			Int32: int32(limit),
			Valid: true,
		},
	})

	if err != nil {
		return nil, false, err
	}

	// write failed tasks as task events
	failedTasks := make([]TaskIdInsertedAtRetryCount, 0)

	for _, task := range res {
		if task.Operation == "FAILED" {
			failedTasks = append(failedTasks, TaskIdInsertedAtRetryCount{
				Id:         task.ID,
				InsertedAt: task.InsertedAt,
				RetryCount: task.RetryCount,
			})
		}
	}

	// failedTaskDatas := make([][]byte, len(failedTasks))

	if len(failedTasks) > 0 {
		// TODO: FIX REASSIGNMENTS
		// _, err = r.createTaskEvents(
		// 	ctx,
		// 	tx,
		// 	tenantId,
		// 	failedTasks,
		// 	failedTaskDatas,
		// 	sqlcv1.V1TaskEventTypeFAILED,
		// 	make([]string, len(failedTasks)),
		// )

		// if err != nil {
		// 	return nil, false, err
		// }
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return nil, false, err
	}

	return res, len(res) == limit, nil
}

func (r *TaskRepositoryImpl) ProcessTaskRetryQueueItems(ctx context.Context, tenantId string) ([]*sqlcv1.V1RetryQueueItem, bool, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, false, err
	}

	defer rollback()

	// TODO: make limit configurable
	limit := 10000

	// get task reassignments
	res, err := r.queries.ProcessRetryQueueItems(ctx, tx, sqlcv1.ProcessRetryQueueItemsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
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

func (r *TaskRepositoryImpl) GetQueueCounts(ctx context.Context, tenantId string) (map[string]int, error) {
	counts, err := r.queries.GetQueuedCounts(ctx, r.pool, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return map[string]int{}, nil
		}

		return nil, err
	}

	res := make(map[string]int)

	for _, count := range counts {
		res[count.Queue] = int(count.Count)
	}

	return res, nil
}

func (r *TaskRepositoryImpl) RefreshTimeoutBy(ctx context.Context, tenantId string, opt RefreshTimeoutBy) (*sqlcv1.V1TaskRuntime, error) {
	if err := r.v.Validate(opt); err != nil {
		return nil, err
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	res, err := r.queries.RefreshTimeoutBy(ctx, tx, sqlcv1.RefreshTimeoutByParams{
		Tenantid:           sqlchelpers.UUIDFromStr(tenantId),
		Externalid:         sqlchelpers.UUIDFromStr(opt.TaskExternalId),
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

func (r *TaskRepositoryImpl) ReleaseSlot(ctx context.Context, tenantId, externalId string) (*sqlcv1.V1TaskRuntime, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	resp, err := r.queries.ManualSlotRelease(
		ctx,
		tx,
		sqlcv1.ManualSlotReleaseParams{
			Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
			Externalid: sqlchelpers.UUIDFromStr(externalId),
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

func (r *sharedRepository) releaseTasks(ctx context.Context, tx sqlcv1.DBTX, tenantId string, tasks []TaskIdInsertedAtRetryCount) ([]*sqlcv1.ReleaseTasksRow, error) {
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
		return nil, fmt.Errorf("failed to release all tasks: %d/%d", len(releasedTasks), len(tasks))
	}

	res := make([]*sqlcv1.ReleaseTasksRow, len(tasks))

	for _, task := range releasedTasks {
		idx := orderedMap[fmt.Sprintf("%d:%d", task.ID, task.RetryCount)]
		res[idx] = task
	}

	return res, nil
}

func (r *sharedRepository) upsertQueues(ctx context.Context, tx sqlcv1.DBTX, tenantId string, queues []string) (func(), error) {
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
		TenantID: sqlchelpers.UUIDFromStr(tenantId),
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

func getQueueCacheKey(tenantId string, queue string) string {
	return fmt.Sprintf("%s:%s", tenantId, queue)
}

func (r *sharedRepository) createTasks(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId string,
	tasks []CreateTaskOpts,
) ([]*sqlcv1.V1Task, error) {
	// list the steps for the tasks
	uniqueStepIds := make(map[string]struct{})
	stepIds := make([]pgtype.UUID, 0)

	for _, task := range tasks {
		if _, ok := uniqueStepIds[task.StepId]; !ok {
			uniqueStepIds[task.StepId] = struct{}{}
			stepIds = append(stepIds, sqlchelpers.UUIDFromStr(task.StepId))
		}
	}

	steps, err := r.queries.ListStepsByIds(ctx, tx, sqlcv1.ListStepsByIdsParams{
		Ids:      stepIds,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	stepIdsToConfig := make(map[string]*sqlcv1.ListStepsByIdsRow)

	for _, step := range steps {
		stepIdsToConfig[sqlchelpers.UUIDToStr(step.ID)] = step
	}

	return r.insertTasks(ctx, tx, tenantId, tasks, stepIdsToConfig)
}

// insertTasks inserts new tasks into the database. note that we're using Postgres rules to automatically insert the created
// tasks into the queue_items table.
func (r *sharedRepository) insertTasks(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId string,
	tasks []CreateTaskOpts,
	stepIdsToConfig map[string]*sqlcv1.ListStepsByIdsRow,
) ([]*sqlcv1.V1Task, error) {
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

	tenantIds := make([]pgtype.UUID, len(tasks))
	queues := make([]string, len(tasks))
	actionIds := make([]string, len(tasks))
	stepIds := make([]pgtype.UUID, len(tasks))
	stepReadableIds := make([]string, len(tasks))
	workflowIds := make([]pgtype.UUID, len(tasks))
	scheduleTimeouts := make([]string, len(tasks))
	stepTimeouts := make([]string, len(tasks))
	priorities := make([]int32, len(tasks))
	stickies := make([]string, len(tasks))
	desiredWorkerIds := make([]pgtype.UUID, len(tasks))
	externalIds := make([]pgtype.UUID, len(tasks))
	displayNames := make([]string, len(tasks))
	inputs := make([][]byte, len(tasks))
	retryCounts := make([]int32, len(tasks))
	additionalMetadatas := make([][]byte, len(tasks))
	initialStates := make([]string, len(tasks))
	initialStateReasons := make([]pgtype.Text, len(tasks))
	dagIds := make([]pgtype.Int8, len(tasks))
	dagInsertedAts := make([]pgtype.Timestamptz, len(tasks))
	strategyIds := make([][]int64, len(tasks))
	concurrencyKeys := make([][]string, len(tasks))
	parentTaskExternalIds := make([]pgtype.UUID, len(tasks))
	parentTaskIds := make([]pgtype.Int8, len(tasks))
	parentTaskInsertedAts := make([]pgtype.Timestamptz, len(tasks))
	childIndices := make([]pgtype.Int8, len(tasks))
	childKeys := make([]pgtype.Text, len(tasks))
	stepIndices := make([]int64, len(tasks))
	retryBackoffFactors := make([]pgtype.Float8, len(tasks))
	retryMaxBackoffs := make([]pgtype.Int4, len(tasks))
	createExpressionOpts := make(map[string][]createTaskExpressionEvalOpt, 0)
	workflowVersionIds := make([]pgtype.UUID, len(tasks))
	workflowRunIds := make([]pgtype.UUID, len(tasks))

	unix := time.Now().UnixMilli()

	for i, task := range tasks {
		stepConfig := stepIdsToConfig[task.StepId]
		tenantIds[i] = sqlchelpers.UUIDFromStr(tenantId)
		queues[i] = stepConfig.ActionId // FIXME: make the queue name dynamic
		actionIds[i] = stepConfig.ActionId
		stepIds[i] = sqlchelpers.UUIDFromStr(task.StepId)
		stepReadableIds[i] = stepConfig.ReadableId.String
		workflowIds[i] = stepConfig.WorkflowId
		workflowVersionIds[i] = stepConfig.WorkflowVersionId
		scheduleTimeouts[i] = stepConfig.ScheduleTimeout
		stepTimeouts[i] = stepConfig.Timeout.String
		externalIds[i] = sqlchelpers.UUIDFromStr(task.ExternalId)
		displayNames[i] = fmt.Sprintf("%s-%d", stepConfig.ReadableId.String, unix)
		stepIndices[i] = int64(task.StepIndex)
		retryBackoffFactors[i] = stepConfig.RetryBackoffFactor
		retryMaxBackoffs[i] = stepConfig.RetryMaxBackoff
		workflowRunIds[i] = sqlchelpers.UUIDFromStr(task.WorkflowRunId)

		// TODO: case on whether this is a v1 or v2 task by looking at the step data. for now,
		// we're assuming a v1 task.
		inputs[i] = r.ToV1StepRunData(task.Input).Bytes()
		retryCounts[i] = 0
		priorities[i] = 1
		stickies[i] = string(sqlcv1.V1StickyStrategyNONE)

		if stepConfig.WorkflowVersionSticky.Valid {
			stickies[i] = string(stepConfig.WorkflowVersionSticky.StickyStrategy)
		}

		desiredWorkerIds[i] = pgtype.UUID{
			Valid: false,
		}

		if task.DesiredWorkerId != nil {
			desiredWorkerIds[i] = sqlchelpers.UUIDFromStr(*task.DesiredWorkerId)
		}

		initialStates[i] = string(task.InitialState)

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

		if task.ParentTaskExternalId != nil {
			parentTaskExternalIds[i] = sqlchelpers.UUIDFromStr(*task.ParentTaskExternalId)
		}

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

		// only check for concurrency if the task is in a queued state, otherwise we don't need to
		// evaluate the expression (and it will likely fail if we do)
		if task.InitialState == sqlcv1.V1TaskInitialStateQUEUED {
			// if we have a step expression, evaluate the expression
			if strats, ok := concurrencyStrats[task.StepId]; ok {
				taskConcurrencyKeys := make([]string, 0)
				taskStrategyIds := make([]int64, 0)
				var failTaskError error

				for _, strat := range strats {
					var additionalMeta map[string]interface{}

					if len(additionalMetadatas[i]) > 0 {
						if err := json.Unmarshal(additionalMetadatas[i], &additionalMeta); err != nil {
							failTaskError = fmt.Errorf("failed to process additional metadata: not a json object")
							break
						}
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
					taskStrategyIds = append(taskStrategyIds, strat.ID)
				}

				if failTaskError != nil {
					// place the task into a failed state
					initialStates[i] = string(sqlcv1.V1TaskInitialStateFAILED)

					initialStateReasons[i] = pgtype.Text{
						String: failTaskError.Error(),
						Valid:  true,
					}
				} else {
					concurrencyKeys[i] = taskConcurrencyKeys
					strategyIds[i] = taskStrategyIds
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

	res, err := r.queries.CreateTasks(ctx, tx, sqlcv1.CreateTasksParams{
		Tenantids:              tenantIds,
		Queues:                 queues,
		Actionids:              actionIds,
		Stepids:                stepIds,
		Stepreadableids:        stepReadableIds,
		Workflowids:            workflowIds,
		Scheduletimeouts:       scheduleTimeouts,
		Steptimeouts:           stepTimeouts,
		Priorities:             priorities,
		Stickies:               stickies,
		Desiredworkerids:       desiredWorkerIds,
		Externalids:            externalIds,
		Displaynames:           displayNames,
		Inputs:                 inputs,
		Retrycounts:            retryCounts,
		Additionalmetadatas:    additionalMetadatas,
		InitialStates:          initialStates,
		InitialStateReasons:    initialStateReasons,
		Dagids:                 dagIds,
		Daginsertedats:         dagInsertedAts,
		ConcurrencyStrategyIds: strategyIds,
		ConcurrencyKeys:        concurrencyKeys,
		ParentTaskExternalIds:  parentTaskExternalIds,
		ParentTaskIds:          parentTaskIds,
		ParentTaskInsertedAts:  parentTaskInsertedAts,
		ChildIndex:             childIndices,
		ChildKey:               childKeys,
		StepIndex:              stepIndices,
		RetryBackoffFactor:     retryBackoffFactors,
		RetryMaxBackoff:        retryMaxBackoffs,
		WorkflowVersionIds:     workflowVersionIds,
		WorkflowRunIds:         workflowRunIds,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create tasks: %w", err)
	}

	if len(createExpressionOpts) > 0 {
		err = r.createExpressionEvals(ctx, tx, res, createExpressionOpts)

		if err != nil {
			return nil, fmt.Errorf("failed to create expression evals: %w", err)
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
	tenantId string,
	tasks []ReplayTaskOpts,
) ([]*sqlcv1.V1Task, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

	uniqueStepIds := make(map[string]struct{})
	stepIds := make([]pgtype.UUID, 0)

	for _, task := range tasks {
		if _, ok := uniqueStepIds[task.StepId]; !ok {
			uniqueStepIds[task.StepId] = struct{}{}
			stepIds = append(stepIds, sqlchelpers.UUIDFromStr(task.StepId))
		}
	}

	steps, err := r.queries.ListStepsByIds(ctx, tx, sqlcv1.ListStepsByIdsParams{
		Ids:      stepIds,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	stepIdsToConfig := make(map[string]*sqlcv1.ListStepsByIdsRow)

	for _, step := range steps {
		stepIdsToConfig[sqlchelpers.UUIDToStr(step.ID)] = step
	}

	concurrencyStrats, err := r.getConcurrencyExpressions(ctx, tx, tenantId, stepIdsToConfig)

	if err != nil {
		return nil, fmt.Errorf("failed to get step expressions: %w", err)
	}

	taskIds := make([]int64, len(tasks))
	taskInsertedAts := make([]pgtype.Timestamptz, len(tasks))
	inputs := make([][]byte, len(tasks))
	initialStates := make([]string, len(tasks))
	initialStateReasons := make([]pgtype.Text, len(tasks))
	strategyIds := make([][]int64, len(tasks))
	concurrencyKeys := make([][]string, len(tasks))
	additionalMetadatas := make([][]byte, len(tasks))
	queues := make([]string, len(tasks))

	for i, task := range tasks {
		stepConfig := stepIdsToConfig[task.StepId]
		queues[i] = stepConfig.ActionId // FIXME: make the queue name dynamic

		taskIds[i] = task.TaskId
		taskInsertedAts[i] = task.InsertedAt

		// TODO: case on whether this is a v1 or v2 task by looking at the step data. for now,
		// we're assuming a v1 task.
		if task.Input != nil {
			inputs[i] = r.ToV1StepRunData(task.Input).Bytes()
		}
		initialStates[i] = string(task.InitialState)

		if len(task.AdditionalMetadata) > 0 {
			additionalMetadatas[i] = task.AdditionalMetadata
		}

		// only check for concurrency if the task is in a queued state, otherwise we don't need to
		// evaluate the expression (and it will likely fail if we do)
		if task.InitialState == sqlcv1.V1TaskInitialStateQUEUED {
			// if we have a step expression, evaluate the expression
			if strats, ok := concurrencyStrats[task.StepId]; ok {
				taskConcurrencyKeys := make([]string, 0)
				taskStrategyIds := make([]int64, 0)
				var failTaskError error

				for _, strat := range strats {
					var additionalMeta map[string]interface{}

					if len(additionalMetadatas[i]) > 0 {
						if err := json.Unmarshal(additionalMetadatas[i], &additionalMeta); err != nil {
							failTaskError = fmt.Errorf("failed to process additional metadata: not a json object")
							break
						}
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
					taskStrategyIds = append(taskStrategyIds, strat.ID)
				}

				if failTaskError != nil {
					// place the task into a failed state
					initialStates[i] = string(sqlcv1.V1TaskInitialStateFAILED)

					initialStateReasons[i] = pgtype.Text{
						String: failTaskError.Error(),
						Valid:  true,
					}
				} else {
					concurrencyKeys[i] = taskConcurrencyKeys
					strategyIds[i] = taskStrategyIds
				}
			}
		}
	}

	saveQueueCache, err := r.upsertQueues(ctx, tx, tenantId, queues)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert queues: %w", err)
	}

	res, err := r.queries.ReplayTasks(ctx, tx, sqlcv1.ReplayTasksParams{
		Taskids:                taskIds,
		Taskinsertedats:        taskInsertedAts,
		Inputs:                 inputs,
		InitialStates:          initialStates,
		InitialStateReasons:    initialStateReasons,
		Concurrencystrategyids: strategyIds,
		Concurrencykeys:        concurrencyKeys,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to replay tasks: %w", err)
	}

	// TODO: this should be moved to after the transaction commits
	saveQueueCache()

	return res, nil
}

func (r *sharedRepository) getConcurrencyExpressions(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId string,
	stepIdsToConfig map[string]*sqlcv1.ListStepsByIdsRow,
) (map[string][]*sqlcv1.V1StepConcurrency, error) {
	stepIdsWithExpressions := make(map[string]struct{})

	for _, step := range stepIdsToConfig {
		if step.ConcurrencyCount > 0 {
			stepIdsWithExpressions[sqlchelpers.UUIDToStr(step.ID)] = struct{}{}
		}
	}

	if len(stepIdsWithExpressions) == 0 {
		return nil, nil
	}

	stepIds := make([]pgtype.UUID, 0, len(stepIdsWithExpressions))

	for stepId := range stepIdsWithExpressions {
		stepIds = append(stepIds, sqlchelpers.UUIDFromStr(stepId))
	}

	strats, err := r.queries.ListConcurrencyStrategiesByStepId(ctx, tx, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Stepids:  stepIds,
	})

	if err != nil {
		return nil, err
	}

	stepIdToStrats := make(map[string][]*sqlcv1.V1StepConcurrency)

	for _, strat := range strats {
		stepId := sqlchelpers.UUIDToStr(strat.StepID)

		if _, ok := stepIdToStrats[stepId]; !ok {
			stepIdToStrats[stepId] = make([]*sqlcv1.V1StepConcurrency, 0)
		}

		stepIdToStrats[stepId] = append(stepIdToStrats[stepId], strat)
	}

	return stepIdToStrats, nil
}

func (r *sharedRepository) getStepExpressions(
	ctx context.Context,
	tx sqlcv1.DBTX,
	stepIdsToConfig map[string]*sqlcv1.ListStepsByIdsRow,
) (map[string][]*sqlcv1.StepExpression, error) {
	stepIdsWithExpressions := make(map[string]struct{})

	for _, step := range stepIdsToConfig {
		if step.ExprCount > 0 {
			stepIdsWithExpressions[sqlchelpers.UUIDToStr(step.ID)] = struct{}{}
		}
	}

	if len(stepIdsWithExpressions) == 0 {
		return map[string][]*sqlcv1.StepExpression{}, nil
	}

	stepIds := make([]pgtype.UUID, 0, len(stepIdsWithExpressions))

	for stepId := range stepIdsWithExpressions {
		stepIds = append(stepIds, sqlchelpers.UUIDFromStr(stepId))
	}

	expressions, err := r.queries.ListStepExpressions(ctx, tx, stepIds)

	if err != nil {
		return nil, err
	}

	stepIdToExpressions := make(map[string][]*sqlcv1.StepExpression)

	for _, expression := range expressions {
		stepId := sqlchelpers.UUIDToStr(expression.StepId)

		if _, ok := stepIdToExpressions[stepId]; !ok {
			stepIdToExpressions[stepId] = make([]*sqlcv1.StepExpression, 0)
		}

		stepIdToExpressions[stepId] = append(stepIdToExpressions[stepId], expression)
	}

	return stepIdToExpressions, nil
}

func (r *sharedRepository) createTaskEvents(
	ctx context.Context,
	dbtx sqlcv1.DBTX,
	tenantId string,
	tasks []TaskIdInsertedAtRetryCount,
	taskExternalIds []string,
	eventDatas [][]byte,
	eventType sqlcv1.V1TaskEventType,
	eventKeys []string,
) ([]InternalTaskEvent, error) {
	if len(tasks) != len(eventDatas) {
		return nil, fmt.Errorf("mismatched task and event data lengths")
	}

	taskIds := make([]int64, len(tasks))
	taskInsertedAts := make([]pgtype.Timestamptz, len(tasks))
	retryCounts := make([]int32, len(tasks))
	eventTypes := make([]string, len(tasks))
	paramDatas := make([][]byte, len(tasks))
	paramKeys := make([]pgtype.Text, len(tasks))

	internalTaskEvents := make([]InternalTaskEvent, len(tasks))

	for i, task := range tasks {
		taskIds[i] = task.Id
		taskInsertedAts[i] = task.InsertedAt
		retryCounts[i] = task.RetryCount
		eventTypes[i] = string(eventType)

		if len(eventDatas[i]) == 0 {
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
			EventType:      eventType,
			EventKey:       eventKeys[i],
			Data:           eventDatas[i],
		}
	}

	err := r.queries.CreateTaskEvents(ctx, dbtx, sqlcv1.CreateTaskEventsParams{
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Retrycounts:     retryCounts,
		Eventtypes:      eventTypes,
		Datas:           paramDatas,
		Eventkeys:       paramKeys,
	})

	if err != nil {
		return nil, err
	}

	return internalTaskEvents, nil
}

func (r *TaskRepositoryImpl) ReplayTasks(ctx context.Context, tenantId string, tasks []TaskIdInsertedAtRetryCount) (*ReplayTasksResult, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

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
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	lockedTaskIds := make([]int64, len(lockedTasks))
	subtreeStepIds := make(map[int64]map[string]bool) // dag id -> step id -> true
	subtreeExternalIds := make(map[string]struct{})
	dagIdsToLockMap := make(map[int64]struct{})

	for i, task := range lockedTasks {
		lockedTaskIds[i] = task.ID

		if task.DagID.Valid {
			if _, ok := subtreeStepIds[task.DagID.Int64]; !ok {
				subtreeStepIds[task.DagID.Int64] = make(map[string]bool)
			}

			dagIdsToLockMap[task.DagID.Int64] = struct{}{}
			subtreeStepIds[task.DagID.Int64][sqlchelpers.UUIDToStr(task.StepID)] = true
			subtreeExternalIds[sqlchelpers.UUIDToStr(task.ExternalID)] = struct{}{}
		}
	}

	// lock all tasks in the DAGs
	dagIdsToLock := make([]int64, 0, len(dagIdsToLockMap))

	for dagId := range dagIdsToLockMap {
		dagIdsToLock = append(dagIdsToLock, dagId)
	}

	successfullyLockedDAGIds, err := r.queries.LockDAGsForReplay(ctx, tx, sqlcv1.LockDAGsForReplayParams{
		Dagids:   dagIdsToLock,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
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
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	for _, dag := range preflightDAGs {
		if dag.StepCount != dag.TaskCount {
			dagIdsFailedPreflight[dag.ID] = true
		}
	}

	tasksFailedPreflight := make(map[int64]bool)

	failedPreflightChecks, err := r.queries.PreflightCheckTasksForReplay(ctx, tx, sqlcv1.PreflightCheckTasksForReplayParams{
		Taskids:  lockedTaskIds,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
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
				if subtreeStepIds[task.DagID.Int64][sqlchelpers.UUIDToStr(parent)] {
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

		replayOpts = append(replayOpts, ReplayTaskOpts{
			TaskId:             task.ID,
			InsertedAt:         task.InsertedAt,
			StepId:             sqlchelpers.UUIDToStr(task.StepID),
			ExternalId:         sqlchelpers.UUIDToStr(task.ExternalID),
			InitialState:       sqlcv1.V1TaskInitialStateQUEUED,
			AdditionalMetadata: task.AdditionalMetadata,
		})
	}

	dagIdsArr := make([]int64, 0, len(dagIds))

	for dagId := range dagIds {
		dagIdsArr = append(dagIdsArr, dagId)
	}

	allTasksInDAGs, err := r.queries.ListAllTasksInDags(ctx, tx, sqlcv1.ListAllTasksInDagsParams{
		Dagids:   dagIdsArr,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	dagIdsToAllTasks := make(map[int64][]*sqlcv1.ListAllTasksInDagsRow)

	for _, task := range allTasksInDAGs {
		if _, ok := dagIdsToAllTasks[task.DagID.Int64]; !ok {
			dagIdsToAllTasks[task.DagID.Int64] = make([]*sqlcv1.ListAllTasksInDagsRow, 0)
		}

		dagIdsToAllTasks[task.DagID.Int64] = append(dagIdsToAllTasks[task.DagID.Int64], task)
	}

	queuedTasks := make([]*sqlcv1.V1Task, 0)

	// NOTE: the tasks which are passed in represent a *subtree* of the DAG.
	if len(replayOpts) > 0 {
		queuedTasks, err = r.replayTasks(ctx, tx, tenantId, replayOpts)

		if err != nil {
			return nil, err
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

			parentExternalId := sqlchelpers.UUIDToStr(task.ParentTaskExternalID)
			k := getChildSignalEventKey(parentExternalId, task.StepIndex, task.ChildIndex.Int64, childKey)

			signalEventKeys = append(signalEventKeys, k)
			parentTaskIds = append(parentTaskIds, task.ParentTaskID.Int64)
			parentTaskInsertedAts = append(parentTaskInsertedAts, task.ParentTaskInsertedAt)

			eventMatches = append(eventMatches, CreateMatchOpts{
				Kind:                 sqlcv1.V1MatchKindSIGNAL,
				Conditions:           getChildWorkflowGroupMatches(sqlchelpers.UUIDToStr(task.ExternalID), task.StepReadableID),
				SignalExternalId:     &parentExternalId,
				SignalTaskId:         &task.ParentTaskID.Int64,
				SignalTaskInsertedAt: task.ParentTaskInsertedAt,
				SignalKey:            &k,
			})
		}

		err = r.queries.DeleteMatchingSignalEvents(ctx, tx, sqlcv1.DeleteMatchingSignalEventsParams{
			Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
			Eventkeys:       signalEventKeys,
			Taskids:         parentTaskIds,
			Taskinsertedats: parentTaskInsertedAts,
			Eventtype:       sqlcv1.V1TaskEventTypeSIGNALCOMPLETED,
		})

		if err != nil {
			return nil, err
		}
	}

	// For any DAGs, reset all match conditions which refer to internal events within the subtree of the DAG.
	// we do not reset other match conditions (for example, ones which refer to completed events for tasks
	// which are outside of this subtree). otherwise, we would end up in a state where these events would
	// never be matched.
	for dagId, tasks := range dagIdsToChildTasks {
		allTasks := dagIdsToAllTasks[dagId]

		for _, task := range tasks {
			taskExternalId := sqlchelpers.UUIDToStr(task.ExternalID)
			stepId := sqlchelpers.UUIDToStr(task.StepID)
			switch {
			case task.JobKind == sqlcv1.JobKindONFAILURE:
				conditions := make([]GroupMatchCondition, 0)
				groupId := uuid.NewString()

				for _, otherTask := range allTasks {
					if sqlchelpers.UUIDToStr(otherTask.StepID) == stepId {
						continue
					}

					otherExternalId := sqlchelpers.UUIDToStr(otherTask.ExternalID)
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

				cancelGroupId := uuid.NewString()

				for _, parent := range task.Parents {
					// FIXME: n^2 complexity here, fix it.
					for _, otherTask := range allTasks {
						if otherTask.StepID == parent {
							parentExternalId := sqlchelpers.UUIDToStr(otherTask.ExternalID)
							readableId := otherTask.StepReadableID

							conditions = append(conditions, getParentInDAGGroupMatch(cancelGroupId, parentExternalId, readableId)...)
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
		return nil, err
	}

	// create the event matches
	err = r.createEventMatches(ctx, tx, tenantId, reconstructedMatches)

	if err != nil {
		return nil, err
	}

	// process event matches
	// TODO: signal the event matches to the caller
	internalMatchResults, err := r.processInternalEventMatches(ctx, tx, tenantId, candidateEvents)

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return &ReplayTasksResult{
		ReplayedTasks:        replayedTasks,
		QueuedTasks:          queuedTasks,
		InternalEventResults: internalMatchResults,
	}, nil
}

func (r *TaskRepositoryImpl) reconstructGroupConditions(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId string,
	subtreeExternalIds map[string]struct{},
	eventMatches []CreateMatchOpts,
) ([]CreateMatchOpts, []CandidateEventMatch, error) {
	// track down completed tasks and failed tasks which represent parents that aren't in the subtree
	// of the DAG. for these tasks, we need to write the match conditions which refer to these tasks
	// as satisfied match conditions.
	// in other words, if the group match condition is an INTERNAL event and refers to a parentExternalId
	// which is NOT in the subtree of what we're replaying, it represent a group condition where we'd like
	// to query the task_events table to ensure the event has already occurred. if it has, we can mark the
	// group condition as satisfied.
	externalIds := make([]pgtype.UUID, 0)
	eventTypes := make([][]string, 0)

	for _, match := range eventMatches {
		if match.TriggerDAGId == nil {
			continue
		}

		for _, groupCondition := range match.Conditions {
			if groupCondition.EventType == sqlcv1.V1EventTypeINTERNAL && groupCondition.EventResourceHint != nil {
				externalId := *groupCondition.EventResourceHint

				// if the parent task is not in the subtree, we need to query the task_events table
				// to ensure the event has already occurred
				if _, ok := subtreeExternalIds[externalId]; !ok {
					externalIds = append(externalIds, sqlchelpers.UUIDFromStr(*groupCondition.EventResourceHint))
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
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
		Taskexternalids: externalIds,
		Eventtypes:      eventTypes,
	})

	if err != nil {
		return nil, nil, err
	}

	foundMatchKeys := make(map[string]*sqlcv1.ListMatchingTaskEventsRow)

	for _, eventMatch := range matchedEvents {
		key := fmt.Sprintf("%s:%s", sqlchelpers.UUIDToStr(eventMatch.ExternalID), string(eventMatch.EventType))

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

					taskExternalId := sqlchelpers.UUIDToStr(match.ExternalID)

					resCandidateEvents = append(resCandidateEvents, CandidateEventMatch{
						ID:             uuid.NewString(),
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

func (r *sharedRepository) createExpressionEvals(ctx context.Context, dbtx sqlcv1.DBTX, createdTasks []*sqlcv1.V1Task, opts map[string][]createTaskExpressionEvalOpt) error {
	if len(opts) == 0 {
		return nil
	}

	// map tasks using their external id
	taskExternalIds := make(map[string]*sqlcv1.V1Task)

	for _, task := range createdTasks {
		taskExternalIds[sqlchelpers.UUIDToStr(task.ExternalID)] = task
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
			r.l.Warn().Str("external_id", externalId).Msg("could not find task for expression eval")
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
