package repository

import (
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

type CreateTenantInviteOpts struct {
	// (required) the invitee email
	InviteeEmail string `validate:"required,email"`

	// (required) the inviter email
	InviterEmail string `validate:"required,email"`

	// (required) when the invite expires
	ExpiresAt time.Time `validate:"required,future"`

	// (required) the role of the invitee
	Role string `validate:"omitempty,oneof=OWNER ADMIN MEMBER"`

	// (optional) the maximum number pending of invites the inviter can have

	MaxPending int `validate:"omitempty"`
}

type UpdateTenantInviteOpts struct {
	Status *string `validate:"omitempty,oneof=ACCEPTED REJECTED"`

	// (optional) the role of the invitee
	Role *string `validate:"omitempty,oneof=OWNER ADMIN MEMBER"`
}

type ListTenantInvitesOpts struct {
	// (optional) the status of the invite
	Status *string `validate:"omitempty,oneof=PENDING ACCEPTED REJECTED"`

	// (optional) whether the invite has expired
	Expired *bool `validate:"omitempty"`
}

type TenantInviteRepository interface {
	// CreateTenantInvite creates a new tenant invite with the given options
	CreateTenantInvite(tenantId string, opts *CreateTenantInviteOpts) (*db.TenantInviteLinkModel, error)

	// GetTenantInvite returns the tenant invite with the given id
	GetTenantInvite(id string) (*db.TenantInviteLinkModel, error)

	// ListTenantInvitesByEmail returns the list of tenant invites for the given invitee email for invites
	// which are not expired
	ListTenantInvitesByEmail(email string) ([]db.TenantInviteLinkModel, error)

	// ListTenantInvitesByTenantId returns the list of tenant invites for the given tenant id
	ListTenantInvitesByTenantId(tenantId string, opts *ListTenantInvitesOpts) ([]db.TenantInviteLinkModel, error)

	// UpdateTenantInvite updates the tenant invite with the given id
	UpdateTenantInvite(id string, opts *UpdateTenantInviteOpts) (*db.TenantInviteLinkModel, error)

	// DeleteTenantInvite deletes the tenant invite with the given id
	DeleteTenantInvite(id string) error
}
