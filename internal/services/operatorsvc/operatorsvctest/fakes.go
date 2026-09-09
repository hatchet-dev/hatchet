// Package operatorsvctest holds in-memory doubles for the stores and the dispatcher an operator
// session talks to. They are shared by the tests of internal/services/operatorsvc and of the
// hosts built on it, so both drive the same behaviour: the fences, the budgets and the session
// bookkeeping are modelled here the way the repository and the dispatcher implement them.
package operatorsvctest

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

var (
	_ operatorsvc.OperatorStore     = (*OperatorStore)(nil)
	_ operatorsvc.WorkerStore       = (*WorkerStore)(nil)
	_ operatorsvc.DispatcherBackend = (*Dispatcher)(nil)
)

// OperatorStore serves operators from memory and upserts by (tenant, name). Sessions authorize
// concurrently, so the store is safe for concurrent use.
type OperatorStore struct {
	mu        sync.Mutex
	operators map[uuid.UUID]*sqlcv1.V1Operator
	getCalls  atomic.Int64
}

// NewOperatorStore seeds a store with the given operators.
func NewOperatorStore(ops ...*sqlcv1.V1Operator) *OperatorStore {
	s := &OperatorStore{operators: map[uuid.UUID]*sqlcv1.V1Operator{}}

	for _, op := range ops {
		s.operators[op.ID] = op
	}

	return s
}

func (f *OperatorStore) GetOperatorById(_ context.Context, operatorId uuid.UUID) (*sqlcv1.V1Operator, error) {
	f.getCalls.Add(1)

	f.mu.Lock()
	defer f.mu.Unlock()

	op, ok := f.operators[operatorId]

	if !ok {
		return nil, pgx.ErrNoRows
	}

	return op, nil
}

func (f *OperatorStore) UpsertGRPCOperator(_ context.Context, tenantId uuid.UUID, name string) (*sqlcv1.V1Operator, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, op := range f.operators {
		if op.TenantID == tenantId && op.Name == name && op.Kind == sqlcv1.V1OperatorKindGRPC {
			return op, nil
		}
	}

	op := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenantId, Name: name, Kind: sqlcv1.V1OperatorKindGRPC}

	if f.operators == nil {
		f.operators = map[uuid.UUID]*sqlcv1.V1Operator{}
	}

	f.operators[op.ID] = op

	return op, nil
}

// UpdateOperator applies the row changes a registration makes: pointing the row at a worker.
func (f *OperatorStore) UpdateOperator(_ context.Context, tenantId, operatorId uuid.UUID, opts repository.UpdateOperatorOpts) (*sqlcv1.V1Operator, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	op, ok := f.operators[operatorId]

	if !ok || op.TenantID != tenantId {
		return nil, pgx.ErrNoRows
	}

	if opts.WorkerId != nil {
		workerId := *opts.WorkerId
		op.WorkerID = &workerId
	}

	if opts.Name != nil {
		op.Name = *opts.Name
	}

	return op, nil
}

// Put adds an operator to the store, as a row created outside the service would be.
func (f *OperatorStore) Put(op *sqlcv1.V1Operator) *sqlcv1.V1Operator {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.operators == nil {
		f.operators = map[uuid.UUID]*sqlcv1.V1Operator{}
	}

	f.operators[op.ID] = op

	return op
}

// GetCalls counts the reads that reached the store, so a test can see the authorization cache
// working.
func (f *OperatorStore) GetCalls() int64 { return f.getCalls.Load() }

// Count is the number of operator rows.
func (f *OperatorStore) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.operators)
}

