package authz

import (
	"errors"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"go.yaml.in/yaml/v3"
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
	} `yaml:"roles"`
}

type RESTPermissions struct {
	roles grantGraph
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

	if err := checkCoverage("operation", specOperations(spec, nil), roles.grantedNames()); err != nil {
		return nil, err
	}

	return &RESTPermissions{roles: roles}, nil
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
	return p.roles.grant(role, operationIDs...)
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
