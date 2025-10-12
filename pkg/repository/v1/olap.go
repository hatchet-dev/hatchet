package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
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

	ParentTaskExternalId *pgtype.UUID

	TriggeringEventExternalId *pgtype.UUID

	IncludePayloads bool
}

type ReadTaskRunMetricsOpts struct {
	CreatedAfter time.Time

	CreatedBefore *time.Time

	WorkflowIds []uuid.UUID

	ParentTaskExternalID *pgtype.UUID

	TriggeringEventExternalId *pgtype.UUID

	AdditionalMetadata map[string]interface{}
}

type WorkflowRunData struct {
	AdditionalMetadata   []byte                      `json:"additional_metadata"`
	CreatedAt            pgtype.Timestamptz          `json:"created_at"`
	DisplayName          string                      `json:"display_name"`
	ErrorMessage         string                      `json:"error_message"`
	ExternalID           pgtype.UUID                 `json:"external_id"`
	FinishedAt           pgtype.Timestamptz          `json:"finished_at"`
	Input                []byte                      `json:"input"`
	InsertedAt           pgtype.Timestamptz          `json:"inserted_at"`
	Kind                 sqlcv1.V1RunKind            `json:"kind"`
	Output               *[]byte                     `json:"output,omitempty"`
	ParentTaskExternalId *pgtype.UUID                `json:"parent_task_external_id,omitempty"`
	ReadableStatus       sqlcv1.V1ReadableStatusOlap `json:"readable_status"`
	StepId               *pgtype.UUID                `json:"step_id,omitempty"`
	StartedAt            pgtype.Timestamptz          `json:"started_at"`
	TaskExternalId       *pgtype.UUID                `json:"task_external_id,omitempty"`
	TaskId               *int64                      `json:"task_id,omitempty"`
	TaskInsertedAt       *pgtype.Timestamptz         `json:"task_inserted_at,omitempty"`
	TenantID             pgtype.UUID                 `json:"tenant_id"`
	WorkflowID           pgtype.UUID                 `json:"workflow_id"`
	WorkflowVersionId    pgtype.UUID                 `json:"workflow_version_id"`
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
	TenantId       pgtype.UUID
	TaskId         int64
	TaskInsertedAt pgtype.Timestamptz
	ReadableStatus sqlcv1.V1ReadableStatusOlap
	ExternalId     pgtype.UUID
	LatestWorkerId pgtype.UUID
	WorkflowId     pgtype.UUID
	IsDAGTask      bool
}

type UpdateDAGStatusRow struct {
	TenantId       pgtype.UUID
	DagId          int64
	DagInsertedAt  pgtype.Timestamptz
	ReadableStatus sqlcv1.V1ReadableStatusOlap
	ExternalId     pgtype.UUID
	WorkflowId     pgtype.UUID
}

type OLAPRepository interface {
	UpdateTablePartitions(ctx context.Context) error
	SetReadReplicaPool(pool *pgxpool.Pool)

	ReadTaskRun(ctx context.Context, taskExternalId string) (*sqlcv1.V1TasksOlap, error)
	ReadWorkflowRun(ctx context.Context, workflowRunExternalId pgtype.UUID) (*V1WorkflowRunPopulator, error)
	ReadTaskRunData(ctx context.Context, tenantId pgtype.UUID, taskId int64, taskInsertedAt pgtype.Timestamptz, retryCount *int) (*sqlcv1.PopulateSingleTaskRunDataRow, pgtype.UUID, error)

	ListTasks(ctx context.Context, tenantId string, opts ListTaskRunOpts) ([]*sqlcv1.PopulateTaskRunDataRow, int, error)
	ListWorkflowRuns(ctx context.Context, tenantId string, opts ListWorkflowRunOpts) ([]*WorkflowRunData, int, error)
	ListTaskRunEvents(ctx context.Context, tenantId string, taskId int64, taskInsertedAt pgtype.Timestamptz, limit, offset int64) ([]*sqlcv1.ListTaskEventsRow, error)
	ListTaskRunEventsByWorkflowRunId(ctx context.Context, tenantId string, workflowRunId pgtype.UUID) ([]*sqlcv1.ListTaskEventsForWorkflowRunRow, error)
	ListWorkflowRunDisplayNames(ctx context.Context, tenantId pgtype.UUID, externalIds []pgtype.UUID) ([]*sqlcv1.ListWorkflowRunDisplayNamesRow, error)
	ReadTaskRunMetrics(ctx context.Context, tenantId string, opts ReadTaskRunMetricsOpts) ([]TaskRunMetric, error)
	CreateTasks(ctx context.Context, tenantId string, tasks []*V1TaskWithPayload) error
	CreateTaskEvents(ctx context.Context, tenantId string, events []sqlcv1.CreateTaskEventsOLAPParams) error
	CreateDAGs(ctx context.Context, tenantId string, dags []*DAGWithData) error
	GetTaskPointMetrics(ctx context.Context, tenantId string, startTimestamp *time.Time, endTimestamp *time.Time, bucketInterval time.Duration) ([]*sqlcv1.GetTaskPointMetricsRow, error)
	UpdateTaskStatuses(ctx context.Context, tenantIds []string) (bool, []UpdateTaskStatusRow, error)
	UpdateDAGStatuses(ctx context.Context, tenantIds []string) (bool, []UpdateDAGStatusRow, error)
	ReadDAG(ctx context.Context, dagExternalId string) (*sqlcv1.V1DagsOlap, error)
	ListTasksByDAGId(ctx context.Context, tenantId string, dagIds []pgtype.UUID, includePayloads bool) ([]*sqlcv1.PopulateTaskRunDataRow, map[int64]uuid.UUID, error)
	ListTasksByIdAndInsertedAt(ctx context.Context, tenantId string, taskMetadata []TaskMetadata) ([]*sqlcv1.PopulateTaskRunDataRow, error)

	// ListTasksByExternalIds returns a list of tasks based on their external ids or the external id of their parent DAG.
	// In the case of a DAG, we flatten the result into the list of tasks which belong to that DAG.
	ListTasksByExternalIds(ctx context.Context, tenantId string, externalIds []string) ([]*sqlcv1.FlattenTasksByExternalIdsRow, error)

	GetTaskTimings(ctx context.Context, tenantId string, workflowRunId pgtype.UUID, depth int32) ([]*sqlcv1.PopulateTaskRunDataRow, map[string]int32, error)
	BulkCreateEventsAndTriggers(ctx context.Context, events sqlcv1.BulkCreateEventsParams, triggers []EventTriggersFromExternalId) error
	ListEvents(ctx context.Context, opts sqlcv1.ListEventsParams) ([]*ListEventsRow, *int64, error)
	GetEvent(ctx context.Context, externalId string) (*sqlcv1.V1EventsOlap, error)
	ListEventKeys(ctx context.Context, tenantId string) ([]string, error)

	GetDAGDurations(ctx context.Context, tenantId string, externalIds []pgtype.UUID, minInsertedAt pgtype.Timestamptz) (map[string]*sqlcv1.GetDagDurationsRow, error)
	GetTaskDurationsByTaskIds(ctx context.Context, tenantId string, taskIds []int64, taskInsertedAts []pgtype.Timestamptz, readableStatuses []sqlcv1.V1ReadableStatusOlap) (map[int64]*sqlcv1.GetTaskDurationsByTaskIdsRow, error)

	CreateIncomingWebhookValidationFailureLogs(ctx context.Context, tenantId string, opts []CreateIncomingWebhookFailureLogOpts) error
	StoreCELEvaluationFailures(ctx context.Context, tenantId string, failures []CELEvaluationFailure) error
	PutPayloads(ctx context.Context, tx sqlcv1.DBTX, tenantId string, payloads []PutOLAPPayloadOpts) error
	ReadPayload(ctx context.Context, tenantId string, externalId pgtype.UUID) ([]byte, error)
	ReadPayloads(ctx context.Context, tenantId string, externalIds []pgtype.UUID) (map[pgtype.UUID][]byte, error)

	AnalyzeOLAPTables(ctx context.Context) error
	OffloadPayloads(ctx context.Context, tenantId string, payloads []OffloadPayloadOpts) error
}

