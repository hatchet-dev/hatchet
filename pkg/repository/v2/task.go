package v2

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
)

const MAX_INTERNAL_RETRIES = 3

type CreateTaskOpts struct {
	// (required) the external id
	ExternalId string `validate:"required,uuid"`

	// (required) the queue
	Queue string

	// (required) the action id
	ActionId string `validate:"required,actionId"`

	// (required) the step id
	StepId string `validate:"required,uuid"`

	// (required) the workflow id
	WorkflowId string `validate:"required,uuid"`

	// (required) the schedule timeout
	ScheduleTimeout string `validate:"required,duration"`

	// (required) the step timeout
	StepTimeout string `validate:"required,duration"`

	// (required) the task display name
	DisplayName string

	// (required) the input bytes to the task
	Input []byte

	// (optional) the additional metadata for the task
	AdditionalMetadata map[string]interface{}

	// (optional) the priority of the task
	Priority *int

	// (optional) the sticky strategy
	// TODO: validation
	StickyStrategy *string

	// (optional) the desired worker id
	DesiredWorkerId *string
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

	CreateTasks(ctx context.Context, tenantId string, tasks []CreateTaskOpts) ([]*sqlcv2.V2Task, error)

	CompleteTasks(ctx context.Context, tenantId string, tasks []TaskIdRetryCount) ([]string, error)

	FailTasks(ctx context.Context, tenantId string, tasks []FailTaskOpts) (retriedTasks []TaskIdRetryCount, queues []string, err error)

	CancelTasks(ctx context.Context, tenantId string, tasks []TaskIdRetryCount) ([]string, error)

	ListTasks(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv2.V2Task, error)

	ListTaskMetas(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv2.ListTaskMetasRow, error)

	ProcessTaskTimeouts(ctx context.Context, tenantId string) ([]*sqlcv2.ProcessTaskTimeoutsRow, bool, error)

	ProcessTaskReassignments(ctx context.Context, tenantId string) ([]*sqlcv2.ProcessTaskReassignmentsRow, bool, error)
}

type TaskRepositoryImpl struct {
	*sharedRepository

	cache *cache.Cache
}

func newTaskRepository(s *sharedRepository) TaskRepository {
	cache := cache.New(5 * time.Minute)

	return &TaskRepositoryImpl{
		sharedRepository: s,
		cache:            cache,
	}
}

