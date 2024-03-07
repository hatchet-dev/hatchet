package tasktypes

type ScheduleStepRunTimeoutTaskPayload struct {
	StepRunId string `json:"step_run_id" validate:"required,uuid"`
	JobRunId  string `json:"job_run_id" validate:"required,uuid"`
	TimeoutAt string `json:"timeout_at" validate:"required"`
}

type ScheduleStepRunTimeoutTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type CancelStepRunTimeoutTaskPayload struct {
	StepRunId string `json:"step_run_id" validate:"required,uuid"`
}

type CancelStepRunTimeoutTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type ScheduleGetGroupKeyRunTimeoutTaskPayload struct {
	GetGroupKeyRunId string `json:"get_group_key_run_id" validate:"required,uuid"`
	WorkflowRunId    string `json:"workflow_run_id" validate:"required,uuid"`
	TimeoutAt        string `json:"timeout_at" validate:"required"`
}

type ScheduleGetGroupKeyRunTimeoutTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type CancelGetGroupKeyRunTimeoutTaskPayload struct {
	GetGroupKeyRunId string `json:"get_group_key_run_id" validate:"required,uuid"`
}

type CancelGetGroupKeyRunTimeoutTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type RemoveTickerTaskPayload struct {
	TickerId string `json:"ticker_id" validate:"required,uuid"`
}

type RemoveTickerTaskMetadata struct{}

type ScheduleCronTaskPayload struct {
	CronParentId      string `json:"cron_parent_id" validate:"required,uuid"`
	Cron              string `json:"cron" validate:"required"`
	WorkflowVersionId string `json:"workflow_version_id" validate:"required,uuid"`
}

type ScheduleCronTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type CancelCronTaskPayload struct {
	CronParentId      string `json:"cron_parent_id" validate:"required,uuid"`
	Cron              string `json:"cron" validate:"required"`
	WorkflowVersionId string `json:"workflow_version_id" validate:"required,uuid"`
}

type CancelCronTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type ScheduleWorkflowTaskPayload struct {
	ScheduledWorkflowId string `json:"scheduled_workflow_id" validate:"required,uuid"`
	TriggerAt           string `json:"trigger_at" validate:"required"`
	WorkflowVersionId   string `json:"workflow_version_id" validate:"required,uuid"`
}

type ScheduleWorkflowTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type CancelWorkflowTaskPayload struct {
	ScheduledWorkflowId string `json:"scheduled_workflow_id" validate:"required,uuid"`
	TriggerAt           string `json:"trigger_at" validate:"required"`
	WorkflowVersionId   string `json:"workflow_version_id" validate:"required,uuid"`
}

type CancelWorkflowTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}
