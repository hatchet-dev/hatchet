package servertel

import (
	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func TenantId(tenantId uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "tenant.id",
		Value: tenantId.String(),
	}
}

func Step(step uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "step.id",
		Value: step.String(),
	}
}

func Job(job uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "job.id",
		Value: job.String(),
	}
}

func WorkflowVersion(workflowVersion uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "workflow_version.id",
		Value: workflowVersion.String(),
	}
}

func WorkerId(workerId uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "worker.id",
		Value: workerId.String(),
	}
}

func EventId(eventId uuid.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "event.id",
		Value: eventId.String(),
	}
}
