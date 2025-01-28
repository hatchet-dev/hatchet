package tasktypes

import "github.com/hatchet-dev/hatchet/internal/msgqueue"

type CheckTenantQueuePayload struct {
	IsStepQueued   bool   `json:"is_step_queued"`
	IsSlotReleased bool   `json:"is_slot_released"`
	QueueName      string `json:"queue_name"`
}

func CheckTenantQueueToTask(tenantId, queueName string, isStepQueued bool, isSlotReleased bool) (*msgqueue.Message, error) {
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"check-tenant-queue",
		CheckTenantQueuePayload{
			IsStepQueued:   isStepQueued,
			IsSlotReleased: isSlotReleased,
			QueueName:      queueName,
		},
		true,
		false,
	)
}

type TaskAssignedBulkTaskPayload struct {
	WorkerIdToTaskIds map[string][]int64 `json:"worker_id_to_task_id" validate:"required"`
}
