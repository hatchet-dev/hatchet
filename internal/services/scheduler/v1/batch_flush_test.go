package scheduler

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	repov1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func TestBatchFlushEmitsStartMessages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mq := newFakeMessageQueue()
	taskRepo := &fakeTaskRepository{}
	repo := &fakeRepository{tasks: taskRepo}

	logger := zerolog.New(io.Discard)

	sched := &Scheduler{
		mq:        mq,
		pubBuffer: msgqueue.NewMQPubBuffer(mq),
		repov1:    repo,
		l:         &logger,
	}

	triggeredAt := time.Now().UTC().Truncate(time.Millisecond)
	const batchKey = "customer-42"
	const maxRuns = 3

	req := &batchFlushRequest{
		TenantID:                "tenant-1",
		StepID:                  "step-1",
		ActionID:                "action-A",
		DispatcherID:            "dispatcher-1",
		WorkerID:                "worker-1",
		BatchID:                 "batch-123",
		BatchKey:                batchKey,
		MaxRuns:                 maxRuns,
		FlushReason:             flushReasonBatchSizeReached,
		ConfiguredBatchSize:     2,
		ConfiguredFlushInterval: 5 * time.Second,
		TriggeredAt:             triggeredAt,
		Items: []*repov1.AssignedItem{
			newAssignedItem(10, "action-A", triggeredAt.Add(-2*time.Second)),
			newAssignedItem(11, "action-A", triggeredAt.Add(-1*time.Second)),
		},
	}

	err := sched.batchFlush(ctx, req)
	require.NoError(t, err)

	require.Len(t, taskRepo.updateCalls, 1)
	update := taskRepo.updateCalls[0]
	assert.Equal(t, "tenant-1", update.tenantID)
	assert.Equal(t, "batch-123", update.batchID)
	assert.Equal(t, "worker-1", update.workerID)
	assert.Equal(t, batchKey, update.batchKey)
	assert.Equal(t, 2, update.batchSize)
	require.Len(t, update.assignments, 2)
	assert.Equal(t, int64(10), update.assignments[0].TaskID)
	assert.Equal(t, int64(11), update.assignments[1].TaskID)
	dispatcherQueue := msgqueue.QueueTypeFromDispatcherID("dispatcher-1")
	messages := mq.MessagesForQueue(dispatcherQueue)

	require.Len(t, messages, 2, "expected start-batch and task-assigned messages")

	var startPayload tasktypes.StartBatchTaskPayload
	err = json.Unmarshal(messages[0].Payloads[0], &startPayload)
	require.NoError(t, err)
	assert.Equal(t, "batch-123", startPayload.BatchId)
	assert.Equal(t, "worker-1", startPayload.WorkerId)
	assert.Equal(t, "tenant-1", startPayload.TenantId)
	assert.Equal(t, 2, startPayload.ExpectedSize)
	assert.Equal(t, "batch size threshold 2 reached", startPayload.TriggerReason)
	assert.WithinDuration(t, triggeredAt, startPayload.TriggerTime, time.Millisecond)
	assert.Equal(t, batchKey, startPayload.BatchKey)
	if assert.NotNil(t, startPayload.MaxRuns) {
		assert.Equal(t, maxRuns, *startPayload.MaxRuns)
	}

	var assignedPayload tasktypes.TaskAssignedBulkTaskPayload
	err = json.Unmarshal(messages[1].Payloads[0], &assignedPayload)
	require.NoError(t, err)
	require.Contains(t, assignedPayload.WorkerBatches, "worker-1")
	require.Len(t, assignedPayload.WorkerBatches["worker-1"], 1)

	batch := assignedPayload.WorkerBatches["worker-1"][0]
	assert.Equal(t, "batch-123", batch.BatchID)
	assert.Equal(t, 2, batch.BatchSize)
	assert.Equal(t, []int64{10, 11}, batch.TaskIds)

	time.Sleep(20 * time.Millisecond)
	monitoringMessages := mq.MessagesForQueue(msgqueue.OLAP_QUEUE)
	require.Len(t, monitoringMessages, 2, "expected monitoring event per task")

	var monitoringPayload tasktypes.CreateMonitoringEventPayload
	err = json.Unmarshal(monitoringMessages[0].Payloads[0], &monitoringPayload)
	require.NoError(t, err)
	assert.Equal(t, sqlcv1.V1EventTypeOlapBATCHFLUSHED, monitoringPayload.EventType)
	var payloadBody map[string]any
	require.NoError(t, json.Unmarshal([]byte(monitoringPayload.EventPayload), &payloadBody))
	assert.Equal(t, "flushed", payloadBody["status"])
	assert.Equal(t, "batch-123", payloadBody["batchId"])
	assert.Equal(t, batchKey, payloadBody["batchKey"])
	assert.EqualValues(t, 2, payloadBody["batchSize"])
	assert.EqualValues(t, 2, payloadBody["expectedSize"])
	assert.EqualValues(t, maxRuns, payloadBody["maxRuns"])
	assert.Contains(t, monitoringPayload.EventMessage, "Batch batch-123 flushed")
	assert.Contains(t, monitoringPayload.EventMessage, batchKey)
	assert.Contains(t, monitoringPayload.EventMessage, "Max concurrent batches per key")
}

