package v2

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
)

const MAX_INTERNAL_RETRIES = 3

type CreateTaskOpts struct {
	// (required) the external id
	ExternalId string `validate:"required,uuid"`

	// (required) the step id
	StepId string `validate:"required,uuid"`

	// (required) the input bytes to the task
	Input *TaskInput

	// (optional) the additional metadata for the task
	AdditionalMetadata []byte

	// (optional) the DAG id for the task
	DagId *int64

	// (optional) the DAG inserted at for the task
	DagInsertedAt pgtype.Timestamptz
}

type TaskIdRetryCount struct {
	// (required) the external id
	Id int64 `validate:"required"`

	// (required) the retry count
	RetryCount int32
}

type FailTaskOpts struct {
	*TaskIdRetryCount

	// (required) whether this is an application-level error or an internal error on the Hatchet side
	IsAppError bool
}

type TaskRepository interface {
	UpdateTablePartitions(ctx context.Context) error

	CompleteTasks(ctx context.Context, tenantId string, tasks []TaskIdRetryCount) ([]*sqlcv2.ReleaseTasksRow, error)

	FailTasks(ctx context.Context, tenantId string, tasks []FailTaskOpts) (retriedTasks []TaskIdRetryCount, queues []*sqlcv2.ReleaseTasksRow, err error)

	CancelTasks(ctx context.Context, tenantId string, tasks []TaskIdRetryCount) ([]*sqlcv2.ReleaseTasksRow, error)

	ListTasks(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv2.V2Task, error)

	ListTaskMetas(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv2.ListTaskMetasRow, error)

	ProcessTaskTimeouts(ctx context.Context, tenantId string) ([]*sqlcv2.ProcessTaskTimeoutsRow, bool, error)

	ProcessTaskReassignments(ctx context.Context, tenantId string) ([]*sqlcv2.ProcessTaskReassignmentsRow, bool, error)
}

type TaskRepositoryImpl struct {
	*sharedRepository
}

func newTaskRepository(s *sharedRepository) TaskRepository {
	return &TaskRepositoryImpl{
		sharedRepository: s,
	}
}

