package servertel

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func TenantId(tenantId pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "tenant.id",
		Value: sqlchelpers.UUIDToStr(tenantId),
	}
}

func Step(step pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "step.id",
		Value: sqlchelpers.UUIDToStr(step),
	}
}

func Job(job pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "job.id",
		Value: sqlchelpers.UUIDToStr(job),
	}
}

func WorkflowVersion(workflowVersion pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "workflow_version.id",
		Value: sqlchelpers.UUIDToStr(workflowVersion),
	}
}

func WorkerId(workerId pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "worker.id",
		Value: sqlchelpers.UUIDToStr(workerId),
	}
}

func EventId(eventId pgtype.UUID) telemetry.AttributeKV {
	return telemetry.AttributeKV{
		Key:   "event.id",
		Value: sqlchelpers.UUIDToStr(eventId),
	}
}
