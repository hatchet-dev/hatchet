// Package authz is the single entry point for authorization decisions across REST and gRPC.
package authz

import (
	"context"
)

type Authorizer interface {
	Authorize(ctx context.Context, req Request) (Decision, error)
}

type Request struct {
	Principal Principal
	Action    Action
}
