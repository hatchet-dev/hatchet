package authz

import (
	"github.com/google/uuid"
)

type ActionType string

const (
	RESTEndpoint ActionType = "rest"
	GRPCEndpoint ActionType = "grpc"
)

type Action interface {
	Type() ActionType
	Name() string // "V1TaskCancel", "/v1.Dispatcher/Listen"
}

type RESTAction struct {
	OperationID string
}

type GRPCAction struct {
	Procedure string
}

func (RESTAction) Type() ActionType { return RESTEndpoint }
func (GRPCAction) Type() ActionType { return GRPCEndpoint }

func (a RESTAction) Name() string { return a.OperationID }
func (a GRPCAction) Name() string { return a.Procedure }

func REST(operationID string) Action {
	return RESTAction{OperationID: operationID}
}

func GRPC(procedure string) Action {
	return GRPCAction{Procedure: procedure}
}

type Scope struct {
	TenantID *uuid.UUID
}

var NoScope = Scope{}

func TenantScope(id uuid.UUID) Scope {
	return Scope{TenantID: &id}
}