// WorkerStore keeps workers and their action sets in memory. Methods record their calls so tests
// can assert what reached the store. mu guards every field: a session's loop and the test
// goroutine touch the store concurrently.
type WorkerStore struct {
	mu sync.Mutex

	workers map[uuid.UUID]*sqlcv1.Worker
	actions map[uuid.UUID]map[string]struct{}

	created     []*repository.CreateWorkerOpts
	activations []uuid.UUID
	// active is the worker's isActive flag as the last activation or deactivation left it
	active map[uuid.UUID]bool
	// paused is the worker's isPaused flag as the last update left it
	paused map[uuid.UUID]bool
	// listenerSessions is the listener session id recorded on each worker by its last activation
	listenerSessions map[uuid.UUID]uuid.UUID
	// sessionIds records the session id passed to each activation and deactivation, in order
	sessionIds []uuid.UUID
	heartbeats int
	// bulkHeartbeats records the worker id sets of each bulk heartbeat write, in order
	bulkHeartbeats [][]uuid.UUID
	labels         map[uuid.UUID][]repository.UpsertWorkerLabelOpts
	dispatchers    map[uuid.UUID]uuid.UUID
	// ops records the writes that change a worker's lifecycle state, in order, so a test can
	// assert that a pause lands before the deactivation that follows it
	ops []string
}

func NewWorkerStore() *WorkerStore {
	return &WorkerStore{
		workers:          map[uuid.UUID]*sqlcv1.Worker{},
		actions:          map[uuid.UUID]map[string]struct{}{},
		active:           map[uuid.UUID]bool{},
		paused:           map[uuid.UUID]bool{},
		listenerSessions: map[uuid.UUID]uuid.UUID{},
		labels:           map[uuid.UUID][]repository.UpsertWorkerLabelOpts{},
		dispatchers:      map[uuid.UUID]uuid.UUID{},
	}
}

// Add seeds a worker row, as a registration outside the test would have created it.
func (f *WorkerStore) Add(w *sqlcv1.Worker) *sqlcv1.Worker {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.workers[w.ID] = w
	f.actions[w.ID] = map[string]struct{}{}

	return w
}

func (f *WorkerStore) CreateNewWorker(_ context.Context, tenantId uuid.UUID, opts *repository.CreateWorkerOpts) (*sqlcv1.Worker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.created = append(f.created, opts)

	dispatcherId := opts.DispatcherId
	w := &sqlcv1.Worker{ID: uuid.New(), TenantId: tenantId, Name: opts.Name, OperatorId: opts.OperatorId, DispatcherId: &dispatcherId}
	f.workers[w.ID] = w
	f.actions[w.ID] = map[string]struct{}{}
	f.dispatchers[w.ID] = dispatcherId

	return w, nil
}

func (f *WorkerStore) GetWorkerForEngine(_ context.Context, tenantId uuid.UUID, workerId uuid.UUID) (*sqlcv1.GetWorkerForEngineRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w, ok := f.workers[workerId]

	if !ok || w.TenantId != tenantId {
		return nil, pgx.ErrNoRows
	}

	return &sqlcv1.GetWorkerForEngineRow{
		ID:           w.ID,
		TenantId:     w.TenantId,
		DispatcherId: w.DispatcherId,
		OperatorId:   w.OperatorId,
	}, nil
}

func (f *WorkerStore) UpdateWorker(_ context.Context, _ uuid.UUID, workerId uuid.UUID, opts *repository.UpdateWorkerOpts) (*sqlcv1.Worker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if opts.DispatcherId != nil {
		f.dispatchers[workerId] = *opts.DispatcherId

		if w, ok := f.workers[workerId]; ok {
			dispatcherId := *opts.DispatcherId
			w.DispatcherId = &dispatcherId
		}
	}

	if opts.IsPaused != nil {
		f.paused[workerId] = *opts.IsPaused

		if *opts.IsPaused {
			f.ops = append(f.ops, "pause")
		} else {
			f.ops = append(f.ops, "unpause")
		}
	}

	return f.workers[workerId], nil
}

func (f *WorkerStore) ActivateWorkerListener(_ context.Context, _ uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.activations = append(f.activations, workerId)
	f.sessionIds = append(f.sessionIds, sessionId)
	f.ops = append(f.ops, "activate")
	f.active[workerId] = true
	f.listenerSessions[workerId] = sessionId

	return f.workers[workerId], nil
}

