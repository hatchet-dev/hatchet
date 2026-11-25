package authz

import (
	"context"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// AllActions grants every action of a type, including ones no spec knows about.
const AllActions = "*"

// Permissions makes the final decision for the requests it handles, so each implementation chooses what it checks.
type Permissions interface {
	Handles(req Request) bool
	Decide(ctx context.Context, req Request) (Decision, error)
}

type RolePermissions interface {
	HasPermission(role string, a Action) bool
}

func RoleBased(perms RolePermissions, resolver RoleResolver, handles func(Request) bool) Permissions {
	return roleBased{perms: perms, resolver: resolver, handles: handles}
}

type roleBased struct {
	perms    RolePermissions
	resolver RoleResolver
	handles  func(Request) bool
}

func (r roleBased) Handles(req Request) bool { return r.handles(req) }

func (r roleBased) Decide(ctx context.Context, req Request) (Decision, error) {
	role, ok, err := r.resolver.ResolveRole(ctx, req.Principal, req.Action)
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

func CapToRole(perms RolePermissions, role string, reason DenyReason, applies func(Principal) bool) Policy {
	return capPolicy{perms: perms, role: role, reason: reason, applies: applies}
}

type capPolicy struct {
	perms   RolePermissions
	role    string
	reason  DenyReason
	applies func(Principal) bool
}

func (c capPolicy) Evaluate(_ context.Context, req Request) (Decision, error) {
	if !c.applies(req.Principal) || c.perms.HasPermission(c.role, req.Action) {
		return Allow(), nil
	}

	return Deny(c.reason), nil
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

// grantGraph is the inheritance model every permission format shares: user roles and token capabilities alike.
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

func (g grantGraph) inherited(name string) []string {
	var out []string

	seen := map[string]bool{name: true}
	queue := append([]string{}, g[name].inherits...)

	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]

		if seen[next] {
			continue
		}

		seen[next] = true
		out = append(out, next)
		queue = append(queue, g[next].inherits...)
	}

	return out
}

func (g grantGraph) grantsAnything(name string) bool {
	return g.has(name, func(string) bool { return true })
}

func keysInFileOrder(node yaml.Node) []string {
	var keys []string

	for i := 0; i+1 < len(node.Content); i += 2 {
		keys = append(keys, node.Content[i].Value)
	}

	return keys
}

func (g grantGraph) defines(name string) bool {
	_, ok := g[name]
	return ok
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
