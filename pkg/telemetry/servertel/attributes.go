package servertel

import (
	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func TenantId(tenantId uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "tenant.id",
		Value: sqlchelpers.UUIDToStr(tenantId),
	}
}

func Step(step uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "step.id",
		Value: sqlchelpers.UUIDToStr(step),
	}
}

func Job(job uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "job.id",
		Value: sqlchelpers.UUIDToStr(job),
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
