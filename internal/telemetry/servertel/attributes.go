package servertel

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"

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

func TenantId(tenantId pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "tenantId",
		Value: sqlchelpers.UUIDToStr(tenantId),
	}
}

func StepRunId(stepRunId pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "stepRunId",
		Value: sqlchelpers.UUIDToStr(stepRunId),
	}
}

func Step(step pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "stepId",
		Value: sqlchelpers.UUIDToStr(step),
	}
}

func JobRunId(jobRunId pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "jobRunId",
		Value: sqlchelpers.UUIDToStr(jobRunId),
	}
}

func Job(job pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "jobId",
		Value: sqlchelpers.UUIDToStr(job),
	}
}

func WorkflowRunId(workflowRunId pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "workflowRunId",
		Value: sqlchelpers.UUIDToStr(workflowRunId),
	}
}

func WorkflowVersion(workflowVersion pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "workflowVersionId",
		Value: sqlchelpers.UUIDToStr(workflowVersion),
	}
}

func WorkerId(workerId pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "workerId",
		Value: sqlchelpers.UUIDToStr(workerId),
	}
}

func EventId(eventId pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "eventId",
		Value: sqlchelpers.UUIDToStr(eventId),
	}
}
