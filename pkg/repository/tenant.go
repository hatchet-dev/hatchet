package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
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
	CreateTenant(opts *CreateTenantOpts) (*dbsqlc.Tenant, error)

	// CreateTenant creates a new tenant.
	UpdateTenant(tenantId string, opts *UpdateTenantOpts) (*db.TenantModel, error)

	// GetTenantByID returns the tenant with the given id
	GetTenantByID(tenantId string) (*db.TenantModel, error)

	// GetTenantBySlug returns the tenant with the given slug
	GetTenantBySlug(slug string) (*db.TenantModel, error)

	// CreateTenantMember creates a new member in the tenant
	CreateTenantMember(tenantId string, opts *CreateTenantMemberOpts) (*db.TenantMemberModel, error)

	// GetTenantMemberByID returns the tenant member with the given id
	GetTenantMemberByID(memberId string) (*db.TenantMemberModel, error)

	// GetTenantMemberByUserID returns the tenant member with the given user id
	GetTenantMemberByUserID(tenantId string, userId string) (*db.TenantMemberModel, error)

	// GetTenantMemberByEmail returns the tenant member with the given email
	GetTenantMemberByEmail(tenantId string, email string) (*db.TenantMemberModel, error)

	// ListTenantMembers returns the list of tenant members for the given tenant
	ListTenantMembers(tenantId string) ([]db.TenantMemberModel, error)

	// UpdateTenantMember updates the tenant member with the given id
	UpdateTenantMember(memberId string, opts *UpdateTenantMemberOpts) (*db.TenantMemberModel, error)

	// DeleteTenantMember deletes the tenant member with the given id
	DeleteTenantMember(memberId string) (*db.TenantMemberModel, error)

	// GetQueueMetrics returns the queue metrics for the given tenant
	GetQueueMetrics(tenantId string, opts *GetQueueMetricsOpts) (*GetQueueMetricsResponse, error)
}

type TenantEngineRepository interface {
	// ListTenants lists all tenants in the instance
	ListTenants(ctx context.Context) ([]*dbsqlc.Tenant, error)

	// ListTenantsByPartition lists all tenants in the given partition
	ListTenantsByControllerPartition(ctx context.Context, controllerPartitionId string) ([]*dbsqlc.Tenant, error)

	ListTenantsByWorkerPartition(ctx context.Context, workerPartitionId string) ([]*dbsqlc.Tenant, error)

	// CreateEnginePartition creates a new partition for tenants within the engine
	CreateControllerPartition(ctx context.Context, id string) error

	DeleteControllerPartition(ctx context.Context, id string) error

	RebalanceAllControllerPartitions(ctx context.Context) error

	RebalanceInactiveControllerPartitions(ctx context.Context) error

	CreateTenantWorkerPartition(ctx context.Context, id string) error

	DeleteTenantWorkerPartition(ctx context.Context, id string) error

	RebalanceAllTenantWorkerPartitions(ctx context.Context) error

	RebalanceInactiveTenantWorkerPartitions(ctx context.Context) error

	// GetTenantByID returns the tenant with the given id
	GetTenantByID(ctx context.Context, tenantId string) (*dbsqlc.Tenant, error)
}