// DeactivateWorkerListener mirrors the repository fence: only the session recorded by the last
// activation may mark the worker inactive, and a superseded session gets pgx.ErrNoRows.
func (f *WorkerStore) DeactivateWorkerListener(_ context.Context, _ uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sessionIds = append(f.sessionIds, sessionId)
	f.ops = append(f.ops, "deactivate")

	if f.listenerSessions[workerId] != sessionId {
		return nil, fmt.Errorf("could not deactivate worker listener: %w", pgx.ErrNoRows)
	}

	f.active[workerId] = false

	return f.workers[workerId], nil
}

func (f *WorkerStore) UpdateWorkerHeartbeat(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.heartbeats++

	return nil
}

// UpdateWorkerHeartbeats records one bulk heartbeat write, the in-process host's liveness.
func (f *WorkerStore) UpdateWorkerHeartbeats(_ context.Context, workerIds []uuid.UUID, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.bulkHeartbeats = append(f.bulkHeartbeats, append([]uuid.UUID(nil), workerIds...))

	return nil
}

// BulkHeartbeats returns the worker id sets of every bulk heartbeat write, in order.
func (f *WorkerStore) BulkHeartbeats() [][]uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([][]uuid.UUID(nil), f.bulkHeartbeats...)
}

func (f *WorkerStore) UpsertWorkerLabels(_ context.Context, workerId uuid.UUID, opts []repository.UpsertWorkerLabelOpts) ([]*sqlcv1.WorkerLabel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.labels[workerId] = opts

	return nil, nil
}

// AddWorkerActions links actions without a budget, for tests that seed an action set.
func (f *WorkerStore) AddWorkerActions(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, actionIds []string) (int, error) {
	return f.AddWorkerActionsWithinBudget(ctx, tenantId, workerId, actionIds, -1)
}

func (f *WorkerStore) AddWorkerActionsWithinBudget(_ context.Context, _ uuid.UUID, workerId uuid.UUID, actionIds []string, maxNewLinks int64) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var fresh []string

	for _, id := range actionIds {
		if _, ok := f.actions[workerId][id]; !ok {
			fresh = append(fresh, id)
		}
	}

	if maxNewLinks >= 0 && int64(len(fresh)) > maxNewLinks {
		return 0, fmt.Errorf("delta would link %d new actions, the budget allows %d: %w", len(fresh), maxNewLinks, repository.ErrWorkerActionBudgetExceeded)
	}

	for _, id := range fresh {
		f.actions[workerId][id] = struct{}{}
	}

	return len(fresh), nil
}

func (f *WorkerStore) RemoveWorkerActions(_ context.Context, _ uuid.UUID, workerId uuid.UUID, actionIds []string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	removed := 0

	for _, id := range actionIds {
		if _, ok := f.actions[workerId][id]; ok {
			delete(f.actions[workerId], id)
			removed++
		}
	}

	return removed, nil
}

func (f *WorkerStore) CountOperatorWorkerActions(_ context.Context, tenantId uuid.UUID, operatorId uuid.UUID) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var n int64

	for workerId, actions := range f.actions {
		w, ok := f.workers[workerId]

		if !ok || w.TenantId != tenantId || w.OperatorId == nil || *w.OperatorId != operatorId {
			continue
		}

		n += int64(len(actions))
	}

	return n, nil
}

// ActionSet is the worker's linked actions, in no particular order.
func (f *WorkerStore) ActionSet(workerId uuid.UUID) []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]string, 0, len(f.actions[workerId]))

	for id := range f.actions[workerId] {
		out = append(out, id)
	}

	return out
}

// SessionLog returns the session ids passed to activation and deactivation calls, in order.
func (f *WorkerStore) SessionLog() []uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]uuid.UUID(nil), f.sessionIds...)
}

