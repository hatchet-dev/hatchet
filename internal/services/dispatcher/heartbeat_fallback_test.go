package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type fakeHeartbeatRepo struct {
	v1.Repository
	workers    *fakeHeartbeatWorkers
	dispatcher *fakeHeartbeatDispatcher
}

func (r *fakeHeartbeatRepo) Workers() v1.WorkerRepository        { return r.workers }
func (r *fakeHeartbeatRepo) Dispatcher() v1.DispatcherRepository { return r.dispatcher }

type fakeHeartbeatWorkers struct {
	v1.WorkerRepository

	mu        sync.Mutex
	getFn     func(ctx context.Context, tenantId, workerId uuid.UUID) (*sqlcv1.GetWorkerForEngineRow, error)
	updateFn  func(ctx context.Context, tenantId, workerId uuid.UUID, lastHeartbeatAt time.Time) error
	flushFn   func(ctx context.Context, heartbeats []v1.WorkerHeartbeat) error
	callOrder *[]string
}

func (w *fakeHeartbeatWorkers) GetWorkerForEngine(ctx context.Context, tenantId, workerId uuid.UUID) (*sqlcv1.GetWorkerForEngineRow, error) {
	w.mu.Lock()
	fn := w.getFn
	w.mu.Unlock()

	if fn != nil {
		return fn(ctx, tenantId, workerId)
	}

	return &sqlcv1.GetWorkerForEngineRow{ID: workerId, TenantId: tenantId, IsActive: true}, nil
}

func (w *fakeHeartbeatWorkers) UpdateWorkerHeartbeat(ctx context.Context, tenantId, workerId uuid.UUID, lastHeartbeatAt time.Time) error {
	w.mu.Lock()
	fn := w.updateFn
	w.mu.Unlock()

	if fn != nil {
		return fn(ctx, tenantId, workerId, lastHeartbeatAt)
	}

	return nil
}

func (w *fakeHeartbeatWorkers) FlushWorkerHeartbeats(ctx context.Context, heartbeats []v1.WorkerHeartbeat) error {
	w.mu.Lock()
	fn := w.flushFn
	order := w.callOrder
	if order != nil {
		*order = append(*order, "flush")
	}
	w.mu.Unlock()

	if fn != nil {
		return fn(ctx, heartbeats)
	}

	return nil
}

type fakeHeartbeatDispatcher struct {
	v1.DispatcherRepository

	mu          sync.Mutex
	heartbeatFn func(ctx context.Context, dispatcherId uuid.UUID) error
	calls       int
	callOrder   *[]string
}

func (d *fakeHeartbeatDispatcher) Heartbeat(ctx context.Context, dispatcherId uuid.UUID) error {
	d.mu.Lock()
	d.calls++
	fn := d.heartbeatFn
	order := d.callOrder
	if order != nil {
		*order = append(*order, "dispatcher-heartbeat")
	}
	d.mu.Unlock()

	if fn != nil {
		return fn(ctx, dispatcherId)
	}

	return nil
}

func newTestHeartbeatDispatcher() (*DispatcherImpl, *fakeHeartbeatWorkers, *fakeHeartbeatDispatcher, *[]string) {
	order := &[]string{}
	workers := &fakeHeartbeatWorkers{callOrder: order}
	disp := &fakeHeartbeatDispatcher{callOrder: order}
	repo := &fakeHeartbeatRepo{workers: workers, dispatcher: disp}
	d := &DispatcherImpl{
		l:            zerologNop(),
		repov1:       repo,
		dispatcherId: uuid.New(),
	}

	return d, workers, disp, order
}

func ctxWithTenant(ctx context.Context, tenantId uuid.UUID) context.Context {
	return context.WithValue(ctx, "tenant", &sqlcv1.Tenant{ID: tenantId})
}

