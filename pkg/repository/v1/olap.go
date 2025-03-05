package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

// TODO: make this dynamic for the instance
const NUM_PARTITIONS = 4

type ListTaskRunOpts struct {
	CreatedAfter time.Time

	Statuses []sqlcv1.V1ReadableStatusOlap

	WorkflowIds []uuid.UUID

	WorkerId *uuid.UUID

	StartedAfter time.Time

	FinishedBefore *time.Time

	AdditionalMetadata map[string]interface{}

	Limit int64

	Offset int64
}

type ListWorkflowRunOpts struct {
	CreatedAfter time.Time

	Statuses []sqlcv1.V1ReadableStatusOlap

	WorkflowIds []uuid.UUID

	StartedAfter time.Time

	FinishedBefore *time.Time

	AdditionalMetadata map[string]interface{}

	Limit int64

	Offset int64

	ParentTaskExternalId *pgtype.UUID
}

type ReadTaskRunMetricsOpts struct {
	CreatedAfter time.Time

	WorkflowIds []uuid.UUID

	ParentTaskExternalID *pgtype.UUID
}

type WorkflowRunData struct {
	AdditionalMetadata []byte                      `json:"additional_metadata"`
	CreatedAt          pgtype.Timestamptz          `json:"created_at"`
	DisplayName        string                      `json:"display_name"`
	ErrorMessage       string                      `json:"error_message"`
	ExternalID         pgtype.UUID                 `json:"external_id"`
	FinishedAt         pgtype.Timestamptz          `json:"finished_at"`
	Input              []byte                      `json:"input"`
	InsertedAt         pgtype.Timestamptz          `json:"inserted_at"`
	Kind               sqlcv1.V1RunKind            `json:"kind"`
	Output             *[]byte                     `json:"output,omitempty"`
	ReadableStatus     sqlcv1.V1ReadableStatusOlap `json:"readable_status"`
	StartedAt          pgtype.Timestamptz          `json:"started_at"`
	TaskExternalId     *pgtype.UUID                `json:"task_external_id,omitempty"`
	TaskId             *int64                      `json:"task_id,omitempty"`
	TaskInsertedAt     *pgtype.Timestamptz         `json:"task_inserted_at,omitempty"`
	TenantID           pgtype.UUID                 `json:"tenant_id"`
	WorkflowID         pgtype.UUID                 `json:"workflow_id"`
	WorkflowVersionId  pgtype.UUID                 `json:"workflow_version_id"`
}

type V1WorkflowRunPopulator struct {
	WorkflowRun  *WorkflowRunData
	TaskMetadata []TaskMetadata
}

type TaskRunMetric struct {
	Status string `json:"status"`
	Count  uint64 `json:"count"`
}
type Sticky string

const (
	STICKY_HARD Sticky = "HARD"
	STICKY_SOFT Sticky = "SOFT"
	STICKY_NONE Sticky = "NONE"
)

type EventType string

const (
	EVENT_TYPE_REQUEUED_NO_WORKER   EventType = "REQUEUED_NO_WORKER"
	EVENT_TYPE_REQUEUED_RATE_LIMIT  EventType = "REQUEUED_RATE_LIMIT"
	EVENT_TYPE_SCHEDULING_TIMED_OUT EventType = "SCHEDULING_TIMED_OUT"
	EVENT_TYPE_ASSIGNED             EventType = "ASSIGNED"
	EVENT_TYPE_STARTED              EventType = "STARTED"
	EVENT_TYPE_FINISHED             EventType = "FINISHED"
	EVENT_TYPE_FAILED               EventType = "FAILED"
	EVENT_TYPE_RETRYING             EventType = "RETRYING"
	EVENT_TYPE_CANCELLED            EventType = "CANCELLED"
	EVENT_TYPE_TIMED_OUT            EventType = "TIMED_OUT"
	EVENT_TYPE_REASSIGNED           EventType = "REASSIGNED"
	EVENT_TYPE_SLOT_RELEASED        EventType = "SLOT_RELEASED"
	EVENT_TYPE_TIMEOUT_REFRESHED    EventType = "TIMEOUT_REFRESHED"
	EVENT_TYPE_RETRIED_BY_USER      EventType = "RETRIED_BY_USER"
	EVENT_TYPE_SENT_TO_WORKER       EventType = "SENT_TO_WORKER"
	EVENT_TYPE_RATE_LIMIT_ERROR     EventType = "RATE_LIMIT_ERROR"
	EVENT_TYPE_ACKNOWLEDGED         EventType = "ACKNOWLEDGED"
	EVENT_TYPE_CREATED              EventType = "CREATED"
	EVENT_TYPE_QUEUED               EventType = "QUEUED"
)

type ReadableTaskStatus string

const (
	READABLE_TASK_STATUS_QUEUED    ReadableTaskStatus = "QUEUED"
	READABLE_TASK_STATUS_RUNNING   ReadableTaskStatus = "RUNNING"
	READABLE_TASK_STATUS_COMPLETED ReadableTaskStatus = "COMPLETED"
	READABLE_TASK_STATUS_CANCELLED ReadableTaskStatus = "CANCELLED"
	READABLE_TASK_STATUS_FAILED    ReadableTaskStatus = "FAILED"
)

