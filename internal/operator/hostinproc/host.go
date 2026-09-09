// Package hostinproc is the in-process operator host: it implements pkg/operator.Host inside
// the dispatcher process over internal/services/operatorsvc, the same session logic the
// OperatorService handlers serve to operators that run outside the engine. A session opened
// here gets assigned actions by direct call, applies action deltas synchronously, reports
// events and opens durable invocations on the tenant-scoped in-engine paths, and is kept alive
// by one host-wide bulk heartbeat per tick.
//
// The host is engine-internal: it runs only where a dispatcher runs and is wired from
// cmd/hatchet-engine/engine. Contract operators are hosted here or over gRPC as a deployment
// choice; the DAG operator is hosted here only, for lifecycle.
package hostinproc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const (
	// defaultHeartbeatInterval is how often the host heartbeats every open session's worker in
	// one statement. The engine treats a worker as gone after a much longer silence.
	defaultHeartbeatInterval = 4 * time.Second

	// heartbeatTimeout bounds one bulk heartbeat write.
	heartbeatTimeout = 5 * time.Second
)

// TenantStore reads the tenant row a session belongs to; the tenant is put on every context
// the host hands the engine, the way the gRPC auth middleware does for a token.
type TenantStore interface {
	GetTenantByID(ctx context.Context, tenantId uuid.UUID) (*sqlcv1.Tenant, error)
}

// HeartbeatStore is the bulk heartbeat write.
type HeartbeatStore interface {
	UpdateWorkerHeartbeats(ctx context.Context, workerIds []uuid.UUID, lastHeartbeatAt time.Time) error
}

// WorkflowStore puts a workflow for a tenant and lists the steps of the version it created,
// which is where the action ids the workflow derives are read from: the engine's stored ids
// are the normalized ones. The admin service and the workflow repository satisfy it together.
type WorkflowStore interface {
	PutWorkflow(ctx context.Context, req *v1.CreateWorkflowVersionRequest) (*v1.CreateWorkflowVersionResponse, error)
	ListStepsByWorkflowVersionId(ctx context.Context, tenantId uuid.UUID, workflowVersionId uuid.UUID) ([]*sqlcv1.ListStepsByWorkflowVersionIdsRow, error)
}

// Deps are the collaborators a Host cannot run without. Workflows is optional: without it
// Session.PutWorkflow reports ErrNotSupported.
type Deps struct {
	Service    *operatorsvc.Service
	Tenants    TenantStore
	Heartbeats HeartbeatStore
	Workflows  WorkflowStore
	Logger     *zerolog.Logger
}

type opts struct {
	heartbeatInterval time.Duration
}

type Opt func(*opts)

// WithHeartbeatInterval sets how often the bulk heartbeat runs.
func WithHeartbeatInterval(d time.Duration) Opt {
	return func(o *opts) { o.heartbeatInterval = d }
}

// Host opens in-process sessions and heartbeats their workers until they close.
type Host struct {
	svc        *operatorsvc.Service
	tenants    TenantStore
	heartbeats HeartbeatStore
	workflows  WorkflowStore
	l          *zerolog.Logger

	// sessions is every open session by worker id; a session is in the set from Open until
	// its Close returns, so a worker draining between Pause and Close keeps heartbeating.
	mu       sync.Mutex
	sessions map[uuid.UUID]*session

	stop     chan struct{}
	stopOnce sync.Once
	done     chan struct{}
}

// New builds a host and starts its heartbeat ticker; Close stops it. Sessions are closed by
// whoever opened them, before the host.
func New(deps Deps, fs ...Opt) (*Host, error) {
	if deps.Service == nil || deps.Tenants == nil || deps.Heartbeats == nil {
		return nil, errors.New("hostinproc: the operator service, tenant store and heartbeat store are required")
	}

	o := &opts{heartbeatInterval: defaultHeartbeatInterval}

	for _, f := range fs {
		f(o)
	}

	l := deps.Logger

	if l == nil {
		defaultLogger := logger.NewDefaultLogger("operator_host")
		l = &defaultLogger
	}

	hl := l.With().Str("service", "operator_host").Logger()

	h := &Host{
		svc:        deps.Service,
		tenants:    deps.Tenants,
		heartbeats: deps.Heartbeats,
		workflows:  deps.Workflows,
		l:          &hl,
		sessions:   map[uuid.UUID]*session{},
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}

	go h.runHeartbeats(o.heartbeatInterval)

	return h, nil
}