// Activations returns the workers that were activated, in order.
func (f *WorkerStore) Activations() []uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]uuid.UUID(nil), f.activations...)
}

// IsActive reports the worker's active flag as the store last left it.
func (f *WorkerStore) IsActive(workerId uuid.UUID) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.active[workerId]
}

// IsPaused reports the worker's paused flag as the store last left it.
func (f *WorkerStore) IsPaused(workerId uuid.UUID) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.paused[workerId]
}

// Created returns the worker creation options the store was called with, in order.
func (f *WorkerStore) Created() []*repository.CreateWorkerOpts {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]*repository.CreateWorkerOpts(nil), f.created...)
}

// Labels returns the labels last upserted for the worker.
func (f *WorkerStore) Labels(workerId uuid.UUID) []repository.UpsertWorkerLabelOpts {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.labels[workerId]
}

// Ops returns the lifecycle writes the store saw, in order: activate, pause, unpause and
// deactivate.
func (f *WorkerStore) Ops() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.ops...)
}

// Heartbeats counts the heartbeat writes that reached the store.
func (f *WorkerStore) Heartbeats() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.heartbeats
}

// DispatcherFor is the dispatcher the worker is pinned to.
func (f *WorkerStore) DispatcherFor(workerId uuid.UUID) uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.dispatchers[workerId]
}

// TenantStore serves tenant rows from memory, for hosts that name the tenant by id.
type TenantStore struct {
	mu      sync.Mutex
	tenants map[uuid.UUID]*sqlcv1.Tenant
}

// NewTenantStore seeds a store with the given tenants.
func NewTenantStore(tenants ...*sqlcv1.Tenant) *TenantStore {
	s := &TenantStore{tenants: map[uuid.UUID]*sqlcv1.Tenant{}}

	for _, tenant := range tenants {
		s.tenants[tenant.ID] = tenant
	}

	return s
}

func (f *TenantStore) GetTenantByID(_ context.Context, tenantId uuid.UUID) (*sqlcv1.Tenant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	tenant, ok := f.tenants[tenantId]

	if !ok {
		return nil, pgx.ErrNoRows
	}

	return tenant, nil
}

// WorkflowStore stands in for the admin service and the workflow repository a host puts
// workflows through: each put is recorded and becomes a version whose steps are the request's
// tasks, with the action ids stored as given.
type WorkflowStore struct {
	mu       sync.Mutex
	puts     []*v1contracts.CreateWorkflowVersionRequest
	versions map[uuid.UUID]*v1contracts.CreateWorkflowVersionRequest
	putErr   error
}

func NewWorkflowStore() *WorkflowStore {
	return &WorkflowStore{versions: map[uuid.UUID]*v1contracts.CreateWorkflowVersionRequest{}}
}

// PutWorkflow records the request and returns a fresh version id for it.
func (f *WorkflowStore) PutWorkflow(_ context.Context, req *v1contracts.CreateWorkflowVersionRequest) (*v1contracts.CreateWorkflowVersionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.putErr != nil {
		return nil, f.putErr
	}

	f.puts = append(f.puts, req)

	versionId := uuid.New()
	f.versions[versionId] = req

	return &v1contracts.CreateWorkflowVersionResponse{Id: versionId.String(), WorkflowId: uuid.NewString()}, nil
}

// ListStepsByWorkflowVersionId returns one step per task of the put request, plus the
// on-failure task; a DAG orchestrator step is never produced by the fake.
func (f *WorkflowStore) ListStepsByWorkflowVersionId(_ context.Context, _ uuid.UUID, versionId uuid.UUID) ([]*sqlcv1.ListStepsByWorkflowVersionIdsRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	req, ok := f.versions[versionId]

	if !ok {
		return nil, pgx.ErrNoRows
	}

	tasks := append([]*v1contracts.CreateTaskOpts(nil), req.Tasks...)

	if req.OnFailureTask != nil {
		tasks = append(tasks, req.OnFailureTask)
	}

	rows := make([]*sqlcv1.ListStepsByWorkflowVersionIdsRow, 0, len(tasks))

	for _, task := range tasks {
		rows = append(rows, &sqlcv1.ListStepsByWorkflowVersionIdsRow{ID: uuid.New(), ActionId: task.Action})
	}

	return rows, nil
}

