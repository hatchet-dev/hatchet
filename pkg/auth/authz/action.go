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
	TenantID    *uuid.UUID // the tenant the route acts on; nil for routes outside any tenant
}

type GRPCAction struct {
	Procedure string
}

func (RESTAction) Type() ActionType { return RESTEndpoint }
func (GRPCAction) Type() ActionType { return GRPCEndpoint }

func (a RESTAction) Name() string { return a.OperationID }
func (a GRPCAction) Name() string { return a.Procedure }

func REST(operationID string, tenantID *uuid.UUID) Action {
	return RESTAction{OperationID: operationID, TenantID: tenantID}
}

func GRPC(procedure string) Action {
	return GRPCAction{Procedure: procedure}
}
