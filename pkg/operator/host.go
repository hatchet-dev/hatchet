package operator

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// This file is the hosting contract: what an operator needs from the engine to receive
// assigned actions for a worker row, report results, drive durable invocations and stay
// alive, independent of whether it runs inside the engine process or over OperatorService.
//
// Every gRPC operator can run in process, but not every operator is a gRPC operator. An
// operator written against Host, Session and DurableChannel is hostable by either host as a
// deployment choice. Engine-internal operators (the DAG operator) are hosted in process for
// lifecycle only and keep TaskEventWriter and repository access, which never exist over gRPC;
// those types are in operator.go and are not part of this contract.

var (
	// ErrNotSupported is returned by a host or session for a capability the transport does
	// not offer: the gRPC host cannot open an already claimed operator row, and an in-process
	// host without an admin service cannot put workflows.
	ErrNotSupported = errors.New("operator: not supported by this host")

	// ErrSessionClosed is returned by session methods once Close was called.
	ErrSessionClosed = errors.New("operator: session closed")

	// ErrRequestInFlight is returned by DurableChannel.Send when an ack-bearing request (memo,
	// trigger_runs, wait_for, evict_invocation) is sent while a previous one is still waiting
	// for its ack. The engine keys pending acks by (task, invocation), so a second one would
	// clobber the first.
	ErrRequestInFlight = errors.New("operator: durable request already in flight for this invocation")

	// ErrChannelClosed is returned by DurableChannel.Send and Recv once Close was called.
	ErrChannelClosed = errors.New("operator: durable channel closed")

	// ErrSessionEnded is returned by DurableChannel.Recv when the engine tore the invocation
	// down without Close being called, which the operator reports as a retryable failure.
	ErrSessionEnded = errors.New("operator: engine durable session ended")
)

// Identity names the operator a session registers as. Exactly one of OperatorId (an existing
// row, as claimed by the in-process claimer for the DAG operator) or Name and Kind (a row the
// host upserts by (tenant, name, kind)) is used. TenantId is always required: a Host spans
// tenants and a Session belongs to one.
type Identity struct {
	TenantId uuid.UUID

	// OperatorId is an existing operator row. The host takes the name and kind from the row
	// and points the row's worker_id at the session's worker, so the claimer keeps recognising
	// the assignment.
	OperatorId *uuid.UUID

	// Name and Kind identify a row the host upserts. Kind defaults to GRPC, the only kind a
	// host registers today.
	Name string
	Kind sqlcv1.V1OperatorKind
}

// OpenOpts describes the worker the session backs and the handler assigned actions go to.
type OpenOpts struct {
	// Handler receives the actions assigned to the session's worker. Required.
	Handler ActionHandler

	// Actions is the initial action set, linked before Open returns so the worker is never
	// live with an action set the caller did not ask for. Later changes go through
	// Session.AddActions and RemoveActions.
	Actions []string

	// SlotConfig maps slot type to max units. The engine defaults it to {"default": 100}.
	SlotConfig map[string]int32

	// Labels are worker labels for affinity assignment (string or int values).
	Labels map[string]interface{}

	// RuntimeInfo describes the process for the dashboard. The gRPC host reports its own
	// process and ignores it.
	RuntimeInfo *contracts.RuntimeInfo

	// ResumeWorkerId names a previous worker of the same operator to resume instead of
	// creating one. Not every host supports it (ErrNotSupported).
	ResumeWorkerId *uuid.UUID

	// WorkerName names the worker row; it defaults to the operator name. Not every host
	// supports it (ErrNotSupported).
	WorkerName string
}

// Registration is the identity the engine assigned to a session.
type Registration struct {
	TenantId   uuid.UUID
	OperatorId uuid.UUID
	WorkerId   uuid.UUID

	// Resumed reports whether WorkerId is the worker ResumeWorkerId named.
	Resumed bool
}

// Host opens sessions. One Host per process; it spans tenants. Implementations:
// internal/operator/hostinproc (inside the dispatcher process, over the engine's own session
// logic) and pkg/operator/hostgrpc (pkg/client over OperatorService with a TokenSource).
type Host interface {
	Open(ctx context.Context, id Identity, opts OpenOpts) (Session, error)
}

// Session is one worker row's worth of traffic: its action set, its events, its durable
// invocations and its liveness, which the host maintains until Close. Methods are safe to call
// concurrently until Close.
type Session interface {
	Registration() Registration

	// AddActions adds ids to the worker's action set; ids already in the set are ignored. A
	// host may apply the delta asynchronously; Flush waits for it.
	AddActions(ctx context.Context, ids []string) error

	// RemoveActions removes ids from the worker's action set; ids not in the set are ignored.
	RemoveActions(ctx context.Context, ids []string) error

	// Flush returns once every delta issued so far is committed by the engine and reports the
	// failure, if any. In-process deltas are synchronous and Flush returns at once.
	Flush(ctx context.Context) error

	// PutWorkflow registers or updates a workflow for the session's tenant and returns the
	// action ids its tasks derive, normalized the way the engine stores them. It does not
	// change the worker's action set.
	PutWorkflow(ctx context.Context, wf *v1.CreateWorkflowVersionRequest) ([]string, error)

	// SendStepActionEvent reports task progress (STARTED, COMPLETED, FAILED, CANCELLED). An
	// empty WorkerId is filled from the registration; another worker's id is refused.
	SendStepActionEvent(ctx context.Context, ev *contracts.StepActionEvent) error

	// OpenDurable opens one invocation's request and response pipe. The register_worker
	// handshake is done by the host; responses that arrive before it completes are held,
	// bounded, and delivered after the ack that names their entry. ctx bounds the handshake;
	// the invocation itself ends on the channel's Close.
	OpenDurable(ctx context.Context, taskExternalId uuid.UUID, invocation int32) (DurableChannel, error)

	// Pause stops the scheduler assigning to the worker and returns once the pause is
	// committed, so a caller that drains afterwards knows no further work will arrive.
	Pause(ctx context.Context) error

	// Close ends the session: the worker is paused if it is not already, the session is
	// released so nothing further is delivered, and the worker is deactivated, fenced on the
	// session id. A host's teardown order is Pause, the operator's Drain, then Close.
	Close(ctx context.Context) error
}

// DurableChannel is one durable invocation's pipe. Send stamps the invocation's task id and
// count on the request and admits one ack-bearing request at a time (ErrRequestInFlight);
// the slot is free again once the ack, or the error that replaces it, has arrived.
// register_worker is the host's and is refused. worker_status may be sent to report the
// entries the invocation is blocked on; a host whose transport reports them itself drops it.
// Responses arrive in engine order, with an entry completion never ahead of the ack that
// names its entry.
type DurableChannel interface {
	Send(ctx context.Context, req *v1.DurableTaskRequest) error
	Recv(ctx context.Context) (*v1.DurableTaskResponse, error)
	Close() error
}
