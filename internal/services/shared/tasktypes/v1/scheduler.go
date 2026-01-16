package v1

import (
	"time"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CheckTenantQueuesPayload struct {
	SlotsReleased bool     `json:"slots_released"`
	QueueNames    []string `json:"queue_name"`
	StrategyIds   []int64  `json:"strategy_ids"`
}

func NotifyTaskReleased(tenantId string, tasks []*sqlcv1.ReleaseTasksRow) (*msgqueue.Message, error) {
	uniqueQueueNames := make(map[string]struct{})
	uniqueStrategies := make(map[int64]struct{})

	for _, task := range tasks {
		uniqueQueueNames[task.Queue] = struct{}{}

		for _, strategyId := range task.ConcurrencyStrategyIds {
			uniqueStrategies[strategyId] = struct{}{}
		}
	}

	payload := CheckTenantQueuesPayload{
		QueueNames:    make([]string, 0, len(uniqueQueueNames)),
		StrategyIds:   make([]int64, 0, len(uniqueStrategies)),
		SlotsReleased: true,
	}

	for queueName := range uniqueQueueNames {
		payload.QueueNames = append(payload.QueueNames, queueName)
	}

	for strategyId := range uniqueStrategies {
		payload.StrategyIds = append(payload.StrategyIds, strategyId)
	}

	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCheckTenantQueue,
		true,
		false,
		payload,
	)
}

func NotifyTaskCreated(tenantId string, tasks []*v1.V1TaskWithPayload) (*msgqueue.Message, error) {
	uniqueQueueNames := make(map[string]struct{})
	uniqueStrategies := make(map[int64]struct{})

	for _, task := range tasks {
		uniqueQueueNames[task.Queue] = struct{}{}

		for _, strategyId := range task.ConcurrencyStrategyIds {
			uniqueStrategies[strategyId] = struct{}{}
		}
	}

	payload := CheckTenantQueuesPayload{
		QueueNames:  make([]string, 0, len(uniqueQueueNames)),
		StrategyIds: make([]int64, 0, len(uniqueStrategies)),
	}

	for queueName := range uniqueQueueNames {
		payload.QueueNames = append(payload.QueueNames, queueName)
	}

	for strategyId := range uniqueStrategies {
		payload.StrategyIds = append(payload.StrategyIds, strategyId)
	}

	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCheckTenantQueue,
		true,
		false,
		payload,
	)
}

type TaskAssignedBulkTaskPayload struct {
	WorkerBatches map[string][]TaskAssignedBatch `json:"worker_batches" validate:"required"`
}

type TaskAssignedBatch struct {
	BatchID    string                 `json:"batch_id"`
	BatchSize  int                    `json:"batch_size"`
	TaskIds    []int64                `json:"task_ids"`
	StartBatch *StartBatchTaskPayload `json:"start_batch,omitempty"`
}

type StartBatchTaskPayload struct {
	TenantId      string    `json:"tenant_id" validate:"required"`
	WorkerId      string    `json:"worker_id" validate:"required"`
	ActionId      string    `json:"action_id" validate:"required"`
	BatchId       string    `json:"batch_id" validate:"required"`
	ExpectedSize  int       `json:"expected_size" validate:"required"`
	BatchKey      string    `json:"batch_key,omitempty"`
	MaxRuns       *int      `json:"max_runs,omitempty"`
	TriggerReason string    `json:"trigger_reason,omitempty"`
	TriggerTime   time.Time `json:"trigger_time" validate:"required"`
}

func StartBatchMessage(tenantId string, payload StartBatchTaskPayload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"batch-start",
		false,
		true,
		payload,
	)
}
