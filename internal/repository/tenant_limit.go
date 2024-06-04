package repository

type TenantLimitConfig struct {
	EnforceLimits bool
}

type TenantLimitRepository interface {
	// CanCreateWorkflowRun checks if the tenant can create a new workflow run
	CanCreateWorkflowRun(tenantId string) (bool, error)

	// MeterWorkflowRun increments the tenant's workflow run count
	MeterWorkflowRun(tenantId string) error

	// CanCreateEvent checks if the tenant can create a new event
	CanCreateEvent(tenantId string) (bool, error)

	// MeterEvent increments the tenant's event count
	MeterEvent(tenantId string) error

	// // CanCreateWorker checks if the tenant can create a new worker
	// CanCreateWorker(tenantId string) (bool, error)
}
