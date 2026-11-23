package authz

import (
	"github.com/google/uuid"
)

type ActionType string

const (
	RESTEndpoint    ActionType = "rest"
	GRPCEndpoint    ActionType = "grpc"
	CapabilityCheck ActionType = "capability"
)

type Action interface {
	Type() ActionType
	Name() string // "V1TaskCancel", "/v1.Dispatcher/Listen", "tenant.payloads.read"
}

type RESTAction struct {
	OperationID string
}

type GRPCAction struct {
	Procedure string
}

type CapabilityAction struct {
	Capability string
}

func (RESTAction) Type() ActionType       { return RESTEndpoint }
func (GRPCAction) Type() ActionType       { return GRPCEndpoint }
func (CapabilityAction) Type() ActionType { return CapabilityCheck }

func (a RESTAction) Name() string       { return a.OperationID }
func (a GRPCAction) Name() string       { return a.Procedure }
func (a CapabilityAction) Name() string { return a.Capability }

func REST(operationID string) Action {
	return RESTAction{OperationID: operationID}
}

func GRPC(procedure string) Action {
	return GRPCAction{Procedure: procedure}
}

func Capability(name string) Action {
	return CapabilityAction{Capability: name}
}

type Scope struct {
	TenantID *uuid.UUID
	OrgID    *uuid.UUID
}

var NoScope = Scope{}

func TenantScope(id uuid.UUID) Scope {
	return Scope{TenantID: &id}
}

func OrgScope(id uuid.UUID) Scope {
	return Scope{OrgID: &id}
}