func TestBatchBufferPerWorkerNoPrematureFlush(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := zerolog.New(io.Discard)

	var mu sync.Mutex
	var requests []*batchFlushRequest

	manager := newBatchBufferManager(&logger, func(_ context.Context, req *batchFlushRequest) error {
		mu.Lock()
		defer mu.Unlock()

		requests = append(requests, req)
		return nil
	}, func(context.Context, *batchFlushRequest) (bool, error) { return true, nil })

	cfg := batchConfig{
		batchSize: 2,
	}

	baseTime := time.Now().UTC()

	add := func(taskID int64, workerID string, insertedAt time.Time) {
		item := newAssignedItem(taskID, "action-A", insertedAt)
		_, err := manager.Add(ctx, "tenant-1", "step-1", "action-A", "dispatcher-1", workerID, "key", cfg, item)
		require.NoError(t, err)
	}

	add(1, "worker-1", baseTime)
	add(2, "worker-2", baseTime.Add(10*time.Millisecond))

	mu.Lock()
	require.Len(t, requests, 0, "expected no flush when switching workers")
	mu.Unlock()

	add(3, "worker-1", baseTime.Add(20*time.Millisecond))

	mu.Lock()
	require.Len(t, requests, 1, "expected flush after worker-1 batch filled")
	first := requests[0]
	mu.Unlock()

	require.Equal(t, "worker-1", first.WorkerID)
	require.Equal(t, flushReasonBatchSizeReached, first.FlushReason)
	require.Len(t, first.Items, 2)

	add(4, "worker-2", baseTime.Add(30*time.Millisecond))

	mu.Lock()
	require.Len(t, requests, 2, "expected flush after worker-2 batch filled")
	second := requests[1]
	mu.Unlock()

	require.Equal(t, "worker-2", second.WorkerID)
	require.Equal(t, flushReasonBatchSizeReached, second.FlushReason)
	require.Len(t, second.Items, 2)
}

func TestBatchBufferDefersUntilFlushAllowed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := zerolog.New(io.Discard)

	var mu sync.Mutex
	var requests []*batchFlushRequest
	callCount := 0

	manager := newBatchBufferManager(
		&logger,
		func(_ context.Context, req *batchFlushRequest) error {
			mu.Lock()
			requests = append(requests, req)
			mu.Unlock()
			return nil
		},
		func(context.Context, *batchFlushRequest) (bool, error) {
			callCount++
			return callCount > 1, nil
		},
	)

	cfg := batchConfig{
		batchSize:     2,
		flushInterval: 20 * time.Millisecond,
	}

	add := func(taskID int64) {
		item := newAssignedItem(taskID, "action-A", time.Now().UTC())
		_, err := manager.Add(
			ctx,
			"tenant-1",
			"step-1",
			"action-A",
			"dispatcher-1",
			"worker-1",
			"key-123",
			cfg,
			item,
		)
		require.NoError(t, err)
	}

	add(1)
	add(2)

	mu.Lock()
	require.Len(t, requests, 0, "flush should be deferred when not allowed")
	mu.Unlock()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	require.Len(t, requests, 1, "flush should occur after retry timer")
	flushed := requests[0]
	mu.Unlock()

	assert.Equal(t, 2, callCount)
	assert.Equal(t, "key-123", flushed.BatchKey)
	assert.Equal(t, flushReasonIntervalElapsed, flushed.FlushReason)
	assert.Len(t, flushed.Items, 2)
}

func newAssignedItem(taskID int64, actionID string, insertedAt time.Time) *repov1.AssignedItem {
	return &repov1.AssignedItem{
		QueueItem: &sqlcv1.V1QueueItem{
			TaskID: taskID,
			TaskInsertedAt: pgtype.Timestamptz{
				Time:  insertedAt,
				Valid: true,
			},
			ActionID:   actionID,
			RetryCount: 0,
		},
	}
}

