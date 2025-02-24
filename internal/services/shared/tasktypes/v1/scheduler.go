package v1

import (
	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
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
		"check-tenant-queue",
		true,
		false,
		payload,
	)
}

func NotifyTaskCreated(tenantId string, tasks []*sqlcv1.V1Task) (*msgqueue.Message, error) {
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
		"check-tenant-queue",
		true,
		false,
		payload,
	)
}

type TaskAssignedBulkTaskPayload struct {
	WorkerIdToTaskIds map[string][]int64 `json:"worker_id_to_task_id" validate:"required"`
}