// FailPut makes every later PutWorkflow fail.
func (f *WorkflowStore) FailPut(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.putErr = err
}

// Puts returns the requests put so far, in order.
func (f *WorkflowStore) Puts() []*v1contracts.CreateWorkflowVersionRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]*v1contracts.CreateWorkflowVersionRequest(nil), f.puts...)
}

// Dispatcher records session registrations, scheduler notifications and delegated calls.
type Dispatcher struct {
	mu sync.Mutex

	sessions []uuid.UUID
	// sessionIds records the session id each registration was keyed on, in order
	sessionIds []uuid.UUID
	// handlers records the handlers registered by handler-backed sessions, in order
	handlers  []operatorsvc.ActionHandler
	released  int
	notifies  []uuid.UUID
	fin       chan bool
	stepCalls []*contracts.StepActionEvent
	// sent records the messages a session sent through the stream handle
	sent    []proto.Message
	sendErr error
	// durableRegister records the first message the delegated durable stream received
	durableRegister *v1contracts.DurableTaskRequest
	durableErr      error

	durables          []*DurableInvocation
	registerDurableEr error
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{fin: make(chan bool)}
}

func (f *Dispatcher) AddOperatorStreamSession(workerId uuid.UUID, sessionId uuid.UUID, _ grpc.ServerStream, _ func(*contracts.AssignedAction) proto.Message) operatorsvc.StreamSession {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sessions = append(f.sessions, workerId)
	f.sessionIds = append(f.sessionIds, sessionId)

	return &streamSession{d: f}
}

func (f *Dispatcher) AddOperatorSession(workerId uuid.UUID, sessionId uuid.UUID, handler operatorsvc.ActionHandler) operatorsvc.HandlerSession {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sessions = append(f.sessions, workerId)
	f.sessionIds = append(f.sessionIds, sessionId)
	f.handlers = append(f.handlers, handler)

	return &handlerSession{d: f}
}

// streamSession is the session handle the fake dispatcher hands to a stream-backed session.
// Sent messages are recorded on the dispatcher.
type streamSession struct {
	d *Dispatcher
}

func (s *streamSession) Fin() <-chan bool { return s.d.fin }

func (s *streamSession) Send(_ context.Context, msg proto.Message) error {
	s.d.mu.Lock()
	defer s.d.mu.Unlock()

	if s.d.sendErr != nil {
		return s.d.sendErr
	}

	s.d.sent = append(s.d.sent, msg)

	return nil
}

func (s *streamSession) Release() {
	s.d.mu.Lock()
	defer s.d.mu.Unlock()
	s.d.released++
}

// handlerSession is the session handle for a handler-backed session.
type handlerSession struct {
	d *Dispatcher
}

func (s *handlerSession) Release() {
	s.d.mu.Lock()
	defer s.d.mu.Unlock()
	s.d.released++
}

func (f *Dispatcher) NotifyNewWorker(_ context.Context, _ *sqlcv1.Tenant, workerId uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.notifies = append(f.notifies, workerId)
}

func (f *Dispatcher) SendStepActionEvent(_ context.Context, req *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.stepCalls = append(f.stepCalls, req)

	return &contracts.ActionEventResponse{WorkerId: req.WorkerId}, nil
}

// DurableTask stands in for the dispatcher's own durable task stream handler: it reads the
// first message and records it.
func (f *Dispatcher) DurableTask(stream v1contracts.V1Dispatcher_DurableTaskServer) error {
	req, err := stream.Recv()

	f.mu.Lock()
	defer f.mu.Unlock()

	if err != nil {
		f.durableErr = err
		return err
	}

	f.durableRegister = req

	return nil
}

