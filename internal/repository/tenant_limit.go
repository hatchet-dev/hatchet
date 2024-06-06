package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type TenantLimitConfig struct {
	EnforceLimits bool
}

type TenantLimitRepository interface {
	GetLimits(ctx context.Context, tenantId string) ([]*dbsqlc.TenantResourceLimit, error)

	// CanCreateWorkflowRun checks if the tenant can create a resource
	CanCreate(ctx context.Context, resource dbsqlc.LimitResource, tenantId string) (bool, int, error)

	// MeterWorkflowRun increments the tenant's resource count
	Meter(ctx context.Context, resource dbsqlc.LimitResource, tenantId string) (*dbsqlc.TenantResourceLimit, error)

	// Create new Tenant Resource Limits for a tenant
	CreateTenantDefaultLimits(ctx context.Context, tenantId string) error

	// Resolve all tenant resource limits
	ResolveAllTenantResourceLimits(ctx context.Context) error
}