func TestHeartbeatBuffersOnDBUnavailableAndFlushesBeforeDispatcherHeartbeat(t *testing.T) {
	d, workers, disp, order := newTestHeartbeatDispatcher()

	tenantId := uuid.New()
	worker1 := uuid.New()
	worker2 := uuid.New()
	worker3 := uuid.New()

	// 1. GetWorkerForEngine fails with a connection error -> buffered, returns OK.
	workers.getFn = func(context.Context, uuid.UUID, uuid.UUID) (*sqlcv1.GetWorkerForEngineRow, error) {
		return nil, errors.New("dial tcp: connection refused")
	}

	before1 := time.Now().UTC()
	resp, err := d.Heartbeat(ctxWithTenant(context.Background(), tenantId), &contracts.HeartbeatRequest{
		WorkerId:    worker1.String(),
		HeartbeatAt: timestamppb.Now(),
	})
	after1 := time.Now().UTC()
	require.NoError(t, err)
	require.NotNil(t, resp)

	// 2. GetWorkerForEngine fails with a fairpool LimitError -> buffered, returns OK.
	workers.getFn = func(context.Context, uuid.UUID, uuid.UUID) (*sqlcv1.GetWorkerForEngineRow, error) {
		return nil, &fairpool.LimitError{Key: tenantId.String(), Limit: 5}
	}

	before2 := time.Now().UTC()
	resp, err = d.Heartbeat(ctxWithTenant(context.Background(), tenantId), &contracts.HeartbeatRequest{
		WorkerId:    worker2.String(),
		HeartbeatAt: timestamppb.Now(),
	})
	after2 := time.Now().UTC()
	require.NoError(t, err)
	require.NotNil(t, resp)

	// 3. Handler context cancelled by client deadline while UpdateWorkerHeartbeat hangs -> still buffered.
	workers.getFn = nil
	workers.updateFn = func(ctx context.Context, _, _ uuid.UUID, _ time.Time) error {
		return ctx.Err()
	}

	cancelledCtx, cancel := context.WithCancel(ctxWithTenant(context.Background(), tenantId))
	cancel()

	before3 := time.Now().UTC()
	resp, err = d.Heartbeat(cancelledCtx, &contracts.HeartbeatRequest{
		WorkerId:    worker3.String(),
		HeartbeatAt: timestamppb.Now(),
	})
	after3 := time.Now().UTC()
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Wait a bit before flushing so we prove the flushed timestamp is the received time, not the flush time.
	time.Sleep(25 * time.Millisecond)
	flushStartedAt := time.Now().UTC()

	var flushed []v1.WorkerHeartbeat
	workers.flushFn = func(_ context.Context, heartbeats []v1.WorkerHeartbeat) error {
		flushed = append([]v1.WorkerHeartbeat(nil), heartbeats...)
		return nil
	}

	d.updateHeartbeat(context.Background())

	require.Equal(t, []string{"flush", "dispatcher-heartbeat"}, *order)
	require.Equal(t, 1, disp.calls)
	require.Len(t, flushed, 3)

	byWorker := make(map[uuid.UUID]v1.WorkerHeartbeat, len(flushed))
	for _, hb := range flushed {
		byWorker[hb.WorkerId] = hb
	}

	for workerId, bounds := range map[uuid.UUID][2]time.Time{
		worker1: {before1, after1},
		worker2: {before2, after2},
		worker3: {before3, after3},
	} {
		hb, ok := byWorker[workerId]
		require.True(t, ok, "missing worker %s", workerId)
		require.Equal(t, tenantId, hb.TenantId)
		require.False(t, hb.LastHeartbeatAt.Before(bounds[0]), "timestamp predates receipt")
		require.False(t, hb.LastHeartbeatAt.After(bounds[1]), "timestamp postdates receipt")
		require.True(t, hb.LastHeartbeatAt.Before(flushStartedAt), "flushed timestamp should be received time, not flush time")
	}

	// Next tick has nothing pending: only the dispatcher heartbeat runs.
	*order = nil
	d.updateHeartbeat(context.Background())
	require.Equal(t, []string{"dispatcher-heartbeat"}, *order)
}

func TestHeartbeatNotFoundIsNotBuffered(t *testing.T) {
	d, workers, _, _ := newTestHeartbeatDispatcher()
	tenantId := uuid.New()
	workerId := uuid.New()

	workers.getFn = func(context.Context, uuid.UUID, uuid.UUID) (*sqlcv1.GetWorkerForEngineRow, error) {
		return nil, pgx.ErrNoRows
	}

	_, err := d.Heartbeat(ctxWithTenant(context.Background(), tenantId), &contracts.HeartbeatRequest{
		WorkerId:    workerId.String(),
		HeartbeatAt: timestamppb.Now(),
	})
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	require.Empty(t, d.pendingHeartbeats.snapshot())

	workers.getFn = nil
	workers.updateFn = func(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
		return fmt.Errorf("could not update worker heartbeat: %w", pgx.ErrNoRows)
	}

	_, err = d.Heartbeat(ctxWithTenant(context.Background(), tenantId), &contracts.HeartbeatRequest{
		WorkerId:    workerId.String(),
		HeartbeatAt: timestamppb.Now(),
	})
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	require.Empty(t, d.pendingHeartbeats.snapshot())
}

