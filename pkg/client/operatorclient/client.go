// Package operatorclient connects an out-of-process operator to the engine's
// OperatorService: one Connect call registers one worker and returns the
// Session that streams its assigned actions, keeps its action set up to date
// and reports task progress. The package speaks to a *grpc.ClientConn the
// caller owns; pkg/client's Operator accessor builds one over its own
// connection.
package operatorclient

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// operatorIdMetadataKey is the gRPC metadata key carrying the operator id on
// every OperatorService RPC after Register. The value mirrors the server-side
// constant so the two packages stay independent.
const operatorIdMetadataKey = "hatchet-operator-id"

// Client connects an out-of-process operator to the engine's OperatorService.
// One Connect call is one live worker.
type Client interface {
	Connect(ctx context.Context, req *ConnectRequest) (Session, error)
}

// ConnectRequest describes the operator and the worker backing the session.
// Workflows and actions are not part of registration: put workflows with
// Session.PutWorkflow and add actions with AddActions.
type ConnectRequest struct {
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

// Registration is the identity the engine assigned to the current Listen
// stream. WorkerId can change across reconnects when the previous worker no
// longer exists or ResumeWorker is false.
type Registration struct {
	TenantId   string
	OperatorId string
	WorkerId   string

	// Resumed reports whether the most recent registration resumed the
	// previous worker. It is false on the first connect.
	Resumed bool
}

// Session is one registered operator worker. Actions may be called once; the
// other methods are safe to call concurrently until Close. The session reads
// its Listen stream from Connect on, so Flush observes delta acknowledgements
// whether or not Actions has been called.
type Session interface {
	// Actions starts the receive and heartbeat loops and returns the assigned
	// action stream. The channels close when ctx is cancelled, the session is
	// closed, or the stream fails permanently (reported on the error channel).
	Actions(ctx context.Context) (<-chan *dispatchercontracts.AssignedAction, <-chan error, error)

	// Registration returns the identity assigned by the most recent
	// successful registration.
	Registration() Registration

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

	// OpenDurableTaskStream opens the OperatorService durable task stream
	// with the session's operator metadata, shaped as the V1Dispatcher
	// stream a durable task listener expects. The register message's worker
	// id is rewritten to the current registration, so a listener built
	// before a reconnect registers the worker the engine now knows. The
	// caller owns the listener it builds over it and stops it before Close.
	OpenDurableTaskStream(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error)

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

// CloseOpt changes how a session is closed.
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

// Opt configures New.
type Opt func(*opts)

type opts struct {
	l                  *zerolog.Logger
	v                  validator.Validator
	headers            map[string]string
	presetWorkerLabels map[string]string
	token              string
}

func defaultOpts() *opts {
	l := zerolog.Nop()

	return &opts{
		l: &l,
		v: validator.NewDefaultValidator(),
	}
}

// WithToken sets the bearer token every RPC carries. It is required.
func WithToken(token string) Opt {
	return func(o *opts) { o.token = token }
}

// WithHeaders adds metadata to every RPC alongside the bearer token.
func WithHeaders(headers map[string]string) Opt {
	return func(o *opts) { o.headers = headers }
}

// WithLogger sets the logger; the default discards everything.
func WithLogger(l *zerolog.Logger) Opt {
	return func(o *opts) { o.l = l }
}

// WithValidator sets the validator Connect checks its request with.
func WithValidator(v validator.Validator) Opt {
	return func(o *opts) { o.v = v }
}

// WithPresetWorkerLabels sets labels every connected worker carries on top of
// the request's own; a preset label wins over a request label of the same
// name.
func WithPresetWorkerLabels(labels map[string]string) Opt {
	return func(o *opts) { o.presetWorkerLabels = labels }
}

// callMetadata is the outgoing metadata every RPC carries: the bearer token
// and any extra headers.
type callMetadata struct {
	headers map[string]string
	token   string
}

func newCallMetadata(token string, headers map[string]string) *callMetadata {
	return &callMetadata{token: token, headers: headers}
}

// context returns ctx with the bearer token and headers as outgoing metadata.
func (m *callMetadata) context(ctx context.Context) context.Context {
	pairs := map[string]string{
		"authorization": "Bearer " + m.token,
	}

	for k, v := range m.headers {
		pairs[k] = v
	}

	return metadata.NewOutgoingContext(ctx, metadata.New(pairs))
}

type clientImpl struct {
	client             v1.OperatorServiceClient
	admin              v1.AdminServiceClient
	l                  *zerolog.Logger
	v                  validator.Validator
	md                 *callMetadata
	presetWorkerLabels map[string]string
}

// New builds a Client over conn. WithToken is required; the other options
// have defaults.
func New(conn *grpc.ClientConn, fs ...Opt) (Client, error) {
	if conn == nil {
		return nil, fmt.Errorf("a gRPC connection is required")
	}

	o := defaultOpts()

	for _, f := range fs {
		f(o)
	}

	if o.token == "" {
		return nil, fmt.Errorf("a token is required. use WithToken")
	}

	if o.l == nil {
		return nil, fmt.Errorf("a logger is required. use WithLogger or omit it for the default")
	}

	if o.v == nil {
		return nil, fmt.Errorf("a validator is required. use WithValidator or omit it for the default")
	}

	return &clientImpl{
		client:             v1.NewOperatorServiceClient(conn),
		admin:              v1.NewAdminServiceClient(conn),
		l:                  o.l,
		v:                  o.v,
		md:                 newCallMetadata(o.token, o.headers),
		presetWorkerLabels: o.presetWorkerLabels,
	}, nil
}

func (o *clientImpl) Connect(ctx context.Context, req *ConnectRequest) (Session, error) {
	if req == nil {
		return nil, fmt.Errorf("connect request is required")
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

	s := newSession(o.client, o.admin, o.md, o.l, register, resume)

	if err := s.connect(ctx); err != nil {
		// nothing has been assigned to a worker that never connected, so there is nothing to
		// drain and possibly no worker to pause
		_ = s.Close(WithoutDrain())
		return nil, fmt.Errorf("could not connect operator %s: %w", req.Name, err)
	}

	return s, nil
}

// mapLabels converts request labels to the contract's typed labels the way
// worker registration does: strings and ints keep their type, anything else
// is formatted as a string.
func mapLabels(req map[string]interface{}) map[string]*dispatchercontracts.WorkerLabels {
	labels := map[string]*dispatchercontracts.WorkerLabels{}

	for k, v := range req {
		label := dispatchercontracts.WorkerLabels{}

		switch value := v.(type) {
		case string:
			strValue := value
			label.StrValue = &strValue
		case int:
			intValue := int32(value) // nolint: gosec
			label.IntValue = &intValue
		case int32:
			label.IntValue = &value
		case int64:
			intValue := int32(value) // nolint: gosec
			label.IntValue = &intValue
		default:
			strValue := fmt.Sprintf("%v", value)
			label.StrValue = &strValue
		}

		labels[k] = &label
	}

	return labels
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
