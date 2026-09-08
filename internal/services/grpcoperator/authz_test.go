package grpcoperator

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// fakeOperatorStore serves operators from memory and upserts by (tenant, name). Listen
// handlers authorize concurrently, so the store is safe for concurrent use.
type fakeOperatorStore struct {
	mu        sync.Mutex
	operators map[uuid.UUID]*sqlcv1.V1Operator
	getCalls  atomic.Int64
}

func (f *fakeOperatorStore) GetOperatorById(_ context.Context, operatorId uuid.UUID) (*sqlcv1.V1Operator, error) {
	f.getCalls.Add(1)

	f.mu.Lock()
	defer f.mu.Unlock()

	op, ok := f.operators[operatorId]

	if !ok {
		return nil, pgx.ErrNoRows
	}

	return op, nil
}

func (f *fakeOperatorStore) UpsertGRPCOperator(_ context.Context, tenantId uuid.UUID, name string) (*sqlcv1.V1Operator, error) {
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

// fakeWorkerStore keeps workers and their action sets in memory. Methods record their calls so
// tests can assert what reached the store. mu guards every field: the Listen handler and the
// test goroutine touch the store concurrently.
type fakeWorkerStore struct {
	mu sync.Mutex

	workers map[uuid.UUID]*sqlcv1.Worker
	actions map[uuid.UUID]map[string]struct{}

	created     []*repository.CreateWorkerOpts
	activations []uuid.UUID
	// active is the worker's isActive flag as the last activation or deactivation left it
	active map[uuid.UUID]bool
	// listenerSessions is the listener session id recorded on each worker by its last activation
	listenerSessions map[uuid.UUID]uuid.UUID
	// sessionIds records the session id passed to each activation and deactivation, in order
	sessionIds  []uuid.UUID
	heartbeats  int
	labels      map[uuid.UUID][]repository.UpsertWorkerLabelOpts
	dispatchers map[uuid.UUID]uuid.UUID
}

func newFakeWorkerStore() *fakeWorkerStore {
	return &fakeWorkerStore{
		workers:          map[uuid.UUID]*sqlcv1.Worker{},
		actions:          map[uuid.UUID]map[string]struct{}{},
		active:           map[uuid.UUID]bool{},
		listenerSessions: map[uuid.UUID]uuid.UUID{},
		labels:           map[uuid.UUID][]repository.UpsertWorkerLabelOpts{},
		dispatchers:      map[uuid.UUID]uuid.UUID{},
	}
}

func (f *fakeWorkerStore) add(w *sqlcv1.Worker) *sqlcv1.Worker {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.workers[w.ID] = w
	f.actions[w.ID] = map[string]struct{}{}

	return w
}

func (f *fakeWorkerStore) CreateNewWorker(_ context.Context, tenantId uuid.UUID, opts *repository.CreateWorkerOpts) (*sqlcv1.Worker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.created = append(f.created, opts)

	dispatcherId := opts.DispatcherId
	w := &sqlcv1.Worker{ID: uuid.New(), TenantId: tenantId, Name: opts.Name, OperatorId: opts.OperatorId, DispatcherId: &dispatcherId}
	f.workers[w.ID] = w
	f.actions[w.ID] = map[string]struct{}{}

	return w, nil
}

func (f *fakeWorkerStore) GetWorkerForEngine(_ context.Context, tenantId uuid.UUID, workerId uuid.UUID) (*sqlcv1.GetWorkerForEngineRow, error) {
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

func (f *fakeWorkerStore) UpdateWorker(_ context.Context, _ uuid.UUID, workerId uuid.UUID, opts *repository.UpdateWorkerOpts) (*sqlcv1.Worker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if opts.DispatcherId != nil {
		f.dispatchers[workerId] = *opts.DispatcherId
	}

	return f.workers[workerId], nil
}

func (f *fakeWorkerStore) ActivateWorkerListener(_ context.Context, _ uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.activations = append(f.activations, workerId)
	f.sessionIds = append(f.sessionIds, sessionId)
	f.active[workerId] = true
	f.listenerSessions[workerId] = sessionId

	return f.workers[workerId], nil
}

// DeactivateWorkerListener mirrors the repository fence: only the session recorded by the last
// activation may mark the worker inactive, and a superseded session gets pgx.ErrNoRows.
func (f *fakeWorkerStore) DeactivateWorkerListener(_ context.Context, _ uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sessionIds = append(f.sessionIds, sessionId)

	if f.listenerSessions[workerId] != sessionId {
		return nil, fmt.Errorf("could not deactivate worker listener: %w", pgx.ErrNoRows)
	}

	f.active[workerId] = false

	return f.workers[workerId], nil
}

func (f *fakeWorkerStore) UpdateWorkerHeartbeat(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.heartbeats++

	return nil
}

func (f *fakeWorkerStore) UpsertWorkerLabels(_ context.Context, workerId uuid.UUID, opts []repository.UpsertWorkerLabelOpts) ([]*sqlcv1.WorkerLabel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.labels[workerId] = opts

	return nil, nil
}

func (f *fakeWorkerStore) AddWorkerActions(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, actionIds []string) (int, error) {
	return f.AddWorkerActionsWithinBudget(ctx, tenantId, workerId, actionIds, -1)
}

func (f *fakeWorkerStore) AddWorkerActionsWithinBudget(_ context.Context, _ uuid.UUID, workerId uuid.UUID, actionIds []string, maxNewLinks int64) (int, error) {
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

func (f *fakeWorkerStore) RemoveWorkerActions(_ context.Context, _ uuid.UUID, workerId uuid.UUID, actionIds []string) (int, error) {
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

func (f *fakeWorkerStore) CountOperatorWorkerActions(_ context.Context, tenantId uuid.UUID, operatorId uuid.UUID) (int64, error) {
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

func (f *fakeWorkerStore) actionSet(workerId uuid.UUID) []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]string, 0, len(f.actions[workerId]))

	for id := range f.actions[workerId] {
		out = append(out, id)
	}

	return out
}

// sessionLog returns the session ids passed to activation and deactivation calls, in order.
func (f *fakeWorkerStore) sessionLog() []uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]uuid.UUID(nil), f.sessionIds...)
}

