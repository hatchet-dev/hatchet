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
	CreateTasks(ctx context.Context, tenantId string, tasks []CreateTaskOpts) error

	CompleteTasks(ctx context.Context, tenantId string, tasks []TaskIdRetryCount) error

	FailTasks(ctx context.Context, tenantId string, tasks []FailTaskOpts) error

	ListTasks(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv2.V2Task, error)

	ListTaskMetas(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv2.ListTaskMetasRow, error)
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

func (r *TaskRepositoryImpl) CreateTasks(ctx context.Context, tenantId string, tasks []CreateTaskOpts) error {
	// TODO: ADD BACK VALIDATION
	// if err := r.v.Validate(tasks); err != nil {
	// 	fmt.Println("FAILED VALIDATION HERE!!!")

	// 	return err
	// }

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return err
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
		return err
	}

	if err := r.createTasks(ctx, tx, tenantId, tasks); err != nil {
		return err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return err
	}

	// TODO: push task created events to the OLAP repository

	return nil
}

func (r *TaskRepositoryImpl) CompleteTasks(ctx context.Context, tenantId string, tasks []TaskIdRetryCount) error {
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
		return err
	}

	defer rollback()

	// release queue items
	err = r.releaseQueueItems(ctx, tx, tenantId, tasks)

	if err != nil {
		return err
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
		return err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return err
	}

	return nil
}

func (r *TaskRepositoryImpl) FailTasks(ctx context.Context, tenantId string, failureOpts []FailTaskOpts) error {
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
		return err
	}

	defer rollback()

	// release queue items
	err = r.releaseQueueItems(ctx, tx, tenantId, tasks)

	if err != nil {
		return err
	}

	// write app failures
	if len(appFailures) > 0 {
		err = r.queries.FailTaskAppFailure(ctx, tx, sqlcv2.FailTaskAppFailureParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			Taskids:  appFailures,
		})

		if err != nil {
			return err
		}
	}

	// write internal failures
	if len(internalFailures) > 0 {
		err = r.queries.FailTaskInternalFailure(ctx, tx, sqlcv2.FailTaskInternalFailureParams{
			Tenantid:           sqlchelpers.UUIDFromStr(tenantId),
			Taskids:            internalFailures,
			Maxinternalretries: MAX_INTERNAL_RETRIES,
		})

		if err != nil {
			return err
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
		return err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return err
	}

	return nil
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

func (r *TaskRepositoryImpl) releaseQueueItems(ctx context.Context, tx sqlcv2.DBTX, tenantId string, tasks []TaskIdRetryCount) error {
	taskIds := make([]int64, len(tasks))
	retryCounts := make([]int32, len(tasks))

	for i, task := range tasks {
		taskIds[i] = task.Id
		retryCounts[i] = task.RetryCount
	}

	return r.queries.ReleaseQueueItems(ctx, tx, sqlcv2.ReleaseQueueItemsParams{
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
func (r *TaskRepositoryImpl) createTasks(ctx context.Context, tx sqlcv2.DBTX, tenantId string, tasks []CreateTaskOpts) error {
	params := make([]sqlcv2.CreateTasksParams, len(tasks))

	for i, task := range tasks {
		p := sqlcv2.CreateTasksParams{
			TenantID:        sqlchelpers.UUIDFromStr(tenantId),
			Queue:           task.Queue,
			ActionID:        task.ActionId,
			StepID:          sqlchelpers.UUIDFromStr(task.StepId),
			ScheduleTimeout: task.ScheduleTimeout,
			StepTimeout:     sqlchelpers.TextFromStr(task.StepTimeout),
			ExternalID:      sqlchelpers.UUIDFromStr(task.ExternalId),
			DisplayName:     task.DisplayName,
			Input:           task.Input,
			RetryCount:      0,
		}

		if task.Priority != nil {
			p.Priority = pgtype.Int4{
				Int32: int32(*task.Priority),
				Valid: true,
			}
		}

		if task.StickyStrategy != nil {
			p.Sticky = sqlcv2.NullStickyStrategy{
				StickyStrategy: sqlcv2.StickyStrategy(*task.StickyStrategy),
				Valid:          true,
			}
		}

		if task.DesiredWorkerId != nil {
			p.DesiredWorkerID = sqlchelpers.UUIDFromStr(*task.DesiredWorkerId)
		}

		params[i] = p
	}

	_, err := r.queries.CreateTasks(ctx, tx, params)

	if err != nil {
		return err
	}

	return nil
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
