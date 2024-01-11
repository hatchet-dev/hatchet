package servertel

import (
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/telemetry"

	"go.opentelemetry.io/otel/trace"
)

func WithStepRunModel(span trace.Span, stepRun *db.StepRunModel) {
	telemetry.WithAttributes(
		span,
		TenantId(stepRun.TenantID),
		StepRunId(stepRun.ID),
		Step(stepRun.StepID),
		JobRunId(stepRun.JobRunID),
	)
}

func WithJobRunModel(span trace.Span, jobRun *db.JobRunModel) {
	telemetry.WithAttributes(
		span,
		TenantId(jobRun.TenantID),
		JobRunId(jobRun.ID),
		Job(jobRun.JobID),
	)
}

func WithWorkflowRunModel(span trace.Span, workflowRun *db.WorkflowRunModel) {
	telemetry.WithAttributes(
		span,
		TenantId(workflowRun.TenantID),
		WorkflowRunId(workflowRun.ID),
		WorkflowVersion(workflowRun.WorkflowVersionID),
	)
}

func TenantId(tenantId string) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "tenantId",
		Value: tenantId,
	}
}

func StepRunId(stepRunId string) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "stepRunId",
		Value: stepRunId,
	}
}

func Step(step string) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "stepId",
		Value: step,
	}
}

func JobRunId(jobRunId string) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "jobRunId",
		Value: jobRunId,
	}
}

func Job(job string) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "jobId",
		Value: job,
	}
}

func WorkflowRunId(workflowRunId string) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "workflowRunId",
		Value: workflowRunId,
	}
}

func WorkflowVersion(workflowVersion string) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "workflowVersionId",
		Value: workflowVersion,
	}
}

func WorkerId(workerId string) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "workerId",
		Value: workerId,
	}
}

func EventId(eventId string) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "eventId",
		Value: eventId,
	}
}