// isActive reports the worker's active flag as the store last left it.
func (f *fakeWorkerStore) isActive(workerId uuid.UUID) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.active[workerId]
}

// fakeDispatcher records session registrations and scheduler notifications.
type fakeDispatcher struct {
	mu sync.Mutex

	sessions []uuid.UUID
	// sessionIds records the session id each registration was keyed on, in order
	sessionIds []uuid.UUID
	released   int
	notifies   []uuid.UUID
	fin        chan bool
	stepCalls  []*contracts.StepActionEvent
	// sent records the messages Listen sent through the session handle
	sent    []proto.Message
	sendErr error
	// durableRegister records the first message the delegated durable stream received
	durableRegister *v1contracts.DurableTaskRequest
	durableErr      error
}

func newFakeDispatcher() *fakeDispatcher {
	return &fakeDispatcher{fin: make(chan bool)}
}

func (f *fakeDispatcher) AddOperatorStreamSession(workerId uuid.UUID, sessionId uuid.UUID, _ grpc.ServerStream, _ func(*contracts.AssignedAction) proto.Message) operatorStreamSession {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sessions = append(f.sessions, workerId)
	f.sessionIds = append(f.sessionIds, sessionId)

	return &fakeStreamSession{d: f}
}

// fakeStreamSession is the session handle the fake dispatcher hands to Listen. Sent messages
// are recorded on the dispatcher.
type fakeStreamSession struct {
	d *fakeDispatcher
}

func (s *fakeStreamSession) Fin() <-chan bool { return s.d.fin }

func (s *fakeStreamSession) Send(_ context.Context, msg proto.Message) error {
	s.d.mu.Lock()
	defer s.d.mu.Unlock()

	if s.d.sendErr != nil {
		return s.d.sendErr
	}

	s.d.sent = append(s.d.sent, msg)

	return nil
}

func (s *fakeStreamSession) Release() {
	s.d.mu.Lock()
	defer s.d.mu.Unlock()
	s.d.released++
}

func (f *fakeDispatcher) NotifyNewWorker(_ context.Context, _ *sqlcv1.Tenant, workerId uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.notifies = append(f.notifies, workerId)
}

func (f *fakeDispatcher) SendStepActionEvent(_ context.Context, req *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.stepCalls = append(f.stepCalls, req)

	return &contracts.ActionEventResponse{WorkerId: req.WorkerId}, nil
}

func (f *fakeDispatcher) DurableTask(stream v1contracts.V1Dispatcher_DurableTaskServer) error {
	req, err := stream.Recv()

	if err != nil {
		f.durableErr = err
		return err
	}

	f.durableRegister = req

	return nil
}

func (f *fakeDispatcher) notifyCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.notifies)
}

// sessionIdLog returns the session ids registrations were keyed on, in order.
func (f *fakeDispatcher) sessionIdLog() []uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]uuid.UUID(nil), f.sessionIds...)
}

func (f *fakeDispatcher) sessionCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.sessions)
}

func (f *fakeDispatcher) releasedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.released
}

type testService struct {
	*OperatorServiceImpl

	operators  *fakeOperatorStore
	workers    *fakeWorkerStore
	dispatcher *fakeDispatcher
}

func newTestService(t *testing.T, operators *fakeOperatorStore) *testService {
	t.Helper()

	if operators == nil {
		operators = &fakeOperatorStore{}
	}

	l := zerolog.Nop()
	workers := newFakeWorkerStore()
	d := newFakeDispatcher()

	svc := &OperatorServiceImpl{
		operators:      operators,
		workers:        workers,
		dispatcher:     d,
		dispatcherId:   uuid.New(),
		cache:          cache.New(time.Minute),
		l:              &l,
		v:              validator.NewDefaultValidator(),
		analytics:      analytics.NoOpAnalytics{},
		notifyInterval: defaultNotifyInterval,

		maxListenStreamsPerOperator: DefaultMaxListenStreamsPerOperator,
		maxActionsPerOperator:       DefaultMaxActionsPerOperator,
		listenStreams:               map[uuid.UUID]int{},
	}

	t.Cleanup(func() { _ = svc.Cleanup() })

	return &testService{OperatorServiceImpl: svc, operators: operators, workers: workers, dispatcher: d}
}

