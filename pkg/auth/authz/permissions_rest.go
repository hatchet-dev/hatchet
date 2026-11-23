package authz

import (
	"errors"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"go.yaml.in/yaml/v3"
)

// restFile is the REST format: roles grant OpenAPI operation ids.
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

// LoadRESTPermissions requires the file to cover exactly the operations in spec.
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

	operations := map[string]struct{}{}
	for _, item := range spec.Paths.Map() {
		for _, op := range item.Operations() {
			operations[op.OperationID] = struct{}{}
		}
	}

	if err := checkCoverage("operation", operations, roles.grantedNames()); err != nil {
		return nil, err
	}

	return &RESTPermissions{roles: roles}, nil
}

func (p *RESTPermissions) Type() ActionType { return RESTEndpoint }

func (p *RESTPermissions) HasPermission(role string, a Action) bool {
	if a.Type() != RESTEndpoint {
		return false
	}

	return p.roles.has(role, func(grant string) bool { return strings.EqualFold(grant, a.Name()) })
}

// Grant adds extension operations that are served outside the spec this file was validated against.
func (p *RESTPermissions) Grant(role string, operationIDs ...string) error {
	return p.roles.grant(role, operationIDs...)
}
