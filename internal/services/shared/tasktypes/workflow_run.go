package tasktypes

type WorkflowRunFailedTask struct {
	WorkflowRunId string `json:"workflow_run_id" validate:"required,uuid"`
	FailedAt      string `json:"failed_at" validate:"required"`
	Reason        string `json:"reason" validate:"required"`
}

type WorkflowRunFailedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}
