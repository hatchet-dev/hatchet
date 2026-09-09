package client

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// operatorIdMetadataKey is the gRPC metadata key carrying the operator id on
// every OperatorService RPC after Register. The value mirrors the server-side
// constant so the two packages stay independent.
const operatorIdMetadataKey = "hatchet-operator-id"

// OperatorClient connects an out-of-process operator to the engine's
// OperatorService. One Connect call is one live worker.
type OperatorClient interface {
	Connect(ctx context.Context, req *ConnectOperatorRequest) (OperatorSession, error)
}

// ConnectOperatorRequest describes the operator and the worker backing the
// session. Workflows and actions are not part of registration: put workflows
// with OperatorSession.PutWorkflow and add actions with AddActions.
type ConnectOperatorRequest struct {
	// Name is the operator name, unique per tenant among GRPC operators. The
	// engine upserts the operator row by this name.
	Name string `validate:"required"`

	// SlotConfig maps slot type to max units. The engine defaults it to
	// {"default": 100} when empty.
	SlotConfig map[string]int32

	// Labels are worker labels for affinity assignment. Values follow the
	// same conventions as GetActionListenerRequest.Labels (string or int).
	Labels map[string]interface{}

	// ResumeWorker controls whether a reconnect resumes the previous worker
	// id. nil means true.
	ResumeWorker *bool
}

// OperatorRegistration is the identity the engine assigned to the current
// Listen stream. WorkerId can change across reconnects when the previous
// worker no longer exists or ResumeWorker is false.
type OperatorRegistration struct {
	TenantId   string
	OperatorId string
	WorkerId   string

	// Resumed reports whether the most recent registration resumed the
	// previous worker. It is false on the first connect.
	Resumed bool
}

// OperatorSession is one registered operator worker. Actions may be called
// once; the other methods are safe to call concurrently until Close. The
// session reads its Listen stream from Connect on, so Flush observes delta
// acknowledgements whether or not Actions has been called.
type OperatorSession interface {
	// Actions starts the receive and heartbeat loops and returns the assigned
	// action stream. The channels close when ctx is cancelled, the session is
	// closed, or the stream fails permanently (reported on the error channel).
	Actions(ctx context.Context) (<-chan *dispatchercontracts.AssignedAction, <-chan error, error)

	// Registration returns the identity assigned by the most recent
	// successful registration.
	Registration() OperatorRegistration

	// SendStepActionEvent reports task progress. An empty WorkerId is filled
	// from the current registration.
	SendStepActionEvent(ctx context.Context, in *dispatchercontracts.StepActionEvent) (*dispatchercontracts.ActionEventResponse, error)

	// PutWorkflow registers or updates a workflow through the admin service on
	// the same connection and returns the action ids its tasks (including the
	// on-failure task) run, normalized the way the engine stores them. It does
	// not change the worker's action set: pass the ids to AddActions once the
	// operator can run them.
	PutWorkflow(ctx context.Context, wf *v1.CreateWorkflowVersionRequest) (*v1.CreateWorkflowVersionResponse, []string, error)

	// AddActions queues action ids to add to the worker's action set. It
	// never blocks: deltas are coalesced and sent in the background. Call
	// Flush to wait for them to reach the engine.
	AddActions(ids ...string)

	// RemoveActions queues action ids to remove from the worker's action set.
	// It never blocks; see AddActions.
	RemoveActions(ids ...string)

	// Flush waits until every queued delta has been acknowledged by the
	// engine, which acknowledges a delta once it is committed to the worker's
	// action set. A delta whose send failed and has not been retried yet is
	// reported as the send error; it stays queued and is replayed by the next
	// reconnect. The scheduler observes a committed change within about a
	// second.
	Flush(ctx context.Context) error

	// NewDurableTaskListener builds a durable task listener bound to this
	// session's worker. The caller starts it with Start; Close stops every
	// listener created here.
	NewDurableTaskListener(opts ...DurableTaskListenerOpt) *DurableTaskListener

	// Pause stops the scheduler assigning to this session's worker. It
	// returns once the engine has committed the pause, so a caller that
	// drains afterwards knows no further action will be assigned. It is its
	// own call rather than a message on the Listen stream so it also works
	// while the stream is reconnecting.
	Pause(ctx context.Context) error

	// Resume lets the scheduler assign to the worker again. Reconnecting also
	// clears the pause, so a session that is resumed after a crash comes back
	// assignable without this call.
	Resume(ctx context.Context) error

	// Close pauses the worker, waits for the actions already handed to the
	// consumer to be reported, flushes pending deltas with a short timeout,
	// and ends the Listen stream, which deactivates the worker. Pass
	// WithoutDrain to hang up at once instead.
	Close(opts ...CloseOpt) error
}

