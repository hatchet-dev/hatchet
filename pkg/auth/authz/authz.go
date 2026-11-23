// Package authz is the single entry point for authorization decisions across REST, gRPC and in-handler checks.
package authz

import (
	"context"
)

type Authorizer interface {
	// Authorize checks the action within its scope, with no specific object.
	Authorize(ctx context.Context, req Request) (Decision, error)

	// AuthorizeResource checks the action against one object.
	AuthorizeResource(ctx context.Context, req Request, r Resource) (Decision, error)

	// AuthorizeResources returns one decision per key; duplicate keys are an error.
	AuthorizeResources(ctx context.Context, req Request, rs []Resource) (Decisions, error)
}

type Request struct {
	Principal Principal
	Action    Action
	Scope     Scope
}