func (r *TaskRepositoryImpl) UpdateTablePartitions(ctx context.Context) error {
	today := time.Now().UTC()
	tomorrow := today.AddDate(0, 0, 1)
	sevenDaysAgo := today.AddDate(0, 0, -7)

	err := r.queries.CreateTaskPartition(ctx, r.pool, pgtype.Date{
		Time:  today,
		Valid: true,
	})

	if err != nil {
		return err
	}

	err = r.queries.CreateTaskPartition(ctx, r.pool, pgtype.Date{
		Time:  tomorrow,
		Valid: true,
	})

	if err != nil {
		return err
	}

	partitions, err := r.queries.ListTaskPartitionsBeforeDate(ctx, r.pool, pgtype.Date{
		Time:  sevenDaysAgo,
		Valid: true,
	})

	if err != nil {
		return err
	}

	for _, partition := range partitions {
		_, err := r.pool.Exec(
			ctx,
			fmt.Sprintf("ALTER TABLE v2_task DETACH PARTITION %s CONCURRENTLY", partition),
		)

		if err != nil {
			return err
		}

		_, err = r.pool.Exec(
			ctx,
			fmt.Sprintf("DROP TABLE %s", partition),
		)

		if err != nil {
			return err
		}
	}

	err = r.queries.CreateDAGPartition(ctx, r.pool, pgtype.Date{
		Time:  today,
		Valid: true,
	})

	if err != nil {
		return err
	}

	err = r.queries.CreateDAGPartition(ctx, r.pool, pgtype.Date{
		Time:  tomorrow,
		Valid: true,
	})

	if err != nil {
		return err
	}

	dagPartitions, err := r.queries.ListDAGPartitionsBeforeDate(ctx, r.pool, pgtype.Date{
		Time:  sevenDaysAgo,
		Valid: true,
	})

	if err != nil {
		return err
	}

	for _, partition := range dagPartitions {
		_, err := r.pool.Exec(
			ctx,
			fmt.Sprintf("ALTER TABLE v2_task DETACH PARTITION %s CONCURRENTLY", partition),
		)

		if err != nil {
			return err
		}

		_, err = r.pool.Exec(
			ctx,
			fmt.Sprintf("DROP TABLE %s", partition),
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *TaskRepositoryImpl) CompleteTasks(ctx context.Context, tenantId string, tasks []TaskIdRetryCount) ([]*sqlcv2.ReleaseTasksRow, error) {
	// TODO: ADD BACK VALIDATION
	// if err := r.v.Validate(tasks); err != nil {
	// 	fmt.Println("FAILED VALIDATION HERE!!!")

	// 	return err
	// }

	datas := make([][]byte, len(tasks))

	for i := range tasks {
		datas[i] = nil
	}

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

	err = r.createTaskEvents(
		ctx,
		tx,
		tenantId,
		tasks,
		datas,
		sqlcv2.V2TaskEventTypeCOMPLETED,
	)

	if err != nil {
		return nil, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return nil, err
	}

	return releasedTasks, nil
}

func (r *TaskRepositoryImpl) FailTasks(ctx context.Context, tenantId string, failureOpts []FailTaskOpts) ([]TaskIdRetryCount, []*sqlcv2.ReleaseTasksRow, error) {
	// TODO: ADD BACK VALIDATION
	// if err := r.v.Validate(tasks); err != nil {
	// 	fmt.Println("FAILED VALIDATION HERE!!!")

	// 	return err
	// }

	tasks := make([]TaskIdRetryCount, len(failureOpts))
	appFailures := make([]int64, 0)
	internalFailures := make([]int64, 0)
	datas := make([][]byte, len(failureOpts))

	for i, failureOpt := range failureOpts {
		tasks[i] = *failureOpt.TaskIdRetryCount
		datas[i] = nil

		if failureOpt.IsAppError {
			appFailures = append(appFailures, failureOpt.Id)
		} else {
			internalFailures = append(internalFailures, failureOpt.Id)
		}
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	// release queue items
	releasedTasks, err := r.releaseTasks(ctx, tx, tenantId, tasks)

	if err != nil {
		return nil, nil, err
	}

	retriedTasks := make([]TaskIdRetryCount, 0)

	// write app failures
	if len(appFailures) > 0 {
		appFailureRetries, err := r.queries.FailTaskAppFailure(ctx, tx, sqlcv2.FailTaskAppFailureParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			Taskids:  appFailures,
		})

		if err != nil {
			return nil, nil, err
		}

		for _, task := range appFailureRetries {
			retriedTasks = append(retriedTasks, TaskIdRetryCount{
				Id:         task.ID,
				RetryCount: task.RetryCount,
			})
		}
	}

	// write internal failures
	if len(internalFailures) > 0 {
		internalFailureRetries, err := r.queries.FailTaskInternalFailure(ctx, tx, sqlcv2.FailTaskInternalFailureParams{
			Tenantid:           sqlchelpers.UUIDFromStr(tenantId),
			Taskids:            internalFailures,
			Maxinternalretries: MAX_INTERNAL_RETRIES,
		})

		if err != nil {
			return nil, nil, err
		}

		for _, task := range internalFailureRetries {
			retriedTasks = append(retriedTasks, TaskIdRetryCount{
				Id:         task.ID,
				RetryCount: task.RetryCount,
			})
		}
	}

	// write task events
	err = r.createTaskEvents(
		ctx,
		tx,
		tenantId,
		tasks,
		datas,
		sqlcv2.V2TaskEventTypeFAILED,
	)

	if err != nil {
		return nil, nil, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return nil, nil, err
	}

	return retriedTasks, releasedTasks, nil
}

func (r *TaskRepositoryImpl) CancelTasks(ctx context.Context, tenantId string, tasks []TaskIdRetryCount) ([]*sqlcv2.ReleaseTasksRow, error) {
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
	releasedTasks, err := r.releaseTasks(ctx, tx, tenantId, tasks)

	if err != nil {
		return nil, err
	}

	// write task events
	err = r.createTaskEvents(
		ctx,
		tx,
		tenantId,
		tasks,
		make([][]byte, len(tasks)),
		sqlcv2.V2TaskEventTypeCANCELLED,
	)

	if err != nil {
		return nil, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return nil, err
	}

	return releasedTasks, nil
}

func (r *TaskRepositoryImpl) ListTasks(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv2.V2Task, error) {
	return r.queries.ListTasks(ctx, r.pool, sqlcv2.ListTasksParams{
		TenantID: sqlchelpers.UUIDFromStr(tenantId),
		Ids:      tasks,
	})
}

func (r *TaskRepositoryImpl) ListTaskMetas(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv2.ListTaskMetasRow, error) {
	return r.queries.ListTaskMetas(ctx, r.pool, sqlcv2.ListTaskMetasParams{
		TenantID: sqlchelpers.UUIDFromStr(tenantId),
		Ids:      tasks,
	})
}

func (r *TaskRepositoryImpl) ProcessTaskTimeouts(ctx context.Context, tenantId string) ([]*sqlcv2.ProcessTaskTimeoutsRow, bool, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, false, err
	}

	defer rollback()

	// TODO: make limit configurable
	limit := 1000

	// get task timeouts
	res, err := r.queries.ProcessTaskTimeouts(ctx, tx, sqlcv2.ProcessTaskTimeoutsParams{
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

func (r *TaskRepositoryImpl) ProcessTaskReassignments(ctx context.Context, tenantId string) ([]*sqlcv2.ProcessTaskReassignmentsRow, bool, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, false, err
	}

	defer rollback()

	// TODO: make limit configurable
	limit := 1000

	// get task reassignments
	res, err := r.queries.ProcessTaskReassignments(ctx, tx, sqlcv2.ProcessTaskReassignmentsParams{
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
	failedTasks := make([]TaskIdRetryCount, 0)

	for _, task := range res {
		if task.Operation == "FAILED" {
			failedTasks = append(failedTasks, TaskIdRetryCount{
				Id:         task.ID,
				RetryCount: task.RetryCount,
			})
		}
	}

	failedTaskDatas := make([][]byte, len(failedTasks))

	if len(failedTasks) > 0 {
		err = r.createTaskEvents(
			ctx,
			tx,
			tenantId,
			failedTasks,
			failedTaskDatas,
			sqlcv2.V2TaskEventTypeFAILED,
		)

		if err != nil {
			return nil, false, err
		}
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return nil, false, err
	}

	return res, len(res) == limit, nil
}

func (r *TaskRepositoryImpl) releaseTasks(ctx context.Context, tx sqlcv2.DBTX, tenantId string, tasks []TaskIdRetryCount) ([]*sqlcv2.ReleaseTasksRow, error) {
	taskIds := make([]int64, len(tasks))
	retryCounts := make([]int32, len(tasks))

	for i, task := range tasks {
		taskIds[i] = task.Id
		retryCounts[i] = task.RetryCount
	}

	return r.queries.ReleaseTasks(ctx, tx, sqlcv2.ReleaseTasksParams{
		Taskids:     taskIds,
		Retrycounts: retryCounts,
	})
}

func (r *sharedRepository) upsertQueues(ctx context.Context, tx sqlcv2.DBTX, tenantId string, queues []string) error {
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

	err := r.queries.UpsertQueues(ctx, tx, sqlcv2.UpsertQueuesParams{
		TenantID: sqlchelpers.UUIDFromStr(tenantId),
		Names:    uniqueQueues,
	})

	if err != nil {
		return err
	}

	// set all the queues to true in the cache
	for _, queue := range uniqueQueues {
		key := getQueueCacheKey(tenantId, queue)
		r.queueCache.Set(key, true)
	}

	return nil
}

func getQueueCacheKey(tenantId string, queue string) string {
	return fmt.Sprintf("%s:%s", tenantId, queue)
}

func (r *sharedRepository) createTasks(
	ctx context.Context,
	tx sqlcv2.DBTX,
	tenantId string,
	tasks []CreateTaskOpts,
) ([]*sqlcv2.V2Task, error) {
	// list the steps for the tasks
	uniqueStepIds := make(map[string]struct{})
	stepIds := make([]pgtype.UUID, 0)

	for _, task := range tasks {
		if _, ok := uniqueStepIds[task.StepId]; !ok {
			uniqueStepIds[task.StepId] = struct{}{}
			stepIds = append(stepIds, sqlchelpers.UUIDFromStr(task.StepId))
		}
	}

	steps, err := r.queries.ListStepsByIds(ctx, tx, sqlcv2.ListStepsByIdsParams{
		Ids:      stepIds,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	stepIdsToConfig := make(map[string]*sqlcv2.ListStepsByIdsRow)

	for _, step := range steps {
		stepIdsToConfig[sqlchelpers.UUIDToStr(step.ID)] = step
	}

	return r.insertTasks(ctx, tx, tenantId, tasks, stepIdsToConfig)
}

// insertTasks inserts new tasks into the database. note that we're using Postgres rules to automatically insert the created
// tasks into the queue_items table.
func (r *sharedRepository) insertTasks(
	ctx context.Context,
	tx sqlcv2.DBTX,
	tenantId string,
	tasks []CreateTaskOpts,
	stepIdsToConfig map[string]*sqlcv2.ListStepsByIdsRow,
) ([]*sqlcv2.V2Task, error) {
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
	dagIds := make([]pgtype.Int8, len(tasks))
	dagInsertedAts := make([]pgtype.Timestamptz, len(tasks))
	unix := time.Now().UnixMilli()

	for i, task := range tasks {
		stepConfig := stepIdsToConfig[task.StepId]
		tenantIds[i] = sqlchelpers.UUIDFromStr(tenantId)
		queues[i] = stepConfig.ActionId // FIXME: make the queue name dynamic
		actionIds[i] = stepConfig.ActionId
		stepIds[i] = sqlchelpers.UUIDFromStr(task.StepId)
		stepReadableIds[i] = stepConfig.ReadableId.String
		workflowIds[i] = stepConfig.WorkflowId
		scheduleTimeouts[i] = stepConfig.ScheduleTimeout
		stepTimeouts[i] = stepConfig.Timeout.String
		externalIds[i] = sqlchelpers.UUIDFromStr(task.ExternalId)
		displayNames[i] = fmt.Sprintf("%s-%d", stepConfig.ReadableId.String, unix)

		// TODO: case on whether this is a v1 or v2 task by looking at the step data. for now,
		// we're assuming a v1 task.
		inputs[i] = r.ToV1StepRunData(task.Input).Bytes()
		retryCounts[i] = 0
		priorities[i] = 1
		stickies[i] = string(sqlcv2.V2StickyStrategyNONE)
		desiredWorkerIds[i] = pgtype.UUID{
			Valid: false,
		}

		if task.AdditionalMetadata != nil {
			additionalMetadatas[i] = task.AdditionalMetadata
		}

		if task.DagId != nil && task.DagInsertedAt.Valid {
			dagIds[i] = pgtype.Int8{
				Int64: *task.DagId,
				Valid: true,
			}

			dagInsertedAts[i] = task.DagInsertedAt
		}
	}

	err := r.upsertQueues(ctx, tx, tenantId, queues)

	if err != nil {
		return nil, err
	}

	res, err := r.queries.CreateTasks(ctx, tx, sqlcv2.CreateTasksParams{
		Tenantids:           tenantIds,
		Queues:              queues,
		Actionids:           actionIds,
		Stepids:             stepIds,
		Stepreadableids:     stepReadableIds,
		Workflowids:         workflowIds,
		Scheduletimeouts:    scheduleTimeouts,
		Steptimeouts:        stepTimeouts,
		Priorities:          priorities,
		Stickies:            stickies,
		Desiredworkerids:    desiredWorkerIds,
		Externalids:         externalIds,
		Displaynames:        displayNames,
		Inputs:              inputs,
		Retrycounts:         retryCounts,
		Additionalmetadatas: additionalMetadatas,
		Dagids:              dagIds,
		Daginsertedats:      dagInsertedAts,
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (r *TaskRepositoryImpl) createTaskEvents(
	ctx context.Context,
	dbtx sqlcv2.DBTX,
	tenantId string,
	tasks []TaskIdRetryCount,
	eventDatas [][]byte,
	eventType sqlcv2.V2TaskEventType,
) error {
	if len(tasks) != len(eventDatas) {
		return fmt.Errorf("mismatched task and event data lengths")
	}

	taskIds := make([]int64, len(tasks))
	retryCounts := make([]int32, len(tasks))
	eventTypes := make([]string, len(tasks))
	paramDatas := make([][]byte, len(tasks))

	for i, task := range tasks {
		taskIds[i] = task.Id
		retryCounts[i] = task.RetryCount
		eventTypes[i] = string(eventType)

		if len(eventDatas[i]) == 0 {
			paramDatas[i] = nil
		} else {
			paramDatas[i] = paramDatas[i]
		}
	}

	return r.queries.CreateTaskEvents(ctx, dbtx, sqlcv2.CreateTaskEventsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Taskids:     taskIds,
		Retrycounts: retryCounts,
		Eventtypes:  eventTypes,
		Datas:       paramDatas,
	})
}