func TestUpdateHeartbeatDoesNotWriteDispatcherHeartbeatWhenFlushFails(t *testing.T) {
	d, workers, disp, order := newTestHeartbeatDispatcher()
	tenantId := uuid.New()
	workerId := uuid.New()

	workers.updateFn = func(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
		return errors.New("database unreachable")
	}

	_, err := d.Heartbeat(ctxWithTenant(context.Background(), tenantId), &contracts.HeartbeatRequest{
		WorkerId:    workerId.String(),
		HeartbeatAt: timestamppb.Now(),
	})
	require.NoError(t, err)

	workers.flushFn = func(context.Context, []v1.WorkerHeartbeat) error {
		return errors.New("still unreachable")
	}

	d.updateHeartbeat(context.Background())

	require.Equal(t, []string{"flush"}, *order)
	require.Zero(t, disp.calls, "dispatcher heartbeat must not be written when the pending flush fails")
	require.Len(t, d.pendingHeartbeats.snapshot(), 1, "pending heartbeat must remain in memory after a failed flush")

	// Once the database recovers, the preserved heartbeat flushes before the dispatcher heartbeat.
	var flushed []v1.WorkerHeartbeat
	workers.flushFn = func(_ context.Context, heartbeats []v1.WorkerHeartbeat) error {
		flushed = append([]v1.WorkerHeartbeat(nil), heartbeats...)
		return nil
	}

	*order = nil
	d.updateHeartbeat(context.Background())

	require.Equal(t, []string{"flush", "dispatcher-heartbeat"}, *order)
	require.Equal(t, 1, disp.calls)
	require.Len(t, flushed, 1)
	require.Equal(t, workerId, flushed[0].WorkerId)
	require.Empty(t, d.pendingHeartbeats.snapshot())
}

func TestFlushPreservesNewerHeartbeatReceivedDuringWrite(t *testing.T) {
	d, workers, _, _ := newTestHeartbeatDispatcher()
	tenantId := uuid.New()
	workerId := uuid.New()

	t1 := time.Now().UTC().Add(-2 * time.Second)
	t2 := time.Now().UTC()

	d.pendingHeartbeats.record(tenantId, workerId, t1)

	inFlush := make(chan struct{})
	releaseFlush := make(chan struct{})

	var batches [][]v1.WorkerHeartbeat
	workers.flushFn = func(_ context.Context, heartbeats []v1.WorkerHeartbeat) error {
		batches = append(batches, append([]v1.WorkerHeartbeat(nil), heartbeats...))
		if len(batches) == 1 {
			close(inFlush)
			<-releaseFlush
		}
		return nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		d.updateHeartbeat(context.Background())
	}()

	<-inFlush
	// A newer heartbeat arrives while the first flush is in flight, followed by an older one.
	d.pendingHeartbeats.record(tenantId, workerId, t2)
	d.pendingHeartbeats.record(tenantId, workerId, t1.Add(-time.Second))
	close(releaseFlush)
	<-done

	require.Len(t, batches, 1)
	require.Equal(t, t1, batches[0][0].LastHeartbeatAt)

	remaining := d.pendingHeartbeats.snapshot()
	require.Len(t, remaining, 1)
	require.Equal(t, t2, remaining[0].LastHeartbeatAt, "newer heartbeat received during flush must not be dropped")

	d.updateHeartbeat(context.Background())
	require.Len(t, batches, 2)
	require.Equal(t, t2, batches[1][0].LastHeartbeatAt)
	require.Empty(t, d.pendingHeartbeats.snapshot())
}

func TestPendingHeartbeatsIsolatesTenants(t *testing.T) {
	var pending pendingHeartbeats
	tenantA := uuid.New()
	tenantB := uuid.New()
	workerId := uuid.New()

	tA := time.Now().UTC().Add(-time.Second)
	tB := time.Now().UTC()

	pending.record(tenantA, workerId, tA)
	pending.record(tenantB, workerId, tB)

	batch := pending.snapshot()
	require.Len(t, batch, 2, "a heartbeat under another tenant must not clobber the owning tenant's entry")
}