func (r *TaskRepositoryImpl) UpdateTablePartitions(ctx context.Context) error {
	today := time.Now().UTC()
	tomorrow := today.AddDate(0, 0, 1)
	sevenDaysAgo := today.AddDate(0, 0, -7)

	err := r.queries.CreateTablePartition(ctx, r.pool, pgtype.Date{
		Time:  today,
		Valid: true,
	})

	if err != nil {
		return err
	}

	err = r.queries.CreateTablePartition(ctx, r.pool, pgtype.Date{
		Time:  tomorrow,
		Valid: true,
	})

	if err != nil {
		return err
	}

	partitions, err := r.queries.ListTablePartitionsBeforeDate(ctx, r.pool, pgtype.Date{
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

	return nil
}

func (r *TaskRepositoryImpl) CreateTasks(ctx context.Context, tenantId string, tasks []CreateTaskOpts) ([]*sqlcv2.V2Task, error) {
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

	// do a cached upsert on the queue
	queueNames := make(map[string]struct{})

	for _, task := range tasks {
		queueNames[task.Queue] = struct{}{}
	}

	queues := make([]string, 0, len(queueNames))

	for queue := range queueNames {
		queues = append(queues, queue)
	}

	if err := r.upsertQueues(ctx, tx, tenantId, queues); err != nil {
		return nil, err
	}

	res, err := r.createTasks(ctx, tx, tenantId, tasks)

	if err != nil {
		return nil, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return nil, err
	}

	// TODO: push task created events to the OLAP repository

	return res, nil
}

func (r *TaskRepositoryImpl) CompleteTasks(ctx context.Context, tenantId string, tasks []TaskIdRetryCount) ([]string, error) {
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
	queues, err := r.releaseTasks(ctx, tx, tenantId, tasks)

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

	return queues, nil
}

func (r *TaskRepositoryImpl) FailTasks(ctx context.Context, tenantId string, failureOpts []FailTaskOpts) ([]TaskIdRetryCount, []string, error) {
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
	queues, err := r.releaseTasks(ctx, tx, tenantId, tasks)

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

	return retriedTasks, queues, nil
}

func (r *TaskRepositoryImpl) CancelTasks(ctx context.Context, tenantId string, tasks []TaskIdRetryCount) ([]string, error) {
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
	queues, err := r.releaseTasks(ctx, tx, tenantId, tasks)

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

	return queues, nil
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

func (r *TaskRepositoryImpl) releaseTasks(ctx context.Context, tx sqlcv2.DBTX, tenantId string, tasks []TaskIdRetryCount) ([]string, error) {
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

func (r *TaskRepositoryImpl) upsertQueues(ctx context.Context, tx sqlcv2.DBTX, tenantId string, queues []string) error {
	queuesToInsert := make([]string, 0)

	for _, queue := range queues {
		key := getQueueCacheKey(tenantId, queue)

		if hasSetQueue, ok := r.cache.Get(key); ok && hasSetQueue.(bool) {
			continue
		}

		queuesToInsert = append(queuesToInsert, queue)
	}

	err := r.queries.UpsertQueues(ctx, tx, sqlcv2.UpsertQueuesParams{
		TenantID: sqlchelpers.UUIDFromStr(tenantId),
		Names:    queuesToInsert,
	})

	if err != nil {
		return err
	}

	// set all the queues to true in the cache
	for _, queue := range queuesToInsert {
		key := getQueueCacheKey(tenantId, queue)
		r.cache.Set(key, true)
	}

	return nil
}

func getQueueCacheKey(tenantId string, queue string) string {
	return fmt.Sprintf("%s:%s", tenantId, queue)
}

// createTasks inserts new tasks into the database. note that we're using Postgres rules to automatically insert the created
// tasks into the queue_items table.
func (r *TaskRepositoryImpl) createTasks(ctx context.Context, tx sqlcv2.DBTX, tenantId string, tasks []CreateTaskOpts) ([]*sqlcv2.V2Task, error) {
	tenantIds := make([]pgtype.UUID, len(tasks))
	queues := make([]string, len(tasks))
	actionIds := make([]string, len(tasks))
	stepIds := make([]pgtype.UUID, len(tasks))
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

	for i, task := range tasks {
		tenantIds[i] = sqlchelpers.UUIDFromStr(tenantId)
		queues[i] = task.Queue
		actionIds[i] = task.ActionId
		stepIds[i] = sqlchelpers.UUIDFromStr(task.StepId)
		workflowIds[i] = sqlchelpers.UUIDFromStr(task.WorkflowId)
		scheduleTimeouts[i] = task.ScheduleTimeout
		stepTimeouts[i] = task.StepTimeout
		externalIds[i] = sqlchelpers.UUIDFromStr(task.ExternalId)
		displayNames[i] = task.DisplayName
		inputs[i] = task.Input
		retryCounts[i] = 0

		if task.Priority != nil {
			priorities[i] = int32(*task.Priority)
		} else {
			priorities[i] = 1
		}

		if task.StickyStrategy != nil {
			stickies[i] = *task.StickyStrategy
		} else {
			stickies[i] = string(sqlcv2.V2StickyStrategyNONE)
		}

		if task.DesiredWorkerId != nil {
			desiredWorkerIds[i] = sqlchelpers.UUIDFromStr(*task.DesiredWorkerId)
		} else {
			desiredWorkerIds[i] = pgtype.UUID{
				Valid: false,
			}
		}
	}

	res, err := r.queries.CreateTasks(ctx, tx, sqlcv2.CreateTasksParams{
		Tenantids:        tenantIds,
		Queues:           queues,
		Actionids:        actionIds,
		Stepids:          stepIds,
		Workflowids:      workflowIds,
		Scheduletimeouts: scheduleTimeouts,
		Steptimeouts:     stepTimeouts,
		Priorities:       priorities,
		Stickies:         stickies,
		Desiredworkerids: desiredWorkerIds,
		Externalids:      externalIds,
		Displaynames:     displayNames,
		Inputs:           inputs,
		Retrycounts:      retryCounts,
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