func tenantContext(tenant *sqlcv1.Tenant) context.Context {
	return context.WithValue(context.Background(), "tenant", tenant)
}

func operatorContext(tenant *sqlcv1.Tenant, operatorId string) context.Context {
	return metadata.NewIncomingContext(tenantContext(tenant), metadata.Pairs(OperatorIdMetadataKey, operatorId))
}

func TestAuthorizeOperator(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	otherTenant := &sqlcv1.Tenant{ID: uuid.New()}

	grpcOp := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "grpc-op", Kind: sqlcv1.V1OperatorKindGRPC}
	dagOp := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "dag-op", Kind: sqlcv1.V1OperatorKindDAG}

	newStore := func() *fakeOperatorStore {
		return &fakeOperatorStore{operators: map[uuid.UUID]*sqlcv1.V1Operator{
			grpcOp.ID: grpcOp,
			dagOp.ID:  dagOp,
		}}
	}

	cases := []struct {
		ctx      context.Context
		name     string
		wantCode codes.Code
	}{
		{
			name:     "missing tenant",
			ctx:      metadata.NewIncomingContext(context.Background(), metadata.Pairs(OperatorIdMetadataKey, grpcOp.ID.String())),
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "no incoming metadata",
			ctx:      tenantContext(tenant),
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "metadata without operator id",
			ctx:      metadata.NewIncomingContext(tenantContext(tenant), metadata.Pairs("other-key", "value")),
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "malformed operator id",
			ctx:      operatorContext(tenant, "not-a-uuid"),
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "unknown operator",
			ctx:      operatorContext(tenant, uuid.NewString()),
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "tenant mismatch",
			ctx:      operatorContext(otherTenant, grpcOp.ID.String()),
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "wrong kind",
			ctx:      operatorContext(tenant, dagOp.ID.String()),
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "authorized",
			ctx:      operatorContext(tenant, grpcOp.ID.String()),
			wantCode: codes.OK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t, newStore())

			op, err := svc.authorizeOperator(tc.ctx)

			if tc.wantCode == codes.OK {
				require.NoError(t, err)
				assert.Equal(t, grpcOp.ID, op.ID)
				return
			}

			require.Error(t, err)
			assert.Nil(t, op)
			assert.Equal(t, tc.wantCode, status.Code(err), err.Error())
		})
	}
}

func TestAuthorizeOperatorCachesLookups(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	grpcOp := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "grpc-op", Kind: sqlcv1.V1OperatorKindGRPC}

	store := &fakeOperatorStore{operators: map[uuid.UUID]*sqlcv1.V1Operator{grpcOp.ID: grpcOp}}
	svc := newTestService(t, store)

	ctx := operatorContext(tenant, grpcOp.ID.String())

	for i := 0; i < 3; i++ {
		op, err := svc.authorizeOperator(ctx)
		require.NoError(t, err)
		assert.Equal(t, grpcOp.ID, op.ID)
	}

	assert.Equal(t, int64(1), store.getCalls.Load(), "cache hit should avoid a second repository call")
}

func TestAuthorizeOperatorDoesNotCacheMisses(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	grpcOp := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "grpc-op", Kind: sqlcv1.V1OperatorKindGRPC}

	store := &fakeOperatorStore{operators: map[uuid.UUID]*sqlcv1.V1Operator{}}
	svc := newTestService(t, store)

	ctx := operatorContext(tenant, grpcOp.ID.String())

	_, err := svc.authorizeOperator(ctx)
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// the operator is created after the first call; the miss must not be sticky
	store.operators[grpcOp.ID] = grpcOp

	op, err := svc.authorizeOperator(ctx)
	require.NoError(t, err)
	assert.Equal(t, grpcOp.ID, op.ID)
	assert.Equal(t, int64(2), store.getCalls.Load())
}

func TestAuthorizeOperatorWorker(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	op := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "grpc-op", Kind: sqlcv1.V1OperatorKindGRPC}
	otherOp := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "other-op", Kind: sqlcv1.V1OperatorKindGRPC}

	svc := newTestService(t, nil)
	ownWorker := svc.workers.add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &op.ID})
	otherWorker := svc.workers.add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &otherOp.ID})
	sdkWorker := svc.workers.add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID})
	ctx := tenantContext(tenant)

	_, err := svc.authorizeOperatorWorker(ctx, op, "not-a-uuid")
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = svc.authorizeOperatorWorker(ctx, op, otherWorker.ID.String())
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	_, err = svc.authorizeOperatorWorker(ctx, op, sdkWorker.ID.String())
	assert.Equal(t, codes.PermissionDenied, status.Code(err), "a worker without an operator is not owned by anyone")

	_, err = svc.authorizeOperatorWorker(ctx, op, uuid.NewString())
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	w, err := svc.authorizeOperatorWorker(ctx, op, ownWorker.ID.String())
	require.NoError(t, err)
	assert.Equal(t, ownWorker.ID, w.ID)
}
