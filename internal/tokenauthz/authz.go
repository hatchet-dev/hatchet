// Package tokenauthz holds the token capability file, which grants tenant API tokens REST operations and gRPC procedures.
package tokenauthz

import (
	_ "embed"
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

//go:embed tokens.yaml
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

// Load returns a fresh copy, since the REST server grants extension operations onto it.
func Load() (*pkgauthz.TokenPermissions, error) {
	spec, err := gen.GetSwagger()
	if err != nil {
		return nil, err
	}

	return pkgauthz.LoadTokenPermissions(permissionsFile, spec, services()...)
}

var shared = sync.OnceValues(Load)

// Defines reports whether capability exists, for validating names when a token is created.
func Defines(capability string) (bool, error) {
	perms, err := shared()
	if err != nil {
		return false, err
	}

	return perms.Defines(capability), nil
}
