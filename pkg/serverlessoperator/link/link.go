// Package link defines the seam between the serverless operator core and the engine. The core
// (leasing, healthchecks, routing, delivery) is the same in and out of process; what differs is
// how a registration reaches the engine. grpclink speaks OperatorService over gRPC with a
// per-tenant token; enginelink (a later phase) calls the dispatcher in process.
package link

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// ErrNoToken is returned by Open when the link cannot authenticate as the tenant. The core
// keeps the unit, marks its endpoints with a status error once, and retries Open on later
// ticks so a token that appears afterwards is picked up without a restart.
var ErrNoToken = errors.New("no token for tenant")

// ErrDurableNotSupported is returned by OpenDurable on links that do not relay durable
// invocations yet. The durable relay is a later phase; until then the core fails durable
// actions with a retryable error.
var ErrDurableNotSupported = errors.New("durable delivery not supported by this link")

// ErrRequestInFlight is returned by DurableChannel.Send when an ack-bearing request (memo,
// trigger_runs, wait_for, evict_invocation) is sent while another one is still waiting for
// its ack. The engine keys pending acks by (task, invocation), so a second one would clobber
// the first; the relay closes the socket with code 4006.
var ErrRequestInFlight = errors.New("durable request already in flight for this invocation")

// ErrChannelClosed is returned by DurableChannel.Recv once Close was called.
var ErrChannelClosed = errors.New("durable channel closed")

// OpenOpts is what a registration advertises to the engine when it opens: the tenant's known
// workflows (already namespaced), the full action set (the union of registered_actions over the
// tenant's enabled endpoints), the process-level slot config and the worker labels.
type OpenOpts struct {
	Workflows  []*v1.CreateWorkflowVersionRequest
	Actions    []string
	SlotConfig map[string]int32
	Labels     map[string]interface{}
}

// Link opens registrations for a tenant. One Link per process; one Registration per owned unit.
type Link interface {
	Open(ctx context.Context, tenantId uuid.UUID, shard int, opts OpenOpts) (Registration, error)
}

// Registration is one engine Worker row's worth of traffic.
type Registration interface {
	// WorkerId is the engine worker id backing this registration.
	WorkerId() string

	// Actions starts the action stream. May be called once. The channels close when ctx is
	// cancelled, the registration is closed, or the link fails permanently, in which case the
	// failure is reported on the error channel.
	Actions(ctx context.Context) (<-chan *contracts.AssignedAction, <-chan error, error)

	// PutWorkflow registers or updates one namespaced workflow and sets the worker's full
	// action set in the same call.
	PutWorkflow(ctx context.Context, wf *v1.CreateWorkflowVersionRequest, fullActions []string) error

	// UpdateActions replaces the worker's action set. fullActions is the full set, not a delta.
	UpdateActions(ctx context.Context, fullActions []string) error

	// SendStepActionEvent reports task progress (STARTED, COMPLETED, FAILED, CANCELLED).
	SendStepActionEvent(ctx context.Context, ev *contracts.StepActionEvent) error

	// OpenDurable opens the request/response pipe of one durable invocation.
	OpenDurable(ctx context.Context, taskExternalId string, invocation int32) (DurableChannel, error)

	// Close ends the registration; the engine deactivates the worker.
	Close() error
}

// DurableChannel is one durable invocation's request/response pipe, what the websocket relay
// drives.
type DurableChannel interface {
	Send(*v1.DurableTaskRequest) error
	Recv() (*v1.DurableTaskResponse, error)
	Close() error
}

// TenantReleaser is implemented by links that hold per-tenant state outside registrations,
// such as grpclink's client cache. The core calls ReleaseTenant when it owns no more units of
// the tenant, after every registration for the tenant is closed.
type TenantReleaser interface {
	ReleaseTenant(tenantId uuid.UUID)
}
