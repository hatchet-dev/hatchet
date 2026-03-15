package rbac

import (
	_ "embed"
	"fmt"
	"maps"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"go.yaml.in/yaml/v3"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func OperationIn(operationId string, operationIds []string) bool {
	for _, id := range operationIds {
		if strings.EqualFold(operationId, id) {
			return true
		}
	}

	return false
}

type Authorizer struct {
	permissionMap PermissionMap
}

func NewAuthorizer() (*Authorizer, error) {
	permMap, err := LoadYaml()
	if err != nil {
		return nil, err
	}
	spec, err := gen.GetSwagger()
	if err != nil {
		return nil, err
	}
	err = permMap.ValidateSpec(*spec)
	if err != nil {
		return nil, err
	}
	return &Authorizer{
		permissionMap: *permMap,
	}, nil
}

func (a *Authorizer) IsAuthorized(role sqlcv1.TenantMemberRole, operation string) bool {
	return a.permissionMap.HasPermission(string(role), operation)
}

type Role struct {
	Inherits    *[]string
	Permissions *[]string
}

type RBACError struct {
	Message string
}

func (e *RBACError) Error() string {
	return e.Message
}

type PermissionMap struct {
	Roles map[string]*Role
}

func (p *PermissionMap) HasPermission(roleName string, operation string) bool {
	curRole := p.Roles[roleName]
	if curRole.Permissions != nil {
		inRole := OperationIn(operation, *curRole.Permissions)
		if inRole {
			return true
		}
	}
	if curRole.Inherits != nil {
		for _, inheritedRoleName := range *curRole.Inherits {
			if p.HasPermission(inheritedRoleName, operation) {
				return true
			}
		}
	}
	return false
}

func (p *PermissionMap) ValidInheritance(roleName string) error {
	if p.Roles[roleName].Inherits == nil {
		return nil
	}
	for _, inheritedRole := range *p.Roles[roleName].Inherits {
		_, ok := p.Roles[inheritedRole]
		if !ok {
			return &RBACError{
				Message: fmt.Sprintf("%s inherits from %s which does not exist", roleName, inheritedRole),
			}
		}
	}
	return nil
}

func (p *PermissionMap) Validate() error {
	for roleName := range p.Roles {
		if err := p.ValidInheritance(roleName); err != nil {
			return err
		}
	}
	return nil
}

func (p *PermissionMap) ValidateSpec(spec openapi3.T) error {
	specOperations := map[string]interface{}{}
	for _, pathItem := range spec.Paths.Map() {
		for _, op := range pathItem.Operations() {
			specOperations[op.OperationID] = struct{}{}
		}
	}

	allOperations := map[string]interface{}{}
	for roleName := range p.Roles {
		if p.Roles[roleName].Permissions == nil {
			continue
		}
		for _, operationId := range *p.Roles[roleName].Permissions {
			allOperations[operationId] = struct{}{}
		}
	}
	completeListOfOps := map[string]interface{}{}
	maps.Copy(completeListOfOps, specOperations)
	maps.Copy(completeListOfOps, allOperations)
	for operationId := range completeListOfOps {
		_, inSpec := specOperations[operationId]
		_, inYaml := allOperations[operationId]
		if inSpec && !inYaml {
			return &RBACError{
				Message: fmt.Sprintf("%s exists in openapi specs but not in rbac.yaml", operationId),
			}
		}
		if !inSpec && inYaml {
			return &RBACError{
				Message: fmt.Sprintf("%s exists in rbac.yaml but not in specs", operationId),
			}
		}
	}
	return nil
}

//go:embed rbac.yaml
var yamlFile []byte

func LoadYaml() (*PermissionMap, error) {
	var yamlContents PermissionMap
	err := yaml.Unmarshal(yamlFile, &yamlContents)
	if err != nil {
		return nil, err
	}
	if err := yamlContents.Validate(); err != nil {
		return nil, err
	}
	return &yamlContents, nil
}
