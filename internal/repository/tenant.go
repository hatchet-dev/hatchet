package repository

import "github.com/hatchet-dev/hatchet/internal/repository/prisma/db"

type CreateTenantOpts struct {
	// (required) the tenant name
	Name string `validate:"required"`

	// (required) the tenant slug
	Slug string `validate:"required,hatchetName"`

	// (optional) the tenant ID
	ID *string `validate:"omitempty,uuid"`
}

type CreateTenantMemberOpts struct {
	Role   string `validate:"required,oneof=OWNER ADMIN MEMBER"`
	UserId string `validate:"required,uuid"`
}

type UpdateTenantMemberOpts struct {
	Role *string `validate:"omitempty,oneof=OWNER ADMIN MEMBER"`
}

type TenantRepository interface {
	// CreateTenant creates a new tenant.
	CreateTenant(opts *CreateTenantOpts) (*db.TenantModel, error)

	// ListTenants lists all tenants in the instance
	ListTenants() ([]db.TenantModel, error)

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
}
