package authz

import (
	_ "embed"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	"github.com/hatchet-dev/hatchet/pkg/auth/rbac"
)

//go:embed rbac.yaml
var yamlFile []byte

//go:embed bearer.yaml
var bearerYamlFile []byte

func newHatchetAuthorizer() (*rbac.Authorizer, error) {
	permMap, err := rbac.LoadPermissionMap(yamlFile)
	if err != nil {
		return nil, err
	}
	spec, err := gen.GetSwagger()
	if err != nil {
		return nil, err
	}

	return rbac.NewAuthorizer(permMap, spec)
}

func newBearerPolicy() (*rbac.BearerPolicy, error) {
	policy, err := rbac.LoadBearerPolicy(bearerYamlFile)
	if err != nil {
		return nil, err
	}

	spec, err := gen.GetSwagger()
	if err != nil {
		return nil, err
	}

	if err := policy.ValidateSpec(*spec); err != nil {
		return nil, err
	}

	return policy, nil
}
