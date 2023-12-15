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

type ScheduleJobRunTimeoutTaskPayload struct {
	JobRunId  string `json:"job_run_id" validate:"required,uuid"`
	TimeoutAt string `json:"timeout_at" validate:"required"`
}

type ScheduleJobRunTimeoutTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

type CancelJobRunTimeoutTaskPayload struct {
	JobRunId string `json:"job_run_id" validate:"required,uuid"`
}

type CancelJobRunTimeoutTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}