type recordedMessage struct {
	queue msgqueue.Queue
	msg   *msgqueue.Message
}

type fakeMessageQueue struct {
	mu       sync.Mutex
	messages []recordedMessage
	isReady  bool
	tenants  map[string]struct{}
}

func newFakeMessageQueue() *fakeMessageQueue {
	return &fakeMessageQueue{
		isReady: true,
		tenants: make(map[string]struct{}),
	}
}

func (f *fakeMessageQueue) Clone() (func() error, msgqueue.MessageQueue) {
	return func() error { return nil }, f
}

func (f *fakeMessageQueue) SetQOS(_ int) {}

func (f *fakeMessageQueue) SendMessage(_ context.Context, queue msgqueue.Queue, msg *msgqueue.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.messages = append(f.messages, recordedMessage{
		queue: queue,
		msg:   msg,
	})

	return nil
}

func (f *fakeMessageQueue) Subscribe(msgqueue.Queue, msgqueue.AckHook, msgqueue.AckHook) (func() error, error) {
	return func() error { return nil }, nil
}

func (f *fakeMessageQueue) RegisterTenant(_ context.Context, tenantID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tenants[tenantID] = struct{}{}
	return nil
}

func (f *fakeMessageQueue) IsReady() bool {
	return f.isReady
}

func (f *fakeMessageQueue) MessagesForQueue(queue msgqueue.Queue) []*msgqueue.Message {
	f.mu.Lock()
	defer f.mu.Unlock()

	result := make([]*msgqueue.Message, 0)
	for _, entry := range f.messages {
		if entry.queue.Name() == queue.Name() {
			result = append(result, entry.msg)
		}
	}

	return result
}

type fakeRepository struct {
	tasks repov1.TaskRepository
}

func (f *fakeRepository) Triggers() repov1.TriggerRepository {
	return nil
}

func (f *fakeRepository) Tasks() repov1.TaskRepository {
	return f.tasks
}

func (f *fakeRepository) Scheduler() repov1.SchedulerRepository {
	return nil
}

func (f *fakeRepository) Matches() repov1.MatchRepository {
	return nil
}

func (f *fakeRepository) OLAP() repov1.OLAPRepository {
	return nil
}

func (f *fakeRepository) OverwriteOLAPRepository(repov1.OLAPRepository) {}

func (f *fakeRepository) Logs() repov1.LogLineRepository {
	return nil
}

func (f *fakeRepository) OverwriteLogsRepository(repov1.LogLineRepository) {}

func (f *fakeRepository) Payloads() repov1.PayloadStoreRepository {
	return nil
}

func (f *fakeRepository) OverwriteExternalPayloadStore(repov1.ExternalStore, time.Duration) {}

func (f *fakeRepository) Workers() repov1.WorkerRepository {
	return nil
}

func (f *fakeRepository) Workflows() repov1.WorkflowRepository {
	return nil
}

func (f *fakeRepository) Ticker() repov1.TickerRepository {
	return nil
}

func (f *fakeRepository) Filters() repov1.FilterRepository {
	return nil
}

func (f *fakeRepository) Webhooks() repov1.WebhookRepository {
	return nil
}

func (f *fakeRepository) Idempotency() repov1.IdempotencyRepository {
	return nil
}

func (f *fakeRepository) IntervalSettings() repov1.IntervalSettingsRepository {
	return nil
}

type fakeTaskRepository struct {
	mu          sync.Mutex
	updateCalls []taskBatchUpdateCall
	completed   []string
}

type taskBatchUpdateCall struct {
	tenantID    string
	batchID     string
	workerID    string
	batchKey    string
	batchSize   int
	assignments []repov1.TaskBatchAssignment
}

func (f *fakeTaskRepository) EnsureTablePartitionsExist(context.Context) (bool, error) {
	return false, nil
}

func (f *fakeTaskRepository) UpdateTablePartitions(context.Context) error {
	return nil
}

func (f *fakeTaskRepository) GetTaskByExternalId(context.Context, string, string, bool) (*sqlcv1.FlattenExternalIdsRow, error) {
	return nil, nil
}

func (f *fakeTaskRepository) FlattenExternalIds(context.Context, string, []string) ([]*sqlcv1.FlattenExternalIdsRow, error) {
	return nil, nil
}