// Open implements operator.Host. It registers the operator (an existing row by id, or an upsert
// by name and kind) and its worker, opens a handler-backed engine session under a fresh session
// id that is both the dispatcher's key and the worker row's listener fence, links the initial
// action set, and adds the worker to the heartbeat set. A failure after the session was opened
// closes it again, so the caller never inherits a half-open worker.
func (h *Host) Open(ctx context.Context, id operator.Identity, o operator.OpenOpts) (operator.Session, error) {
	if o.Handler == nil {
		return nil, errors.New("hostinproc: an action handler is required")
	}

	if id.OperatorId == nil && id.Name == "" {
		return nil, errors.New("hostinproc: the identity names neither an operator id nor an operator name")
	}

	tenant, err := h.tenants.GetTenantByID(ctx, id.TenantId)

	if err != nil {
		return nil, fmt.Errorf("hostinproc: could not load tenant %s: %w", id.TenantId, err)
	}

	kind := id.Kind

	if kind == "" {
		kind = sqlcv1.V1OperatorKindGRPC
	}

	reg, err := h.svc.Register(ctx, tenant, operatorsvc.RegisterOpts{
		OperatorId:     id.OperatorId,
		Name:           id.Name,
		Kind:           kind,
		WorkerName:     o.WorkerName,
		SlotConfig:     o.SlotConfig,
		Labels:         labelsToProto(o.Labels),
		RuntimeInfo:    o.RuntimeInfo,
		ResumeWorkerId: o.ResumeWorkerId,
	})

	if err != nil {
		return nil, err
	}

	ss, err := h.svc.OpenSession(ctx, tenant, reg.Operator, reg.WorkerId, operatorsvc.OpenOpts{Handler: o.Handler})

	if err != nil {
		return nil, err
	}

	s := &session{
		host: h,
		ss:   ss,
		reg: operator.Registration{
			TenantId:   reg.TenantId,
			OperatorId: reg.OperatorId,
			WorkerId:   reg.WorkerId,
			Resumed:    reg.Resumed,
		},
	}

	if len(o.Actions) > 0 {
		if err := s.applyDelta(ctx, o.Actions, nil); err != nil {
			_ = ss.Close(ctx)
			return nil, fmt.Errorf("hostinproc: could not link the initial actions: %w", err)
		}
	}

	h.mu.Lock()
	h.sessions[reg.WorkerId] = s
	h.mu.Unlock()

	return s, nil
}

// forget drops a closed session from the heartbeat set.
func (h *Host) forget(s *session) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.sessions[s.reg.WorkerId] == s {
		delete(h.sessions, s.reg.WorkerId)
	}
}

// SessionCount is the number of open sessions.
func (h *Host) SessionCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.sessions)
}

// Close stops the heartbeat ticker. Open sessions keep working but are no longer kept alive,
// so they are closed first.
func (h *Host) Close() {
	h.stopOnce.Do(func() { close(h.stop) })
	<-h.done
}

// runHeartbeats writes one bulk heartbeat per tick for every open session's worker.
func (h *Host) runHeartbeats(interval time.Duration) {
	defer close(h.done)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stop:
			return
		case <-ticker.C:
			h.heartbeat()
		}
	}
}

// heartbeat writes the heartbeat for every open session's worker in one statement. It runs on
// its own bounded context: the sessions it keeps alive may be draining during a shutdown that
// has already cancelled everything else.
func (h *Host) heartbeat() {
	workerIds := h.workerIds()

	if len(workerIds) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), heartbeatTimeout)
	defer cancel()

	if err := h.heartbeats.UpdateWorkerHeartbeats(ctx, workerIds, time.Now().UTC()); err != nil {
		h.l.Error().Err(err).Msg("could not heartbeat operator workers")
	}
}

func (h *Host) workerIds() []uuid.UUID {
	h.mu.Lock()
	defer h.mu.Unlock()

	ids := make([]uuid.UUID, 0, len(h.sessions))

	for workerId := range h.sessions {
		ids = append(ids, workerId)
	}

	return ids
}

// labelsToProto converts the contract's labels into the registration's, the way pkg/client
// does for a worker: strings are string labels, integers are int labels, anything else is its
// string form.
func labelsToProto(labels map[string]interface{}) map[string]*contracts.WorkerLabels {
	if len(labels) == 0 {
		return nil
	}

	out := make(map[string]*contracts.WorkerLabels, len(labels))

	for key, value := range labels {
		label := &contracts.WorkerLabels{}

		switch v := value.(type) {
		case string:
			label.StrValue = &v
		case int:
			intValue := int32(v) // nolint:gosec // label values are small integers
			label.IntValue = &intValue
		case int32:
			label.IntValue = &v
		case int64:
			intValue := int32(v) // nolint:gosec // label values are small integers
			label.IntValue = &intValue
		default:
			strValue := fmt.Sprintf("%v", v)
			label.StrValue = &strValue
		}

		out[key] = label
	}

	return out
}