type OLAPRepositoryImpl struct {
	*sharedRepository

	readPool *pgxpool.Pool

	eventCache *lru.Cache[string, bool]

	olapRetentionPeriod time.Duration

	shouldPartitionEventsTables bool
}

func NewOLAPRepositoryFromPool(pool *pgxpool.Pool, l *zerolog.Logger, olapRetentionPeriod time.Duration, entitlements repository.EntitlementsRepository, shouldPartitionEventsTables bool, payloadStoreOpts PayloadStoreRepositoryOpts) (OLAPRepository, func() error) {
	v := validator.NewDefaultValidator()

	shared, cleanupShared := newSharedRepository(pool, v, l, entitlements, payloadStoreOpts)

	return newOLAPRepository(shared, olapRetentionPeriod, shouldPartitionEventsTables), cleanupShared
}

func newOLAPRepository(shared *sharedRepository, olapRetentionPeriod time.Duration, shouldPartitionEventsTables bool) OLAPRepository {
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

func (r *OLAPRepositoryImpl) SetReadReplicaPool(pool *pgxpool.Pool) {
	r.readPool = pool
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

func (r *OLAPRepositoryImpl) ReadTaskRun(ctx context.Context, taskExternalId string) (*sqlcv1.V1TasksOlap, error) {
	row, err := r.queries.ReadTaskByExternalID(ctx, r.readPool, sqlchelpers.UUIDFromStr(taskExternalId))

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

func (r *OLAPRepositoryImpl) ReadWorkflowRun(ctx context.Context, workflowRunExternalId pgtype.UUID) (*V1WorkflowRunPopulator, error) {
	row, err := r.queries.ReadWorkflowRunByExternalId(ctx, r.readPool, workflowRunExternalId)

	if err != nil {
		return nil, err
	}

	taskMetadata, err := ParseTaskMetadata(row.TaskMetadata)

	if err != nil {
		return nil, err
	}

	inputPayload, err := r.ReadPayload(ctx, row.TenantID.String(), row.ExternalID)

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
			ParentTaskExternalId: &row.ParentTaskExternalID,
		},
		TaskMetadata: taskMetadata,
	}, nil
}

func (r *OLAPRepositoryImpl) ReadTaskRunData(ctx context.Context, tenantId pgtype.UUID, taskId int64, taskInsertedAt pgtype.Timestamptz, retryCount *int) (*sqlcv1.PopulateSingleTaskRunDataRow, pgtype.UUID, error) {
	emptyUUID := pgtype.UUID{}

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

	workflowRunId := pgtype.UUID{}

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

	return taskRun, workflowRunId, nil
}

func (r *OLAPRepositoryImpl) ListTasks(ctx context.Context, tenantId string, opts ListTaskRunOpts) ([]*sqlcv1.PopulateTaskRunDataRow, int, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-tasks-olap")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.readPool, r.l, 10000)

	if err != nil {
		return nil, 0, err
	}

	defer rollback()

	params := sqlcv1.ListTasksOlapParams{
		Tenantid:                  sqlchelpers.UUIDFromStr(tenantId),
		Since:                     sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		Tasklimit:                 int32(opts.Limit),
		Taskoffset:                int32(opts.Offset),
		TriggeringEventExternalId: pgtype.UUID{},
	}

	countParams := sqlcv1.CountTasksParams{
		Tenantid:                  sqlchelpers.UUIDFromStr(tenantId),
		Since:                     sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		TriggeringEventExternalId: pgtype.UUID{},
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

	triggeringEventExternalId := opts.TriggeringEventExternalId

	if triggeringEventExternalId != nil {
		params.TriggeringEventExternalId = sqlchelpers.UUIDFromStr(triggeringEventExternalId.String())
		countParams.TriggeringEventExternalId = sqlchelpers.UUIDFromStr(triggeringEventExternalId.String())
	}

	rows, err := r.queries.ListTasksOlap(ctx, tx, params)

	if err != nil {
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

	count, err := r.queries.CountTasks(ctx, tx, countParams)

	if err != nil {
		count = int64(len(tasksWithData))
	}

	if err := commit(ctx); err != nil {
		return nil, 0, err
	}

	return tasksWithData, int(count), nil
}

func (r *OLAPRepositoryImpl) ListTasksByDAGId(ctx context.Context, tenantId string, dagids []pgtype.UUID, includePayloads bool) ([]*sqlcv1.PopulateTaskRunDataRow, map[int64]uuid.UUID, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-tasks-by-dag-id-olap")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.readPool, r.l, 15000)
	taskIdToDagExternalId := make(map[int64]uuid.UUID)

	if err != nil {
		return nil, taskIdToDagExternalId, err
	}

	defer rollback()

	tasks, err := r.queries.ListTasksByDAGIds(ctx, tx, sqlcv1.ListTasksByDAGIdsParams{
		Dagids:   dagids,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, taskIdToDagExternalId, err
	}

	idsInsertedAts := make([]IdInsertedAt, 0, len(tasks))

	for _, row := range tasks {
		taskIdToDagExternalId[row.TaskID] = uuid.MustParse(sqlchelpers.UUIDToStr(row.DagExternalID))
		idsInsertedAts = append(idsInsertedAts, IdInsertedAt{
			ID:         row.TaskID,
			InsertedAt: row.TaskInsertedAt,
		})
	}

	tasksWithData, err := r.populateTaskRunData(ctx, tx, tenantId, idsInsertedAts, includePayloads)

	if err != nil {
		return nil, taskIdToDagExternalId, err
	}

	if err := commit(ctx); err != nil {
		return nil, taskIdToDagExternalId, err
	}

	return tasksWithData, taskIdToDagExternalId, nil
}

func (r *OLAPRepositoryImpl) ListTasksByIdAndInsertedAt(ctx context.Context, tenantId string, taskMetadata []TaskMetadata) ([]*sqlcv1.PopulateTaskRunDataRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-tasks-by-id-and-inserted-at-olap")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.readPool, r.l, 15000)

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

	tasksWithData, err := r.populateTaskRunData(ctx, tx, tenantId, idsInsertedAts, true)

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return tasksWithData, nil
}

func (r *OLAPRepositoryImpl) ListWorkflowRuns(ctx context.Context, tenantId string, opts ListWorkflowRunOpts) ([]*WorkflowRunData, int, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-workflow-runs-olap")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.readPool, r.l, 30000)

	if err != nil {
		return nil, 0, err
	}

	defer rollback()

	params := sqlcv1.FetchWorkflowRunIdsParams{
		Tenantid:                  sqlchelpers.UUIDFromStr(tenantId),
		Since:                     sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		Listworkflowrunslimit:     int32(opts.Limit),
		Listworkflowrunsoffset:    int32(opts.Offset),
		ParentTaskExternalId:      pgtype.UUID{},
		TriggeringEventExternalId: pgtype.UUID{},
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

	if opts.TriggeringEventExternalId != nil {
		params.TriggeringEventExternalId = *opts.TriggeringEventExternalId
		countParams.TriggeringEventExternalId = *opts.TriggeringEventExternalId
	}

	workflowRunIds, err := r.queries.FetchWorkflowRunIds(ctx, tx, params)

	if err != nil {
		return nil, 0, err
	}

	dagIdsInsertedAts := make([]IdInsertedAt, 0, len(workflowRunIds))
	idsInsertedAts := make([]IdInsertedAt, 0, len(workflowRunIds))

	for _, row := range workflowRunIds {
		if row.Kind == sqlcv1.V1RunKindDAG {
			dagIdsInsertedAts = append(dagIdsInsertedAts, IdInsertedAt{
				ID:         row.ID,
				InsertedAt: row.InsertedAt,
			})
		} else {
			idsInsertedAts = append(idsInsertedAts, IdInsertedAt{
				ID:         row.ID,
				InsertedAt: row.InsertedAt,
			})
		}
	}

	dagsToPopulated := make(map[string]*sqlcv1.PopulateDAGMetadataRow)

	for _, idInsertedAt := range dagIdsInsertedAts {
		dag, err := r.queries.PopulateDAGMetadata(ctx, tx, sqlcv1.PopulateDAGMetadataParams{
			ID:         idInsertedAt.ID,
			Insertedat: idInsertedAt.InsertedAt,
			Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, fmt.Errorf("error populating dag metadata for id %d: %v", idInsertedAt.ID, err)
		}

		if errors.Is(err, pgx.ErrNoRows) {
			r.l.Error().Msgf("could not find dag with id %d and inserted at %s", idInsertedAt.ID, idInsertedAt.InsertedAt.Time.Format(time.RFC3339))
			continue
		}

		dagsToPopulated[sqlchelpers.UUIDToStr(dag.ExternalID)] = dag
	}

	count, err := r.queries.CountWorkflowRuns(ctx, tx, countParams)

	if err != nil {
		r.l.Error().Msgf("error counting workflow runs: %v", err)
		count = int64(len(workflowRunIds))
	}

	tasksToPopulated := make(map[string]*sqlcv1.PopulateTaskRunDataRow)

	populatedTasks, err := r.populateTaskRunData(ctx, tx, tenantId, idsInsertedAts, opts.IncludePayloads)

	if err != nil {
		return nil, 0, err
	}

	for _, task := range populatedTasks {
		externalId := sqlchelpers.UUIDToStr(task.ExternalID)
		tasksToPopulated[externalId] = task
	}

	if err := commit(ctx); err != nil {
		return nil, 0, err
	}

	externalIds := make([]pgtype.UUID, 0, len(workflowRunIds))
	for _, row := range workflowRunIds {
		externalIds = append(externalIds, row.ExternalID)
	}

	externalIdToPayload, err := r.ReadPayloads(ctx, tenantId, externalIds)

	res := make([]*WorkflowRunData, 0)

	for _, row := range workflowRunIds {
		externalId := sqlchelpers.UUIDToStr(row.ExternalID)

		if row.Kind == sqlcv1.V1RunKindDAG {
			dag, ok := dagsToPopulated[externalId]

			outputPayload := externalIdToPayload[dag.OutputEventExternalID]
			inputPayload := externalIdToPayload[dag.ExternalID]

			if !ok {
				r.l.Error().Msgf("could not find dag with external id %s", externalId)
				continue
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
				ErrorMessage:         dag.ErrorMessage,
				Kind:                 sqlcv1.V1RunKindDAG,
				WorkflowVersionId:    dag.WorkflowVersionID,
				TaskExternalId:       nil,
				TaskId:               nil,
				TaskInsertedAt:       nil,
				Output:               &outputPayload,
				Input:                inputPayload,
				ParentTaskExternalId: &dag.ParentTaskExternalID,
				RetryCount:           &retryCount,
			})
		} else {
			task, ok := tasksToPopulated[externalId]

			if !ok {
				r.l.Error().Msgf("could not find task with external id %s", externalId)
				continue
			}

			retryCount := int(task.RetryCount)

			outputPayload := externalIdToPayload[task.OutputEventExternalID]
			inputPayload := externalIdToPayload[task.ExternalID]

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
				ErrorMessage:       task.ErrorMessage,
				Kind:               sqlcv1.V1RunKindTASK,
				TaskExternalId:     &task.ExternalID,
				TaskId:             &task.ID,
				TaskInsertedAt:     &task.InsertedAt,
				Output:             &outputPayload,
				Input:              inputPayload,
				StepId:             &task.StepID,
				RetryCount:         &retryCount,
			})
		}
	}

	return res, int(count), nil
}

func (r *OLAPRepositoryImpl) ListTaskRunEvents(ctx context.Context, tenantId string, taskId int64, taskInsertedAt pgtype.Timestamptz, limit, offset int64) ([]*sqlcv1.ListTaskEventsRow, error) {
	rows, err := r.queries.ListTaskEvents(ctx, r.readPool, sqlcv1.ListTaskEventsParams{
		Tenantid:       sqlchelpers.UUIDFromStr(tenantId),
		Taskid:         taskId,
		Taskinsertedat: taskInsertedAt,
	})

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *OLAPRepositoryImpl) ListTaskRunEventsByWorkflowRunId(ctx context.Context, tenantId string, workflowRunId pgtype.UUID) ([]*sqlcv1.ListTaskEventsForWorkflowRunRow, error) {
	rows, err := r.queries.ListTaskEventsForWorkflowRun(ctx, r.readPool, sqlcv1.ListTaskEventsForWorkflowRunParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Workflowrunid: workflowRunId,
	})

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *OLAPRepositoryImpl) ReadTaskRunMetrics(ctx context.Context, tenantId string, opts ReadTaskRunMetricsOpts) ([]TaskRunMetric, error) {
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

	var triggeringEventExternalId pgtype.UUID
	if opts.TriggeringEventExternalId != nil {
		triggeringEventExternalId = *opts.TriggeringEventExternalId
	}

	var additionalMetaKeys []string
	var additionalMetaValues []string

	for key, value := range opts.AdditionalMetadata {
		additionalMetaKeys = append(additionalMetaKeys, key)
		additionalMetaValues = append(additionalMetaValues, value.(string))
	}

	params := sqlcv1.GetTenantStatusMetricsParams{
		Tenantid:                  sqlchelpers.UUIDFromStr(tenantId),
		Createdafter:              sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		WorkflowIds:               workflowIds,
		ParentTaskExternalId:      parentTaskExternalId,
		TriggeringEventExternalId: triggeringEventExternalId,
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

func (r *OLAPRepositoryImpl) writeTaskEventBatch(ctx context.Context, tenantId string, events []sqlcv1.CreateTaskEventsOLAPParams) error {
	// skip any events which have a corresponding event already
	eventsToWrite := make([]sqlcv1.CreateTaskEventsOLAPParams, 0)
	tmpEventsToWrite := make([]sqlcv1.CreateTaskEventsOLAPTmpParams, 0)
	payloadsToWrite := make([]PutOLAPPayloadOpts, 0)

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

		if len(event.Output) > 0 {
			payloadsToWrite = append(payloadsToWrite, PutOLAPPayloadOpts{
				StoreOLAPPayloadOpts: &StoreOLAPPayloadOpts{
					ExternalId: event.ExternalID,
					InsertedAt: event.TaskInsertedAt,
					Payload:    event.Output,
				},
				Location: sqlcv1.V1PayloadLocationOlapINLINE,
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

	err = r.PutPayloads(ctx, tx, tenantId, payloadsToWrite)

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	r.saveEventsToCache(eventsToWrite)

	return nil
}

func (r *OLAPRepositoryImpl) UpdateTaskStatuses(ctx context.Context, tenantIds []string) (bool, []UpdateTaskStatusRow, error) {
	var limit int32 = 10000

	// each partition gets its own goroutine
	eg := &errgroup.Group{}
	mu := sync.Mutex{}
	rows := make([]UpdateTaskStatusRow, 0)

	// if any of the partitions are saturated, we return true
	isSaturated := false

	tenantIdUUIDs := make([]pgtype.UUID, len(tenantIds))
	for i, tenantId := range tenantIds {
		tenantIdUUIDs[i] = sqlchelpers.UUIDFromStr(tenantId)
	}

	for i := 0; i < NUM_PARTITIONS; i++ {
		partitionNumber := i

		eg.Go(func() error {
			tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 15000)

			if err != nil {
				return err
			}

			defer rollback()

			minInsertedAt, err := r.queries.FindMinInsertedAtForTaskStatusUpdates(ctx, tx, sqlcv1.FindMinInsertedAtForTaskStatusUpdatesParams{
				Partitionnumber: int32(partitionNumber), // nolint: gosec
				Tenantids:       tenantIdUUIDs,
				Eventlimit:      limit,
			})

			if err != nil {
				return fmt.Errorf("failed to find min inserted at for task status updates: %w", err)
			}

			statusUpdateRes, err := r.queries.UpdateTaskStatuses(ctx, tx, sqlcv1.UpdateTaskStatusesParams{
				Partitionnumber: int32(partitionNumber), // nolint: gosec
				Tenantids:       tenantIdUUIDs,
				Eventlimit:      limit,
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

				rows = append(rows, UpdateTaskStatusRow{
					TenantId:       row.TenantID,
					TaskId:         row.ID,
					TaskInsertedAt: row.InsertedAt,
					ReadableStatus: row.ReadableStatus,
					ExternalId:     row.ExternalID,
					LatestWorkerId: row.LatestWorkerID,
					WorkflowId:     row.WorkflowID,
					IsDAGTask:      row.IsDagTask,
				})
			}

			isSaturated = isSaturated || eventCount == int(limit)

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return false, nil, err
	}

	return isSaturated, rows, nil
}

func (r *OLAPRepositoryImpl) UpdateDAGStatuses(ctx context.Context, tenantIds []string) (bool, []UpdateDAGStatusRow, error) {
	var limit int32 = 10000

	// each partition gets its own goroutine
	eg := &errgroup.Group{}
	mu := sync.Mutex{}
	rows := make([]UpdateDAGStatusRow, 0)

	// if any of the partitions are saturated, we return true
	isSaturated := false

	tenantIdUUIDs := make([]pgtype.UUID, len(tenantIds))
	for i, tenantId := range tenantIds {
		tenantIdUUIDs[i] = sqlchelpers.UUIDFromStr(tenantId)
	}

	for i := 0; i < NUM_PARTITIONS; i++ {
		partitionNumber := i

		eg.Go(func() error {
			tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 15000)

			if err != nil {
				return fmt.Errorf("failed to prepare transaction: %w", err)
			}

			defer rollback()

			minInsertedAt, err := r.queries.FindMinInsertedAtForDAGStatusUpdates(ctx, tx, sqlcv1.FindMinInsertedAtForDAGStatusUpdatesParams{
				Partitionnumber: int32(partitionNumber), // nolint: gosec
				Tenantids:       tenantIdUUIDs,
				Eventlimit:      limit,
			})

			if err != nil {
				return fmt.Errorf("failed to find min inserted at for DAG status updates: %w", err)
			}

			statusUpdateRes, err := r.queries.UpdateDAGStatuses(ctx, tx, sqlcv1.UpdateDAGStatusesParams{
				Partitionnumber: int32(partitionNumber), // nolint: gosec
				Tenantids:       tenantIdUUIDs,
				Eventlimit:      limit,
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

			isSaturated = isSaturated || eventCount == int(limit)

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return false, nil, fmt.Errorf("failed to wait for status update goroutines: %w", err)
	}

	return isSaturated, rows, nil
}

func (r *OLAPRepositoryImpl) writeTaskBatch(ctx context.Context, tenantId string, tasks []*V1TaskWithPayload) error {
	params := make([]sqlcv1.CreateTasksOLAPParams, 0)
	putPayloadOpts := make([]PutOLAPPayloadOpts, 0)

	for _, task := range tasks {
		payload := task.Payload

		// fall back to input if payload is empty
		// for backwards compatibility
		if len(payload) == 0 {
			r.l.Error().Msgf("writeTaskBatch: task %s with ID %d and inserted_at %s has empty payload, falling back to input", task.ExternalID.String(), task.ID, task.InsertedAt.Time)
			payload = task.Input
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
			Input:                payload,
		})

		putPayloadOpts = append(putPayloadOpts, PutOLAPPayloadOpts{
			StoreOLAPPayloadOpts: &StoreOLAPPayloadOpts{
				ExternalId: task.ExternalID,
				InsertedAt: task.InsertedAt,
				Payload:    payload,
			},
			Location: sqlcv1.V1PayloadLocationOlapINLINE,
		})
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)
	if err != nil {
		return err
	}
	defer rollback()

	_, err = r.queries.CreateTasksOLAP(ctx, tx, params)
	if err != nil {
		return err
	}

	err = r.PutPayloads(ctx, tx, tenantId, putPayloadOpts)

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	return nil
}

func (r *OLAPRepositoryImpl) writeDAGBatch(ctx context.Context, tenantId string, dags []*DAGWithData) error {
	params := make([]sqlcv1.CreateDAGsOLAPParams, 0)
	putPayloadOpts := make([]PutOLAPPayloadOpts, 0)

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
			AdditionalMetadata:   dag.AdditionalMetadata,
			ParentTaskExternalID: parentTaskExternalID,
			TotalTasks:           int32(dag.TotalTasks), // nolint: gosec
			Input:                dag.Input,
		})

		putPayloadOpts = append(putPayloadOpts, PutOLAPPayloadOpts{
			StoreOLAPPayloadOpts: &StoreOLAPPayloadOpts{
				ExternalId: dag.ExternalID,
				InsertedAt: dag.InsertedAt,
				Payload:    dag.Input,
			},
			Location: sqlcv1.V1PayloadLocationOlapINLINE,
		})
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)
	if err != nil {
		return err
	}
	defer rollback()

	_, err = r.queries.CreateDAGsOLAP(ctx, tx, params)
	if err != nil {
		return err
	}

	err = r.PutPayloads(ctx, tx, tenantId, putPayloadOpts)

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	return nil
}

func (r *OLAPRepositoryImpl) CreateTaskEvents(ctx context.Context, tenantId string, events []sqlcv1.CreateTaskEventsOLAPParams) error {
	return r.writeTaskEventBatch(ctx, tenantId, events)
}

func (r *OLAPRepositoryImpl) CreateTasks(ctx context.Context, tenantId string, tasks []*V1TaskWithPayload) error {
	return r.writeTaskBatch(ctx, tenantId, tasks)
}

func (r *OLAPRepositoryImpl) CreateDAGs(ctx context.Context, tenantId string, dags []*DAGWithData) error {
	return r.writeDAGBatch(ctx, tenantId, dags)
}

func (r *OLAPRepositoryImpl) GetTaskPointMetrics(ctx context.Context, tenantId string, startTimestamp *time.Time, endTimestamp *time.Time, bucketInterval time.Duration) ([]*sqlcv1.GetTaskPointMetricsRow, error) {
	rows, err := r.queries.GetTaskPointMetrics(ctx, r.readPool, sqlcv1.GetTaskPointMetricsParams{
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

func (r *OLAPRepositoryImpl) ReadDAG(ctx context.Context, dagExternalId string) (*sqlcv1.V1DagsOlap, error) {
	return r.queries.ReadDAGByExternalID(ctx, r.readPool, sqlchelpers.UUIDFromStr(dagExternalId))
}

func (r *OLAPRepositoryImpl) ListTasksByExternalIds(ctx context.Context, tenantId string, externalIds []string) ([]*sqlcv1.FlattenTasksByExternalIdsRow, error) {
	externalUUIDs := make([]pgtype.UUID, 0)

	for _, id := range externalIds {
		externalUUIDs = append(externalUUIDs, sqlchelpers.UUIDFromStr(id))
	}

	return r.queries.FlattenTasksByExternalIds(ctx, r.readPool, sqlcv1.FlattenTasksByExternalIdsParams{
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

func (r *OLAPRepositoryImpl) ListWorkflowRunDisplayNames(ctx context.Context, tenantId pgtype.UUID, externalIds []pgtype.UUID) ([]*sqlcv1.ListWorkflowRunDisplayNamesRow, error) {
	return r.queries.ListWorkflowRunDisplayNames(ctx, r.readPool, sqlcv1.ListWorkflowRunDisplayNamesParams{
		Tenantid:    tenantId,
		Externalids: externalIds,
	})
}

func (r *OLAPRepositoryImpl) GetTaskTimings(ctx context.Context, tenantId string, workflowRunId pgtype.UUID, depth int32) ([]*sqlcv1.PopulateTaskRunDataRow, map[string]int32, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-task-timings-olap")
	defer span.End()

	if depth > 10 {
		return nil, nil, fmt.Errorf("depth too large")
	}

	// start out by getting a list of task external ids for the workflow run id
	rootTaskExternalIds := make([]pgtype.UUID, 0)
	sevenDaysAgo := time.Now().Add(-time.Hour * 24 * 7)
	minInsertedAt := time.Now()

	rootTasks, err := r.queries.FlattenTasksByExternalIds(ctx, r.readPool, sqlcv1.FlattenTasksByExternalIdsParams{
		Externalids: []pgtype.UUID{workflowRunId},
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
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
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
		Depth:           depth,
		Createdafter:    sqlchelpers.TimestamptzFromTime(minInsertedAt),
	})

	if err != nil {
		return nil, nil, err
	}

	// associate each run external id with a depth
	idsToDepth := make(map[string]int32)
	idsInsertedAts := make([]IdInsertedAt, 0, len(runsList))

	for _, row := range runsList {
		idsToDepth[sqlchelpers.UUIDToStr(row.ExternalID)] = row.Depth
		idsInsertedAts = append(idsInsertedAts, IdInsertedAt{
			ID:         row.ID,
			InsertedAt: row.InsertedAt,
		})
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.readPool, r.l, 30000)
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
	EventExternalId pgtype.UUID        `json:"event_external_id"`
	EventSeenAt     pgtype.Timestamptz `json:"event_seen_at"`
	FilterId        pgtype.UUID        `json:"filter_id"`
}

func (r *OLAPRepositoryImpl) BulkCreateEventsAndTriggers(ctx context.Context, events sqlcv1.BulkCreateEventsParams, triggers []EventTriggersFromExternalId) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return fmt.Errorf("error beginning transaction: %v", err)
	}

	defer rollback()

	insertedEvents, err := r.queries.BulkCreateEvents(ctx, tx, events)

	if err != nil {
		return fmt.Errorf("error creating events: %v", err)
	}

	eventExternalIdToId := make(map[pgtype.UUID]int64)

	for _, event := range insertedEvents {
		eventExternalIdToId[event.ExternalID] = event.ID
	}

	bulkCreateTriggersParams := make([]sqlcv1.BulkCreateEventTriggersParams, 0)

	for _, trigger := range triggers {
		eventId, ok := eventExternalIdToId[trigger.EventExternalId]

		if !ok {
			return fmt.Errorf("event external id %s not found in events", sqlchelpers.UUIDToStr(trigger.EventExternalId))
		}

		bulkCreateTriggersParams = append(bulkCreateTriggersParams, sqlcv1.BulkCreateEventTriggersParams{
			RunID:         trigger.RunID,
			RunInsertedAt: trigger.RunInsertedAt,
			EventID:       eventId,
			EventSeenAt:   trigger.EventSeenAt,
			FilterID:      trigger.FilterId,
		})
	}

	_, err = r.queries.BulkCreateEventTriggers(ctx, tx, bulkCreateTriggersParams)

	if err != nil {
		return fmt.Errorf("error creating event triggers: %v", err)
	}

	if err := commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	return nil
}

func (r *OLAPRepositoryImpl) GetEvent(ctx context.Context, externalId string) (*sqlcv1.V1EventsOlap, error) {
	return r.queries.GetEventByExternalId(ctx, r.readPool, sqlchelpers.UUIDFromStr(externalId))
}

type ListEventsRow struct {
	TenantID                pgtype.UUID        `json:"tenant_id"`
	EventID                 int64              `json:"event_id"`
	EventExternalID         pgtype.UUID        `json:"event_external_id"`
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

func (r *OLAPRepositoryImpl) ListEvents(ctx context.Context, opts sqlcv1.ListEventsParams) ([]*ListEventsRow, *int64, error) {
	events, err := r.queries.ListEvents(ctx, r.readPool, opts)

	if err != nil {
		return nil, nil, err
	}

	eventCount, err := r.queries.CountEvents(ctx, r.readPool, sqlcv1.CountEventsParams{
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
		return nil, nil, err
	}

	eventExternalIds := make([]pgtype.UUID, len(events))

	for i, event := range events {
		eventExternalIds[i] = event.ExternalID
	}

	eventData, err := r.queries.PopulateEventData(ctx, r.readPool, sqlcv1.PopulateEventDataParams{
		Eventexternalids: eventExternalIds,
		Tenantid:         opts.Tenantid,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("error populating event data: %v", err)
	}

	externalIdToEventData := make(map[pgtype.UUID][]*sqlcv1.PopulateEventDataRow)

	for _, data := range eventData {
		externalIdToEventData[data.ExternalID] = append(externalIdToEventData[data.ExternalID], data)
	}

	result := make([]*ListEventsRow, 0)

	for _, event := range events {
		data, exists := externalIdToEventData[event.ExternalID]
		var triggeringWebhookName *string

		if event.TriggeringWebhookName.Valid {
			triggeringWebhookName = &event.TriggeringWebhookName.String
		}

		if !exists || len(data) == 0 {

			result = append(result, &ListEventsRow{
				TenantID:                event.TenantID,
				EventID:                 event.ID,
				EventExternalID:         event.ExternalID,
				EventSeenAt:             event.SeenAt,
				EventKey:                event.Key,
				EventPayload:            event.Payload,
				EventAdditionalMetadata: event.AdditionalMetadata,
				EventScope:              event.Scope.String,
				QueuedCount:             0,
				RunningCount:            0,
				CompletedCount:          0,
				CancelledCount:          0,
				FailedCount:             0,
				TriggeringWebhookName:   triggeringWebhookName,
			})
		} else {
			for _, d := range data {
				result = append(result, &ListEventsRow{
					TenantID:                event.TenantID,
					EventID:                 event.ID,
					EventExternalID:         event.ExternalID,
					EventSeenAt:             event.SeenAt,
					EventKey:                event.Key,
					EventPayload:            event.Payload,
					EventAdditionalMetadata: event.AdditionalMetadata,
					EventScope:              event.Scope.String,
					QueuedCount:             d.QueuedCount,
					RunningCount:            d.RunningCount,
					CompletedCount:          d.CompletedCount,
					CancelledCount:          d.CancelledCount,
					FailedCount:             d.FailedCount,
					TriggeredRuns:           d.TriggeredRuns,
					TriggeringWebhookName:   triggeringWebhookName,
				})
			}
		}
	}

	return result, &eventCount, nil
}

func (r *OLAPRepositoryImpl) ListEventKeys(ctx context.Context, tenantId string) ([]string, error) {
	keys, err := r.queries.ListEventKeys(ctx, r.pool, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (r *OLAPRepositoryImpl) GetDAGDurations(ctx context.Context, tenantId string, externalIds []pgtype.UUID, minInsertedAt pgtype.Timestamptz) (map[string]*sqlcv1.GetDagDurationsRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "olap_repository.get_dag_durations")
	defer span.End()

	span.SetAttributes(attribute.KeyValue{
		Key:   "olap_repository.get_dag_durations.batch_size",
		Value: attribute.IntValue(len(externalIds)),
	})

	rows, err := r.queries.GetDagDurations(ctx, r.readPool, sqlcv1.GetDagDurationsParams{
		Externalids:   externalIds,
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Mininsertedat: minInsertedAt,
	})

	if err != nil {
		return nil, err
	}

	dagDurations := make(map[string]*sqlcv1.GetDagDurationsRow)

	for _, row := range rows {
		dagDurations[sqlchelpers.UUIDToStr(row.ExternalID)] = row
	}

	return dagDurations, nil
}

func (r *OLAPRepositoryImpl) GetTaskDurationsByTaskIds(ctx context.Context, tenantId string, taskIds []int64, taskInsertedAts []pgtype.Timestamptz, readableStatuses []sqlcv1.V1ReadableStatusOlap) (map[int64]*sqlcv1.GetTaskDurationsByTaskIdsRow, error) {
	rows, err := r.queries.GetTaskDurationsByTaskIds(ctx, r.readPool, sqlcv1.GetTaskDurationsByTaskIdsParams{
		Taskids:          taskIds,
		Taskinsertedats:  taskInsertedAts,
		Tenantid:         sqlchelpers.UUIDFromStr(tenantId),
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

func (r *OLAPRepositoryImpl) CreateIncomingWebhookValidationFailureLogs(ctx context.Context, tenantId string, opts []CreateIncomingWebhookFailureLogOpts) error {
	incomingWebhookNames := make([]string, len(opts))
	errors := make([]string, len(opts))

	for i, opt := range opts {
		incomingWebhookNames[i] = opt.WebhookName
		errors[i] = opt.ErrorText
	}

	params := sqlcv1.CreateIncomingWebhookValidationFailureLogsParams{
		Tenantid:             sqlchelpers.UUIDFromStr(tenantId),
		Incomingwebhooknames: incomingWebhookNames,
		Errors:               errors,
	}

	return r.queries.CreateIncomingWebhookValidationFailureLogs(ctx, r.pool, params)
}

type CELEvaluationFailure struct {
	Source       sqlcv1.V1CelEvaluationFailureSource `json:"source"`
	ErrorMessage string                              `json:"error_message"`
}

func (r *OLAPRepositoryImpl) StoreCELEvaluationFailures(ctx context.Context, tenantId string, failures []CELEvaluationFailure) error {
	errorMessages := make([]string, len(failures))
	sources := make([]string, len(failures))

	for i, failure := range failures {
		errorMessages[i] = failure.ErrorMessage
		sources[i] = string(failure.Source)
	}

	return r.queries.StoreCELEvaluationFailures(ctx, r.pool, sqlcv1.StoreCELEvaluationFailuresParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Sources:  sources,
		Errors:   errorMessages,
	})
}

type OffloadPayloadOpts struct {
	ExternalId          pgtype.UUID
	ExternalLocationKey string
}

type PutOLAPPayloadOpts struct {
	*StoreOLAPPayloadOpts
	Location sqlcv1.V1PayloadLocationOlap
}

func (r *OLAPRepositoryImpl) PutPayloads(ctx context.Context, tx sqlcv1.DBTX, tenantId string, putPayloadOpts []PutOLAPPayloadOpts) error {
	insertedAts := make([]pgtype.Timestamptz, len(putPayloadOpts))
	tenantIds := make([]pgtype.UUID, len(putPayloadOpts))
	externalIds := make([]pgtype.UUID, len(putPayloadOpts))
	payloads := make([][]byte, len(putPayloadOpts))
	locations := make([]string, len(putPayloadOpts))

	for i, opt := range putPayloadOpts {
		externalIds[i] = opt.ExternalId
		insertedAts[i] = opt.InsertedAt
		tenantIds[i] = sqlchelpers.UUIDFromStr(tenantId)
		payloads[i] = opt.Payload
		locations[i] = string(opt.Location)
	}

	return r.queries.PutPayloads(ctx, tx, sqlcv1.PutPayloadsParams{
		Externalids: externalIds,
		Insertedats: insertedAts,
		Tenantids:   tenantIds,
		Payloads:    payloads,
		Locations:   locations,
	})
}

func (r *OLAPRepositoryImpl) ReadPayload(ctx context.Context, tenantId string, externalId pgtype.UUID) ([]byte, error) {
	payloads, err := r.ReadPayloads(ctx, tenantId, []pgtype.UUID{externalId})

	if err != nil {
		return nil, err
	}

	payload, exists := payloads[externalId]

	if !exists {
		r.l.Warn().Msgf("payload for external ID %s not found", sqlchelpers.UUIDToStr(externalId))
	}

	return payload, nil
}

func (r *OLAPRepositoryImpl) ReadPayloads(ctx context.Context, tenantId string, externalIds []pgtype.UUID) (map[pgtype.UUID][]byte, error) {
	payloads, err := r.queries.ReadPayloadsOLAP(ctx, r.readPool, sqlcv1.ReadPayloadsOLAPParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Externalids: externalIds,
	})

	if err != nil {
		return nil, err
	}

	externalIdToPayload := make(map[pgtype.UUID][]byte)
	externalIdToExternalKey := make(map[pgtype.UUID]ExternalPayloadLocationKey)
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

	keyToPayload, err := r.payloadStore.RetrieveFromExternal(ctx, externalKeys)

	if err != nil {
		return nil, err
	}

	for externalId, externalKey := range externalIdToExternalKey {
		externalIdToPayload[externalId] = keyToPayload[externalKey]
	}

	return externalIdToPayload, nil
}

func (r *OLAPRepositoryImpl) OffloadPayloads(ctx context.Context, tenantId string, payloads []OffloadPayloadOpts) error {
	tenantIds := make([]pgtype.UUID, len(payloads))
	externalIds := make([]pgtype.UUID, len(payloads))
	externalLocationKeys := make([]string, len(payloads))

	for i, opt := range payloads {
		externalIds[i] = opt.ExternalId
		tenantIds[i] = sqlchelpers.UUIDFromStr(tenantId)
		externalLocationKeys[i] = opt.ExternalLocationKey
	}

	return r.queries.OffloadPayloads(ctx, r.pool, sqlcv1.OffloadPayloadsParams{
		Externalids:          externalIds,
		Tenantids:            tenantIds,
		Externallocationkeys: externalLocationKeys,
	})
}

func (r *OLAPRepositoryImpl) AnalyzeOLAPTables(ctx context.Context) error {
	const timeout = 1000 * 60 * 60 // 60 minute timeout
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, timeout)

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

	if err := commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	return nil
}

type IdInsertedAt struct {
	ID         int64              `json:"id"`
	InsertedAt pgtype.Timestamptz `json:"inserted_at"`
}

func (r *OLAPRepositoryImpl) populateTaskRunData(ctx context.Context, tx pgx.Tx, tenantId string, opts []IdInsertedAt, includePayloads bool) ([]*sqlcv1.PopulateTaskRunDataRow, error) {
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

	idInsertedAtToData := make(map[IdInsertedAt]*sqlcv1.PopulateTaskRunDataRow)

	for idInsertedAt := range uniqueTaskIdInsertedAts {
		taskData, err := r.queries.PopulateTaskRunData(ctx, tx, sqlcv1.PopulateTaskRunDataParams{
			Taskid:         idInsertedAt.ID,
			Taskinsertedat: idInsertedAt.InsertedAt,
			Tenantid:       sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}

		if errors.Is(err, pgx.ErrNoRows) {
			r.l.Warn().Msgf("task %d not found with inserted at %s", idInsertedAt.ID, idInsertedAt.InsertedAt.Time)
			continue
		}

		idInsertedAtToData[idInsertedAt] = taskData
	}

	result := make([]*sqlcv1.PopulateTaskRunDataRow, 0)
	for _, taskData := range idInsertedAtToData {
		result = append(result, taskData)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].InsertedAt.Time.Equal(result[j].InsertedAt.Time) {
			return result[i].ID < result[j].ID
		}

		return result[i].InsertedAt.Time.After(result[j].InsertedAt.Time)
	})

	return result, nil

}