func (f *fakeTaskRepository) CompleteTasks(context.Context, string, []repov1.CompleteTaskOpts) (*repov1.FinalizedTaskResponse, error) {
	return nil, nil
}

func (f *fakeTaskRepository) FailTasks(context.Context, string, []repov1.FailTaskOpts) (*repov1.FailTasksResponse, error) {
	return nil, nil
}

func (f *fakeTaskRepository) CancelTasks(context.Context, string, []repov1.TaskIdInsertedAtRetryCount) (*repov1.FinalizedTaskResponse, error) {
	return nil, nil
}

func (f *fakeTaskRepository) ListTasks(context.Context, string, []int64) ([]*repov1.TaskWithRuntime, error) {
	return nil, nil
}

func (f *fakeTaskRepository) ListTaskMetas(context.Context, string, []int64) ([]*sqlcv1.ListTaskMetasRow, error) {
	return nil, nil
}

func (f *fakeTaskRepository) ListFinalizedWorkflowRuns(context.Context, string, []string) ([]*repov1.ListFinalizedWorkflowRunsResponse, error) {
	return nil, nil
}

func (f *fakeTaskRepository) ListTaskParentOutputs(context.Context, string, []*sqlcv1.V1Task) (map[int64][]*repov1.TaskOutputEvent, error) {
	return nil, nil
}

func (f *fakeTaskRepository) DefaultTaskActivityGauge(context.Context, string) (int, error) {
	return 0, nil
}

func (f *fakeTaskRepository) ProcessTaskTimeouts(context.Context, string) (*repov1.TimeoutTasksResponse, bool, error) {
	return nil, false, nil
}

func (f *fakeTaskRepository) ProcessTaskReassignments(context.Context, string) (*repov1.FailTasksResponse, bool, error) {
	return nil, false, nil
}

func (f *fakeTaskRepository) ProcessTaskRetryQueueItems(context.Context, string) ([]*sqlcv1.V1RetryQueueItem, bool, error) {
	return nil, false, nil
}

func (f *fakeTaskRepository) ProcessDurableSleeps(context.Context, string) (*repov1.EventMatchResults, bool, error) {
	return nil, false, nil
}

func (f *fakeTaskRepository) GetQueueCounts(context.Context, string) (map[string]interface{}, error) {
	return nil, nil
}

func (f *fakeTaskRepository) ReplayTasks(context.Context, string, []repov1.TaskIdInsertedAtRetryCount) (*repov1.ReplayTasksResult, error) {
	return nil, nil
}

func (f *fakeTaskRepository) RefreshTimeoutBy(context.Context, string, repov1.RefreshTimeoutBy) (*sqlcv1.V1TaskRuntime, error) {
	return nil, nil
}

func (f *fakeTaskRepository) ReleaseSlot(context.Context, string, string) (*sqlcv1.V1TaskRuntime, error) {
	return nil, nil
}

func (f *fakeTaskRepository) ListSignalCompletedEvents(context.Context, string, []repov1.TaskIdInsertedAtSignalKey) ([]*repov1.V1TaskEventWithPayload, error) {
	return nil, nil
}

func (f *fakeTaskRepository) CountActiveTaskBatchRuns(context.Context, string, string, string) (int, error) {
	return 0, nil
}

func (f *fakeTaskRepository) ReserveTaskBatchRun(context.Context, string, string, string, string, string, int) (bool, error) {
	return true, nil
}

func (f *fakeTaskRepository) CompleteTaskBatchRun(_ context.Context, _ string, batchId string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.completed = append(f.completed, batchId)
	return nil
}

func (f *fakeTaskRepository) AnalyzeTaskTables(context.Context) error {
	return nil
}

func (f *fakeTaskRepository) Cleanup(context.Context) (bool, error) {
	return false, nil
}

func (f *fakeTaskRepository) GetTaskStats(context.Context, string) (map[string]repov1.TaskStat, error) {
	return nil, nil
}

func (f *fakeTaskRepository) UpdateTaskBatchMetadata(_ context.Context, tenantID, batchID, workerID, batchKey string, batchSize int, assignments []repov1.TaskBatchAssignment) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	copied := make([]repov1.TaskBatchAssignment, len(assignments))
	copy(copied, assignments)

	f.updateCalls = append(f.updateCalls, taskBatchUpdateCall{
		tenantID:    tenantID,
		batchID:     batchID,
		workerID:    workerID,
		batchKey:    batchKey,
		batchSize:   batchSize,
		assignments: copied,
	})

	return nil
}