func (s ReadableTaskStatus) EnumValue() int {
	switch s {
	case READABLE_TASK_STATUS_QUEUED:
		return 1
	case READABLE_TASK_STATUS_RUNNING:
		return 2
	case READABLE_TASK_STATUS_COMPLETED:
		return 3
	case READABLE_TASK_STATUS_CANCELLED:
		return 4
	case READABLE_TASK_STATUS_FAILED:
		return 5
	default:
		return -1
	}
}

type OLAPRepository interface {
	UpdateTablePartitions(ctx context.Context) error
	ReadTaskRun(ctx context.Context, taskExternalId string) (*sqlcv1.V1TasksOlap, error)
	ReadWorkflowRun(ctx context.Context, workflowRunExternalId pgtype.UUID) (*V1WorkflowRunPopulator, error)
	ReadTaskRunData(ctx context.Context, tenantId pgtype.UUID, taskId int64, taskInsertedAt pgtype.Timestamptz) (*sqlcv1.PopulateSingleTaskRunDataRow, *pgtype.UUID, error)

	ListTasks(ctx context.Context, tenantId string, opts ListTaskRunOpts) ([]*sqlcv1.PopulateTaskRunDataRow, int, error)
	ListWorkflowRuns(ctx context.Context, tenantId string, opts ListWorkflowRunOpts) ([]*WorkflowRunData, int, error)
	ListTaskRunEvents(ctx context.Context, tenantId string, taskId int64, taskInsertedAt pgtype.Timestamptz, limit, offset int64) ([]*sqlcv1.ListTaskEventsRow, error)
	ListTaskRunEventsByWorkflowRunId(ctx context.Context, tenantId string, workflowRunId pgtype.UUID) ([]*sqlcv1.ListTaskEventsForWorkflowRunRow, error)
	ReadTaskRunMetrics(ctx context.Context, tenantId string, opts ReadTaskRunMetricsOpts) ([]TaskRunMetric, error)
	CreateTasks(ctx context.Context, tenantId string, tasks []*sqlcv1.V1Task) error
	CreateTaskEvents(ctx context.Context, tenantId string, events []sqlcv1.CreateTaskEventsOLAPParams) error
	CreateDAGs(ctx context.Context, tenantId string, dags []*DAGWithData) error
	GetTaskPointMetrics(ctx context.Context, tenantId string, startTimestamp *time.Time, endTimestamp *time.Time, bucketInterval time.Duration) ([]*sqlcv1.GetTaskPointMetricsRow, error)
	UpdateTaskStatuses(ctx context.Context, tenantId string) (bool, error)
	UpdateDAGStatuses(ctx context.Context, tenantId string) (bool, error)
	ReadDAG(ctx context.Context, dagExternalId string) (*sqlcv1.V1DagsOlap, error)
	ListTasksByDAGId(ctx context.Context, tenantId string, dagIds []pgtype.UUID) ([]*sqlcv1.PopulateTaskRunDataRow, map[int64]uuid.UUID, error)
	ListTasksByIdAndInsertedAt(ctx context.Context, tenantId string, taskMetadata []TaskMetadata) ([]*sqlcv1.PopulateTaskRunDataRow, error)

	// ListTasksByExternalIds returns a list of tasks based on their external ids or the external id of their parent DAG.
	// In the case of a DAG, we flatten the result into the list of tasks which belong to that DAG.
	ListTasksByExternalIds(ctx context.Context, tenantId string, externalIds []string) ([]*sqlcv1.FlattenTasksByExternalIdsRow, error)
}

type olapRepository struct {
	*sharedRepository

	eventCache *lru.Cache[string, bool]
}

func newOLAPRepository(shared *sharedRepository) OLAPRepository {
	eventCache, err := lru.New[string, bool](100000)

	if err != nil {
		log.Fatal(err)
	}

	return &olapRepository{
		sharedRepository: shared,
		eventCache:       eventCache,
	}
}

func (o *olapRepository) UpdateTablePartitions(ctx context.Context) error {
	err := o.queries.CreateOLAPTaskEventTmpPartitions(ctx, o.pool, NUM_PARTITIONS)

	if err != nil {
		return err
	}

	err = o.queries.CreateOLAPTaskStatusUpdateTmpPartitions(ctx, o.pool, NUM_PARTITIONS)

	if err != nil {
		return err
	}

	err = o.setupRangePartition(
		ctx,
		o.queries.CreateOLAPTaskPartition,
		o.queries.ListOLAPTaskPartitionsBeforeDate,
		"v2_tasks_olap",
	)

	if err != nil {
		return err
	}

	err = o.setupRangePartition(
		ctx,
		o.queries.CreateOLAPDAGPartition,
		o.queries.ListOLAPDAGPartitionsBeforeDate,
		"v2_dags_olap",
	)

	if err != nil {
		return err
	}

	err = o.setupRangePartition(
		ctx,
		o.queries.CreateOLAPRunsPartition,
		o.queries.ListOLAPRunsPartitionsBeforeDate,
		"v2_runs_olap",
	)

	if err != nil {
		return err
	}

	return nil
}

