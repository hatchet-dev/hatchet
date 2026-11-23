// Package permissions holds hatchet's authorization rules: user roles in users.yaml and token capabilities in tokens.yaml.
package permissions

import (
	_ "embed"
	"fmt"
	"sync"

	collectortracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	ingestorcontracts "github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
)

//go:embed users.yaml
var usersFile []byte

//go:embed tokens.yaml
var tokensFile []byte

const (
	// RoleUnverifiedEmail caps users who have not verified their email.
	RoleUnverifiedEmail = "UNVERIFIED_EMAIL"

	// RoleNoAuth caps every request in authdisabled builds.
	RoleNoAuth = "NOAUTH"
)

// requiredRoles are referenced from code, so loading fails if users.yaml drops one.
var requiredRoles = []string{RoleUnverifiedEmail, RoleNoAuth}

// Users loads a fresh copy of the user roles, since the REST server grants extension operations onto it.
func Users() (*pkgauthz.RESTPermissions, error) {
	spec, err := gen.GetSwagger()
	if err != nil {
		return nil, err
	}

	perms, err := pkgauthz.LoadRESTPermissions(usersFile, spec)
	if err != nil {
		return nil, err
	}

	for _, role := range requiredRoles {
		if !perms.Defines(role) {
			return nil, fmt.Errorf("users.yaml must define role %s", role)
		}
	}

	return perms, nil
}

// grpcServices lists every service the gRPC server mounts, so a new procedure fails startup until it has a capability.
func grpcServices() []protoreflect.ServiceDescriptor {
	return []protoreflect.ServiceDescriptor{
		ingestorcontracts.File_events_proto.Services().ByName("EventsService"),
		dispatchercontracts.File_dispatcher_proto.Services().ByName("Dispatcher"),
		admincontracts.File_workflows_proto.Services().ByName("WorkflowService"),
		v1contracts.File_v1_dispatcher_proto.Services().ByName("V1Dispatcher"),
		v1contracts.File_v1_workflows_proto.Services().ByName("AdminService"),
		v1contracts.File_v1_streams_proto.Services().ByName("V1Streams"),
		v1contracts.File_v1_operator_proto.Services().ByName("OperatorService"),
		collectortracev1.File_opentelemetry_proto_collector_trace_v1_trace_service_proto.Services().ByName("TraceService"),
	}
}

// Tokens loads a fresh copy of the token capabilities, since the REST server grants extension operations onto it.
func Tokens() (*pkgauthz.TokenPermissions, error) {
	spec, err := gen.GetSwagger()
	if err != nil {
		return nil, err
	}

	return pkgauthz.LoadTokenPermissions(tokensFile, spec, grpcServices()...)
}

var sharedTokens = sync.OnceValues(Tokens)

// TokenCapabilityExists validates capability names when a token is created.
func TokenCapabilityExists(capability string) (bool, error) {
	perms, err := sharedTokens()
	if err != nil {
		return false, err
	}

	return perms.Defines(capability), nil
}
