package authz

import (
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/enums"
)

// Kind is an open set so cloud and control-plane can register their own resource kinds.
type Kind string

const (
	KindTenant       Kind = "tenant"
	KindTenantMember Kind = "tenant-member"
	KindOrgMember    Kind = "org-member"
	KindOperator     Kind = "operator"
	KindWorker       Kind = "worker"
	KindStreamTopic  Kind = "stream-topic"
)

// Key is unique within a Scope; an empty ID means the object doesn't exist yet.
type Key struct {
	Kind Kind
	ID   string
}

func (k Key) String() string {
	return string(k.Kind) + "/" + k.ID
}

type Resource interface {
	Key() Key
}

// OrganizationMemberRole mirrors the control-plane enum, which OSS hatchet has no table for.
type OrganizationMemberRole string

const (
	OrganizationMemberRoleOWNER  OrganizationMemberRole = "OWNER"
	OrganizationMemberRoleMEMBER OrganizationMemberRole = "MEMBER"
)

type TenantResource struct {
	TenantID *uuid.UUID
	Tags     []string // proposed tags when TenantID is nil
}

type TenantMemberResource struct {
	MemberID    *uuid.UUID // nil for invites
	GrantedRole *enums.TenantMemberRole
}

type OrgMemberResource struct {
	MemberID    *uuid.UUID // nil for invites
	GrantedRole *OrganizationMemberRole
}

type OperatorResource struct {
	OperatorID uuid.UUID
}

type WorkerResource struct {
	OperatorID uuid.UUID // verified against the loaded row, not part of the key
	WorkerID   uuid.UUID
}

type StreamTopicResource struct {
	Namespace string
	Topic     string
}

func (r TenantResource) Key() Key {
	return Key{Kind: KindTenant, ID: optionalID(r.TenantID)}
}

func (r TenantMemberResource) Key() Key {
	return Key{Kind: KindTenantMember, ID: optionalID(r.MemberID)}
}

func (r OrgMemberResource) Key() Key {
	return Key{Kind: KindOrgMember, ID: optionalID(r.MemberID)}
}

func (r OperatorResource) Key() Key {
	return Key{Kind: KindOperator, ID: r.OperatorID.String()}
}

func (r WorkerResource) Key() Key {
	return Key{Kind: KindWorker, ID: r.WorkerID.String()}
}

func (r StreamTopicResource) Key() Key {
	return Key{Kind: KindStreamTopic, ID: r.Namespace + "/" + r.Topic}
}

func optionalID(id *uuid.UUID) string {
	if id == nil {
		return ""
	}

	return id.String()
}
