package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type CreateTenantOpts struct {
	// (required) the tenant name
	Name string `validate:"required"`

	// (required) the tenant slug
	Slug string `validate:"required,hatchetName"`

	// (optional) the tenant ID
	ID *string `validate:"omitempty,uuid"`

	// (optional) the tenant data retention period
	DataRetentionPeriod *string `validate:"omitempty,duration"`
}

type UpdateTenantOpts struct {
	Name *string

	AnalyticsOptOut *bool `validate:"omitempty"`

	AlertMemberEmails *bool `validate:"omitempty"`

	Version *dbsqlc.NullTenantMajorEngineVersion `validate:"omitempty"`

	UIVersion *dbsqlc.NullTenantMajorUIVersion `validate:"omitempty"`
}

type CreateTenantMemberOpts struct {
	Role   string `validate:"required,oneof=OWNER ADMIN MEMBER"`
	UserId string `validate:"required,uuid"`
}

type UpdateTenantMemberOpts struct {
	Role *string `validate:"omitempty,oneof=OWNER ADMIN MEMBER"`
}

type GetQueueMetricsOpts struct {
	// (optional) a list of workflow ids to filter by
	WorkflowIds []string `validate:"omitempty,dive,uuid"`

	// (optional) exact metadata to filter by
	AdditionalMetadata map[string]interface{} `validate:"omitempty"`
}

type QueueMetric struct {
	// the total number of PENDING_ASSIGNMENT step runs in the queue
	PendingAssignment int `json:"pending_assignment"`

	// the total number of PENDING step runs in the queue
	Pending int `json:"pending"`

	// the total number of RUNNING step runs in the queue
	Running int `json:"running"`
}

type GetQueueMetricsResponse struct {
	Total QueueMetric `json:"total"`

	ByWorkflowId map[string]QueueMetric `json:"by_workflow"`
}

type TenantAPIRepository interface {
	// CreateTenant creates a new tenant.
	CreateTenant(ctx context.Context, opts *CreateTenantOpts) (*dbsqlc.Tenant, error)

	// UpdateTenant updates an existing tenant in the db.
	UpdateTenant(ctx context.Context, tenantId string, opts *UpdateTenantOpts) (*dbsqlc.Tenant, error)

	// GetTenantByID returns the tenant with the given id
	GetTenantByID(ctx context.Context, tenantId string) (*dbsqlc.Tenant, error)

	// GetTenantBySlug returns the tenant with the given slug
	GetTenantBySlug(ctx context.Context, slug string) (*dbsqlc.Tenant, error)

	// CreateTenantMember creates a new member in the tenant
	CreateTenantMember(ctx context.Context, tenantId string, opts *CreateTenantMemberOpts) (*dbsqlc.PopulateTenantMembersRow, error)

	// GetTenantMemberByID returns the tenant member with the given id
	GetTenantMemberByID(ctx context.Context, memberId string) (*dbsqlc.PopulateTenantMembersRow, error)

	// GetTenantMemberByUserID returns the tenant member with the given user id
	GetTenantMemberByUserID(ctx context.Context, tenantId string, userId string) (*dbsqlc.PopulateTenantMembersRow, error)

	// GetTenantMemberByEmail returns the tenant member with the given email
	GetTenantMemberByEmail(ctx context.Context, tenantId string, email string) (*dbsqlc.PopulateTenantMembersRow, error)

	// ListTenantMembers returns the list of tenant members for the given tenant
	ListTenantMembers(ctx context.Context, tenantId string) ([]*dbsqlc.PopulateTenantMembersRow, error)

	// UpdateTenantMember updates the tenant member with the given id
	UpdateTenantMember(ctx context.Context, memberId string, opts *UpdateTenantMemberOpts) (*dbsqlc.PopulateTenantMembersRow, error)

	// DeleteTenantMember deletes the tenant member with the given id
	DeleteTenantMember(ctx context.Context, memberId string) error

	// GetQueueMetrics returns the queue metrics for the given tenant
	GetQueueMetrics(ctx context.Context, tenantId string, opts *GetQueueMetricsOpts) (*GetQueueMetricsResponse, error)
}

type TenantEngineRepository interface {
	// ListTenants lists all tenants in the instance
	ListTenants(ctx context.Context) ([]*dbsqlc.Tenant, error)

	// Gets the tenant corresponding to the "internal" tenant if it's assigned to this controller.
	// Returns nil if the tenant is not assigned to this controller.
	GetInternalTenantForController(ctx context.Context, controllerPartitionId string) (*dbsqlc.Tenant, error)

	// ListTenantsByPartition lists all tenants in the given partition
	ListTenantsByControllerPartition(ctx context.Context, controllerPartitionId string, majorVersion dbsqlc.TenantMajorEngineVersion) ([]*dbsqlc.Tenant, error)

	ListTenantsByWorkerPartition(ctx context.Context, workerPartitionId string, majorVersion dbsqlc.TenantMajorEngineVersion) ([]*dbsqlc.Tenant, error)

	ListTenantsBySchedulerPartition(ctx context.Context, schedulerPartitionId string, majorVersion dbsqlc.TenantMajorEngineVersion) ([]*dbsqlc.Tenant, error)

	// CreateEnginePartition creates a new partition for tenants within the engine
	CreateControllerPartition(ctx context.Context) (string, error)

	// UpdateControllerPartitionHeartbeat updates the heartbeat for the given partition. If the partition no longer exists,
	// it creates a new partition and returns the new partition id. Otherwise, it returns the existing partition id.
	UpdateControllerPartitionHeartbeat(ctx context.Context, partitionId string) (string, error)

	DeleteControllerPartition(ctx context.Context, id string) error

	RebalanceAllControllerPartitions(ctx context.Context) error

	RebalanceInactiveControllerPartitions(ctx context.Context) error

	CreateSchedulerPartition(ctx context.Context) (string, error)

	UpdateSchedulerPartitionHeartbeat(ctx context.Context, partitionId string) (string, error)

	DeleteSchedulerPartition(ctx context.Context, id string) error

	RebalanceAllSchedulerPartitions(ctx context.Context) error

	RebalanceInactiveSchedulerPartitions(ctx context.Context) error

	CreateTenantWorkerPartition(ctx context.Context) (string, error)

	UpdateWorkerPartitionHeartbeat(ctx context.Context, partitionId string) (string, error)

	DeleteTenantWorkerPartition(ctx context.Context, id string) error

	RebalanceAllTenantWorkerPartitions(ctx context.Context) error

	RebalanceInactiveTenantWorkerPartitions(ctx context.Context) error

	// GetTenantByID returns the tenant with the given id
	GetTenantByID(ctx context.Context, tenantId string) (*dbsqlc.Tenant, error)
}
