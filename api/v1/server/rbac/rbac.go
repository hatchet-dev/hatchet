package rbac

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.yaml.in/yaml/v4"

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

func LoadYaml() (*PermissionMap, error) {
	_, yamlPath, _, _ := runtime.Caller(0)
	yamlFile, err := os.ReadFile(filepath.Join(filepath.Dir(yamlPath), "rbac.yaml"))
	if err != nil {
		return nil, err
	}
	var yamlContents PermissionMap
	err = yaml.Unmarshal(yamlFile, &yamlContents)
	if err != nil {
		return nil, err
	}
	if err := yamlContents.Validate(); err != nil {
		return nil, err
	}
	return &yamlContents, nil
}
