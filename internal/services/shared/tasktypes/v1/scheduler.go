package v1

import (
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CheckTenantQueuesPayload struct {
	QueueNames    []string `json:"queue_name"`
	StrategyIds   []int64  `json:"strategy_ids"`
	SlotsReleased bool     `json:"slots_released"`
}

func NotifyTaskReleased(tenantId uuid.UUID, tasks []*sqlcv1.ReleaseTasksRow) (*msgqueue.Message, error) {
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

func NotifyTaskCreated(tenantId uuid.UUID, tasks []*v1.V1TaskWithPayload) (*msgqueue.Message, error) {
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
	WorkerIdToTaskIds map[uuid.UUID][]int64 `json:"worker_id_to_task_id" validate:"required"`
}
