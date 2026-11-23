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

func WithResolver(r ResourceResolver) Option {
	return func(a *authorizer) { a.resolvers[r.Kind()] = r }
}

type authorizer struct {
	perms     map[ActionType]ActionPermissions
	policies  []Policy
	resolvers map[Kind]ResourceResolver
}

func New(opts ...Option) (Authorizer, error) {
	a := &authorizer{
		perms:     map[ActionType]ActionPermissions{},
		resolvers: map[Kind]ResourceResolver{},
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

func (a *authorizer) AuthorizeResource(ctx context.Context, req Request, r Resource) (Decision, error) {
	ds, err := a.AuthorizeResources(ctx, req, []Resource{r})
	if err != nil {
		return Decision{}, err
	}

	return ds[r.Key()], nil
}

func (a *authorizer) AuthorizeResources(ctx context.Context, req Request, rs []Resource) (Decisions, error) {
	byKind := map[Kind][]Resource{}
	seen := make(map[Key]struct{}, len(rs))

	for _, r := range rs {
		k := r.Key()

		if _, dup := seen[k]; dup {
			return nil, fmt.Errorf("authz: duplicate resource %s", k)
		}

		seen[k] = struct{}{}
		byKind[k.Kind] = append(byKind[k.Kind], r)
	}

	actionDecision, err := a.Authorize(ctx, req)
	if err != nil {
		return nil, err
	}

	out := make(Decisions, len(rs))

	if !actionDecision.Allowed {
		for k := range seen {
			out[k] = actionDecision
		}

		return out, nil
	}

	for kind, group := range byKind {
		resolver, ok := a.resolvers[kind]
		if !ok {
			return nil, fmt.Errorf("authz: no resolver registered for kind %s", kind)
		}

		ds, err := resolver.Check(ctx, req, group)
		if err != nil {
			return nil, err
		}

		for _, r := range group {
			d, ok := ds[r.Key()]
			if !ok {
				return nil, fmt.Errorf("authz: resolver for %s returned no decision for %s", kind, r.Key())
			}

			out[r.Key()] = d
		}
	}

	return out, nil
}