// DurableInvocation is one channel-backed durable task the service registered. Requests carries
// what the session sent, Responses is what the test sends back; the response channel is closed
// when the session's context ends or the test calls End, as the engine closes its side.
type DurableInvocation struct {
	ExternalId uuid.UUID
	Requests   chan *v1contracts.DurableTaskRequest
	Responses  chan *v1contracts.DurableTaskResponse

	end     chan struct{}
	endOnce sync.Once
}

// End closes the engine's side of the invocation, as the engine does when it tears the
// invocation down on its own.
func (inv *DurableInvocation) End() {
	inv.endOnce.Do(func() { close(inv.end) })
}

// RegisterDurableTask hands out a channel pair for the invocation and records it.
func (f *Dispatcher) RegisterDurableTask(ctx context.Context, externalId uuid.UUID) (chan<- *v1contracts.DurableTaskRequest, <-chan *v1contracts.DurableTaskResponse, error) {
	f.mu.Lock()

	if err := f.registerDurableEr; err != nil {
		f.mu.Unlock()
		return nil, nil, err
	}

	inv := &DurableInvocation{
		ExternalId: externalId,
		Requests:   make(chan *v1contracts.DurableTaskRequest, 64),
		Responses:  make(chan *v1contracts.DurableTaskResponse, 512),
		end:        make(chan struct{}),
	}

	f.durables = append(f.durables, inv)
	f.mu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
		case <-inv.end:
		}

		close(inv.Responses)
	}()

	return inv.Requests, inv.Responses, nil
}

// FailRegisterDurableTask makes the next RegisterDurableTask calls fail.
func (f *Dispatcher) FailRegisterDurableTask(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.registerDurableEr = err
}

// Durables returns the invocations registered so far, in order.
func (f *Dispatcher) Durables() []*DurableInvocation {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]*DurableInvocation(nil), f.durables...)
}

// Fin is the channel the dispatcher uses to ask a stream-backed session to hang up.
func (f *Dispatcher) Fin() chan bool { return f.fin }

// SetSendErr makes every later send on a stream-backed session fail.
func (f *Dispatcher) SetSendErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sendErr = err
}

// NotifyCount is the number of scheduler notifications the dispatcher received.
func (f *Dispatcher) NotifyCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.notifies)
}

// SessionIdLog returns the session ids registrations were keyed on, in order.
func (f *Dispatcher) SessionIdLog() []uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]uuid.UUID(nil), f.sessionIds...)
}

// SessionCount is the number of sessions registered so far.
func (f *Dispatcher) SessionCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.sessions)
}

// Handlers returns the handlers registered by handler-backed sessions, in order.
func (f *Dispatcher) Handlers() []operatorsvc.ActionHandler {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]operatorsvc.ActionHandler(nil), f.handlers...)
}

// ReleasedCount is the number of sessions that were released.
func (f *Dispatcher) ReleasedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.released
}

// StepCalls returns the step action events that reached the dispatcher, in order.
func (f *Dispatcher) StepCalls() []*contracts.StepActionEvent {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]*contracts.StepActionEvent(nil), f.stepCalls...)
}

// Sent returns the messages sessions sent through the stream handle, in order.
func (f *Dispatcher) Sent() []proto.Message {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]proto.Message(nil), f.sent...)
}

// AckedSequences lists the delta sequences acknowledged through the stream handle, in order.
func (f *Dispatcher) AckedSequences() []uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []uint64

	for _, msg := range f.sent {
		if resp, ok := msg.(*v1contracts.OperatorListenResponse); ok && resp.GetAck() != nil {
			out = append(out, resp.GetAck().Sequence)
		}
	}

	return out
}

// DurableRegister is the first message the delegated durable stream received.
func (f *Dispatcher) DurableRegister() *v1contracts.DurableTaskRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.durableRegister
}

// DurableErr is the error the delegated durable stream ended with.
func (f *Dispatcher) DurableErr() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.durableErr
}
