package authz

import (
	"context"
	"errors"
	"fmt"
)

type Option func(*authorizer)

// WithPermissions registers permission sets in order; the first that handles a request decides it.
func WithPermissions(ps ...Permissions) Option {
	return func(a *authorizer) { a.perms = append(a.perms, ps...) }
}

func WithPolicies(ps ...Policy) Option {
	return func(a *authorizer) { a.policies = append(a.policies, ps...) }
}

type authorizer struct {
	perms    []Permissions
	policies []Policy
}

func New(opts ...Option) (Authorizer, error) {
	a := &authorizer{}

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

	perms := a.handler(req)
	if perms == nil {
		return Decision{}, fmt.Errorf("authz: no permissions handle %T calling %s %s", req.Principal, req.Action.Type(), req.Action.Name())
	}

	for _, p := range a.policies {
		d, err := p.Evaluate(ctx, req)
		if err != nil || !d.Allowed {
			return d, err
		}
	}

	return perms.Decide(ctx, req)
}

func (a *authorizer) handler(req Request) Permissions {
	for _, p := range a.perms {
		if p.Handles(req) {
			return p
		}
	}

	return nil
}
