package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type TenantLimitConfig struct {
	EnforceLimits bool
}

type TenantLimitRepository interface {
	GetLimits(tenantId string) ([]*dbsqlc.TenantResourceLimit, error)

	// CanCreateWorkflowRun checks if the tenant can create a new workflow run
	CanCreateWorkflowRun(tenantId string) (bool, error)

	// MeterWorkflowRun increments the tenant's workflow run count
	MeterWorkflowRun(tenantId string) error

	// CanCreateEvent checks if the tenant can create a new event
	CanCreateEvent(tenantId string) (bool, error)

	// MeterEvent increments the tenant's event count
	MeterEvent(tenantId string) error

	// CanCreateWorker checks if the tenant can create a new worker
	CanCreateWorker(tenantId string) (bool, error)

	// Create new Tenant Resource Limits for a tenant
	CreateTenantDefaultLimits(tenantId string) error

	// Resolve the tenant's resource limits
	ResolveAllTenantResourceLimits(ctx context.Context) error
}
