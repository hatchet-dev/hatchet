package authz

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"go.yaml.in/yaml/v3"

	"github.com/hatchet-dev/hatchet/internal/enumerables"
)

// restFile is the REST role format: roles grant OpenAPI operation ids.
//
//	roles:
//	  ADMIN:
//	    inherits: [MEMBER]
//	    permissions: [TenantMemberUpdate]
type restFile struct {
	Roles map[string]struct {
		Inherits    []string `yaml:"inherits"`
		Permissions []string `yaml:"permissions"`
		Assignable  bool     `yaml:"assignable"`
		Label       string   `yaml:"label"`
		Description string   `yaml:"description"`
	} `yaml:"roles"`
}

type AssignableRole struct {
	Name        string
	Label       string
	Description string
	Operations  []string
}

type RESTPermissions struct {
	roles      grantGraph
	assignable []AssignableRole
	operations map[string]struct{}
}

func LoadRESTPermissions(yamlBytes []byte, spec *openapi3.T) (*RESTPermissions, error) {
	if spec == nil {
		return nil, errors.New("authz: an OpenAPI spec is required to load REST permissions")
	}

	var f restFile
	if err := yaml.Unmarshal(yamlBytes, &f); err != nil {
		return nil, err
	}

	roles := grantGraph{}
	for name, r := range f.Roles {
		roles[name] = &grantDef{inherits: r.Inherits, grants: r.Permissions}
	}

	if err := roles.validate(); err != nil {
		return nil, err
	}

	operations := specOperations(spec, nil)
	if err := checkCoverage("operation", operations, roles.grantedNames()); err != nil {
		return nil, err
	}

	var order struct {
		Roles yaml.Node `yaml:"roles"`
	}
	if err := yaml.Unmarshal(yamlBytes, &order); err != nil {
		return nil, err
	}

	p := &RESTPermissions{roles: roles, operations: operations}

	for _, name := range keysInFileOrder(order.Roles) {
		r := f.Roles[name]
		if !r.Assignable {
			continue
		}

		if r.Label == "" || r.Description == "" {
			return nil, fmt.Errorf("assignable role %s needs a label and a description", name)
		}

		p.assignable = append(p.assignable, AssignableRole{Name: name, Label: r.Label, Description: r.Description})
	}

	return p, nil
}

func (p *RESTPermissions) HasPermission(role string, a Action) bool {
	if a.Type() != RESTEndpoint {
		return false
	}

	return p.roles.has(role, func(grant string) bool { return strings.EqualFold(grant, a.Name()) })
}

func (p *RESTPermissions) Defines(role string) bool {
	return p.roles.defines(role)
}

func (p *RESTPermissions) Grant(role string, operationIDs ...string) error {
	if err := p.roles.grant(role, operationIDs...); err != nil {
		return err
	}

	for _, op := range operationIDs {
		p.operations[op] = struct{}{}
	}

	return nil
}

func (p *RESTPermissions) AssignableRoles() []AssignableRole {
	operations := slices.Sorted(maps.Keys(p.operations))

	// resolved per call rather than at load, so operations the extension grants afterwards are included
	return enumerables.Map(p.assignable, func(r AssignableRole) AssignableRole {
		r.Operations = enumerables.Filter(operations, func(op string) bool {
			return p.HasPermission(r.Name, REST(op, nil))
		})

		return r
	})
}

// specOperations lists operation ids in spec, keeping only those accepting scheme when one is given.
func specOperations(spec *openapi3.T, scheme *string) map[string]struct{} {
	out := map[string]struct{}{}

	for _, item := range spec.Paths.Map() {
		for _, op := range item.Operations() {
			if scheme != nil && !acceptsScheme(spec, op, *scheme) {
				continue
			}

			out[op.OperationID] = struct{}{}
		}
	}

	return out
}

func acceptsScheme(spec *openapi3.T, op *openapi3.Operation, scheme string) bool {
	reqs := spec.Security
	if op.Security != nil {
		reqs = *op.Security
	}

	for _, req := range reqs {
		if _, ok := req[scheme]; ok {
			return true
		}
	}

	return false
}
