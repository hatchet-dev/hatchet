// Package authz holds the gRPC permission file and builds the authorizer the gRPC request gate uses.
package authz

import (
	"context"
	_ "embed"
	"sync"

	collectortracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/reflect/protoreflect"

	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	ingestorcontracts "github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
)

//go:embed grpc.yaml
var permissionsFile []byte

// services lists every service the gRPC server mounts, so a new procedure fails startup until it has a capability.
func services() []protoreflect.ServiceDescriptor {
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

// Permissions is loaded once because both the gRPC gate and REST token creation (to validate names) need it.
var Permissions = sync.OnceValues(func() (*pkgauthz.GRPCPermissions, error) {
	return pkgauthz.LoadGRPCPermissions(permissionsFile, services()...)
})

func New() (pkgauthz.Authorizer, error) {
	perms, err := Permissions()
	if err != nil {
		return nil, err
	}

	return pkgauthz.New(
		pkgauthz.WithPermissions(perms),
		pkgauthz.WithPolicies(readOnlyPolicy{}),
	)
}

// readOnlyPolicy keeps read-only tokens off gRPC entirely, whatever capabilities they hold.
type readOnlyPolicy struct{}

func (readOnlyPolicy) Evaluate(_ context.Context, req pkgauthz.Request) (pkgauthz.Decision, error) {
	if t, ok := req.Principal.(pkgauthz.TokenPrincipal); ok && t.ReadOnly {
		return pkgauthz.Deny(pkgauthz.TokenRestricted), nil
	}

	return pkgauthz.Allow(), nil
}
