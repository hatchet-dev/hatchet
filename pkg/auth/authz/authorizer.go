package authz

import (
	"context"
	"errors"
	"fmt"
)

type Option func(*authorizer)

// WithPermissions registers one permission set per ActionType; a later set for the same type replaces the earlier one.
func WithPermissions(ps ...ActionPermissions) Option {
	return func(a *authorizer) {
		for _, p := range ps {
			a.perms[p.Type()] = p
		}
	}
}

func WithPolicies(ps ...Policy) Option {
	return func(a *authorizer) { a.policies = append(a.policies, ps...) }
}

type authorizer struct {
	perms    map[ActionType]ActionPermissions
	policies []Policy
}

func New(opts ...Option) (Authorizer, error) {
	a := &authorizer{
		perms: map[ActionType]ActionPermissions{},
	}

	for _, opt := range opts {
		opt(a)
	}

	if len(a.perms) == 0 {
		return nil, errors.New("authz: permissions are required")
	}

	return a, nil
}

func (a *authorizer) Authorize(ctx context.Context, req Request) (Decision, error) {
	if req.Principal == nil {
		return Decision{}, errors.New("authz: no principal on request")
	}

	if req.Action == nil {
		return Decision{}, errors.New("authz: no action on request")
	}

	perms, ok := a.perms[req.Action.Type()]
	if !ok {
		return Decision{}, fmt.Errorf("authz: no permissions loaded for %s actions", req.Action.Type())
	}

	for _, p := range a.policies {
		d, err := p.Evaluate(ctx, req)
		if err != nil || !d.Allowed {
			return d, err
		}
	}

	return perms.Decide(ctx, req)
}
