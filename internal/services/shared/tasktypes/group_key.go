package tasktypes

// import (
// 	"github.com/hatchet-dev/hatchet/internal/datautils"
// 	"github.com/hatchet-dev/hatchet/internal/msgqueue"
// 	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
// )

// type GroupKeyActionAssignedTaskPayload struct {
// 	WorkflowRunId string `json:"workflow_run_id" validate:"required,uuid"`
// 	WorkerId      string `json:"worker_id" validate:"required,uuid"`
// }

// type GroupKeyActionAssignedTaskMetadata struct {
// 	TenantId     string `json:"tenant_id" validate:"required,uuid"`
// 	DispatcherId string `json:"dispatcher_id" validate:"required,uuid"`
// }

// type GroupKeyActionRequeueTaskPayload struct {
// 	TenantId string `json:"tenant_id" validate:"required,uuid"`
// }

// type GroupKeyActionRequeueTaskMetadata struct {
// 	TenantId string `json:"tenant_id" validate:"required,uuid"`
// }

// func TenantToGroupKeyActionRequeueTask(tenant db.TenantModel) *msgqueue.Message {
// 	payload, _ := datautils.ToJSONMap(GroupKeyActionRequeueTaskPayload{
// 		TenantId: tenant.ID,
// 	})

// 	metadata, _ := datautils.ToJSONMap(GroupKeyActionRequeueTaskMetadata{
// 		TenantId: tenant.ID,
// 	})

// 	return &msgqueue.Message{
// 		ID:       "group-key-action-requeue-ticker",
// 		Payload:  payload,
// 		Metadata: metadata,
// 		Retries:  3,
// 	}
// }

// type GetGroupKeyRunStartedTaskPayload struct {
// 	GetGroupKeyRunId string `json:"get_group_key_run_id" validate:"required,uuid"`
// 	StartedAt        string `json:"started_at" validate:"required"`
// }

// type GetGroupKeyRunStartedTaskMetadata struct {
// 	TenantId string `json:"tenant_id" validate:"required,uuid"`
// }

// type GetGroupKeyRunFinishedTaskPayload struct {
// 	GetGroupKeyRunId string `json:"get_group_key_run_id" validate:"required,uuid"`
// 	FinishedAt       string `json:"finished_at" validate:"required"`
// 	GroupKey         string `json:"group_key"`
// }

// type GetGroupKeyRunFinishedTaskMetadata struct {
// 	TenantId string `json:"tenant_id" validate:"required,uuid"`
// }

// type GetGroupKeyRunFailedTaskPayload struct {
// 	GetGroupKeyRunId string `json:"get_group_key_run_id" validate:"required,uuid"`
// 	FailedAt         string `json:"failed_at" validate:"required"`
// 	Error            string `json:"error" validate:"required"`
// }

// type GetGroupKeyRunFailedTaskMetadata struct {
// 	TenantId string `json:"tenant_id" validate:"required,uuid"`
// }

// type GetGroupKeyRunTimedOutTaskPayload struct {
// 	GetGroupKeyRunId string `json:"get_group_key_run_id" validate:"required,uuid"`
// }

// type GetGroupKeyRunTimedOutTaskMetadata struct {
// 	TenantId string `json:"tenant_id" validate:"required,uuid"`
// }
