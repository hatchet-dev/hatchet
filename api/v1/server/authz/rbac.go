package authz

import (
	_ "embed"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
)

//go:embed rbac.yaml
var yamlFile []byte

func newHatchetPermissions() (*pkgauthz.RESTPermissions, error) {
	spec, err := gen.GetSwagger()
	if err != nil {
		return nil, err
	}

	return pkgauthz.LoadRESTPermissions(yamlFile, spec)
}
