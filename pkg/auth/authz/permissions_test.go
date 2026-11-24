package authz

import (
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
)

func testSpec(operationIDs ...string) *openapi3.T {
	paths := openapi3.NewPaths()
	for _, id := range operationIDs {
		paths.Set("/"+id, &openapi3.PathItem{Get: &openapi3.Operation{OperationID: id}})
	}

	return &openapi3.T{Paths: paths}
}

// The shipped yaml files only exercise the success path, so these cover the startup failures that keep a
// new endpoint from being silently unauthorized.
func TestInvalidPermissionFilesFailToLoad(t *testing.T) {
	const roles = `
roles:
  ADMIN:
    inherits: [VIEWER]
    permissions: [Write]
  VIEWER:
    permissions: [Read]
`

	_, err := LoadRESTPermissions([]byte(roles), testSpec("Read", "Write"))
	require.NoError(t, err)

	for name, load := range map[string]func() error{
		"operation missing from the file": func() error {
			_, err := LoadRESTPermissions([]byte(roles), testSpec("Read", "Write", "Delete"))
			return err
		},
		"operation missing from the spec": func() error {
			_, err := LoadRESTPermissions([]byte(roles), testSpec("Read"))
			return err
		},
		"inheritance cycle": func() error {
			_, err := LoadRESTPermissions([]byte("roles:\n  A:\n    inherits: [B]\n  B:\n    inherits: [A]\n"), testSpec())
			return err
		},
	} {
		require.Error(t, load(), name)
	}

	// Read and Write accept bearer tokens; Login is cookie-only, so token files need not mention it.
	spec := testSpec("Read", "Write", "Mint")
	spec.Security = openapi3.SecurityRequirements{{"bearerAuth": []string{}}, {"cookieAuth": []string{}}}
	cookieOnly := openapi3.SecurityRequirements{{"cookieAuth": []string{}}}
	spec.Paths.Set("/Login", &openapi3.PathItem{Post: &openapi3.Operation{OperationID: "Login", Security: &cookieOnly}})

	svc := contracts.File_events_proto.Services().Get(0)
	var procedures []string
	for i := 0; i < svc.Methods().Len(); i++ {
		procedures = append(procedures, `"/`+string(svc.FullName())+"/"+string(svc.Methods().Get(i).Name())+`"`)
	}

	tokens := `unavailable:
  rest: [Mint]
capabilities:
  all:
    rest: ["*"]
    grpc: ["*"]
  read-only:
    rest: [Read]
  events:
    rest: [Write]
    grpc: [` + strings.Join(procedures, ", ") + `]
`

	_, err = LoadTokenPermissions([]byte(tokens), spec, svc)
	require.NoError(t, err)

	for name, file := range map[string]string{
		"bearer operation and procedures not listed": strings.Replace(tokens, "    rest: [Write]\n    grpc: ["+strings.Join(procedures, ", ")+"]\n", "    rest: []\n", 1),
		"no all capability":                          strings.Replace(tokens, "all:", "everything:", 1),
		"granted and unavailable":                    strings.Replace(tokens, "rest: [Read]", "rest: [Read, Mint]", 1),
	} {
		_, err := LoadTokenPermissions([]byte(file), spec, svc)
		require.Error(t, err, name)
	}
}
