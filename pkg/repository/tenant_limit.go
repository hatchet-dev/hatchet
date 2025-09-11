package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type TenantLimitConfig struct {
	EnforceLimits bool
}

type Limit struct {
	Resource         dbsqlc.LimitResource
	Limit            int32
	Alarm            int32
	Window           *time.Duration
	CustomValueMeter bool
}

type PlanLimitMap map[string][]Limit

type TenantLimitRepository interface {
	GetLimits(ctx context.Context, tenantId string) ([]*dbsqlc.TenantResourceLimit, error)

	// CanCreateWorkflowRun checks if the tenant can create a resource
	CanCreate(ctx context.Context, resource dbsqlc.LimitResource, tenantId string, numberOfResources int32) (bool, int, error)

	// MeterWorkflowRun increments the tenant's resource count
	Meter(ctx context.Context, resource dbsqlc.LimitResource, tenantId string, numberOfResources int32) (*dbsqlc.TenantResourceLimit, error)

	// Create new Tenant Resource Limits for a tenant
	SelectOrInsertTenantLimits(ctx context.Context, tenantId string, plan *string) error

	// UpsertTenantLimits updates or inserts new tenant limits
	UpsertTenantLimits(ctx context.Context, tenantId string, plan *string) error

	// Resolve all tenant resource limits
	ResolveAllTenantResourceLimits(ctx context.Context) error

	// SetPlanLimitMap sets the plan limit map
	SetPlanLimitMap(planLimitMap PlanLimitMap) error

	DefaultLimits() []Limit
}