// CloseOpt changes how an operator session is closed.
type CloseOpt func(*closeOpts)

type closeOpts struct {
	drain        bool
	drainTimeout time.Duration
}

// WithoutDrain closes the session at once, without pausing the worker or
// waiting for in-flight actions. Use it when the process is going away and the
// work it holds will be retried by the engine anyway.
func WithoutDrain() CloseOpt {
	return func(o *closeOpts) { o.drain = false }
}

// WithDrainTimeout bounds the wait for in-flight actions on Close. When it
// elapses the session hangs up with work still outstanding, which the engine
// retries once the task times out.
func WithDrainTimeout(d time.Duration) CloseOpt {
	return func(o *closeOpts) { o.drainTimeout = d }
}

type operatorClientImpl struct {
	client             v1.OperatorServiceClient
	admin              v1.AdminServiceClient
	l                  *zerolog.Logger
	v                  validator.Validator
	ctx                *contextLoader
	presetWorkerLabels map[string]string
}

func newOperatorClient(conn *grpc.ClientConn, opts *sharedClientOpts, presetWorkerLabels map[string]string) OperatorClient {
	return &operatorClientImpl{
		client:             v1.NewOperatorServiceClient(conn),
		admin:              v1.NewAdminServiceClient(conn),
		l:                  opts.l,
		v:                  opts.v,
		ctx:                opts.ctxLoader,
		presetWorkerLabels: presetWorkerLabels,
	}
}

func (o *operatorClientImpl) Connect(ctx context.Context, req *ConnectOperatorRequest) (OperatorSession, error) {
	if req == nil {
		return nil, fmt.Errorf("connect operator request is required")
	}

	if err := o.v.Validate(req); err != nil {
		return nil, err
	}

	labels := map[string]*dispatchercontracts.WorkerLabels{}

	if req.Labels != nil {
		labels = mapLabels(req.Labels)
	}

	for k, v := range o.presetWorkerLabels {
		value := v
		labels[k] = &dispatchercontracts.WorkerLabels{StrValue: &value}
	}

	register := &v1.OperatorRegisterRequest{
		Name:        req.Name,
		SlotConfig:  req.SlotConfig,
		Labels:      labels,
		RuntimeInfo: goRuntimeInfo(),
	}

	resume := req.ResumeWorker == nil || *req.ResumeWorker

	session := newOperatorSession(o.client, o.admin, o.ctx, o.l, register, resume)

	if err := session.connect(ctx); err != nil {
		// nothing has been assigned to a worker that never connected, so there is nothing to
		// drain and possibly no worker to pause
		_ = session.Close(WithoutDrain())
		return nil, fmt.Errorf("could not connect operator %s: %w", req.Name, err)
	}

	return session, nil
}

// goRuntimeInfo describes this process the same way worker registration does,
// so operator workers show up in the dashboard with language and versions.
func goRuntimeInfo() *dispatchercontracts.RuntimeInfo {
	var goVersion string
	var hatchetVersion string

	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		goVersion = buildInfo.GoVersion

		for _, dep := range buildInfo.Deps {
			if dep.Path == "github.com/hatchet-dev/hatchet" {
				hatchetVersion = dep.Version
				break
			}
		}
	}

	os := runtime.GOOS

	return &dispatchercontracts.RuntimeInfo{
		Language:        dispatchercontracts.SDKS_GO.Enum(),
		LanguageVersion: &goVersion,
		Os:              &os,
		SdkVersion:      &hatchetVersion,
	}
}
