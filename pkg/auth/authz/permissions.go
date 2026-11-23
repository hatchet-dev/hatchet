package authz

import (
	"context"
	"fmt"
	"strings"
)

// AllActions grants every action of a type, including ones no spec knows about.
const AllActions = "*"

// ActionPermissions makes the final decision for one ActionType, so each type chooses what it checks (roles, token capabilities).
type ActionPermissions interface {
	Type() ActionType
	Decide(ctx context.Context, req Request) (Decision, error)
}

// RolePermissions is implemented by the role-based formats (REST, capability checks); gRPC has no roles.
type RolePermissions interface {
	Type() ActionType
	HasPermission(role string, a Action) bool
}

// RoleBased decides by resolving the principal's role in the request scope and looking it up in perms.
func RoleBased(perms RolePermissions, resolver RoleResolver) ActionPermissions {
	return roleBased{perms: perms, resolver: resolver}
}

type roleBased struct {
	perms    RolePermissions
	resolver RoleResolver
}

func (r roleBased) Type() ActionType { return r.perms.Type() }

func (r roleBased) Decide(ctx context.Context, req Request) (Decision, error) {
	role, ok, err := r.resolver.ResolveRole(ctx, req.Principal, req.Scope)
	if err != nil {
		return Decision{}, err
	}

	if !ok {
		return Deny(NotMember), nil
	}

	if role == NoRole {
		return Allow(), nil
	}

	if !r.perms.HasPermission(role, req.Action) {
		return Deny(RoleInsufficient), nil
	}

	return Allow(), nil
}

func OperationIn(operationId string, operationIds []string) bool {
	for _, id := range operationIds {
		if strings.EqualFold(operationId, id) {
			return true
		}
	}

	return false
}

type grantDef struct {
	inherits []string
	grants   []string
}

// grantGraph is the inheritance model every permission format shares: REST roles and gRPC capabilities alike.
type grantGraph map[string]*grantDef

func (g grantGraph) validate() error {
	for name, role := range g {
		for _, inherited := range role.inherits {
			if _, ok := g[inherited]; !ok {
				return fmt.Errorf("%s inherits from %s which does not exist", name, inherited)
			}
		}
	}

	for name := range g {
		if err := g.checkCycle(name, map[string]bool{}); err != nil {
			return err
		}
	}

	return nil
}

func (g grantGraph) checkCycle(name string, path map[string]bool) error {
	if path[name] {
		return fmt.Errorf("%s inherits from itself", name)
	}

	path[name] = true
	defer delete(path, name)

	for _, inherited := range g[name].inherits {
		if err := g.checkCycle(inherited, path); err != nil {
			return err
		}
	}

	return nil
}

// has is false for unknown names so a bad role or capability denies instead of panicking.
func (g grantGraph) has(role string, matches func(grant string) bool) bool {
	def, ok := g[role]
	if !ok {
		return false
	}

	for _, grant := range def.grants {
		if grant == AllActions || matches(grant) {
			return true
		}
	}

	for _, inherited := range def.inherits {
		if g.has(inherited, matches) {
			return true
		}
	}

	return false
}

func (g grantGraph) grant(role string, names ...string) error {
	def, ok := g[role]
	if !ok {
		return fmt.Errorf("cannot grant to %s: it does not exist", role)
	}

	def.grants = append(def.grants, names...)

	return nil
}

func (g grantGraph) grantedNames() map[string]struct{} {
	out := map[string]struct{}{}

	for _, def := range g {
		for _, grant := range def.grants {
			if grant != AllActions {
				out[grant] = struct{}{}
			}
		}
	}

	return out
}

// checkCoverage requires the file and the source of truth to list exactly the same names.
func checkCoverage(kind string, known, granted map[string]struct{}) error {
	for name := range known {
		if _, ok := granted[name]; !ok {
			return fmt.Errorf("%s %s exists in the spec but not in the permission file", kind, name)
		}
	}

	for name := range granted {
		if _, ok := known[name]; !ok {
			return fmt.Errorf("%s %s exists in the permission file but not in the spec", kind, name)
		}
	}

	return nil
}
