package authz

import (
	"context"
)

// NoRole is resolved for a principal that is authenticated but outside any scope that grants roles.
const NoRole = ""

type RoleResolver interface {
	// ResolveRole returns ok=false when the principal has no access to the scope at all.
	ResolveRole(ctx context.Context, p Principal, scope Scope) (role string, ok bool, err error)
}

// Policy runs before the action type decides and can only deny, so plug-ins can never widen access.
type Policy interface {
	Evaluate(ctx context.Context, req Request) (Decision, error)
}