func (o *olapRepository) setupRangePartition(
	ctx context.Context,
	create func(ctx context.Context, db sqlcv1.DBTX, date pgtype.Date) error,
	listBeforeDate func(ctx context.Context, db sqlcv1.DBTX, date pgtype.Date) ([]string, error),
	tableName string,
) error {
	today := time.Now().UTC()
	tomorrow := today.AddDate(0, 0, 1)
	sevenDaysAgo := today.AddDate(0, 0, -7)

	err := create(ctx, o.pool, pgtype.Date{
		Time:  today,
		Valid: true,
	})

	if err != nil {
		return err
	}

	err = create(ctx, o.pool, pgtype.Date{
		Time:  tomorrow,
		Valid: true,
	})

	if err != nil {
		return err
	}

	partitions, err := listBeforeDate(ctx, o.pool, pgtype.Date{
		Time:  sevenDaysAgo,
		Valid: true,
	})

	if err != nil {
		return err
	}

	for _, partition := range partitions {
		_, err := o.pool.Exec(
			ctx,
			fmt.Sprintf("ALTER TABLE %s DETACH PARTITION %s CONCURRENTLY", tableName, partition),
		)

		if err != nil {
			return err
		}

		_, err = o.pool.Exec(
			ctx,
			fmt.Sprintf("DROP TABLE %s", partition),
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func StringToReadableStatus(status string) ReadableTaskStatus {
	switch status {
	case "QUEUED":
		return READABLE_TASK_STATUS_QUEUED
	case "RUNNING":
		return READABLE_TASK_STATUS_RUNNING
	case "COMPLETED":
		return READABLE_TASK_STATUS_COMPLETED
	case "CANCELLED":
		return READABLE_TASK_STATUS_CANCELLED
	case "FAILED":
		return READABLE_TASK_STATUS_FAILED
	default:
		return READABLE_TASK_STATUS_QUEUED
	}
}

func (r *olapRepository) ReadTaskRun(ctx context.Context, taskExternalId string) (*sqlcv1.V1TasksOlap, error) {
	row, err := r.queries.ReadTaskByExternalID(ctx, r.pool, sqlchelpers.UUIDFromStr(taskExternalId))

	if err != nil {
		return nil, err
	}

	return &sqlcv1.V1TasksOlap{
		TenantID:           row.TenantID,
		ID:                 row.ID,
		InsertedAt:         row.InsertedAt,
		Queue:              row.Queue,
		ActionID:           row.ActionID,
		StepID:             row.StepID,
		WorkflowID:         row.WorkflowID,
		ScheduleTimeout:    row.ScheduleTimeout,
		StepTimeout:        row.StepTimeout,
		Priority:           row.Priority,
		Sticky:             row.Sticky,
		DesiredWorkerID:    row.DesiredWorkerID,
		DisplayName:        row.DisplayName,
		Input:              row.Input,
		AdditionalMetadata: row.AdditionalMetadata,
		DagID:              row.DagID,
		DagInsertedAt:      row.DagInsertedAt,
		ReadableStatus:     row.ReadableStatus,
		ExternalID:         row.ExternalID,
		LatestRetryCount:   row.LatestRetryCount,
		LatestWorkerID:     row.LatestWorkerID,
	}, nil
}

type TaskMetadata struct {
	TaskID         int64     `json:"task_id"`
	TaskInsertedAt time.Time `json:"task_inserted_at"`
}

func ParseTaskMetadata(jsonData []byte) ([]TaskMetadata, error) {
	var tasks []TaskMetadata
	err := json.Unmarshal(jsonData, &tasks)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *olapRepository) ReadWorkflowRun(ctx context.Context, workflowRunExternalId pgtype.UUID) (*V1WorkflowRunPopulator, error) {
	row, err := r.queries.ReadWorkflowRunByExternalId(ctx, r.pool, workflowRunExternalId)

	if err != nil {
		return nil, err
	}

	taskMetadata, err := ParseTaskMetadata(row.TaskMetadata)

	if err != nil {
		return nil, err
	}

	return &V1WorkflowRunPopulator{
		WorkflowRun: &WorkflowRunData{
			TenantID:           row.TenantID,
			InsertedAt:         row.InsertedAt,
			ExternalID:         row.ExternalID,
			ReadableStatus:     row.ReadableStatus,
			Kind:               row.Kind,
			WorkflowID:         row.WorkflowID,
			DisplayName:        row.DisplayName,
			AdditionalMetadata: row.AdditionalMetadata,
			CreatedAt:          row.CreatedAt,
			StartedAt:          row.StartedAt,
			FinishedAt:         row.FinishedAt,
			ErrorMessage:       row.ErrorMessage.String,
			WorkflowVersionId:  row.WorkflowVersionID,
			Input:              row.Input,
		},
		TaskMetadata: taskMetadata,
	}, nil
}

func (r *olapRepository) ReadTaskRunData(ctx context.Context, tenantId pgtype.UUID, taskId int64, taskInsertedAt pgtype.Timestamptz) (*sqlcv1.PopulateSingleTaskRunDataRow, *pgtype.UUID, error) {
	taskRun, err := r.queries.PopulateSingleTaskRunData(ctx, r.pool, sqlcv1.PopulateSingleTaskRunDataParams{
		Taskid:         taskId,
		Tenantid:       tenantId,
		Taskinsertedat: taskInsertedAt,
	})

	if err != nil {
		return nil, nil, err
	}

	workflowRunId := taskRun.ExternalID

	if taskRun.DagID.Valid {
		dagId := taskRun.DagID.Int64
		dagInsertedAt := taskRun.DagInsertedAt

		workflowRunId, err = r.queries.GetWorkflowRunIdFromDagIdInsertedAt(ctx, r.pool, sqlcv1.GetWorkflowRunIdFromDagIdInsertedAtParams{
			Dagid:         dagId,
			Daginsertedat: dagInsertedAt,
		})

		if err != nil {
			return nil, nil, err
		}
	}

	return taskRun, &workflowRunId, nil
}

func (r *olapRepository) ListTasks(ctx context.Context, tenantId string, opts ListTaskRunOpts) ([]*sqlcv1.PopulateTaskRunDataRow, int, error) {
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, 0, err
	}

	defer tx.Rollback(ctx)

	params := sqlcv1.ListTasksOlapParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Since:      sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		Tasklimit:  int32(opts.Limit),
		Taskoffset: int32(opts.Offset),
	}

	countParams := sqlcv1.CountTasksParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Since:    sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
	}

	statuses := make([]string, 0)

	for _, status := range opts.Statuses {
		statuses = append(statuses, string(status))
	}

	if len(statuses) == 0 {
		statuses = []string{
			string(sqlcv1.V1ReadableStatusOlapQUEUED),
			string(sqlcv1.V1ReadableStatusOlapRUNNING),
			string(sqlcv1.V1ReadableStatusOlapCOMPLETED),
			string(sqlcv1.V1ReadableStatusOlapCANCELLED),
			string(sqlcv1.V1ReadableStatusOlapFAILED),
		}
	}

	params.Statuses = statuses
	countParams.Statuses = statuses

	if len(opts.WorkflowIds) > 0 {
		workflowIdParams := make([]pgtype.UUID, 0)

		for _, id := range opts.WorkflowIds {
			workflowIdParams = append(workflowIdParams, sqlchelpers.UUIDFromStr(id.String()))
		}

		params.WorkflowIds = workflowIdParams
		countParams.WorkflowIds = workflowIdParams
	}

	until := opts.FinishedBefore

	if until != nil {
		params.Until = sqlchelpers.TimestamptzFromTime(*until)
		countParams.Until = sqlchelpers.TimestamptzFromTime(*until)
	}

	workerId := opts.WorkerId

	if workerId != nil {
		params.WorkerId = sqlchelpers.UUIDFromStr(workerId.String())
	}

	for key, value := range opts.AdditionalMetadata {
		params.Keys = append(params.Keys, key)
		params.Values = append(params.Values, value.(string))
		countParams.Keys = append(countParams.Keys, key)
		countParams.Values = append(countParams.Values, value.(string))
	}

	rows, err := r.queries.ListTasksOlap(ctx, tx, params)

	if err != nil {
		return nil, 0, err
	}

	taskIds := make([]int64, 0)
	taskInsertedAts := make([]pgtype.Timestamptz, 0)

	for _, row := range rows {
		taskIds = append(taskIds, row.ID)
		taskInsertedAts = append(taskInsertedAts, row.InsertedAt)
	}

	tasksWithData, err := r.queries.PopulateTaskRunData(ctx, tx, sqlcv1.PopulateTaskRunDataParams{
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, err
	}

	count, err := r.queries.CountTasks(ctx, tx, countParams)

	if err != nil {
		count = int64(len(tasksWithData))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}

	return tasksWithData, int(count), nil
}

func (r *olapRepository) ListTasksByDAGId(ctx context.Context, tenantId string, dagids []pgtype.UUID) ([]*sqlcv1.PopulateTaskRunDataRow, map[int64]uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	taskIdToDagExternalId := make(map[int64]uuid.UUID)

	if err != nil {
		return nil, taskIdToDagExternalId, err
	}

	defer tx.Rollback(ctx)

	tasks, err := r.queries.ListTasksByDAGIds(ctx, tx, sqlcv1.ListTasksByDAGIdsParams{
		Dagids:   dagids,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, taskIdToDagExternalId, err
	}

	for _, row := range tasks {
		taskIdToDagExternalId[row.TaskID] = uuid.MustParse(sqlchelpers.UUIDToStr(row.DagExternalID))
	}

	taskIds := make([]int64, 0)
	taskInsertedAts := make([]pgtype.Timestamptz, 0)

	for _, row := range tasks {
		taskIds = append(taskIds, row.TaskID)
		taskInsertedAts = append(taskInsertedAts, row.TaskInsertedAt)
	}

	tasksWithData, err := r.queries.PopulateTaskRunData(ctx, tx, sqlcv1.PopulateTaskRunDataParams{
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, taskIdToDagExternalId, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, taskIdToDagExternalId, err
	}

	return tasksWithData, taskIdToDagExternalId, nil
}

func (r *olapRepository) ListTasksByIdAndInsertedAt(ctx context.Context, tenantId string, taskMetadata []TaskMetadata) ([]*sqlcv1.PopulateTaskRunDataRow, error) {
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer tx.Rollback(ctx)

	taskIds := make([]int64, 0)
	taskInsertedAts := make([]pgtype.Timestamptz, 0)

	for _, metadata := range taskMetadata {
		taskIds = append(taskIds, metadata.TaskID)
		taskInsertedAts = append(taskInsertedAts, sqlchelpers.TimestamptzFromTime(metadata.TaskInsertedAt))
	}

	tasksWithData, err := r.queries.PopulateTaskRunData(ctx, tx, sqlcv1.PopulateTaskRunDataParams{
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return tasksWithData, nil
}

func (r *olapRepository) ListWorkflowRuns(ctx context.Context, tenantId string, opts ListWorkflowRunOpts) ([]*WorkflowRunData, int, error) {
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, 0, err
	}

	defer tx.Rollback(ctx)

	params := sqlcv1.FetchWorkflowRunIdsParams{
		Tenantid:               sqlchelpers.UUIDFromStr(tenantId),
		Since:                  sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		Listworkflowrunslimit:  int32(opts.Limit),
		Listworkflowrunsoffset: int32(opts.Offset),
		ParentTaskExternalId:   pgtype.UUID{},
	}

	countParams := sqlcv1.CountWorkflowRunsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Since:    sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
	}

	statuses := make([]string, 0)

	for _, status := range opts.Statuses {
		statuses = append(statuses, string(status))
	}

	if len(statuses) == 0 {
		statuses = []string{
			string(sqlcv1.V1ReadableStatusOlapQUEUED),
			string(sqlcv1.V1ReadableStatusOlapRUNNING),
			string(sqlcv1.V1ReadableStatusOlapCOMPLETED),
			string(sqlcv1.V1ReadableStatusOlapCANCELLED),
			string(sqlcv1.V1ReadableStatusOlapFAILED),
		}
	}

	params.Statuses = statuses
	countParams.Statuses = statuses

	if len(opts.WorkflowIds) > 0 {
		workflowIdParams := make([]pgtype.UUID, 0)

		for _, id := range opts.WorkflowIds {
			workflowIdParams = append(workflowIdParams, sqlchelpers.UUIDFromStr(id.String()))
		}

		params.WorkflowIds = workflowIdParams
		countParams.WorkflowIds = workflowIdParams
	}

	until := opts.FinishedBefore

	if until != nil {
		params.Until = sqlchelpers.TimestamptzFromTime(*until)
		countParams.Until = sqlchelpers.TimestamptzFromTime(*until)
	}

	for key, value := range opts.AdditionalMetadata {
		params.Keys = append(params.Keys, key)
		params.Values = append(params.Values, value.(string))
		countParams.Keys = append(countParams.Keys, key)
		countParams.Values = append(countParams.Values, value.(string))
	}

	if opts.ParentTaskExternalId != nil {
		params.ParentTaskExternalId = *opts.ParentTaskExternalId
	}

	workflowRunIds, err := r.queries.FetchWorkflowRunIds(ctx, tx, params)

	if err != nil {
		return nil, 0, err
	}

	runIdsWithDAGs := make([]int64, 0)
	runInsertedAtsWithDAGs := make([]pgtype.Timestamptz, 0)
	runIdsWithTasks := make([]int64, 0)
	runInsertedAtsWithTasks := make([]pgtype.Timestamptz, 0)

	for _, row := range workflowRunIds {
		if row.Kind == sqlcv1.V1RunKindDAG {
			runIdsWithDAGs = append(runIdsWithDAGs, row.ID)
			runInsertedAtsWithDAGs = append(runInsertedAtsWithDAGs, row.InsertedAt)
		} else {
			runIdsWithTasks = append(runIdsWithTasks, row.ID)
			runInsertedAtsWithTasks = append(runInsertedAtsWithTasks, row.InsertedAt)
		}
	}

	populatedDAGs, err := r.queries.PopulateDAGMetadata(ctx, tx, sqlcv1.PopulateDAGMetadataParams{
		Ids:         runIdsWithDAGs,
		Insertedats: runInsertedAtsWithDAGs,
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, err
	}

	dagsToPopulated := make(map[string]*sqlcv1.PopulateDAGMetadataRow)

	for _, dag := range populatedDAGs {
		externalId := sqlchelpers.UUIDToStr(dag.ExternalID)

		dagsToPopulated[externalId] = dag
	}

	populatedTasks, err := r.queries.PopulateTaskRunData(ctx, tx, sqlcv1.PopulateTaskRunDataParams{
		Taskids:         runIdsWithTasks,
		Taskinsertedats: runInsertedAtsWithTasks,
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, err
	}

	tasksToPopulated := make(map[string]*sqlcv1.PopulateTaskRunDataRow)

	for _, task := range populatedTasks {
		externalId := sqlchelpers.UUIDToStr(task.ExternalID)
		tasksToPopulated[externalId] = task
	}

	count, err := r.queries.CountWorkflowRuns(ctx, tx, countParams)

	if err != nil {
		r.l.Error().Msgf("error counting workflow runs: %v", err)
		count = int64(len(workflowRunIds))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}

	res := make([]*WorkflowRunData, 0)

	for _, row := range workflowRunIds {
		externalId := sqlchelpers.UUIDToStr(row.ExternalID)

		if row.Kind == sqlcv1.V1RunKindDAG {
			dag, ok := dagsToPopulated[externalId]

			if !ok {
				r.l.Error().Msgf("could not find dag with external id %s", externalId)
				continue
			}

			res = append(res, &WorkflowRunData{
				TenantID:           dag.TenantID,
				InsertedAt:         dag.InsertedAt,
				ExternalID:         dag.ExternalID,
				WorkflowID:         dag.WorkflowID,
				DisplayName:        dag.DisplayName,
				ReadableStatus:     dag.ReadableStatus,
				AdditionalMetadata: dag.AdditionalMetadata,
				CreatedAt:          dag.CreatedAt,
				StartedAt:          dag.StartedAt,
				FinishedAt:         dag.FinishedAt,
				ErrorMessage:       dag.ErrorMessage.String,
				Kind:               sqlcv1.V1RunKindDAG,
				WorkflowVersionId:  dag.WorkflowVersionID,
				TaskExternalId:     nil,
				TaskId:             nil,
				TaskInsertedAt:     nil,
				Output:             &dag.Output,
				Input:              dag.Input,
			})
		} else {
			task, ok := tasksToPopulated[externalId]

			if !ok {
				r.l.Error().Msgf("could not find task with external id %s", externalId)
				continue
			}

			res = append(res, &WorkflowRunData{
				TenantID:           task.TenantID,
				InsertedAt:         task.InsertedAt,
				ExternalID:         task.ExternalID,
				WorkflowID:         task.WorkflowID,
				DisplayName:        task.DisplayName,
				ReadableStatus:     task.Status,
				AdditionalMetadata: task.AdditionalMetadata,
				CreatedAt:          task.InsertedAt,
				StartedAt:          task.StartedAt,
				FinishedAt:         task.FinishedAt,
				ErrorMessage:       task.ErrorMessage.String,
				Kind:               sqlcv1.V1RunKindTASK,
				TaskExternalId:     &task.ExternalID,
				TaskId:             &task.ID,
				TaskInsertedAt:     &task.InsertedAt,
				Output:             &task.Output,
				Input:              task.Input,
			})
		}
	}

	return res, int(count), nil
}

func (r *olapRepository) ListTaskRunEvents(ctx context.Context, tenantId string, taskId int64, taskInsertedAt pgtype.Timestamptz, limit, offset int64) ([]*sqlcv1.ListTaskEventsRow, error) {
	rows, err := r.queries.ListTaskEvents(ctx, r.pool, sqlcv1.ListTaskEventsParams{
		Tenantid:       sqlchelpers.UUIDFromStr(tenantId),
		Taskid:         taskId,
		Taskinsertedat: taskInsertedAt,
	})

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *olapRepository) ListTaskRunEventsByWorkflowRunId(ctx context.Context, tenantId string, workflowRunId pgtype.UUID) ([]*sqlcv1.ListTaskEventsForWorkflowRunRow, error) {
	rows, err := r.queries.ListTaskEventsForWorkflowRun(ctx, r.pool, sqlcv1.ListTaskEventsForWorkflowRunParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Workflowrunid: workflowRunId,
	})

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *olapRepository) ReadTaskRunMetrics(ctx context.Context, tenantId string, opts ReadTaskRunMetricsOpts) ([]TaskRunMetric, error) {
	var workflowIds []pgtype.UUID

	if len(opts.WorkflowIds) > 0 {
		workflowIds = make([]pgtype.UUID, 0)

		for _, id := range opts.WorkflowIds {
			workflowIds = append(workflowIds, sqlchelpers.UUIDFromStr(id.String()))
		}
	}

	var parentTaskExternalId pgtype.UUID
	if opts.ParentTaskExternalID != nil {
		parentTaskExternalId = *opts.ParentTaskExternalID
	}

	res, err := r.queries.GetTenantStatusMetrics(context.Background(), r.pool, sqlcv1.GetTenantStatusMetricsParams{
		Tenantid:             sqlchelpers.UUIDFromStr(tenantId),
		Createdafter:         sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		WorkflowIds:          workflowIds,
		ParentTaskExternalId: parentTaskExternalId,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []TaskRunMetric{}, nil
		}

		return nil, err
	}

	metrics := make([]TaskRunMetric, 0)

	metrics = append(metrics, TaskRunMetric{
		Status: "QUEUED",
		Count:  uint64(res.TotalQueued),
	})

	metrics = append(metrics, TaskRunMetric{
		Status: "RUNNING",
		Count:  uint64(res.TotalRunning),
	})

	metrics = append(metrics, TaskRunMetric{
		Status: "COMPLETED",
		Count:  uint64(res.TotalCompleted),
	})

	metrics = append(metrics, TaskRunMetric{
		Status: "CANCELLED",
		Count:  uint64(res.TotalCancelled),
	})

	metrics = append(metrics, TaskRunMetric{
		Status: "FAILED",
		Count:  uint64(res.TotalFailed),
	})

	return metrics, nil
}

func (r *olapRepository) saveEventsToCache(events []sqlcv1.CreateTaskEventsOLAPParams) {
	for _, event := range events {
		key := getCacheKey(event)
		r.eventCache.Add(key, true)
	}
}

func getCacheKey(event sqlcv1.CreateTaskEventsOLAPParams) string {
	// key on the task_id, retry_count, and event_type
	return fmt.Sprintf("%d-%s-%d", event.TaskID, event.EventType, event.RetryCount)
}

func (r *olapRepository) writeTaskEventBatch(ctx context.Context, tenantId string, events []sqlcv1.CreateTaskEventsOLAPParams) error {
	// skip any events which have a corresponding event already
	eventsToWrite := make([]sqlcv1.CreateTaskEventsOLAPParams, 0)
	tmpEventsToWrite := make([]sqlcv1.CreateTaskEventsOLAPTmpParams, 0)

	for _, event := range events {
		key := getCacheKey(event)

		if _, ok := r.eventCache.Get(key); !ok {
			eventsToWrite = append(eventsToWrite, event)

			tmpEventsToWrite = append(tmpEventsToWrite, sqlcv1.CreateTaskEventsOLAPTmpParams{
				TenantID:       event.TenantID,
				TaskID:         event.TaskID,
				TaskInsertedAt: event.TaskInsertedAt,
				EventType:      event.EventType,
				RetryCount:     event.RetryCount,
				ReadableStatus: event.ReadableStatus,
				WorkerID:       event.WorkerID,
			})
		}
	}

	if len(eventsToWrite) == 0 {
		return nil
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return err
	}

	defer rollback()

	_, err = r.queries.CreateTaskEventsOLAP(ctx, tx, eventsToWrite)

	if err != nil {
		return err
	}

	_, err = r.queries.CreateTaskEventsOLAPTmp(ctx, tx, tmpEventsToWrite)

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	r.saveEventsToCache(eventsToWrite)

	return nil
}

func (r *olapRepository) UpdateTaskStatuses(ctx context.Context, tenantId string) (bool, error) {
	var limit int32 = 10000

	// each partition gets its own goroutine
	eg := &errgroup.Group{}
	mu := sync.Mutex{}

	// if any of the partitions are saturated, we return true
	isSaturated := false

	for i := 0; i < NUM_PARTITIONS; i++ {
		partitionNumber := i

		eg.Go(func() error {
			tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 15000)

			if err != nil {
				return err
			}

			defer rollback()

			count, err := r.queries.UpdateTaskStatuses(ctx, tx, sqlcv1.UpdateTaskStatusesParams{
				Partitionnumber: int32(partitionNumber), // nolint: gosec
				Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
				Eventlimit:      limit,
			})

			if err != nil {
				return err
			}

			if err := commit(ctx); err != nil {
				return err
			}

			mu.Lock()
			isSaturated = isSaturated || count == int64(limit)
			mu.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return false, err
	}

	return isSaturated, nil
}

func (r *olapRepository) UpdateDAGStatuses(ctx context.Context, tenantId string) (bool, error) {
	var limit int32 = 10000

	// each partition gets its own goroutine
	eg := &errgroup.Group{}
	mu := sync.Mutex{}

	// if any of the partitions are saturated, we return true
	isSaturated := false

	for i := 0; i < NUM_PARTITIONS; i++ {
		partitionNumber := i

		eg.Go(func() error {
			tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

			if err != nil {
				return err
			}

			defer rollback()

			count, err := r.queries.UpdateDAGStatuses(ctx, tx, sqlcv1.UpdateDAGStatusesParams{
				Partitionnumber: int32(partitionNumber), // nolint: gosec
				Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
				Eventlimit:      limit,
			})

			if err != nil {
				return err
			}

			if err := commit(ctx); err != nil {
				return err
			}

			mu.Lock()
			isSaturated = isSaturated || count == int64(limit)
			mu.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return false, err
	}

	return isSaturated, nil
}

func (r *olapRepository) writeTaskBatch(ctx context.Context, tenantId string, tasks []*sqlcv1.V1Task) error {
	params := make([]sqlcv1.CreateTasksOLAPParams, 0)

	for _, task := range tasks {
		params = append(params, sqlcv1.CreateTasksOLAPParams{
			TenantID:             task.TenantID,
			ID:                   task.ID,
			InsertedAt:           task.InsertedAt,
			Queue:                task.Queue,
			ActionID:             task.ActionID,
			StepID:               task.StepID,
			WorkflowID:           task.WorkflowID,
			WorkflowVersionID:    task.WorkflowVersionID,
			ScheduleTimeout:      task.ScheduleTimeout,
			StepTimeout:          task.StepTimeout,
			Priority:             task.Priority,
			Sticky:               sqlcv1.V1StickyStrategyOlap(task.Sticky),
			DesiredWorkerID:      task.DesiredWorkerID,
			ExternalID:           task.ExternalID,
			DisplayName:          task.DisplayName,
			Input:                task.Input,
			AdditionalMetadata:   task.AdditionalMetadata,
			DagID:                task.DagID,
			DagInsertedAt:        task.DagInsertedAt,
			ParentTaskExternalID: task.ParentTaskExternalID,
		})
	}

	_, err := r.queries.CreateTasksOLAP(ctx, r.pool, params)

	return err
}

func (r *olapRepository) writeDAGBatch(ctx context.Context, tenantId string, dags []*DAGWithData) error {
	params := make([]sqlcv1.CreateDAGsOLAPParams, 0)

	for _, dag := range dags {
		var parentTaskExternalID = pgtype.UUID{}
		if dag.ParentTaskExternalID != nil {
			parentTaskExternalID = *dag.ParentTaskExternalID
		}

		params = append(params, sqlcv1.CreateDAGsOLAPParams{
			TenantID:             dag.TenantID,
			ID:                   dag.ID,
			InsertedAt:           dag.InsertedAt,
			WorkflowID:           dag.WorkflowID,
			WorkflowVersionID:    dag.WorkflowVersionID,
			ExternalID:           dag.ExternalID,
			DisplayName:          dag.DisplayName,
			Input:                dag.Input,
			AdditionalMetadata:   dag.AdditionalMetadata,
			ParentTaskExternalID: parentTaskExternalID,
		})
	}

	_, err := r.queries.CreateDAGsOLAP(ctx, r.pool, params)

	return err
}

func (r *olapRepository) CreateTaskEvents(ctx context.Context, tenantId string, events []sqlcv1.CreateTaskEventsOLAPParams) error {
	return r.writeTaskEventBatch(ctx, tenantId, events)
}

func (r *olapRepository) CreateTasks(ctx context.Context, tenantId string, tasks []*sqlcv1.V1Task) error {
	return r.writeTaskBatch(ctx, tenantId, tasks)
}

func (r *olapRepository) CreateDAGs(ctx context.Context, tenantId string, dags []*DAGWithData) error {
	return r.writeDAGBatch(ctx, tenantId, dags)
}

func (r *olapRepository) GetTaskPointMetrics(ctx context.Context, tenantId string, startTimestamp *time.Time, endTimestamp *time.Time, bucketInterval time.Duration) ([]*sqlcv1.GetTaskPointMetricsRow, error) {
	rows, err := r.queries.GetTaskPointMetrics(ctx, r.pool, sqlcv1.GetTaskPointMetricsParams{
		Interval:      durationToPgInterval(bucketInterval),
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Createdafter:  sqlchelpers.TimestamptzFromTime(*startTimestamp),
		Createdbefore: sqlchelpers.TimestamptzFromTime(*endTimestamp),
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*sqlcv1.GetTaskPointMetricsRow{}, nil
		}

		return nil, err
	}

	return rows, nil
}

func (r *olapRepository) ReadDAG(ctx context.Context, dagExternalId string) (*sqlcv1.V1DagsOlap, error) {
	return r.queries.ReadDAGByExternalID(ctx, r.pool, sqlchelpers.UUIDFromStr(dagExternalId))
}

func (r *olapRepository) ListTasksByExternalIds(ctx context.Context, tenantId string, externalIds []string) ([]*sqlcv1.FlattenTasksByExternalIdsRow, error) {
	externalUUIDs := make([]pgtype.UUID, 0)

	for _, id := range externalIds {
		externalUUIDs = append(externalUUIDs, sqlchelpers.UUIDFromStr(id))
	}

	return r.queries.FlattenTasksByExternalIds(ctx, r.pool, sqlcv1.FlattenTasksByExternalIdsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Externalids: externalUUIDs,
	})
}

func durationToPgInterval(d time.Duration) pgtype.Interval {
	// Convert the time.Duration to microseconds
	microseconds := d.Microseconds()

	return pgtype.Interval{
		Microseconds: microseconds,
		Valid:        true,
	}
}
