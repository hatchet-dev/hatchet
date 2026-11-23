package authz

import (
	"context"
)

// NoRole is resolved for a principal that is authenticated but needs no role for the action.
const NoRole = ""

// RoleResolver reads whatever it needs (such as a target tenant) from the concrete action type it handles.
type RoleResolver interface {
	// ResolveRole returns ok=false when the principal has no access to what the action targets.
	ResolveRole(ctx context.Context, p Principal, a Action) (role string, ok bool, err error)
}

// Policy runs before the action type decides and can only deny, so plug-ins can never widen access.
type Policy interface {
	Evaluate(ctx context.Context, req Request) (Decision, error)
}
