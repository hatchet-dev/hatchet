package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/config/limits"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/validator"
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

	TriggeringEventExternalId *uuid.UUID

	Limit int64

	Offset int64

	IncludePayloads bool
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

	ParentTaskExternalId *uuid.UUID

	TriggeringEventExternalId *uuid.UUID

	IncludePayloads bool
}

type ReadTaskRunMetricsOpts struct {
	CreatedAfter time.Time

	CreatedBefore *time.Time

	WorkflowIds []uuid.UUID

	ParentTaskExternalID *uuid.UUID

	TriggeringEventExternalId *uuid.UUID

	AdditionalMetadata map[string]interface{}
}

type WorkflowRunData struct {
	AdditionalMetadata   []byte                      `json:"additional_metadata"`
	CreatedAt            pgtype.Timestamptz          `json:"created_at"`
	DisplayName          string                      `json:"display_name"`
	ErrorMessage         string                      `json:"error_message"`
	ExternalID           uuid.UUID                   `json:"external_id"`
	FinishedAt           pgtype.Timestamptz          `json:"finished_at"`
	Input                []byte                      `json:"input"`
	InsertedAt           pgtype.Timestamptz          `json:"inserted_at"`
	Kind                 sqlcv1.V1RunKind            `json:"kind"`
	Output               []byte                      `json:"output,omitempty"`
	ParentTaskExternalId *uuid.UUID                  `json:"parent_task_external_id,omitempty"`
	ReadableStatus       sqlcv1.V1ReadableStatusOlap `json:"readable_status"`
	StepId               *uuid.UUID                  `json:"step_id,omitempty"`
	StartedAt            pgtype.Timestamptz          `json:"started_at"`
	TaskExternalId       *uuid.UUID                  `json:"task_external_id,omitempty"`
	TaskId               *int64                      `json:"task_id,omitempty"`
	TaskInsertedAt       *pgtype.Timestamptz         `json:"task_inserted_at,omitempty"`
	TenantID             uuid.UUID                   `json:"tenant_id"`
	WorkflowID           uuid.UUID                   `json:"workflow_id"`
	WorkflowVersionId    uuid.UUID                   `json:"workflow_version_id"`
	RetryCount           *int                        `json:"retry_count,omitempty"`
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

type UpdateTaskStatusRow struct {
	TenantId       uuid.UUID
	TaskId         int64
	TaskInsertedAt pgtype.Timestamptz
	ReadableStatus sqlcv1.V1ReadableStatusOlap
	ExternalId     uuid.UUID
	LatestWorkerId uuid.UUID
	WorkflowId     uuid.UUID
	IsDAGTask      bool
}

type UpdateDAGStatusRow struct {
	TenantId       uuid.UUID
	DagId          int64
	DagInsertedAt  pgtype.Timestamptz
	ReadableStatus sqlcv1.V1ReadableStatusOlap
	ExternalId     uuid.UUID
	WorkflowId     uuid.UUID
}

type TaskWithPayloads struct {
	*sqlcv1.PopulateTaskRunDataRow
	InputPayload       []byte
	OutputPayload      []byte
	NumSpawnedChildren int64
}

type TaskEventWithPayloads struct {
	*sqlcv1.ListTaskEventsForWorkflowRunRow
	OutputPayload []byte
}

type OLAPRepository interface {
	UpdateTablePartitions(ctx context.Context) error
	SetReadReplicaPool(pool *pgxpool.Pool)

	ReadTaskRun(ctx context.Context, taskExternalId uuid.UUID) (*sqlcv1.V1TasksOlap, error)
	ReadWorkflowRun(ctx context.Context, workflowRunExternalId uuid.UUID) (*V1WorkflowRunPopulator, error)
	ReadTaskRunData(ctx context.Context, tenantId uuid.UUID, taskId int64, taskInsertedAt pgtype.Timestamptz, retryCount *int) (*TaskWithPayloads, uuid.UUID, error)

	ListTasks(ctx context.Context, tenantId uuid.UUID, opts ListTaskRunOpts) ([]*TaskWithPayloads, int, error)
	ListWorkflowRuns(ctx context.Context, tenantId uuid.UUID, opts ListWorkflowRunOpts) ([]*WorkflowRunData, int, error)
	ListTaskRunEvents(ctx context.Context, tenantId uuid.UUID, taskId int64, taskInsertedAt pgtype.Timestamptz, limit, offset int64) ([]*sqlcv1.ListTaskEventsRow, error)
	ListTaskRunEventsByWorkflowRunId(ctx context.Context, tenantId uuid.UUID, workflowRunId uuid.UUID) ([]*TaskEventWithPayloads, error)
	ListWorkflowRunDisplayNames(ctx context.Context, tenantId uuid.UUID, externalIds []uuid.UUID) ([]*sqlcv1.ListWorkflowRunDisplayNamesRow, error)
	ReadTaskRunMetrics(ctx context.Context, tenantId uuid.UUID, opts ReadTaskRunMetricsOpts) ([]TaskRunMetric, error)
	CreateTasks(ctx context.Context, tenantId uuid.UUID, tasks []*V1TaskWithPayload) error
	CreateTaskEvents(ctx context.Context, tenantId uuid.UUID, events []sqlcv1.CreateTaskEventsOLAPParams) error
	CreateDAGs(ctx context.Context, tenantId uuid.UUID, dags []*DAGWithData) error
	GetTaskPointMetrics(ctx context.Context, tenantId uuid.UUID, startTimestamp *time.Time, endTimestamp *time.Time, bucketInterval time.Duration) ([]*sqlcv1.GetTaskPointMetricsRow, error)
	UpdateTaskStatuses(ctx context.Context, tenantIds []uuid.UUID) (bool, []UpdateTaskStatusRow, error)
	UpdateDAGStatuses(ctx context.Context, tenantIds []uuid.UUID) (bool, []UpdateDAGStatusRow, error)
	ReadDAG(ctx context.Context, dagExternalId uuid.UUID) (*sqlcv1.V1DagsOlap, error)
	ListTasksByDAGId(ctx context.Context, tenantId uuid.UUID, dagIds []uuid.UUID, includePayloads bool) ([]*TaskWithPayloads, map[int64]uuid.UUID, error)
	ListTasksByIdAndInsertedAt(ctx context.Context, tenantId uuid.UUID, taskMetadata []TaskMetadata, includePayloads bool) ([]*TaskWithPayloads, error)

	// ListTasksByExternalIds returns a list of tasks based on their external ids or the external id of their parent DAG.
	// In the case of a DAG, we flatten the result into the list of tasks which belong to that DAG.
	ListTasksByExternalIds(ctx context.Context, tenantId uuid.UUID, externalIds []uuid.UUID) ([]*sqlcv1.FlattenTasksByExternalIdsRow, error)

	GetTaskTimings(ctx context.Context, tenantId, workflowRunId uuid.UUID, depth int32) ([]*sqlcv1.PopulateTaskRunDataRow, map[uuid.UUID]int32, error)

	// Events queries
	BulkCreateEventsAndTriggers(ctx context.Context, events sqlcv1.BulkCreateEventsOLAPParams, triggers []EventTriggersFromExternalId) error
	ListEvents(ctx context.Context, opts sqlcv1.ListEventsParams) ([]*EventWithPayload, *int64, error)
	GetEvent(ctx context.Context, externalId uuid.UUID) (*sqlcv1.V1EventsOlap, error)
	GetEventWithPayload(ctx context.Context, externalId, tenantId uuid.UUID) (*EventWithPayload, error)
	ListEventKeys(ctx context.Context, tenantId uuid.UUID) ([]string, error)

	GetDAGDurations(ctx context.Context, tenantId uuid.UUID, externalIds []uuid.UUID, minInsertedAt pgtype.Timestamptz) (map[string]*sqlcv1.GetDagDurationsRow, error)
	GetTaskDurationsByTaskIds(ctx context.Context, tenantId uuid.UUID, taskIds []int64, taskInsertedAts []pgtype.Timestamptz, readableStatuses []sqlcv1.V1ReadableStatusOlap) (map[int64]*sqlcv1.GetTaskDurationsByTaskIdsRow, error)

	CreateIncomingWebhookValidationFailureLogs(ctx context.Context, tenantId uuid.UUID, opts []CreateIncomingWebhookFailureLogOpts) error
	StoreCELEvaluationFailures(ctx context.Context, tenantId uuid.UUID, failures []CELEvaluationFailure) error
	PutPayloads(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, putPayloadOpts ...StoreOLAPPayloadOpts) (map[uuid.UUID]ExternalPayloadLocationKey, error)
	ReadPayload(ctx context.Context, tenantId uuid.UUID, externalId uuid.UUID) ([]byte, error)
	ReadPayloads(ctx context.Context, tenantId uuid.UUID, externalIds ...uuid.UUID) (map[uuid.UUID][]byte, error)

	AnalyzeOLAPTables(ctx context.Context) error
	OffloadPayloads(ctx context.Context, tenantId uuid.UUID, payloads []OffloadPayloadOpts) error

	PayloadStore() PayloadStoreRepository
	StatusUpdateBatchSizeLimits() StatusUpdateBatchSizeLimits

	ListWorkflowRunExternalIds(ctx context.Context, tenantId uuid.UUID, opts ListWorkflowRunOpts) ([]uuid.UUID, error)

	ProcessOLAPPayloadCutovers(ctx context.Context, externalStoreEnabled bool, inlineStoreTTL *time.Duration, externalCutoverBatchSize, externalCutoverNumConcurrentOffloads int32) error

	CountOLAPTempTableSizeForDAGStatusUpdates(ctx context.Context) (int64, error)
	CountOLAPTempTableSizeForTaskStatusUpdates(ctx context.Context) (int64, error)
	ListYesterdayRunCountsByStatus(ctx context.Context) (map[sqlcv1.V1ReadableStatusOlap]int64, error)
}

type StatusUpdateBatchSizeLimits struct {
	Task int32
	DAG  int32
}
type OLAPRepositoryImpl struct {
	*sharedRepository

	readPool *pgxpool.Pool

	eventCache *lru.Cache[string, bool]

	olapRetentionPeriod time.Duration

	shouldPartitionEventsTables bool

	statusUpdateBatchSizeLimits StatusUpdateBatchSizeLimits
}

func NewOLAPRepositoryFromPool(
	pool *pgxpool.Pool,
	l *zerolog.Logger,
	olapRetentionPeriod time.Duration,
	tenantLimitConfig limits.LimitConfigFile, enforceLimits bool,
	shouldPartitionEventsTables bool,
	payloadStoreOpts PayloadStoreRepositoryOpts,
	statusUpdateBatchSizeLimits StatusUpdateBatchSizeLimits,
	cacheDuration time.Duration,
	enableDurableUserEventLog bool,
) (OLAPRepository, func() error) {
	v := validator.NewDefaultValidator()

	shared, cleanupShared := newSharedRepository(pool, v, l, payloadStoreOpts, tenantLimitConfig, enforceLimits, cacheDuration, enableDurableUserEventLog)

	return newOLAPRepository(shared, olapRetentionPeriod, shouldPartitionEventsTables, statusUpdateBatchSizeLimits), cleanupShared
}

func newOLAPRepository(shared *sharedRepository, olapRetentionPeriod time.Duration, shouldPartitionEventsTables bool, statusUpdateBatchSizeLimits StatusUpdateBatchSizeLimits) OLAPRepository {
	eventCache, err := lru.New[string, bool](100000)

	if err != nil {
		log.Fatal(err)
	}

	return &OLAPRepositoryImpl{
		sharedRepository:            shared,
		readPool:                    shared.pool,
		eventCache:                  eventCache,
		olapRetentionPeriod:         olapRetentionPeriod,
		shouldPartitionEventsTables: shouldPartitionEventsTables,
		statusUpdateBatchSizeLimits: statusUpdateBatchSizeLimits,
	}
}

func (r *OLAPRepositoryImpl) UpdateTablePartitions(ctx context.Context) error {
	today := time.Now().UTC()
	tomorrow := today.AddDate(0, 0, 1)
	removeBefore := today.Add(-1 * r.olapRetentionPeriod)

	err := r.queries.CreateOLAPPartitions(ctx, r.pool, sqlcv1.CreateOLAPPartitionsParams{
		Date: pgtype.Date{
			Time:  today,
			Valid: true,
		},
		Partitions: NUM_PARTITIONS,
	})

	if err != nil {
		return err
	}

	if r.shouldPartitionEventsTables {
		err = r.queries.CreateOLAPEventPartitions(ctx, r.pool, pgtype.Date{
			Time:  today,
			Valid: true,
		})

		if err != nil {
			return err
		}
	}

	err = r.queries.CreateOLAPPartitions(ctx, r.pool, sqlcv1.CreateOLAPPartitionsParams{
		Date: pgtype.Date{
			Time:  tomorrow,
			Valid: true,
		},
		Partitions: NUM_PARTITIONS,
	})

	if err != nil {
		return err
	}

	if r.shouldPartitionEventsTables {
		err = r.queries.CreateOLAPEventPartitions(ctx, r.pool, pgtype.Date{
			Time:  tomorrow,
			Valid: true,
		})

		if err != nil {
			return err
		}
	}

	params := sqlcv1.ListOLAPPartitionsBeforeDateParams{
		Shouldpartitioneventstables: r.shouldPartitionEventsTables,
		Date: pgtype.Date{
			Time:  removeBefore,
			Valid: true,
		},
	}

	partitions, err := r.queries.ListOLAPPartitionsBeforeDate(ctx, r.pool, params)

	if err != nil {
		return err
	}

	if len(partitions) > 0 {
		r.l.Warn().Msgf("removing partitions before %s using retention period of %s", removeBefore.Format(time.RFC3339), r.olapRetentionPeriod)
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

	return nil
}

func (r *OLAPRepositoryImpl) SetReadReplicaPool(pool *pgxpool.Pool) {
	r.readPool = pool
}

func (r *OLAPRepositoryImpl) PayloadStore() PayloadStoreRepository {
	return r.payloadStore
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

func (r *OLAPRepositoryImpl) ReadTaskRun(ctx context.Context, taskExternalId uuid.UUID) (*sqlcv1.V1TasksOlap, error) {
	row, err := r.queries.ReadTaskByExternalID(ctx, r.readPool, taskExternalId)

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

	if len(jsonData) == 0 {
		return tasks, nil
	}

	err := json.Unmarshal(jsonData, &tasks)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *OLAPRepositoryImpl) ReadWorkflowRun(ctx context.Context, workflowRunExternalId uuid.UUID) (*V1WorkflowRunPopulator, error) {
	row, err := r.queries.ReadWorkflowRunByExternalId(ctx, r.readPool, workflowRunExternalId)

	if err != nil {
		return nil, err
	}

	taskMetadata, err := ParseTaskMetadata(row.TaskMetadata)

	if err != nil {
		return nil, err
	}

	inputPayload, err := r.ReadPayload(ctx, row.TenantID, row.ExternalID)

	if err != nil {
		return nil, err
	}

	return &V1WorkflowRunPopulator{
		WorkflowRun: &WorkflowRunData{
			TenantID:             row.TenantID,
			InsertedAt:           row.InsertedAt,
			ExternalID:           row.ExternalID,
			ReadableStatus:       row.ReadableStatus,
			Kind:                 row.Kind,
			WorkflowID:           row.WorkflowID,
			DisplayName:          row.DisplayName,
			AdditionalMetadata:   row.AdditionalMetadata,
			CreatedAt:            row.CreatedAt,
			StartedAt:            row.StartedAt,
			FinishedAt:           row.FinishedAt,
			ErrorMessage:         row.ErrorMessage.String,
			WorkflowVersionId:    row.WorkflowVersionID,
			Input:                inputPayload,
			ParentTaskExternalId: row.ParentTaskExternalID,
		},
		TaskMetadata: taskMetadata,
	}, nil
}

func (r *OLAPRepositoryImpl) ReadTaskRunData(ctx context.Context, tenantId uuid.UUID, taskId int64, taskInsertedAt pgtype.Timestamptz, retryCount *int) (*TaskWithPayloads, uuid.UUID, error) {
	emptyUUID := uuid.UUID{}

	params := sqlcv1.PopulateSingleTaskRunDataParams{
		Taskid:         taskId,
		Tenantid:       tenantId,
		Taskinsertedat: taskInsertedAt,
	}

	if retryCount != nil {
		params.RetryCount = pgtype.Int4{Int32: int32(*retryCount), Valid: true}
	}

	taskRun, err := r.queries.PopulateSingleTaskRunData(ctx, r.readPool, params)

	if err != nil {
		return nil, emptyUUID, err
	}

	var workflowRunId uuid.UUID

	if taskRun.DagID.Valid {
		dagId := taskRun.DagID.Int64
		dagInsertedAt := taskRun.DagInsertedAt

		workflowRunId, err = r.queries.GetWorkflowRunIdFromDagIdInsertedAt(ctx, r.readPool, sqlcv1.GetWorkflowRunIdFromDagIdInsertedAtParams{
			Dagid:         dagId,
			Daginsertedat: dagInsertedAt,
		})

		if err != nil {
			return nil, emptyUUID, err
		}
	} else {
		workflowRunId = taskRun.ExternalID
	}

	externalIds := make([]uuid.UUID, 0)

	if taskRun.OutputEventExternalID != nil {
		externalIds = append(externalIds, *taskRun.OutputEventExternalID)
	}

	payloads, err := r.readPayloads(ctx, r.readPool, tenantId, externalIds...)

	if err != nil {
		return nil, emptyUUID, err
	}

	input, exists := payloads[workflowRunId]

	if !exists {
		input = taskRun.Input
	}

	var output []byte
	if taskRun.OutputEventExternalID != nil {
		output, exists = payloads[*taskRun.OutputEventExternalID]

		if !exists {
			output = taskRun.Output
		}
	} else {
		output = taskRun.Output
	}

	return &TaskWithPayloads{
		&sqlcv1.PopulateTaskRunDataRow{
			TenantID:              taskRun.TenantID,
			ID:                    taskRun.ID,
			InsertedAt:            taskRun.InsertedAt,
			ExternalID:            taskRun.ExternalID,
			Queue:                 taskRun.Queue,
			ActionID:              taskRun.ActionID,
			StepID:                taskRun.StepID,
			WorkflowID:            taskRun.WorkflowID,
			WorkflowVersionID:     taskRun.WorkflowVersionID,
			ScheduleTimeout:       taskRun.ScheduleTimeout,
			StepTimeout:           taskRun.StepTimeout,
			Priority:              taskRun.Priority,
			Sticky:                taskRun.Sticky,
			DisplayName:           taskRun.DisplayName,
			AdditionalMetadata:    taskRun.AdditionalMetadata,
			ParentTaskExternalID:  taskRun.ParentTaskExternalID,
			Status:                taskRun.Status,
			WorkflowRunID:         workflowRunId,
			FinishedAt:            taskRun.FinishedAt,
			StartedAt:             taskRun.StartedAt,
			QueuedAt:              taskRun.QueuedAt,
			ErrorMessage:          taskRun.ErrorMessage,
			RetryCount:            taskRun.RetryCount,
			OutputEventExternalID: taskRun.OutputEventExternalID,
		},
		input,
		output,
		taskRun.SpawnedChildren.Int64,
	}, workflowRunId, nil
}

func (r *OLAPRepositoryImpl) ListTasks(ctx context.Context, tenantId uuid.UUID, opts ListTaskRunOpts) ([]*TaskWithPayloads, int, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-tasks-olap")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.readPool, r.l)

	if err != nil {
		return nil, 0, err
	}

	defer rollback()

	params := sqlcv1.ListTasksOlapParams{
		Tenantid:                  tenantId,
		Since:                     sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		Tasklimit:                 int32(opts.Limit),
		Taskoffset:                int32(opts.Offset),
		TriggeringEventExternalId: opts.TriggeringEventExternalId,
		WorkerId:                  opts.WorkerId,
	}

	countParams := sqlcv1.CountTasksParams{
		Tenantid:                  tenantId,
		Since:                     sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		TriggeringEventExternalId: opts.TriggeringEventExternalId,
		WorkerId:                  opts.WorkerId,
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
		workflowIdParams := make([]uuid.UUID, 0)

		for _, id := range opts.WorkflowIds {
			workflowIdParams = append(workflowIdParams, id)
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

	var (
		rows     []*sqlcv1.ListTasksOlapRow
		count    int64
		countErr error
	)

	// A pgx.Tx must not be used concurrently, so we run the count query against the pool.
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		rows, err = r.queries.ListTasksOlap(gctx, tx, params)
		return err
	})

	g.Go(func() error {
		count, countErr = r.queries.CountTasks(gctx, r.readPool, countParams)
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, 0, err
	}

	idsInsertedAts := make([]IdInsertedAt, 0, len(rows))

	for _, row := range rows {
		idsInsertedAts = append(idsInsertedAts, IdInsertedAt{
			ID:         row.ID,
			InsertedAt: row.InsertedAt,
		})
	}

	tasksWithData, err := r.populateTaskRunData(ctx, tx, tenantId, idsInsertedAts, opts.IncludePayloads)

	if err != nil {
		return nil, 0, err
	}

	payloads := make(map[uuid.UUID][]byte)

	if opts.IncludePayloads {
		externalIds := make([]uuid.UUID, 0)
		for _, task := range tasksWithData {
			externalIds = append(externalIds, task.ExternalID)

			if task.OutputEventExternalID != nil {
				externalIds = append(externalIds, *task.OutputEventExternalID)
			}
		}

		payloads, err = r.readPayloads(ctx, tx, tenantId, externalIds...)

		if err != nil {
			return nil, 0, err
		}
	}

	result := make([]*TaskWithPayloads, 0, len(tasksWithData))

	for _, task := range tasksWithData {
		input, exists := payloads[task.ExternalID]

		if !exists {
			input = task.Input
		}

		var output []byte

		if task.OutputEventExternalID != nil {
			output, exists = payloads[*task.OutputEventExternalID]

			if !exists {
				output = task.Output
			}
		} else {
			output = task.Output
		}

		result = append(result, &TaskWithPayloads{
			task,
			input,
			output,
			int64(0),
		})
	}

	if countErr != nil {
		count = int64(len(tasksWithData))
	}

	if err := commit(ctx); err != nil {
		return nil, 0, err
	}

	return result, int(count), nil
}

func (r *OLAPRepositoryImpl) ListTasksByDAGId(ctx context.Context, tenantId uuid.UUID, dagids []uuid.UUID, includePayloads bool) ([]*TaskWithPayloads, map[int64]uuid.UUID, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-tasks-by-dag-id-olap")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.readPool, r.l)
	taskIdToDagExternalId := make(map[int64]uuid.UUID)

	if err != nil {
		return nil, taskIdToDagExternalId, err
	}

	defer rollback()

	tasks, err := r.queries.ListTasksByDAGIds(ctx, tx, sqlcv1.ListTasksByDAGIdsParams{
		Dagids:   dagids,
		Tenantid: tenantId,
	})

	if err != nil {
		return nil, taskIdToDagExternalId, err
	}

	idsInsertedAts := make([]IdInsertedAt, 0, len(tasks))

	for _, row := range tasks {
		taskIdToDagExternalId[row.TaskID] = row.DagExternalID
		idsInsertedAts = append(idsInsertedAts, IdInsertedAt{
			ID:         row.TaskID,
			InsertedAt: row.TaskInsertedAt,
		})
	}

	tasksWithData, err := r.populateTaskRunData(ctx, tx, tenantId, idsInsertedAts, includePayloads)

	if err != nil {
		return nil, taskIdToDagExternalId, err
	}

	payloads := make(map[uuid.UUID][]byte)

	if includePayloads {
		externalIds := make([]uuid.UUID, 0)
		for _, task := range tasksWithData {
			externalIds = append(externalIds, task.ExternalID)

			if task.OutputEventExternalID != nil {
				externalIds = append(externalIds, *task.OutputEventExternalID)
			}
		}

		payloads, err = r.readPayloads(ctx, tx, tenantId, externalIds...)

		if err != nil {
			return nil, taskIdToDagExternalId, err
		}
	}

	result := make([]*TaskWithPayloads, 0, len(tasksWithData))

	for _, task := range tasksWithData {
		input, exists := payloads[task.ExternalID]

		if !exists {
			input = task.Input
		}

		var output []byte

		if task.OutputEventExternalID != nil {
			output, exists = payloads[*task.OutputEventExternalID]

			if !exists {
				output = task.Output
			}
		} else {
			output = task.Output
		}

		result = append(result, &TaskWithPayloads{
			task,
			input,
			output,
			int64(0),
		})
	}

	if err := commit(ctx); err != nil {
		return nil, taskIdToDagExternalId, err
	}

	return result, taskIdToDagExternalId, nil
}

func (r *OLAPRepositoryImpl) ListTasksByIdAndInsertedAt(ctx context.Context, tenantId uuid.UUID, taskMetadata []TaskMetadata, includePayloads bool) ([]*TaskWithPayloads, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-tasks-by-id-and-inserted-at-olap")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.readPool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	idsInsertedAts := make([]IdInsertedAt, 0, len(taskMetadata))

	for _, metadata := range taskMetadata {
		idsInsertedAts = append(idsInsertedAts, IdInsertedAt{
			ID:         metadata.TaskID,
			InsertedAt: pgtype.Timestamptz{Time: metadata.TaskInsertedAt, Valid: true},
		})
	}

	tasksWithData, err := r.populateTaskRunData(ctx, tx, tenantId, idsInsertedAts, includePayloads)

	if err != nil {
		return nil, err
	}

	payloads := make(map[uuid.UUID][]byte)

	if includePayloads {
		externalIds := make([]uuid.UUID, 0)
		for _, task := range tasksWithData {
			externalIds = append(externalIds, task.ExternalID)

			if task.OutputEventExternalID != nil {

				externalIds = append(externalIds, *task.OutputEventExternalID)
			}
		}

		payloads, err = r.readPayloads(ctx, tx, tenantId, externalIds...)

		if err != nil {
			return nil, err
		}
	}

	result := make([]*TaskWithPayloads, 0, len(tasksWithData))

	for _, task := range tasksWithData {
		input, exists := payloads[task.ExternalID]

		if !exists {
			input = task.Input
		}

		var output []byte
		if task.OutputEventExternalID != nil {
			output, exists = payloads[*task.OutputEventExternalID]

			if !exists {
				output = task.Output
			}
		} else {
			output = task.Output
		}

		result = append(result, &TaskWithPayloads{
			task,
			input,
			output,
			int64(0),
		})
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *OLAPRepositoryImpl) ListWorkflowRuns(ctx context.Context, tenantId uuid.UUID, opts ListWorkflowRunOpts) ([]*WorkflowRunData, int, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-workflow-runs-olap")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.readPool, r.l)

	if err != nil {
		return nil, 0, err
	}

	defer rollback()

	params := sqlcv1.FetchWorkflowRunIdsParams{
		Tenantid:                  tenantId,
		Since:                     sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		Listworkflowrunslimit:     int32(opts.Limit),
		Listworkflowrunsoffset:    int32(opts.Offset),
		ParentTaskExternalId:      opts.ParentTaskExternalId,
		TriggeringEventExternalId: opts.TriggeringEventExternalId,
	}

	countParams := sqlcv1.CountWorkflowRunsParams{
		Tenantid: tenantId,
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
		workflowIdParams := make([]uuid.UUID, 0)

		for _, id := range opts.WorkflowIds {
			workflowIdParams = append(workflowIdParams, id)
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

	var (
		workflowRunIds []*sqlcv1.FetchWorkflowRunIdsRow
		count          int64
		countErr       error
	)

	// A pgx.Tx must not be used concurrently; run count on the pool in the background while we do tx work.
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		count, countErr = r.queries.CountWorkflowRuns(gctx, r.readPool, countParams)
		return nil
	})

	workflowRunIds, err = r.queries.FetchWorkflowRunIds(ctx, tx, params)
	if err != nil {
		return nil, 0, err
	}

	runIdsWithDAGs := make([]int64, 0)
	runInsertedAtsWithDAGs := make([]pgtype.Timestamptz, 0)
	idsInsertedAts := make([]IdInsertedAt, 0, len(workflowRunIds))
	externalIdsForPayloads := make([]uuid.UUID, 0)

	for _, row := range workflowRunIds {
		if row.Kind == sqlcv1.V1RunKindDAG {
			runIdsWithDAGs = append(runIdsWithDAGs, row.ID)
			runInsertedAtsWithDAGs = append(runInsertedAtsWithDAGs, row.InsertedAt)
		} else {
			idsInsertedAts = append(idsInsertedAts, IdInsertedAt{
				ID:         row.ID,
				InsertedAt: row.InsertedAt,
			})
		}
	}

	populatedDAGs, err := r.queries.PopulateDAGMetadata(ctx, tx, sqlcv1.PopulateDAGMetadataParams{
		Ids:             runIdsWithDAGs,
		Insertedats:     runInsertedAtsWithDAGs,
		Tenantid:        tenantId,
		Includepayloads: opts.IncludePayloads,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, err
	}

	dagsToPopulated := make(map[string]*sqlcv1.PopulateDAGMetadataRow)

	for _, dag := range populatedDAGs {
		externalId := dag.ExternalID.String()

		dagsToPopulated[externalId] = dag
		externalIdsForPayloads = append(externalIdsForPayloads, dag.ExternalID)

		if dag.OutputEventExternalID != nil {
			externalIdsForPayloads = append(externalIdsForPayloads, *dag.OutputEventExternalID)
		}
	}

	tasksToPopulated := make(map[string]*sqlcv1.PopulateTaskRunDataRow)

	populatedTasks, err := r.populateTaskRunData(ctx, tx, tenantId, idsInsertedAts, opts.IncludePayloads)

	if err != nil {
		return nil, 0, err
	}

	for _, task := range populatedTasks {
		externalId := task.ExternalID.String()
		tasksToPopulated[externalId] = task

		externalIdsForPayloads = append(externalIdsForPayloads, task.ExternalID)

		if task.OutputEventExternalID != nil {
			externalIdsForPayloads = append(externalIdsForPayloads, *task.OutputEventExternalID)
		}
	}

	if err := commit(ctx); err != nil {
		return nil, 0, err
	}

	// Join the count goroutine before returning.
	if err := g.Wait(); err != nil {
		return nil, 0, err
	}

	if countErr != nil {
		r.l.Error().Msgf("error counting workflow runs: %v", countErr)
		count = int64(len(workflowRunIds))
	}

	externalIdToPayload := make(map[uuid.UUID][]byte)

	if opts.IncludePayloads {
		externalIdToPayload, err = r.readPayloads(ctx, r.readPool, tenantId, externalIdsForPayloads...)

		if err != nil {
			return nil, 0, err
		}
	}

	res := make([]*WorkflowRunData, 0)

	for _, row := range workflowRunIds {
		externalId := row.ExternalID.String()

		if row.Kind == sqlcv1.V1RunKindDAG {
			dag, ok := dagsToPopulated[externalId]

			if !ok {
				r.l.Error().Msgf("could not find dag with external id %s", externalId)
				continue
			}

			var outputPayload []byte
			var exists bool

			if dag.OutputEventExternalID != nil {
				outputPayload, exists = externalIdToPayload[*dag.OutputEventExternalID]

				if !exists {
					if opts.IncludePayloads && dag.OutputEventExternalID != nil && dag.ReadableStatus == sqlcv1.V1ReadableStatusOlapCOMPLETED {
						r.l.Error().Msgf("ListWorkflowRuns-1: dag with external_id %s and inserted_at %s has empty payload, falling back to output", dag.ExternalID, dag.InsertedAt.Time)
					}
					outputPayload = dag.Output
				}
			} else {
				outputPayload = dag.Output
			}

			inputPayload, exists := externalIdToPayload[dag.ExternalID]
			if !exists {
				if opts.IncludePayloads && dag.ExternalID != uuid.Nil {
					r.l.Error().Msgf("ListWorkflowRuns-2: dag with external_id %s and inserted_at %s has empty payload, falling back to input", dag.ExternalID, dag.InsertedAt.Time)
				}
				inputPayload = dag.Input
			}

			// TODO !IMPORTANT: verify this is correct
			retryCount := int(dag.RetryCount)

			res = append(res, &WorkflowRunData{
				TenantID:             dag.TenantID,
				InsertedAt:           dag.InsertedAt,
				ExternalID:           dag.ExternalID,
				WorkflowID:           dag.WorkflowID,
				DisplayName:          dag.DisplayName,
				ReadableStatus:       dag.ReadableStatus,
				AdditionalMetadata:   dag.AdditionalMetadata,
				CreatedAt:            dag.CreatedAt,
				StartedAt:            dag.StartedAt,
				FinishedAt:           dag.FinishedAt,
				ErrorMessage:         dag.ErrorMessage.String,
				Kind:                 sqlcv1.V1RunKindDAG,
				WorkflowVersionId:    dag.WorkflowVersionID,
				TaskExternalId:       nil,
				TaskId:               nil,
				TaskInsertedAt:       nil,
				Output:               outputPayload,
				Input:                inputPayload,
				ParentTaskExternalId: dag.ParentTaskExternalID,
				RetryCount:           &retryCount,
			})
		} else {
			task, ok := tasksToPopulated[externalId]

			if !ok {
				r.l.Error().Msgf("could not find task with external id %s", externalId)
				continue
			}

			retryCount := int(task.RetryCount)

			var outputPayload []byte
			var exists bool

			if task.OutputEventExternalID != nil {
				outputPayload, exists = externalIdToPayload[*task.OutputEventExternalID]

				if !exists {
					if opts.IncludePayloads && task.OutputEventExternalID != nil && task.Status == sqlcv1.V1ReadableStatusOlapCOMPLETED {
						r.l.Error().Msgf("ListWorkflowRuns-3: task with external_id %s and inserted_at %s has empty payload, falling back to output", task.ExternalID, task.InsertedAt.Time)
					}
					outputPayload = task.Output
				}
			} else {
				outputPayload = task.Output
			}

			inputPayload, exists := externalIdToPayload[task.ExternalID]

			if !exists {
				if opts.IncludePayloads && task.ExternalID != uuid.Nil {
					r.l.Error().Msgf("ListWorkflowRuns-4: task with external_id %s and inserted_at %s has empty payload, falling back to input", task.ExternalID, task.InsertedAt.Time)
				}
				inputPayload = task.Input
			}

			res = append(res, &WorkflowRunData{
				TenantID:           task.TenantID,
				InsertedAt:         task.InsertedAt,
				ExternalID:         task.ExternalID,
				WorkflowID:         task.WorkflowID,
				WorkflowVersionId:  task.WorkflowVersionID,
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
				Output:             outputPayload,
				Input:              inputPayload,
				StepId:             &task.StepID,
				RetryCount:         &retryCount,
			})
		}
	}

	return res, int(count), nil
}

func (r *OLAPRepositoryImpl) ListWorkflowRunExternalIds(ctx context.Context, tenantId uuid.UUID, opts ListWorkflowRunOpts) ([]uuid.UUID, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-workflow-run-external-ids-olap")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.readPool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	params := sqlcv1.ListWorkflowRunExternalIdsParams{
		Tenantid: tenantId,
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

	if len(opts.WorkflowIds) > 0 {
		workflowIdParams := make([]uuid.UUID, 0)

		for _, id := range opts.WorkflowIds {
			workflowIdParams = append(workflowIdParams, id)
		}

		params.WorkflowIds = workflowIdParams
	}

	until := opts.FinishedBefore

	if until != nil {
		params.Until = sqlchelpers.TimestamptzFromTime(*until)
	}

	for key, value := range opts.AdditionalMetadata {
		params.AdditionalMetaKeys = append(params.AdditionalMetaKeys, key)
		params.AdditionalMetaValues = append(params.AdditionalMetaValues, value.(string))
	}

	externalIds, err := r.queries.ListWorkflowRunExternalIds(ctx, tx, params)

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return externalIds, nil
}

func (r *OLAPRepositoryImpl) ListTaskRunEvents(ctx context.Context, tenantId uuid.UUID, taskId int64, taskInsertedAt pgtype.Timestamptz, limit, offset int64) ([]*sqlcv1.ListTaskEventsRow, error) {
	rows, err := r.queries.ListTaskEvents(ctx, r.readPool, sqlcv1.ListTaskEventsParams{
		Tenantid:       tenantId,
		Taskid:         taskId,
		Taskinsertedat: taskInsertedAt,
	})

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *OLAPRepositoryImpl) ListTaskRunEventsByWorkflowRunId(ctx context.Context, tenantId uuid.UUID, workflowRunId uuid.UUID) ([]*TaskEventWithPayloads, error) {
	rows, err := r.queries.ListTaskEventsForWorkflowRun(ctx, r.readPool, sqlcv1.ListTaskEventsForWorkflowRunParams{
		Tenantid:      tenantId,
		Workflowrunid: workflowRunId,
	})

	if err != nil {
		return nil, err
	}

	externalIds := make([]uuid.UUID, len(rows))

	for i, row := range rows {
		eventExternalId := uuid.Nil
		if row.EventExternalID != nil {
			eventExternalId = *row.EventExternalID
		}

		externalIds[i] = eventExternalId
	}

	payloads, err := r.readPayloads(ctx, r.readPool, tenantId, externalIds...)

	if err != nil {
		return nil, err
	}

	taskEventWithPayloads := make([]*TaskEventWithPayloads, 0, len(rows))

	for _, row := range rows {
		eventExternalId := uuid.Nil
		if row.EventExternalID != nil {
			eventExternalId = *row.EventExternalID
		}
		payload, exists := payloads[eventExternalId]
		if !exists {
			r.l.Error().Msgf("ListTaskRunEventsByWorkflowRunId: event with external_id %s and task_inserted_at %s has empty payload, falling back to payload", row.EventExternalID, row.TaskInsertedAt.Time)
			payload = row.Output
		}

		taskEventWithPayloads = append(taskEventWithPayloads, &TaskEventWithPayloads{
			row,
			payload,
		})
	}

	return taskEventWithPayloads, nil
}

func (r *OLAPRepositoryImpl) ReadTaskRunMetrics(ctx context.Context, tenantId uuid.UUID, opts ReadTaskRunMetricsOpts) ([]TaskRunMetric, error) {
	var workflowIds []uuid.UUID

	if len(opts.WorkflowIds) > 0 {
		workflowIds = make([]uuid.UUID, 0)

		for _, id := range opts.WorkflowIds {
			workflowIds = append(workflowIds, id)
		}
	}

	var additionalMetaKeys []string
	var additionalMetaValues []string

	for key, value := range opts.AdditionalMetadata {
		additionalMetaKeys = append(additionalMetaKeys, key)
		additionalMetaValues = append(additionalMetaValues, value.(string))
	}

	params := sqlcv1.GetTenantStatusMetricsParams{
		Tenantid:                  tenantId,
		Createdafter:              sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		WorkflowIds:               workflowIds,
		ParentTaskExternalId:      opts.ParentTaskExternalID,
		TriggeringEventExternalId: opts.TriggeringEventExternalId,
		AdditionalMetaKeys:        additionalMetaKeys,
		AdditionalMetaValues:      additionalMetaValues,
	}

	if opts.CreatedBefore != nil {
		params.CreatedBefore = sqlchelpers.TimestamptzFromTime(*opts.CreatedBefore)
	}

	res, err := r.queries.GetTenantStatusMetrics(ctx, r.readPool, params)
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

func (r *OLAPRepositoryImpl) saveEventsToCache(events []sqlcv1.CreateTaskEventsOLAPParams) {
	for _, event := range events {
		key := getCacheKey(event)
		r.eventCache.Add(key, true)
	}
}

func getCacheKey(event sqlcv1.CreateTaskEventsOLAPParams) string {
	// key on the task_id, retry_count, and event_type
	return fmt.Sprintf("%d-%s-%d", event.TaskID, event.EventType, event.RetryCount)
}

func (r *OLAPRepositoryImpl) writeTaskEventBatch(ctx context.Context, tenantId uuid.UUID, events []sqlcv1.CreateTaskEventsOLAPParams) error {
	// skip any events which have a corresponding event already
	eventsToWrite := make([]sqlcv1.CreateTaskEventsOLAPParams, 0)
	tmpEventsToWrite := make([]sqlcv1.CreateTaskEventsOLAPTmpParams, 0)
	payloadsToWrite := make([]StoreOLAPPayloadOpts, 0)

	for _, event := range events {
		key := getCacheKey(event)
		output := event.Output

		if _, ok := r.eventCache.Get(key); !ok {
			if !r.payloadStore.OLAPDualWritesEnabled() && event.Output != nil {
				event.Output = []byte("{}")
			}

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

		if event.ExternalID != nil {
			// randomly jitter the inserted at time by +/- 300ms to make collisions virtually impossible
			dummyInsertedAt := time.Now().Add(time.Duration(rand.Intn(2*300+1)-300) * time.Millisecond)

			payloadsToWrite = append(payloadsToWrite, StoreOLAPPayloadOpts{
				ExternalId: *event.ExternalID,
				InsertedAt: sqlchelpers.TimestamptzFromTime(dummyInsertedAt),
				Payload:    output,
			})
		}
	}

	if len(eventsToWrite) == 0 {
		return nil
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

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

	_, err = r.PutPayloads(ctx, tx, tenantId, payloadsToWrite...)

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	r.saveEventsToCache(eventsToWrite)

	return nil
}

func (r *OLAPRepositoryImpl) UpdateTaskStatuses(ctx context.Context, tenantIds []uuid.UUID) (bool, []UpdateTaskStatusRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "olap_repository.update_task_statuses")
	defer span.End()

	// each partition gets its own goroutine
	eg := &errgroup.Group{}
	mu := sync.Mutex{}
	rows := make([]UpdateTaskStatusRow, 0)
	batchSizeLimit := r.statusUpdateBatchSizeLimits.Task

	// if any of the partitions are saturated, we return true
	isSaturated := false

	for i := 0; i < NUM_PARTITIONS; i++ {
		partitionNumber := i

		innerCtx, innerSpan := telemetry.NewSpan(ctx, "olap_repository.update_task_statuses.partition")
		defer innerSpan.End()

		innerSpan.SetAttributes(
			attribute.Int("olap_repository.update_task_statuses.partition.number", partitionNumber),
		)

		eg.Go(func() error {
			ctx := innerCtx
			tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

			if err != nil {
				return err
			}

			defer rollback()

			minInsertedAt, err := r.queries.FindMinInsertedAtForTaskStatusUpdates(ctx, tx, sqlcv1.FindMinInsertedAtForTaskStatusUpdatesParams{
				Partitionnumber: int32(partitionNumber), // nolint: gosec
				Tenantids:       tenantIds,
				Eventlimit:      batchSizeLimit,
			})

			if err != nil {
				return fmt.Errorf("failed to find min inserted at for task status updates: %w", err)
			}

			statusUpdateRes, err := r.queries.UpdateTaskStatuses(ctx, tx, sqlcv1.UpdateTaskStatusesParams{
				Partitionnumber: int32(partitionNumber), // nolint: gosec
				Tenantids:       tenantIds,
				Eventlimit:      batchSizeLimit,
				Mininsertedat:   minInsertedAt,
			})

			if err != nil {
				return err
			}

			if err := commit(ctx); err != nil {
				return err
			}

			mu.Lock()
			defer mu.Unlock()

			eventCount := 0

			for _, row := range statusUpdateRes {
				if row.Count > 0 {
					// not a bug: the total count is actually attached to each row
					eventCount = int(row.Count)
				}

				latestWorkerId := uuid.Nil
				if row.LatestWorkerID != nil {
					latestWorkerId = *row.LatestWorkerID
				}

				rows = append(rows, UpdateTaskStatusRow{
					TenantId:       row.TenantID,
					TaskId:         row.ID,
					TaskInsertedAt: row.InsertedAt,
					ReadableStatus: row.ReadableStatus,
					ExternalId:     row.ExternalID,
					LatestWorkerId: latestWorkerId,
					WorkflowId:     row.WorkflowID,
					IsDAGTask:      row.IsDagTask,
				})
			}

			// not super precise, but good enough to know whether to iterate
			isSaturated = isSaturated || eventCount > int(batchSizeLimit)

			innerSpan.SetAttributes(
				attribute.Int("olap_repository.update_task_statuses.partition.events_processed", eventCount),
				attribute.Bool("olap_repository.update_task_statuses.partition.is_saturated", eventCount > int(batchSizeLimit)),
			)

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return false, nil, err
	}

	span.SetAttributes(
		attribute.Bool("olap_repository.update_task_statuses.is_saturated", isSaturated),
	)

	return isSaturated, rows, nil
}

func (r *OLAPRepositoryImpl) UpdateDAGStatuses(ctx context.Context, tenantIds []uuid.UUID) (bool, []UpdateDAGStatusRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "olap_repository.update_dag_statuses")
	defer span.End()

	// each partition gets its own goroutine
	eg := &errgroup.Group{}
	mu := sync.Mutex{}
	rows := make([]UpdateDAGStatusRow, 0)

	// if any of the partitions are saturated, we return true
	isSaturated := false

	batchSizeLimit := r.statusUpdateBatchSizeLimits.DAG

	for i := 0; i < NUM_PARTITIONS; i++ {
		partitionNumber := i

		innerCtx, innerSpan := telemetry.NewSpan(ctx, "olap_repository.update_dag_statuses.partition")
		defer innerSpan.End()

		innerSpan.SetAttributes(
			attribute.Int("olap_repository.update_dag_statuses.partition.number", partitionNumber),
		)

		eg.Go(func() error {
			ctx := innerCtx
			tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

			if err != nil {
				return fmt.Errorf("failed to prepare transaction: %w", err)
			}

			defer rollback()

			minInsertedAt, err := r.queries.FindMinInsertedAtForDAGStatusUpdates(ctx, tx, sqlcv1.FindMinInsertedAtForDAGStatusUpdatesParams{
				Partitionnumber: int32(partitionNumber), // nolint: gosec
				Tenantids:       tenantIds,
				Eventlimit:      batchSizeLimit,
			})

			if err != nil {
				return fmt.Errorf("failed to find min inserted at for DAG status updates: %w", err)
			}

			statusUpdateRes, err := r.queries.UpdateDAGStatuses(ctx, tx, sqlcv1.UpdateDAGStatusesParams{
				Partitionnumber: int32(partitionNumber), // nolint: gosec
				Tenantids:       tenantIds,
				Eventlimit:      batchSizeLimit,
				Mininsertedat:   minInsertedAt,
			})

			if err != nil {
				return fmt.Errorf("failed to update DAG statuses: %w", err)
			}

			if err := commit(ctx); err != nil {
				return fmt.Errorf("failed to commit transaction: %w", err)
			}

			mu.Lock()
			defer mu.Unlock()

			eventCount := 0

			for _, row := range statusUpdateRes {
				if row.Count > 0 {
					// not a bug: the total count is actually attached to each row
					eventCount = int(row.Count)
				}

				rows = append(rows, UpdateDAGStatusRow{
					TenantId:       row.TenantID,
					DagId:          row.ID,
					DagInsertedAt:  row.InsertedAt,
					ReadableStatus: row.ReadableStatus,
					ExternalId:     row.ExternalID,
					WorkflowId:     row.WorkflowID,
				})
			}

			// not super precise, but good enough to know whether to iterate
			isSaturated = isSaturated || eventCount > int(batchSizeLimit)

			innerSpan.SetAttributes(
				attribute.Int("olap_repository.update_dag_statuses.partition.events_processed", eventCount),
				attribute.Bool("olap_repository.update_dag_statuses.partition.is_saturated", eventCount > int(batchSizeLimit)),
			)

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return false, nil, fmt.Errorf("failed to wait for status update goroutines: %w", err)
	}

	span.SetAttributes(
		attribute.Bool("olap_repository.update_dag_statuses.is_saturated", isSaturated),
	)

	return isSaturated, rows, nil
}

func (r *OLAPRepositoryImpl) writeTaskBatch(ctx context.Context, tenantId uuid.UUID, tasks []*V1TaskWithPayload) error {
	params := make([]sqlcv1.CreateTasksOLAPParams, 0)
	putPayloadOpts := make([]StoreOLAPPayloadOpts, 0)

	for _, task := range tasks {
		payload := task.Payload

		// fall back to input if payload is empty
		// for backwards compatibility
		if len(payload) == 0 {
			r.l.Error().Msgf("writeTaskBatch: task %s with ID %d and inserted_at %s has empty payload, falling back to input", task.ExternalID.String(), task.ID, task.InsertedAt.Time)
			payload = task.Input
		}

		// todo: remove this when we remove dual writes
		payloadToWriteToTask := payload
		if !r.payloadStore.OLAPDualWritesEnabled() {
			payloadToWriteToTask = []byte("{}")
		}

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
			AdditionalMetadata:   task.AdditionalMetadata,
			DagID:                task.DagID,
			DagInsertedAt:        task.DagInsertedAt,
			ParentTaskExternalID: task.ParentTaskExternalID,
			WorkflowRunID:        task.WorkflowRunID,
			Input:                payloadToWriteToTask,
		})

		putPayloadOpts = append(putPayloadOpts, StoreOLAPPayloadOpts{
			ExternalId: task.ExternalID,
			InsertedAt: task.InsertedAt,
			Payload:    payload,
		})
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)
	if err != nil {
		return err
	}
	defer rollback()

	_, err = r.queries.CreateTasksOLAP(ctx, tx, params)
	if err != nil {
		return err
	}

	_, err = r.PutPayloads(ctx, tx, tenantId, putPayloadOpts...)

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	return nil
}

func (r *OLAPRepositoryImpl) writeDAGBatch(ctx context.Context, tenantId uuid.UUID, dags []*DAGWithData) error {
	params := make([]sqlcv1.CreateDAGsOLAPParams, 0)
	putPayloadOpts := make([]StoreOLAPPayloadOpts, 0)

	for _, dag := range dags {
		// todo: remove this when we remove dual writes
		input := dag.Input
		if !r.payloadStore.OLAPDualWritesEnabled() {
			input = []byte("{}")
		}

		params = append(params, sqlcv1.CreateDAGsOLAPParams{
			TenantID:             dag.TenantID,
			ID:                   dag.ID,
			InsertedAt:           dag.InsertedAt,
			WorkflowID:           dag.WorkflowID,
			WorkflowVersionID:    dag.WorkflowVersionID,
			ExternalID:           dag.ExternalID,
			DisplayName:          dag.DisplayName,
			AdditionalMetadata:   dag.AdditionalMetadata,
			ParentTaskExternalID: dag.ParentTaskExternalID,
			TotalTasks:           int32(dag.TotalTasks), // nolint: gosec
			Input:                input,
		})

		putPayloadOpts = append(putPayloadOpts, StoreOLAPPayloadOpts{
			ExternalId: dag.ExternalID,
			InsertedAt: dag.InsertedAt,
			Payload:    dag.Input,
		})
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)
	if err != nil {
		return err
	}
	defer rollback()

	_, err = r.queries.CreateDAGsOLAP(ctx, tx, params)
	if err != nil {
		return err
	}

	_, err = r.PutPayloads(ctx, tx, tenantId, putPayloadOpts...)

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	return nil
}

func (r *OLAPRepositoryImpl) CreateTaskEvents(ctx context.Context, tenantId uuid.UUID, events []sqlcv1.CreateTaskEventsOLAPParams) error {
	return r.writeTaskEventBatch(ctx, tenantId, events)
}

func (r *OLAPRepositoryImpl) CreateTasks(ctx context.Context, tenantId uuid.UUID, tasks []*V1TaskWithPayload) error {
	return r.writeTaskBatch(ctx, tenantId, tasks)
}

func (r *OLAPRepositoryImpl) CreateDAGs(ctx context.Context, tenantId uuid.UUID, dags []*DAGWithData) error {
	return r.writeDAGBatch(ctx, tenantId, dags)
}

func (r *OLAPRepositoryImpl) GetTaskPointMetrics(ctx context.Context, tenantId uuid.UUID, startTimestamp *time.Time, endTimestamp *time.Time, bucketInterval time.Duration) ([]*sqlcv1.GetTaskPointMetricsRow, error) {
	rows, err := r.queries.GetTaskPointMetrics(ctx, r.readPool, sqlcv1.GetTaskPointMetricsParams{
		Interval:      durationToPgInterval(bucketInterval),
		Tenantid:      tenantId,
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

func (r *OLAPRepositoryImpl) ReadDAG(ctx context.Context, dagExternalId uuid.UUID) (*sqlcv1.V1DagsOlap, error) {
	return r.queries.ReadDAGByExternalID(ctx, r.readPool, dagExternalId)
}

func (r *OLAPRepositoryImpl) ListTasksByExternalIds(ctx context.Context, tenantId uuid.UUID, externalIds []uuid.UUID) ([]*sqlcv1.FlattenTasksByExternalIdsRow, error) {
	externalUUIDs := make([]uuid.UUID, 0)

	for _, id := range externalIds {
		externalUUIDs = append(externalUUIDs, id)
	}

	return r.queries.FlattenTasksByExternalIds(ctx, r.readPool, sqlcv1.FlattenTasksByExternalIdsParams{
		Tenantid:    tenantId,
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

func (r *OLAPRepositoryImpl) ListWorkflowRunDisplayNames(ctx context.Context, tenantId uuid.UUID, externalIds []uuid.UUID) ([]*sqlcv1.ListWorkflowRunDisplayNamesRow, error) {
	return r.queries.ListWorkflowRunDisplayNames(ctx, r.readPool, sqlcv1.ListWorkflowRunDisplayNamesParams{
		Tenantid:    tenantId,
		Externalids: externalIds,
	})
}

func (r *OLAPRepositoryImpl) GetTaskTimings(ctx context.Context, tenantId uuid.UUID, workflowRunId uuid.UUID, depth int32) ([]*sqlcv1.PopulateTaskRunDataRow, map[uuid.UUID]int32, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-task-timings-olap")
	defer span.End()

	if depth > 10 {
		return nil, nil, fmt.Errorf("depth too large")
	}

	// start out by getting a list of task external ids for the workflow run id
	rootTaskExternalIds := make([]uuid.UUID, 0)
	sevenDaysAgo := time.Now().Add(-time.Hour * 24 * 7)
	minInsertedAt := time.Now()

	rootTasks, err := r.queries.FlattenTasksByExternalIds(ctx, r.readPool, sqlcv1.FlattenTasksByExternalIdsParams{
		Externalids: []uuid.UUID{workflowRunId},
		Tenantid:    tenantId,
	})

	if err != nil {
		return nil, nil, err
	}

	for _, task := range rootTasks {
		rootTaskExternalIds = append(rootTaskExternalIds, task.ExternalID)

		if task.InsertedAt.Time.Before(minInsertedAt) {
			minInsertedAt = task.InsertedAt.Time
		}
	}

	// Setting the maximum lookback period to 7 days
	// to prevent scanning a zillion partitions on the tasks,
	// runs, and dags tables.
	if minInsertedAt.Before(sevenDaysAgo) {
		minInsertedAt = sevenDaysAgo
	}

	runsList, err := r.queries.GetRunsListRecursive(ctx, r.readPool, sqlcv1.GetRunsListRecursiveParams{
		Taskexternalids: rootTaskExternalIds,
		Tenantid:        tenantId,
		Depth:           depth,
		Createdafter:    sqlchelpers.TimestamptzFromTime(minInsertedAt),
	})

	if err != nil {
		return nil, nil, err
	}

	// associate each run external id with a depth
	idsToDepth := make(map[uuid.UUID]int32)
	idsInsertedAts := make([]IdInsertedAt, 0, len(runsList))

	for _, row := range runsList {
		idsToDepth[row.ExternalID] = row.Depth
		idsInsertedAts = append(idsInsertedAts, IdInsertedAt{
			ID:         row.ID,
			InsertedAt: row.InsertedAt,
		})
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.readPool, r.l)
	defer rollback()

	if err != nil {
		return nil, nil, fmt.Errorf("error beginning transaction: %v", err)
	}

	tasksWithData, err := r.populateTaskRunData(ctx, tx, tenantId, idsInsertedAts, false)

	if err != nil {
		return nil, nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("error committing transaction: %v", err)
	}

	return tasksWithData, idsToDepth, nil
}

type EventTriggersFromExternalId struct {
	RunID           int64              `json:"run_id"`
	RunInsertedAt   pgtype.Timestamptz `json:"run_inserted_at"`
	EventExternalId uuid.UUID          `json:"event_external_id"`
	EventSeenAt     pgtype.Timestamptz `json:"event_seen_at"`
	FilterId        uuid.UUID          `json:"filter_id"`
}

func (r *OLAPRepositoryImpl) BulkCreateEventsAndTriggers(ctx context.Context, events sqlcv1.BulkCreateEventsOLAPParams, triggers []EventTriggersFromExternalId) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return fmt.Errorf("error beginning transaction: %v", err)
	}

	defer rollback()

	eventsToInsert := events
	eventExternalIdToPayload := make(map[uuid.UUID][]byte)

	for i, payload := range eventsToInsert.Payloads {
		eventExternalIdToPayload[eventsToInsert.Externalids[i]] = payload
	}

	// todo: remove this when we remove dual writes
	if !r.payloadStore.OLAPDualWritesEnabled() {
		payloads := make([][]byte, len(eventsToInsert.Payloads))

		for i := range eventsToInsert.Payloads {
			payloads[i] = []byte("{}")
		}

		eventsToInsert.Payloads = payloads
	}

	insertedEvents, err := r.queries.BulkCreateEventsOLAP(ctx, tx, eventsToInsert)

	if err != nil {
		return fmt.Errorf("error creating events: %v", err)
	}

	eventExternalIdToId := make(map[uuid.UUID]int64)

	for _, event := range insertedEvents {
		eventExternalIdToId[event.ExternalID] = event.ID
	}

	bulkCreateTriggersParams := make([]sqlcv1.BulkCreateEventTriggersParams, 0)

	for _, trigger := range triggers {
		eventId, ok := eventExternalIdToId[trigger.EventExternalId]

		if !ok {
			return fmt.Errorf("event external id %s not found in events", trigger.EventExternalId.String())
		}

		bulkCreateTriggersParams = append(bulkCreateTriggersParams, sqlcv1.BulkCreateEventTriggersParams{
			RunID:         trigger.RunID,
			RunInsertedAt: trigger.RunInsertedAt,
			EventID:       eventId,
			EventSeenAt:   trigger.EventSeenAt,
			FilterID:      &trigger.FilterId,
		})
	}

	_, err = r.queries.BulkCreateEventTriggers(ctx, tx, bulkCreateTriggersParams)

	if err != nil {
		return fmt.Errorf("error creating event triggers: %v", err)
	}

	tenantIdToPutPayloadOpts := make(map[uuid.UUID][]StoreOLAPPayloadOpts)

	for _, event := range insertedEvents {
		if event == nil {
			continue
		}

		payload := eventExternalIdToPayload[event.ExternalID]

		tenantIdToPutPayloadOpts[event.TenantID] = append(tenantIdToPutPayloadOpts[event.TenantID], StoreOLAPPayloadOpts{
			ExternalId: event.ExternalID,
			InsertedAt: event.SeenAt,
			Payload:    payload,
		})
	}

	for tenantId, putPayloadOpts := range tenantIdToPutPayloadOpts {
		_, err = r.PutPayloads(ctx, tx, tenantId, putPayloadOpts...)

		if err != nil {
			return fmt.Errorf("error putting event payloads: %v", err)
		}
	}

	if err := commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	return nil
}

func (r *OLAPRepositoryImpl) GetEvent(ctx context.Context, externalId uuid.UUID) (*sqlcv1.V1EventsOlap, error) {
	return r.queries.GetEventByExternalId(ctx, r.readPool, externalId)
}

func (r *OLAPRepositoryImpl) PopulateEventData(ctx context.Context, tenantId uuid.UUID, eventExternalIds []uuid.UUID) (map[uuid.UUID]sqlcv1.PopulateEventDataRow, error) {
	eventData, err := r.queries.PopulateEventData(ctx, r.readPool, sqlcv1.PopulateEventDataParams{
		Eventexternalids: eventExternalIds,
		Tenantid:         tenantId,
	})

	if err != nil {
		return nil, fmt.Errorf("error populating event data: %v", err)
	}

	externalIdToEventData := make(map[uuid.UUID]sqlcv1.PopulateEventDataRow)

	for _, data := range eventData {
		externalIdToEventData[data.ExternalID] = *data
	}

	return externalIdToEventData, nil
}

func (r *OLAPRepositoryImpl) GetEventWithPayload(ctx context.Context, externalId, tenantId uuid.UUID) (*EventWithPayload, error) {
	event, err := r.queries.GetEventByExternalIdUsingTenantId(ctx, r.readPool, sqlcv1.GetEventByExternalIdUsingTenantIdParams{
		Tenantid:        tenantId,
		Eventexternalid: externalId,
	})

	if err != nil {
		return nil, err
	}

	payload, err := r.ReadPayload(ctx, tenantId, event.ExternalID)

	if err != nil {
		return nil, fmt.Errorf("error reading event payload: %v", err)
	}

	eventExternalIds := []uuid.UUID{event.ExternalID}

	eventExternalIdToData, err := r.PopulateEventData(ctx, event.TenantID, eventExternalIds)

	if err != nil {
		return nil, fmt.Errorf("error populating event data: %v", err)
	}

	eventData, exists := eventExternalIdToData[event.ExternalID]
	var triggeredRuns []byte
	var queuedCount, runningCount, completedCount, cancelledCount, failedCount int64

	if exists {
		triggeredRuns = eventData.TriggeredRuns
		queuedCount = eventData.QueuedCount
		runningCount = eventData.RunningCount
		completedCount = eventData.CompletedCount
		cancelledCount = eventData.CancelledCount
		failedCount = eventData.FailedCount
	}

	return &EventWithPayload{
		ListEventsRow: &ListEventsRow{
			TenantID:                event.TenantID,
			EventID:                 event.ID,
			EventExternalID:         event.ExternalID,
			EventSeenAt:             event.SeenAt,
			EventKey:                event.Key,
			EventPayload:            payload,
			EventAdditionalMetadata: event.AdditionalMetadata,
			EventScope:              event.Scope.String,
			QueuedCount:             queuedCount,
			RunningCount:            runningCount,
			CompletedCount:          completedCount,
			CancelledCount:          cancelledCount,
			FailedCount:             failedCount,
			TriggeredRuns:           triggeredRuns,
			TriggeringWebhookName:   &event.TriggeringWebhookName.String,
		},
		Payload: payload,
	}, nil
}

type ListEventsRow struct {
	TenantID                uuid.UUID          `json:"tenant_id"`
	EventID                 int64              `json:"event_id"`
	EventExternalID         uuid.UUID          `json:"event_external_id"`
	EventSeenAt             pgtype.Timestamptz `json:"event_seen_at"`
	EventKey                string             `json:"event_key"`
	EventPayload            []byte             `json:"event_payload"`
	EventAdditionalMetadata []byte             `json:"event_additional_metadata"`
	EventScope              string             `json:"event_scope"`
	QueuedCount             int64              `json:"queued_count"`
	RunningCount            int64              `json:"running_count"`
	CompletedCount          int64              `json:"completed_count"`
	CancelledCount          int64              `json:"cancelled_count"`
	FailedCount             int64              `json:"failed_count"`
	TriggeredRuns           []byte             `json:"triggered_runs"`
	TriggeringWebhookName   *string            `json:"triggering_webhook_name,omitempty"`
}

type EventWithPayload struct {
	*ListEventsRow
	Payload []byte `json:"payload"`
}

func (r *OLAPRepositoryImpl) ListEvents(ctx context.Context, opts sqlcv1.ListEventsParams) ([]*EventWithPayload, *int64, error) {
	var (
		events     []*sqlcv1.V1EventsOlap
		eventCount int64
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		c, err := r.queries.CountEvents(gctx, r.readPool, sqlcv1.CountEventsParams{
			Tenantid:           opts.Tenantid,
			Keys:               opts.Keys,
			Since:              opts.Since,
			Until:              opts.Until,
			WorkflowIds:        opts.WorkflowIds,
			EventIds:           opts.EventIds,
			AdditionalMetadata: opts.AdditionalMetadata,
			Statuses:           opts.Statuses,
			Scopes:             opts.Scopes,
		})

		if err != nil {
			return err
		}

		eventCount = c
		return nil
	})

	// We need the events list to proceed; keep it in-line while the count runs in the background.
	var err error
	events, err = r.queries.ListEvents(gctx, r.readPool, opts)
	if err != nil {
		return nil, nil, err
	}

	eventExternalIds := make([]uuid.UUID, len(events))

	for i, event := range events {
		eventExternalIds[i] = event.ExternalID
	}

	eventExternalIdToData, err := r.PopulateEventData(
		ctx,
		opts.Tenantid,
		eventExternalIds,
	)

	if err != nil {
		return nil, nil, fmt.Errorf("error populating event data: %v", err)
	}

	externalIdToPayload, err := r.readPayloads(ctx, r.readPool, opts.Tenantid, eventExternalIds...)

	if err != nil {
		return nil, nil, fmt.Errorf("error reading event payloads: %v", err)
	}

	result := make([]*EventWithPayload, 0)

	for _, event := range events {
		payload, exists := externalIdToPayload[event.ExternalID]

		if !exists {
			r.l.Error().Msgf("ListEvents: payload for event %s not found", event.ExternalID.String())
			payload = event.Payload
		}

		var triggeringWebhookName *string

		if event.TriggeringWebhookName.Valid {
			triggeringWebhookName = &event.TriggeringWebhookName.String
		}

		data, exists := eventExternalIdToData[event.ExternalID]

		var triggeredRuns []byte
		var queuedCount, runningCount, completedCount, cancelledCount, failedCount int64

		if exists {
			triggeredRuns = data.TriggeredRuns
			queuedCount = data.QueuedCount
			runningCount = data.RunningCount
			completedCount = data.CompletedCount
			cancelledCount = data.CancelledCount
			failedCount = data.FailedCount
		}

		result = append(result, &EventWithPayload{
			ListEventsRow: &ListEventsRow{
				TenantID:                event.TenantID,
				EventID:                 event.ID,
				EventExternalID:         event.ExternalID,
				EventSeenAt:             event.SeenAt,
				EventKey:                event.Key,
				EventPayload:            payload,
				EventAdditionalMetadata: event.AdditionalMetadata,
				EventScope:              event.Scope.String,
				QueuedCount:             queuedCount,
				RunningCount:            runningCount,
				CompletedCount:          completedCount,
				CancelledCount:          cancelledCount,
				FailedCount:             failedCount,
				TriggeredRuns:           triggeredRuns,
				TriggeringWebhookName:   triggeringWebhookName,
			},
			Payload: payload,
		})
	}

	// Ensure count is complete (and propagate any count error) before returning.
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	return result, &eventCount, nil
}

func (r *OLAPRepositoryImpl) ListEventKeys(ctx context.Context, tenantId uuid.UUID) ([]string, error) {
	keys, err := r.queries.ListEventKeys(ctx, r.pool, tenantId)

	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (r *OLAPRepositoryImpl) GetDAGDurations(ctx context.Context, tenantId uuid.UUID, externalIds []uuid.UUID, minInsertedAt pgtype.Timestamptz) (map[string]*sqlcv1.GetDagDurationsRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "olap_repository.get_dag_durations")
	defer span.End()

	span.SetAttributes(attribute.KeyValue{
		Key:   "olap_repository.get_dag_durations.batch_size",
		Value: attribute.IntValue(len(externalIds)),
	})

	rows, err := r.queries.GetDagDurations(ctx, r.readPool, sqlcv1.GetDagDurationsParams{
		Externalids:   externalIds,
		Tenantid:      tenantId,
		Mininsertedat: minInsertedAt,
	})

	if err != nil {
		return nil, err
	}

	dagDurations := make(map[string]*sqlcv1.GetDagDurationsRow)

	for _, row := range rows {
		dagDurations[row.ExternalID.String()] = row
	}

	return dagDurations, nil
}

func (r *OLAPRepositoryImpl) GetTaskDurationsByTaskIds(ctx context.Context, tenantId uuid.UUID, taskIds []int64, taskInsertedAts []pgtype.Timestamptz, readableStatuses []sqlcv1.V1ReadableStatusOlap) (map[int64]*sqlcv1.GetTaskDurationsByTaskIdsRow, error) {
	rows, err := r.queries.GetTaskDurationsByTaskIds(ctx, r.readPool, sqlcv1.GetTaskDurationsByTaskIdsParams{
		Taskids:          taskIds,
		Taskinsertedats:  taskInsertedAts,
		Tenantid:         tenantId,
		Readablestatuses: readableStatuses,
	})
	if err != nil {
		return nil, err
	}

	taskDurations := make(map[int64]*sqlcv1.GetTaskDurationsByTaskIdsRow)

	for i, row := range rows {
		taskDurations[taskIds[i]] = row
	}

	return taskDurations, nil
}

type CreateIncomingWebhookFailureLogOpts struct {
	WebhookName string
	ErrorText   string
}

func (r *OLAPRepositoryImpl) CreateIncomingWebhookValidationFailureLogs(ctx context.Context, tenantId uuid.UUID, opts []CreateIncomingWebhookFailureLogOpts) error {
	incomingWebhookNames := make([]string, len(opts))
	errors := make([]string, len(opts))

	for i, opt := range opts {
		incomingWebhookNames[i] = opt.WebhookName
		errors[i] = opt.ErrorText
	}

	params := sqlcv1.CreateIncomingWebhookValidationFailureLogsParams{
		Tenantid:             tenantId,
		Incomingwebhooknames: incomingWebhookNames,
		Errors:               errors,
	}

	return r.queries.CreateIncomingWebhookValidationFailureLogs(ctx, r.pool, params)
}

type CELEvaluationFailure struct {
	Source       sqlcv1.V1CelEvaluationFailureSource `json:"source"`
	ErrorMessage string                              `json:"error_message"`
}

func (r *OLAPRepositoryImpl) StoreCELEvaluationFailures(ctx context.Context, tenantId uuid.UUID, failures []CELEvaluationFailure) error {
	errorMessages := make([]string, len(failures))
	sources := make([]string, len(failures))

	for i, failure := range failures {
		errorMessages[i] = failure.ErrorMessage
		sources[i] = string(failure.Source)
	}

	return r.queries.StoreCELEvaluationFailures(ctx, r.pool, sqlcv1.StoreCELEvaluationFailuresParams{
		Tenantid: tenantId,
		Sources:  sources,
		Errors:   errorMessages,
	})
}

type OffloadPayloadOpts struct {
	ExternalId          uuid.UUID
	ExternalLocationKey string
}

func (r *OLAPRepositoryImpl) PutPayloads(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, putPayloadOpts ...StoreOLAPPayloadOpts) (map[uuid.UUID]ExternalPayloadLocationKey, error) {
	ctx, span := telemetry.NewSpan(ctx, "OLAPRepository.PutPayloads")
	defer span.End()

	span.SetAttributes(attribute.Int("olap_repository.put_payloads.batch_size", len(putPayloadOpts)))

	localTx := false
	var (
		commit   func(context.Context) error
		rollback func()
		err      error
	)

	if tx == nil {
		localTx = true
		tx, commit, rollback, err = sqlchelpers.PrepareTx(ctx, r.pool, r.l)

		if err != nil {
			return nil, fmt.Errorf("error beginning transaction in `PutPayload`: %v", err)
		}

		defer rollback()
	}

	externalIdToKey := make(map[uuid.UUID]ExternalPayloadLocationKey)

	if r.payloadStore.ExternalStoreEnabled() && r.payloadStore.ImmediateOffloadsEnabled() {
		storeExternalPayloadOpts := make([]OffloadToExternalStoreOpts, len(putPayloadOpts))

		for i, opt := range putPayloadOpts {
			storeOpts := OffloadToExternalStoreOpts{
				TenantId:   tenantId,
				ExternalID: uuid.UUID(opt.ExternalId),
				InsertedAt: opt.InsertedAt,
				Payload:    opt.Payload,
			}

			storeExternalPayloadOpts[i] = storeOpts
		}

		externalIdToKey, err = r.payloadStore.ExternalStore().Store(ctx, storeExternalPayloadOpts...)

		if err != nil {
			return nil, fmt.Errorf("error offloading payloads to external store: %v", err)
		}
	}

	insertedAts := make([]pgtype.Timestamptz, 0, len(putPayloadOpts))
	tenantIds := make([]uuid.UUID, 0, len(putPayloadOpts))
	externalIds := make([]uuid.UUID, 0, len(putPayloadOpts))
	payloads := make([][]byte, 0, len(putPayloadOpts))
	locations := make([]string, 0, len(putPayloadOpts))
	externalKeys := make([]string, 0, len(putPayloadOpts))

	for _, opt := range putPayloadOpts {
		key, ok := externalIdToKey[opt.ExternalId]

		externalIds = append(externalIds, opt.ExternalId)
		insertedAts = append(insertedAts, opt.InsertedAt)
		tenantIds = append(tenantIds, tenantId)

		if ok {
			payloads = append(payloads, nil)
			locations = append(locations, string(sqlcv1.V1PayloadLocationOlapEXTERNAL))
			externalKeys = append(externalKeys, string(key))
		} else {
			payloads = append(payloads, opt.Payload)
			locations = append(locations, string(sqlcv1.V1PayloadLocationOlapINLINE))
			externalKeys = append(externalKeys, "")
		}
	}

	err = r.queries.PutPayloads(ctx, tx, sqlcv1.PutPayloadsParams{
		Externalids:          externalIds,
		Insertedats:          insertedAts,
		Tenantids:            tenantIds,
		Payloads:             payloads,
		Locations:            locations,
		Externallocationkeys: externalKeys,
	})

	if err != nil {
		return nil, fmt.Errorf("error putting payloads: %w", err)
	}

	if localTx {
		if err := commit(ctx); err != nil {
			return nil, fmt.Errorf("error committing transaction in `PutPayload`: %v", err)
		}
	}

	return externalIdToKey, nil
}

func (r *OLAPRepositoryImpl) ReadPayload(ctx context.Context, tenantId uuid.UUID, externalId uuid.UUID) ([]byte, error) {
	payloads, err := r.readPayloads(ctx, r.readPool, tenantId, externalId)

	if err != nil {
		return nil, err
	}

	payload, exists := payloads[externalId]

	if !exists {
		r.l.Debug().Msgf("payload for external ID %s not found", externalId.String())
	}

	return payload, nil
}

func (r *OLAPRepositoryImpl) ReadPayloads(ctx context.Context, tenantId uuid.UUID, externalIds ...uuid.UUID) (map[uuid.UUID][]byte, error) {
	return r.readPayloads(ctx, r.readPool, tenantId, externalIds...)
}

func (r *OLAPRepositoryImpl) readPayloads(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, externalIds ...uuid.UUID) (map[uuid.UUID][]byte, error) {
	payloads, err := r.queries.ReadPayloadsOLAP(ctx, tx, sqlcv1.ReadPayloadsOLAPParams{
		Tenantid:    tenantId,
		Externalids: externalIds,
	})

	if err != nil {
		return nil, err
	}

	externalIdToPayload := make(map[uuid.UUID][]byte)
	externalIdToExternalKey := make(map[uuid.UUID]ExternalPayloadLocationKey)
	externalKeys := make([]ExternalPayloadLocationKey, 0)

	for _, payload := range payloads {
		if payload.Location == sqlcv1.V1PayloadLocationOlapINLINE {
			externalIdToPayload[payload.ExternalID] = payload.InlineContent
		} else {
			key := ExternalPayloadLocationKey(payload.ExternalLocationKey.String)

			externalIdToExternalKey[payload.ExternalID] = key
			externalKeys = append(externalKeys, key)
		}
	}

	if len(externalKeys) > 0 && r.payloadStore.ExternalStoreEnabled() {
		keyToPayload, err := r.payloadStore.RetrieveFromExternal(ctx, externalKeys...)

		if err != nil {
			return nil, err
		}

		for externalId, externalKey := range externalIdToExternalKey {
			externalIdToPayload[externalId] = keyToPayload[externalKey]
		}
	}

	return externalIdToPayload, nil
}

func (r *OLAPRepositoryImpl) OffloadPayloads(ctx context.Context, tenantId uuid.UUID, payloads []OffloadPayloadOpts) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return fmt.Errorf("error beginning transaction: %v", err)
	}

	defer rollback()

	tenantIds := make([]uuid.UUID, len(payloads))
	externalIds := make([]uuid.UUID, len(payloads))
	externalLocationKeys := make([]string, len(payloads))

	for i, opt := range payloads {
		externalIds[i] = opt.ExternalId
		tenantIds[i] = tenantId
		externalLocationKeys[i] = opt.ExternalLocationKey
	}

	err = r.queries.OffloadPayloads(ctx, tx, sqlcv1.OffloadPayloadsParams{
		Externalids:          externalIds,
		Tenantids:            tenantIds,
		Externallocationkeys: externalLocationKeys,
	})

	if err != nil {
		return fmt.Errorf("error offloading payloads: %v", err)
	}

	if err := commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	return nil
}

func (r *OLAPRepositoryImpl) AnalyzeOLAPTables(ctx context.Context) error {
	const timeout = 1000 * 60 * 60 // 60 minute timeout
	tx, commit, rollback, err := sqlchelpers.PrepareTxWithStatementTimeout(ctx, r.pool, r.l, timeout)

	if err != nil {
		return fmt.Errorf("error beginning transaction: %v", err)
	}

	defer rollback()

	acquired, err := r.queries.TryAdvisoryLock(ctx, tx, hash("analyze-olap-tables"))

	if err != nil {
		return fmt.Errorf("error acquiring advisory lock: %v", err)
	}

	if !acquired {
		r.l.Info().Msg("advisory lock already held, skipping OLAP table analysis")
		return nil
	}

	err = r.queries.AnalyzeV1RunsOLAP(ctx, tx)

	if err != nil {
		return fmt.Errorf("error analyzing v1_runs_olap: %v", err)
	}

	err = r.queries.AnalyzeV1TasksOLAP(ctx, tx)

	if err != nil {
		return fmt.Errorf("error analyzing v1_tasks_olap: %v", err)
	}

	err = r.queries.AnalyzeV1DAGsOLAP(ctx, tx)

	if err != nil {
		return fmt.Errorf("error analyzing v1_dags_olap: %v", err)
	}

	err = r.queries.AnalyzeV1DAGToTaskOLAP(ctx, tx)

	if err != nil {
		return fmt.Errorf("error analyzing v1_dag_to_task_olap: %v", err)
	}

	err = r.queries.AnalyzeV1PayloadsOLAP(ctx, tx)

	if err != nil {
		return fmt.Errorf("error analyzing v1_payloads_olap: %v", err)
	}

	err = r.queries.AnalyzeV1LookupTableOLAP(ctx, tx)

	if err != nil {
		return fmt.Errorf("error analyzing v1_lookup_table_olap: %v", err)
	}

	if err := commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	return nil
}

type IdInsertedAt struct {
	ID         int64              `json:"id"`
	InsertedAt pgtype.Timestamptz `json:"inserted_at"`
}

func (r *OLAPRepositoryImpl) populateTaskRunData(ctx context.Context, tx pgx.Tx, tenantId uuid.UUID, opts []IdInsertedAt, includePayloads bool) ([]*sqlcv1.PopulateTaskRunDataRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "populate-task-run-data-olap")
	defer span.End()

	uniqueTaskIdInsertedAts := make(map[IdInsertedAt]struct{})

	for _, opt := range opts {
		uniqueTaskIdInsertedAts[IdInsertedAt{
			ID:         opt.ID,
			InsertedAt: opt.InsertedAt,
		}] = struct{}{}
	}

	span.SetAttributes(attribute.KeyValue{
		Key:   "populate-task-run-data-olap.batch_size",
		Value: attribute.IntValue(len(uniqueTaskIdInsertedAts)),
	})

	if len(uniqueTaskIdInsertedAts) == 0 {
		r.l.Debug().Msg("populateTaskRunData called with empty opts, returning empty result")
		return []*sqlcv1.PopulateTaskRunDataRow{}, nil
	}

	taskIds := make([]int64, 0)
	taskInsertedAts := make([]pgtype.Timestamptz, 0)

	for idInsertedAt := range uniqueTaskIdInsertedAts {
		taskIds = append(taskIds, idInsertedAt.ID)
		taskInsertedAts = append(taskInsertedAts, idInsertedAt.InsertedAt)
	}

	taskData, err := r.queries.PopulateTaskRunData(ctx, tx, sqlcv1.PopulateTaskRunDataParams{
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Tenantid:        tenantId,
		Includepayloads: includePayloads,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	sort.Slice(taskData, func(i, j int) bool {
		if taskData[i].InsertedAt.Time.Equal(taskData[j].InsertedAt.Time) {
			return taskData[i].ID < taskData[j].ID
		}

		return taskData[i].InsertedAt.Time.After(taskData[j].InsertedAt.Time)
	})

	return taskData, nil

}

func (r *OLAPRepositoryImpl) StatusUpdateBatchSizeLimits() StatusUpdateBatchSizeLimits {
	return r.statusUpdateBatchSizeLimits
}

func (r *OLAPRepositoryImpl) CountOLAPTempTableSizeForDAGStatusUpdates(ctx context.Context) (int64, error) {
	return r.queries.CountOLAPTempTableSizeForDAGStatusUpdates(ctx, r.readPool)
}

func (r *OLAPRepositoryImpl) CountOLAPTempTableSizeForTaskStatusUpdates(ctx context.Context) (int64, error) {
	return r.queries.CountOLAPTempTableSizeForTaskStatusUpdates(ctx, r.readPool)
}

func (r *OLAPRepositoryImpl) ListYesterdayRunCountsByStatus(ctx context.Context) (map[sqlcv1.V1ReadableStatusOlap]int64, error) {
	rows, err := r.queries.ListYesterdayRunCountsByStatus(ctx, r.readPool)

	if err != nil {
		return nil, err
	}

	statusToCount := make(map[sqlcv1.V1ReadableStatusOlap]int64)

	for _, row := range rows {
		statusToCount[row.ReadableStatus] = row.Count
	}

	return statusToCount, nil
}

type BulkCutOverOLAPPayload struct {
	TenantID            uuid.UUID
	InsertedAt          pgtype.Timestamptz
	ExternalId          uuid.UUID
	ExternalLocationKey ExternalPayloadLocationKey
}

type OLAPPaginationParams struct {
	LastTenantId   uuid.UUID
	LastInsertedAt pgtype.Timestamptz
	LastExternalId uuid.UUID
	Limit          int32
}

type OLAPCutoverJobRunMetadata struct {
	ShouldRun      bool
	Pagination     OLAPPaginationParams
	PartitionDate  PartitionDate
	LeaseProcessId uuid.UUID
}

type OLAPCutoverBatchOutcome struct {
	ShouldContinue bool
	NextPagination OLAPPaginationParams
}

func (p *OLAPRepositoryImpl) OptimizeOLAPPayloadWindowSize(ctx context.Context, partitionDate PartitionDate, candidateBatchNumRows int32, pagination OLAPPaginationParams) (*int32, error) {
	if candidateBatchNumRows <= 0 {
		// trivial case that we'll never hit, but to prevent infinite recursion
		zero := int32(0)
		return &zero, nil
	}

	proposedBatchSizeBytes, err := p.queries.ComputeOLAPPayloadBatchSize(ctx, p.pool, sqlcv1.ComputeOLAPPayloadBatchSizeParams{
		Partitiondate:  pgtype.Date(partitionDate),
		Lasttenantid:   pagination.LastTenantId,
		Lastinsertedat: pagination.LastInsertedAt,
		Lastexternalid: pagination.LastExternalId,
		Batchsize:      candidateBatchNumRows,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to compute olap payload batch size: %w", err)
	}

	if proposedBatchSizeBytes < MAX_BATCH_SIZE_BYTES {
		return &candidateBatchNumRows, nil
	}

	// if the proposed batch size is too large, then
	// cut it in half and try again
	return p.OptimizeOLAPPayloadWindowSize(
		ctx,
		partitionDate,
		candidateBatchNumRows/2,
		pagination,
	)
}

func (p *OLAPRepositoryImpl) processOLAPPayloadCutoverBatch(ctx context.Context, processId uuid.UUID, partitionDate PartitionDate, pagination OLAPPaginationParams, externalCutoverBatchSize, externalCutoverNumConcurrentOffloads int32) (*OLAPCutoverBatchOutcome, error) {
	ctx, span := telemetry.NewSpan(ctx, "OLAPRepository.processOLAPPayloadCutoverBatch")
	defer span.End()

	tableName := fmt.Sprintf("v1_payloads_olap_offload_tmp_%s", partitionDate.String())

	windowSizePtr, err := p.OptimizeOLAPPayloadWindowSize(
		ctx,
		partitionDate,
		externalCutoverBatchSize*externalCutoverNumConcurrentOffloads,
		pagination,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to optimize olap payload window size: %w", err)
	}

	windowSize := *windowSizePtr

	payloadRanges, err := p.queries.CreateOLAPPayloadRangeChunks(ctx, p.pool, sqlcv1.CreateOLAPPayloadRangeChunksParams{
		Chunksize:      externalCutoverBatchSize,
		Partitiondate:  pgtype.Date(partitionDate),
		Windowsize:     windowSize,
		Lasttenantid:   pagination.LastTenantId,
		Lastexternalid: pagination.LastExternalId,
		Lastinsertedat: pagination.LastInsertedAt,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to create payload range chunks: %w", err)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return &OLAPCutoverBatchOutcome{
			ShouldContinue: false,
			NextPagination: pagination,
		}, nil
	}

	mu := sync.Mutex{}
	eg := errgroup.Group{}

	externalIdToPayload := make(map[uuid.UUID]sqlcv1.ListPaginatedOLAPPayloadsForOffloadRow)
	alreadyExternalPayloads := make(map[uuid.UUID]ExternalPayloadLocationKey)
	offloadToExternalStoreOpts := make([]OffloadToExternalStoreOpts, 0)

	numPayloads := 0

	for _, payloadRange := range payloadRanges {
		pr := payloadRange
		eg.Go(func() error {
			payloads, err := p.queries.ListPaginatedOLAPPayloadsForOffload(ctx, p.pool, sqlcv1.ListPaginatedOLAPPayloadsForOffloadParams{
				Partitiondate:  pgtype.Date(partitionDate),
				Lasttenantid:   pr.LowerTenantID,
				Lastexternalid: pr.LowerExternalID,
				Lastinsertedat: pr.LowerInsertedAt,
				Nexttenantid:   pr.UpperTenantID,
				Nextexternalid: pr.UpperExternalID,
				Nextinsertedat: pr.UpperInsertedAt,
				Batchsize:      externalCutoverBatchSize,
			})

			if err != nil {
				return fmt.Errorf("failed to list paginated payloads for offload")
			}

			alreadyExternalPayloadsInner := make(map[uuid.UUID]ExternalPayloadLocationKey)
			externalIdToPayloadInner := make(map[uuid.UUID]sqlcv1.ListPaginatedOLAPPayloadsForOffloadRow)
			offloadToExternalStoreOptsInner := make([]OffloadToExternalStoreOpts, 0)

			for _, payload := range payloads {
				externalId := uuid.UUID(payload.ExternalID)
				externalIdToPayloadInner[externalId] = *payload

				if payload.Location != sqlcv1.V1PayloadLocationOlapINLINE {
					alreadyExternalPayloadsInner[externalId] = ExternalPayloadLocationKey(payload.ExternalLocationKey)
				} else {
					offloadToExternalStoreOptsInner = append(offloadToExternalStoreOptsInner, OffloadToExternalStoreOpts{
						TenantId:   payload.TenantID,
						ExternalID: externalId,
						InsertedAt: payload.InsertedAt,
						Payload:    payload.InlineContent,
					})
				}
			}

			mu.Lock()
			maps.Copy(externalIdToPayload, externalIdToPayloadInner)
			maps.Copy(alreadyExternalPayloads, alreadyExternalPayloadsInner)
			offloadToExternalStoreOpts = append(offloadToExternalStoreOpts, offloadToExternalStoreOptsInner...)
			numPayloads += len(payloads)
			mu.Unlock()

			return nil
		})
	}

	err = eg.Wait()

	if err != nil {
		return nil, err
	}

	externalIdToKey, err := p.PayloadStore().ExternalStore().Store(ctx, offloadToExternalStoreOpts...)

	if err != nil {
		return nil, fmt.Errorf("failed to offload payloads to external store: %w", err)
	}

	maps.Copy(externalIdToKey, alreadyExternalPayloads)

	span.SetAttributes(attribute.Int("num_payloads_read", numPayloads))
	payloadsToInsert := make([]sqlcv1.CutoverOLAPPayloadToInsert, 0, numPayloads)

	for externalId, key := range externalIdToKey {
		payload := externalIdToPayload[externalId]
		payloadsToInsert = append(payloadsToInsert, sqlcv1.CutoverOLAPPayloadToInsert{
			TenantID:            payload.TenantID,
			InsertedAt:          payload.InsertedAt,
			ExternalID:          externalId,
			ExternalLocationKey: string(key),
			Location:            sqlcv1.V1PayloadLocationOlapEXTERNAL,
		})
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare transaction for copying offloaded payloads: %w", err)
	}

	defer rollback()

	inserted, err := sqlcv1.InsertCutOverOLAPPayloadsIntoTempTable(ctx, tx, tableName, payloadsToInsert)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to copy offloaded payloads into temp table: %w", err)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return &OLAPCutoverBatchOutcome{
			ShouldContinue: false,
			NextPagination: pagination,
		}, nil
	}

	extendedLease, err := p.acquireOrExtendJobLease(ctx, tx, processId, partitionDate, OLAPPaginationParams{
		LastTenantId:   inserted.TenantId,
		LastInsertedAt: inserted.InsertedAt,
		LastExternalId: inserted.ExternalId,
		Limit:          pagination.Limit,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to extend cutover job lease: %w", err)
	}

	if err := commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit copy offloaded payloads transaction: %w", err)
	}

	if numPayloads < int(windowSize) {
		return &OLAPCutoverBatchOutcome{
			ShouldContinue: false,
			NextPagination: extendedLease.Pagination,
		}, nil
	}

	return &OLAPCutoverBatchOutcome{
		ShouldContinue: true,
		NextPagination: extendedLease.Pagination,
	}, nil
}

func (p *OLAPRepositoryImpl) acquireOrExtendJobLease(ctx context.Context, tx pgx.Tx, processId uuid.UUID, partitionDate PartitionDate, pagination OLAPPaginationParams) (*OLAPCutoverJobRunMetadata, error) {
	leaseInterval := 2 * time.Minute
	leaseExpiresAt := sqlchelpers.TimestamptzFromTime(time.Now().Add(leaseInterval))

	lease, err := p.queries.AcquireOrExtendOLAPCutoverJobLease(ctx, tx, sqlcv1.AcquireOrExtendOLAPCutoverJobLeaseParams{
		Key:            pgtype.Date(partitionDate),
		Lasttenantid:   pagination.LastTenantId,
		Lastexternalid: pagination.LastExternalId,
		Lastinsertedat: pagination.LastInsertedAt,
		Leaseprocessid: processId,
		Leaseexpiresat: leaseExpiresAt,
	})

	if err != nil {
		// ErrNoRows here means that something else is holding the lease
		// since we did not insert a new record, and the `UPDATE` returned an empty set
		if errors.Is(err, pgx.ErrNoRows) {
			return &OLAPCutoverJobRunMetadata{
				ShouldRun:      false,
				PartitionDate:  partitionDate,
				LeaseProcessId: processId,
			}, nil
		}
		return nil, fmt.Errorf("failed to create initial cutover job lease: %w", err)
	}

	if lease.LeaseProcessID != processId || lease.IsCompleted {
		return &OLAPCutoverJobRunMetadata{
			ShouldRun: false,
			Pagination: OLAPPaginationParams{
				LastTenantId:   lease.LastTenantID,
				LastInsertedAt: lease.LastInsertedAt,
				LastExternalId: lease.LastExternalID,
				Limit:          pagination.Limit,
			},
			PartitionDate:  partitionDate,
			LeaseProcessId: lease.LeaseProcessID,
		}, nil
	}

	return &OLAPCutoverJobRunMetadata{
		ShouldRun: true,
		Pagination: OLAPPaginationParams{
			LastTenantId:   lease.LastTenantID,
			LastInsertedAt: lease.LastInsertedAt,
			LastExternalId: lease.LastExternalID,
			Limit:          pagination.Limit,
		},
		PartitionDate:  partitionDate,
		LeaseProcessId: processId,
	}, nil
}

func (p *OLAPRepositoryImpl) prepareCutoverTableJob(ctx context.Context, processId uuid.UUID, partitionDate PartitionDate, inlineStoreTTL *time.Duration, externalCutoverBatchSize int32) (*OLAPCutoverJobRunMetadata, error) {
	if inlineStoreTTL == nil {
		return nil, fmt.Errorf("inline store TTL is not set")
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	var zeroUuid uuid.UUID

	lease, err := p.acquireOrExtendJobLease(ctx, tx, processId, partitionDate, OLAPPaginationParams{
		LastTenantId:   zeroUuid,
		LastExternalId: zeroUuid,
		LastInsertedAt: sqlchelpers.TimestamptzFromTime(time.Unix(0, 0)),
		Limit:          externalCutoverBatchSize,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to acquire or extend cutover job lease: %w", err)
	}

	if !lease.ShouldRun {
		return lease, nil
	}

	err = p.queries.CreateV1PayloadOLAPCutoverTemporaryTable(ctx, tx, pgtype.Date(partitionDate))

	if err != nil {
		return nil, fmt.Errorf("failed to create payload cutover temporary table: %w", err)
	}

	if err := commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit copy offloaded payloads transaction: %w", err)
	}

	return &OLAPCutoverJobRunMetadata{
		ShouldRun:      true,
		Pagination:     lease.Pagination,
		PartitionDate:  partitionDate,
		LeaseProcessId: processId,
	}, nil
}

func (p *OLAPRepositoryImpl) processSinglePartition(ctx context.Context, processId uuid.UUID, partitionDate PartitionDate, inlineStoreTTL *time.Duration, externalCutoverBatchSize, externalCutoverNumConcurrentOffloads int32) error {
	ctx, span := telemetry.NewSpan(ctx, "olap_repository.processSinglePartition")
	defer span.End()

	jobMeta, err := p.prepareCutoverTableJob(ctx, processId, partitionDate, inlineStoreTTL, externalCutoverBatchSize)

	if err != nil {
		return fmt.Errorf("failed to prepare cutover table job: %w", err)
	}

	if !jobMeta.ShouldRun {
		return nil
	}

	pagination := jobMeta.Pagination

	for {
		outcome, err := p.processOLAPPayloadCutoverBatch(ctx, processId, partitionDate, pagination, externalCutoverBatchSize, externalCutoverNumConcurrentOffloads)

		if err != nil {
			return fmt.Errorf("failed to process payload cutover batch: %w", err)
		}

		if !outcome.ShouldContinue {
			break
		}

		pagination = outcome.NextPagination
	}

	tempPartitionName := fmt.Sprintf("v1_payloads_olap_offload_tmp_%s", partitionDate.String())
	sourcePartitionName := fmt.Sprintf("v1_payloads_olap_%s", partitionDate.String())

	reconciliationDoneChan := make(chan struct{})
	reconciliationCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-reconciliationCtx.Done():
				return
			case <-reconciliationDoneChan:
				return
			case <-ticker.C:
				tx, commit, rollback, err := sqlchelpers.PrepareTx(reconciliationCtx, p.pool, p.l)

				if err != nil {
					p.l.Error().Err(err).Msg("failed to prepare transaction for extending cutover job lease during reconciliation")
					return
				}

				defer rollback()

				lease, err := p.acquireOrExtendJobLease(reconciliationCtx, tx, processId, partitionDate, pagination)

				if err != nil {
					return
				}

				if err := commit(reconciliationCtx); err != nil {
					p.l.Error().Err(err).Msg("failed to commit extend cutover job lease transaction during reconciliation")
					return
				}

				if !lease.ShouldRun {
					return
				}
			}
		}
	}()

	connStatementTimeout := 30 * 60 * 1000 // 30 minutes

	conn, release, err := sqlchelpers.AcquireConnectionWithStatementTimeout(ctx, p.pool, p.l, connStatementTimeout)

	if err != nil {
		return fmt.Errorf("failed to acquire connection with statement timeout: %w", err)
	}

	defer release()

	rowCounts, err := sqlcv1.ComparePartitionRowCounts(ctx, conn, tempPartitionName, sourcePartitionName)

	if err != nil {
		return fmt.Errorf("failed to compare partition row counts: %w", err)
	}

	const maxCountDiff = 5000

	if rowCounts.SourcePartitionCount-rowCounts.TempPartitionCount > maxCountDiff {
		return fmt.Errorf("row counts do not match between temp and source partitions for date %s. off by more than %d", partitionDate.String(), maxCountDiff)
	} else if rowCounts.SourcePartitionCount > rowCounts.TempPartitionCount {
		missingRows, err := p.queries.DiffOLAPPayloadSourceAndTargetPartitions(ctx, conn, pgtype.Date(partitionDate))

		if err != nil {
			return fmt.Errorf("failed to diff source and target partitions: %w", err)
		}

		missingPayloadsToInsert := make([]sqlcv1.CutoverOLAPPayloadToInsert, 0, len(missingRows))

		for _, p := range missingRows {
			missingPayloadsToInsert = append(missingPayloadsToInsert, sqlcv1.CutoverOLAPPayloadToInsert{
				TenantID:            p.TenantID,
				InsertedAt:          p.InsertedAt,
				ExternalID:          p.ExternalID,
				ExternalLocationKey: p.ExternalLocationKey,
				InlineContent:       p.InlineContent,
				Location:            p.Location,
			})
		}

		_, err = sqlcv1.InsertCutOverOLAPPayloadsIntoTempTable(ctx, conn, tempPartitionName, missingPayloadsToInsert)

		if err != nil {
			return fmt.Errorf("failed to insert missing payloads into temp partition: %w", err)
		}

		rowCounts, err := sqlcv1.ComparePartitionRowCounts(ctx, conn, tempPartitionName, sourcePartitionName)

		if err != nil {
			return fmt.Errorf("failed to compare partition row counts: %w", err)
		}

		if rowCounts.SourcePartitionCount != rowCounts.TempPartitionCount {
			return fmt.Errorf("row counts still do not match between temp and source partitions for date %s after inserting missing rows", partitionDate.String())
		}
	}

	close(reconciliationDoneChan)

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l)

	if err != nil {
		return fmt.Errorf("failed to prepare transaction for swapping payload cutover temp table: %w", err)
	}

	defer rollback()

	err = p.queries.SwapV1PayloadOLAPPartitionWithTemp(ctx, tx, pgtype.Date(partitionDate))

	if err != nil {
		return fmt.Errorf("failed to swap payload cutover temp table: %w", err)
	}

	err = p.queries.MarkOLAPCutoverJobAsCompleted(ctx, tx, pgtype.Date(partitionDate))

	if err != nil {
		return fmt.Errorf("failed to mark cutover job as completed: %w", err)
	}

	if err := commit(ctx); err != nil {
		return fmt.Errorf("failed to commit swap payload cutover temp table transaction: %w", err)
	}

	return nil
}

func (p *OLAPRepositoryImpl) ProcessOLAPPayloadCutovers(ctx context.Context, externalStoreEnabled bool, inlineStoreTTL *time.Duration, externalCutoverBatchSize, externalCutoverNumConcurrentOffloads int32) error {
	if !externalStoreEnabled {
		return nil
	}

	ctx, span := telemetry.NewSpan(ctx, "olap_repository.ProcessOLAPPayloadCutovers")
	defer span.End()

	if inlineStoreTTL == nil {
		return fmt.Errorf("inline store TTL is not set")
	}

	mostRecentPartitionToOffload := pgtype.Date{
		Time:  time.Now().Add(-1 * (*inlineStoreTTL + 12*time.Hour)),
		Valid: true,
	}

	partitions, err := p.queries.FindV1OLAPPayloadPartitionsBeforeDate(ctx, p.pool, MAX_PARTITIONS_TO_OFFLOAD, mostRecentPartitionToOffload)

	if err != nil {
		return fmt.Errorf("failed to find payload partitions before date %s: %w", mostRecentPartitionToOffload.Time.String(), err)
	}

	processId := uuid.New()

	for _, partition := range partitions {
		p.l.Info().Str("partition", partition.PartitionName).Msg("processing payload cutover for partition")
		err = p.processSinglePartition(ctx, processId, PartitionDate(partition.PartitionDate), inlineStoreTTL, externalCutoverBatchSize, externalCutoverNumConcurrentOffloads)

		if err != nil {
			return fmt.Errorf("failed to process partition %s: %w", partition.PartitionName, err)
		}
	}

	return nil
}
