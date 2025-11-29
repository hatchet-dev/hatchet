package servertel

import (
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"

	"go.opentelemetry.io/otel/trace"
)

func WithStepRunModel(span trace.Span, stepRun *dbsqlc.GetStepRunForEngineRow) {
	telemetry.WithAttributes(
		span,
		TenantId(stepRun.SRTenantId),
		StepRunId(stepRun.SRID),
		Step(stepRun.StepId),
		JobRunId(stepRun.JobRunId),
	)
}

func WithJobRunModel(span trace.Span, jobRun *dbsqlc.JobRun) {
	telemetry.WithAttributes(
		span,
		TenantId(jobRun.TenantId),
		JobRunId(jobRun.ID),
		Job(jobRun.JobId),
	)
}

func WithWorkflowRunModel(span trace.Span, workflowRun *dbsqlc.GetWorkflowRunRow) {
	telemetry.WithAttributes(
		span,
		TenantId(workflowRun.WorkflowRun.TenantId),
		WorkflowRunId(workflowRun.WorkflowRun.ID),
		WorkflowVersion(workflowRun.WorkflowVersion.ID),
	)
}

func TenantId(tenantId uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "tenant.id",
		Value: sqlchelpers.UUIDToStr(tenantId),
	}
}

func StepRunId(stepRunId uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "step_run.id",
		Value: sqlchelpers.UUIDToStr(stepRunId),
	}
}

func Step(step uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "step.id",
		Value: sqlchelpers.UUIDToStr(step),
	}
}

func JobRunId(jobRunId uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "job_run.id",
		Value: sqlchelpers.UUIDToStr(jobRunId),
	}
}

func Job(job uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "job.id",
		Value: sqlchelpers.UUIDToStr(job),
	}
}

func WorkflowRunId(workflowRunId uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "workflow_run.id",
		Value: sqlchelpers.UUIDToStr(workflowRunId),
	}
}

func WorkflowVersion(workflowVersion uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "workflow_version.id",
		Value: sqlchelpers.UUIDToStr(workflowVersion),
	}
}

func WorkerId(workerId uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "worker.id",
		Value: sqlchelpers.UUIDToStr(workerId),
	}
}

func EventId(eventId uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "event.id",
		Value: sqlchelpers.UUIDToStr(eventId),
	}
}
