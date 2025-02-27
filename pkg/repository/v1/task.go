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

	// (required) the initial state for the task
	InitialState sqlcv1.V1TaskInitialState

	// (optional) a list of concurrency keys for the task
	ConcurrencyKeys []string

	// (optional) the step retry backoff factor
	RetryBackoffFactor *float64 `validate:"omitnil,min=1,max=1000"`

	// (optional) the step retry backoff max seconds (can't be greater than 86400)
	RetryBackoffMaxSeconds *int `validate:"omitnil,min=1,max=86400"`
}

type TaskIdRetryCount struct {
	// (required) the external id
	Id int64 `validate:"required"`

	// (required) the retry count
	RetryCount int32
}

type CompleteTaskOpts struct {
	*TaskIdRetryCount

	// (required) the output bytes for the task
	Output []byte
}

type FailTaskOpts struct {
	*TaskIdRetryCount

	// (required) whether this is an application-level error or an internal error on the Hatchet side
	IsAppError bool

	// (optional) the error message for the task
	ErrorMessage string
}

type TaskIdEventKeyTuple struct {
	Id int64 `validate:"required"`

	EventKey string `validate:"required"`
}

type TaskRepository interface {
	UpdateTablePartitions(ctx context.Context) error

	CompleteTasks(ctx context.Context, tenantId string, tasks []CompleteTaskOpts) ([]*sqlcv1.ReleaseTasksRow, error)

	FailTasks(ctx context.Context, tenantId string, tasks []FailTaskOpts) (retriedTasks []TaskIdRetryCount, queues []*sqlcv1.ReleaseTasksRow, err error)

	CancelTasks(ctx context.Context, tenantId string, tasks []TaskIdRetryCount) ([]*sqlcv1.ReleaseTasksRow, []*sqlcv1.V1Task, error)

	ListTasks(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv1.V1Task, error)

	ListCompletedTaskSignals(ctx context.Context, tenantId string, tasks []TaskIdEventKeyTuple) ([]*sqlcv1.V1TaskEvent, error)

	ListTaskMetas(ctx context.Context, tenantId string, tasks []int64) ([]*sqlcv1.ListTaskMetasRow, error)

	ProcessTaskTimeouts(ctx context.Context, tenantId string) ([]*sqlcv1.ProcessTaskTimeoutsRow, bool, error)

	ProcessTaskReassignments(ctx context.Context, tenantId string) ([]*sqlcv1.ProcessTaskReassignmentsRow, bool, error)

	GetQueueCounts(ctx context.Context, tenantId string) (map[string]int, error)
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
			fmt.Sprintf("ALTER TABLE v1_task DETACH PARTITION %s CONCURRENTLY", partition),
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
			fmt.Sprintf("ALTER TABLE v1_dag DETACH PARTITION %s CONCURRENTLY", partition),
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

	err = r.queries.CreateConcurrencyPartition(ctx, r.pool, pgtype.Date{
		Time:  today,
		Valid: true,
	})

	if err != nil {
		return err
	}

	err = r.queries.CreateConcurrencyPartition(ctx, r.pool, pgtype.Date{
		Time:  tomorrow,
		Valid: true,
	})

	if err != nil {
		return err
	}

	concurrencyPartitions, err := r.queries.ListConcurrencyPartitionsBeforeDate(ctx, r.pool, pgtype.Date{
		Time:  sevenDaysAgo,
		Valid: true,
	})

	if err != nil {
		return err
	}

	for _, partition := range concurrencyPartitions {
		_, err := r.pool.Exec(
			ctx,
			fmt.Sprintf("ALTER TABLE v1_concurrency_slot DETACH PARTITION %s CONCURRENTLY", partition),
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

func (r *TaskRepositoryImpl) CompleteTasks(ctx context.Context, tenantId string, tasks []CompleteTaskOpts) ([]*sqlcv1.ReleaseTasksRow, error) {
	// TODO: ADD BACK VALIDATION
	// if err := r.v.Validate(tasks); err != nil {
	// 	fmt.Println("FAILED VALIDATION HERE!!!")

	// 	return err
	// }

	datas := make([][]byte, len(tasks))
	taskIdRetryCounts := make([]TaskIdRetryCount, len(tasks))

	for i, task := range tasks {
		datas[i] = task.Output
		taskIdRetryCounts[i] = *task.TaskIdRetryCount
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	// release queue items
	releasedTasks, err := r.releaseTasks(ctx, tx, tenantId, taskIdRetryCounts)

	if err != nil {
		return nil, err
	}

	err = r.createTaskEvents(
		ctx,
		tx,
		tenantId,
		taskIdRetryCounts,
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

	return releasedTasks, nil
}

type FailTaskData struct {
	ErrorMessage string `json:"error_message"`

	// We use this to disambiguate in event matches downstream -- if this is true, we know that
	// this is an error payload and not a task output which happens to have an `error_message` field.
	IsErrorPayload bool `json:"is_error_payload"`
}

func (r *TaskRepositoryImpl) FailTasks(ctx context.Context, tenantId string, failureOpts []FailTaskOpts) ([]TaskIdRetryCount, []*sqlcv1.ReleaseTasksRow, error) {
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

		dataBytes, err := json.Marshal(FailTaskData{
			ErrorMessage:   failureOpt.ErrorMessage,
			IsErrorPayload: true,
		})

		if err == nil {
			datas[i] = dataBytes
		}

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
		appFailureRetries, err := r.queries.FailTaskAppFailure(ctx, tx, sqlcv1.FailTaskAppFailureParams{
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
		internalFailureRetries, err := r.queries.FailTaskInternalFailure(ctx, tx, sqlcv1.FailTaskInternalFailureParams{
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
		sqlcv1.V1TaskEventTypeFAILED,
		make([]string, len(tasks)),
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

func (r *TaskRepositoryImpl) CancelTasks(ctx context.Context, tenantId string, tasks []TaskIdRetryCount) ([]*sqlcv1.ReleaseTasksRow, []*sqlcv1.V1Task, error) {
	// TODO: ADD BACK VALIDATION
	// if err := r.v.Validate(tasks); err != nil {
	// 	fmt.Println("FAILED VALIDATION HERE!!!")

	// 	return err
	// }

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	// release queue items
	releasedTasks, err := r.cancelTasks(ctx, tx, tenantId, tasks)

	if err != nil {
		return nil, nil, err
	}

	taskIds := make([]int64, len(tasks))

	for i, task := range tasks {
		taskIds[i] = task.Id
	}

	resTasks, err := r.listTasks(ctx, tx, tenantId, taskIds)

	if err != nil {
		return nil, nil, err
	}

	// commit the transaction
	if err := commit(ctx); err != nil {
		return nil, nil, err
	}

	return releasedTasks, resTasks, nil
}

func (r *sharedRepository) cancelTasks(ctx context.Context, dbtx sqlcv1.DBTX, tenantId string, tasks []TaskIdRetryCount) ([]*sqlcv1.ReleaseTasksRow, error) {
	// release queue items
	releasedTasks, err := r.releaseTasks(ctx, dbtx, tenantId, tasks)

	if err != nil {
		return nil, err
	}

	// write task events
	err = r.createTaskEvents(
		ctx,
		dbtx,
		tenantId,
		tasks,
		make([][]byte, len(tasks)),
		sqlcv1.V1TaskEventTypeCANCELLED,
		make([]string, len(tasks)),
	)

	if err != nil {
		return nil, err
	}

	return releasedTasks, nil
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

func (r *TaskRepositoryImpl) ListCompletedTaskSignals(ctx context.Context, tenantId string, tasks []TaskIdEventKeyTuple) ([]*sqlcv1.V1TaskEvent, error) {
	taskIds := make([]int64, len(tasks))
	eventKeys := make([]string, len(tasks))

	for i, task := range tasks {
		taskIds[i] = task.Id
		eventKeys[i] = task.EventKey
	}

	return r.queries.ListMatchingSignalEvents(ctx, r.pool, sqlcv1.ListMatchingSignalEventsParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Taskids:    taskIds,
		Signalkeys: eventKeys,
		Eventtype:  sqlcv1.V1TaskEventTypeSIGNALCOMPLETED,
	})
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
			sqlcv1.V1TaskEventTypeFAILED,
			make([]string, len(failedTasks)),
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

func (r *sharedRepository) releaseTasks(ctx context.Context, tx sqlcv1.DBTX, tenantId string, tasks []TaskIdRetryCount) ([]*sqlcv1.ReleaseTasksRow, error) {
	taskIds := make([]int64, len(tasks))
	retryCounts := make([]int32, len(tasks))

	for i, task := range tasks {
		taskIds[i] = task.Id
		retryCounts[i] = task.RetryCount
	}

	return r.queries.ReleaseTasks(ctx, tx, sqlcv1.ReleaseTasksParams{
		Taskids:     taskIds,
		Retrycounts: retryCounts,
	})
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
	concurrencyStrats, err := r.getConcurrencyExpressions(ctx, tx, tenantId, stepIdsToConfig)

	if err != nil {
		return nil, fmt.Errorf("failed to get step expressions: %w", err)
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
		stickies[i] = string(sqlcv1.V1StickyStrategyNONE)
		desiredWorkerIds[i] = pgtype.UUID{
			Valid: false,
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
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create tasks: %w", err)
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

// func (r *sharedRepository) getStepExpressions(
// 	ctx context.Context,
// 	tx sqlcv1.DBTX,
// 	stepIdsToConfig map[string]*sqlcv1.ListStepsByIdsRow,
// ) (map[string][]*sqlcv1.StepExpression, error) {
// 	stepIdsWithExpressions := make(map[string]struct{})

// 	for _, step := range stepIdsToConfig {
// 		if step.ExpressionCount > 0 {
// 			stepIdsWithExpressions[sqlchelpers.UUIDToStr(step.ID)] = struct{}{}
// 		}
// 	}

// 	if len(stepIdsWithExpressions) == 0 {
// 		return nil, nil
// 	}

// 	stepIds := make([]pgtype.UUID, 0, len(stepIdsWithExpressions))

// 	for stepId := range stepIdsWithExpressions {
// 		stepIds = append(stepIds, sqlchelpers.UUIDFromStr(stepId))
// 	}

// 	expressions, err := r.queries.ListStepExpressions(ctx, tx, stepIds)

// 	if err != nil {
// 		return nil, err
// 	}

// 	stepIdToExpressions := make(map[string][]*sqlcv1.StepExpression)

// 	for _, expression := range expressions {
// 		stepId := sqlchelpers.UUIDToStr(expression.StepId)

// 		if _, ok := stepIdToExpressions[stepId]; !ok {
// 			stepIdToExpressions[stepId] = make([]*sqlcv1.StepExpression, 0)
// 		}

// 		stepIdToExpressions[stepId] = append(stepIdToExpressions[stepId], expression)
// 	}

// 	return stepIdToExpressions, nil
// }

func (r *sharedRepository) createTaskEvents(
	ctx context.Context,
	dbtx sqlcv1.DBTX,
	tenantId string,
	tasks []TaskIdRetryCount,
	eventDatas [][]byte,
	eventType sqlcv1.V1TaskEventType,
	eventKeys []string,
) error {
	if len(tasks) != len(eventDatas) {
		return fmt.Errorf("mismatched task and event data lengths")
	}

	taskIds := make([]int64, len(tasks))
	retryCounts := make([]int32, len(tasks))
	eventTypes := make([]string, len(tasks))
	paramDatas := make([][]byte, len(tasks))
	paramKeys := make([]pgtype.Text, len(tasks))

	for i, task := range tasks {
		taskIds[i] = task.Id
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
	}

	return r.queries.CreateTaskEvents(ctx, dbtx, sqlcv1.CreateTaskEventsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Taskids:     taskIds,
		Retrycounts: retryCounts,
		Eventtypes:  eventTypes,
		Datas:       paramDatas,
		Eventkeys:   paramKeys,
	})
}

// TODO: IF A REPLAYED TASK HAS A PARENT TASK, WE NEED TO DELETE AND RE-INIT THE PARENT TASK'S SIGNAL, AND
// THEN REPROCESS THE SIGNAL...
func (r *TaskRepositoryImpl) ReplayTasks(ctx context.Context, tenantId string, tasks []TaskIdRetryCount) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return err
	}

	defer rollback()
	defer commit(ctx)

	taskIds := make([]int64, len(tasks))

	for i, task := range tasks {
		taskIds[i] = task.Id
	}

	// list tasks and place a lock on the tasks, and join on steps to get the structure of the DAG
	lockedTasks, err := r.queries.LockTasksForReplay(ctx, tx, sqlcv1.LockTasksForReplayParams{
		Taskids:  taskIds,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return err
	}

	// group tasks by their dag_id, if it exists
	dagIdsToChildTasks := make(map[int64][]*sqlcv1.LockTasksForReplayRow)
	dagIds := make(map[int64]struct{}, 0)

	// figure out which tasks to delete and which to increment
	tasksToDelete := make([]int64, 0)
	tasksToIncrementRetries := make([]int64, 0)

	for _, task := range lockedTasks {
		if task.DagID.Valid && len(task.Parents) > 0 {
			if _, ok := dagIdsToChildTasks[task.DagID.Int64]; !ok {
				dagIdsToChildTasks[task.DagID.Int64] = make([]*sqlcv1.LockTasksForReplayRow, 0)
			}

			dagIdsToChildTasks[task.DagID.Int64] = append(dagIdsToChildTasks[task.DagID.Int64], task)
			tasksToDelete = append(tasksToDelete, task.ID)
		} else {
			tasksToIncrementRetries = append(tasksToIncrementRetries, task.ID)
		}

		if task.DagID.Valid {
			dagIds[task.DagID.Int64] = struct{}{}
		}
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
		return err
	}

	dagIdsToAllTasks := make(map[int64][]*sqlcv1.ListAllTasksInDagsRow)

	for _, task := range allTasksInDAGs {
		if _, ok := dagIdsToAllTasks[task.DagID.Int64]; !ok {
			dagIdsToAllTasks[task.DagID.Int64] = make([]*sqlcv1.ListAllTasksInDagsRow, 0)
		}

		dagIdsToAllTasks[task.DagID.Int64] = append(dagIdsToAllTasks[task.DagID.Int64], task)
	}

	// NOTE: the tasks which are passed in represent a *subtree* of the DAG.
	// If this is a DAG, delete all tasks and task events which are in the subtree of the DAG
	if len(tasksToDelete) > 0 {
		err = r.queries.DeleteTasksForReplay(ctx, tx, sqlcv1.DeleteTasksForReplayParams{
			Taskids:  tasksToDelete,
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil {
			return err
		}
	}

	if len(tasksToIncrementRetries) > 0 {
		err = r.queries.UpdateTasksForReplay(ctx, tx, sqlcv1.UpdateTasksForReplayParams{
			Taskids:  tasksToIncrementRetries,
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil {
			return err
		}
	}

	// For any DAGs, reset all match conditions which refer to internal events within the subtree of the DAG.
	// we do not reset other match conditions (for example, ones which refer to completed events for tasks
	// which are outside of this subtree). otherwise, we would end up in a state where these events would
	// never be matched.
	subtreeExternalIds := make(map[string]struct{})
	eventMatches := make([]CreateMatchOpts, 0)

	for dagId, tasks := range dagIdsToChildTasks {
		allTasks := dagIdsToAllTasks[dagId]

		for _, task := range tasks {
			taskExternalId := sqlchelpers.UUIDToStr(task.ExternalID)
			subtreeExternalIds[taskExternalId] = struct{}{}
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
					Kind:                 sqlcv1.V1MatchKindTRIGGER,
					Conditions:           conditions,
					TriggerExternalId:    &taskExternalId,
					TriggerStepId:        &stepId,
					TriggerDAGId:         &task.DagID.Int64,
					TriggerDAGInsertedAt: task.DagInsertedAt,
				})
			default:
				conditions := make([]GroupMatchCondition, 0)

				createGroupId := uuid.NewString()

				for _, parent := range task.Parents {
					// FIXME: n^2 complexity here, fix it.
					for _, otherTask := range allTasks {
						if otherTask.StepID == parent {
							parentExternalId := sqlchelpers.UUIDToStr(otherTask.ExternalID)
							readableId := otherTask.StepReadableID

							conditions = append(conditions, getParentInDAGGroupMatch(createGroupId, parentExternalId, readableId)...)
						}
					}
				}

				// create an event match
				eventMatches = append(eventMatches, CreateMatchOpts{
					Kind:                 sqlcv1.V1MatchKindTRIGGER,
					Conditions:           conditions,
					TriggerExternalId:    &taskExternalId,
					TriggerStepId:        &stepId,
					TriggerDAGId:         &task.DagID.Int64,
					TriggerDAGInsertedAt: task.DagInsertedAt,
				})
			}
		}
	}

	// reconstruct group conditions
	reconstructedMatches, candidateEvents, err := r.reconstructGroupConditions(ctx, tx, tenantId, subtreeExternalIds, eventMatches)

	if err != nil {
		return err
	}

	// create the event matches
	err = r.createEventMatches(ctx, tx, tenantId, reconstructedMatches)

	if err != nil {
		return err
	}

	// process event matches
	// TODO: signal the event matches to the caller
	_, err = r.processInternalEventMatches(ctx, tx, tenantId, candidateEvents)

	if err != nil {
		return err
	}

	return nil
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
	dagIds := make([]int64, 0)
	externalIds := make([]pgtype.UUID, 0)
	eventKeys := make([]sqlcv1.V1TaskEventType, 0)

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
					dagIds = append(dagIds, *match.TriggerDAGId)
					externalIds = append(externalIds, sqlchelpers.UUIDFromStr(*groupCondition.EventResourceHint))
					eventKeys = append(eventKeys, sqlcv1.V1TaskEventType(groupCondition.EventType))
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
		Dagids:          dagIds,
		Taskexternalids: externalIds,
		Eventkeys:       eventKeys,
	})

	if err != nil {
		return nil, nil, err
	}

	foundMatchKeys := make(map[string]*sqlcv1.ListMatchingTaskEventsRow)

	for _, eventMatch := range matchedEvents {
		key := fmt.Sprintf("%s:%s", sqlchelpers.UUIDToStr(eventMatch.TaskExternalID), eventMatch.EventKey)

		foundMatchKeys[key] = eventMatch
	}

	resMatches := make([]CreateMatchOpts, 0)
	resCandidateEvents := make([]CandidateEventMatch, 0)

	// for each group condition, if we have a match, mark the group condition as satisfied and use
	// the data from the match to update the group condition.
	for _, match := range eventMatches {
		if match.TriggerDAGId == nil {
			continue
		}

		conditions := make([]GroupMatchCondition, 0)

		for _, groupCondition := range match.Conditions {
			cond := groupCondition

			if groupCondition.EventType == sqlcv1.V1EventTypeINTERNAL && groupCondition.EventResourceHint != nil {
				key := fmt.Sprintf("%s:%s", *groupCondition.EventResourceHint, string(groupCondition.EventKey))

				if match, ok := foundMatchKeys[key]; ok {
					cond.IsSatisfied = true
					cond.Data = match.Data

					taskExternalId := sqlchelpers.UUIDToStr(match.TaskExternalID)

					resCandidateEvents = append(resCandidateEvents, CandidateEventMatch{
						ID:             uuid.NewString(),
						EventTimestamp: match.CreatedAt.Time,
						Key:            match.EventKey.String,
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
