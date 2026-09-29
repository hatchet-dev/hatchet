//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type mockLeaseRepo struct {
	mock.Mock
}

func (m *mockLeaseRepo) ListQueues(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.V1Queue, error) {
	args := m.Called(ctx, tenantId)
	return args.Get(0).([]*sqlcv1.V1Queue), args.Error(1)
}

func (m *mockLeaseRepo) ListActiveWorkers(ctx context.Context, tenantId uuid.UUID) ([]*v1.ListActiveWorkersResult, error) {
	args := m.Called(ctx, tenantId)
	return args.Get(0).([]*v1.ListActiveWorkersResult), args.Error(1)
}

func (m *mockLeaseRepo) ListConcurrencyStrategies(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.V1StepConcurrency, error) {
	args := m.Called(ctx, tenantId)
	return args.Get(0).([]*sqlcv1.V1StepConcurrency), args.Error(1)
}

func (m *mockLeaseRepo) AcquireOrExtendLeases(ctx context.Context, tenantId uuid.UUID, kind sqlcv1.LeaseKind, resourceIds []string, existingLeases []*sqlcv1.Lease) ([]*sqlcv1.Lease, error) {
	args := m.Called(ctx, kind, resourceIds, existingLeases)
	return args.Get(0).([]*sqlcv1.Lease), args.Error(1)
}

func (m *mockLeaseRepo) RenewLeases(ctx context.Context, tenantId uuid.UUID, leases []*sqlcv1.Lease) ([]*sqlcv1.Lease, error) {
	args := m.Called(ctx, leases)
	return args.Get(0).([]*sqlcv1.Lease), args.Error(1)
}

func (m *mockLeaseRepo) ReleaseLeases(ctx context.Context, tenantId uuid.UUID, leases []*sqlcv1.Lease) error {
	args := m.Called(ctx, leases)
	return args.Error(0)
}

type fakeBatchQueueFactory struct {
	resources []*sqlcv1.ListDistinctBatchResourcesRow
}

func (f *fakeBatchQueueFactory) NewBatchQueue(uuid.UUID) v1repo.BatchQueueRepository {
	return &fakeBatchQueueRepo{resources: f.resources}
}

type fakeBatchQueueRepo struct {
	resources []*sqlcv1.ListDistinctBatchResourcesRow
}

func (f *fakeBatchQueueRepo) ListBatchResources(context.Context) ([]*sqlcv1.ListDistinctBatchResourcesRow, error) {
	return f.resources, nil
}

func (f *fakeBatchQueueRepo) ListBatchedQueueItems(context.Context, uuid.UUID, []int64, int32) ([]*sqlcv1.V1BatchedQueueItem, error) {
	return nil, nil
}

func (f *fakeBatchQueueRepo) GetBatchedQueueItemsByIds(context.Context, []int64) ([]*sqlcv1.V1BatchedQueueItem, error) {
	return nil, nil
}

func (f *fakeBatchQueueRepo) DeleteBatchedQueueItems(context.Context, []int64) error {
	return nil
}

func (f *fakeBatchQueueRepo) MoveBatchedQueueItems(context.Context, []int64) ([]*sqlcv1.MoveBatchedQueueItemsRow, error) {
	return nil, nil
}

func (f *fakeBatchQueueRepo) ListExistingBatchedQueueItemIds(context.Context, []int64) (map[int64]struct{}, error) {
	return map[int64]struct{}{}, nil
}

func (f *fakeBatchQueueRepo) CommitAssignments(context.Context, []*v1repo.BatchAssignment) ([]*v1repo.BatchAssignment, error) {
	return nil, nil
}

func (f *fakeBatchQueueRepo) ReserveAndCommitBatchRun(
	context.Context,
	uuid.UUID, uuid.UUID,
	string, string, string,
	int,
	[]*v1repo.BatchAssignment,
) (bool, []*v1repo.BatchAssignment, error) {
	return true, nil, nil
}

type fakeLeaseRepo struct {
	resourceSets [][]string
}

func (f *fakeLeaseRepo) ListQueues(context.Context, uuid.UUID) ([]*sqlcv1.V1Queue, error) {
	return nil, nil
}

func (f *fakeLeaseRepo) ListActiveWorkers(context.Context, uuid.UUID) ([]*v1.ListActiveWorkersResult, error) {
	return nil, nil
}

func (f *fakeLeaseRepo) GetActiveWorker(context.Context, uuid.UUID, uuid.UUID) (*v1.ListActiveWorkersResult, error) {
	return nil, nil
}

func (f *fakeLeaseRepo) ListConcurrencyStrategies(context.Context, uuid.UUID) ([]*sqlcv1.V1StepConcurrency, error) {
	return nil, nil
}

func (f *fakeLeaseRepo) GetConcurrencyStrategy(context.Context, uuid.UUID, int64) (*sqlcv1.V1StepConcurrency, error) {
	return nil, nil
}

func (f *fakeLeaseRepo) AcquireOrExtendLeases(_ context.Context, _ uuid.UUID, _ sqlcv1.LeaseKind, resourceIds []string, _ []*sqlcv1.Lease) ([]*sqlcv1.Lease, error) {
	f.resourceSets = append(f.resourceSets, resourceIds)
	leases := make([]*sqlcv1.Lease, len(resourceIds))
	for i, id := range resourceIds {
		leases[i] = &sqlcv1.Lease{ResourceId: id}
	}
	return leases, nil
}

func (f *fakeLeaseRepo) ReleaseLeases(context.Context, uuid.UUID, []*sqlcv1.Lease) error {
	return nil
}

type fakeSchedulerRepo struct {
	leaseRepo *fakeLeaseRepo
	batchRepo *fakeBatchQueueFactory
}

func (f *fakeSchedulerRepo) Optimistic() v1repo.OptimisticSchedulingRepository {
	return nil
}

func (f *fakeSchedulerRepo) Concurrency() v1repo.ConcurrencyRepository {
	return nil
}

func (f *fakeSchedulerRepo) Lease() v1repo.LeaseRepository {
	return f.leaseRepo
}

func (f *fakeSchedulerRepo) QueueFactory() v1repo.QueueFactoryRepository {
	return nil
}

func (f *fakeSchedulerRepo) BatchQueue() v1repo.BatchQueueFactoryRepository {
	return f.batchRepo
}

func (f *fakeSchedulerRepo) RateLimit() v1repo.RateLimitRepository {
	return nil
}

func (f *fakeSchedulerRepo) Assignment() v1repo.AssignmentRepository {
	return nil
}

func (m *mockLeaseRepo) GetActiveWorker(ctx context.Context, tenantId, workerId uuid.UUID) (*v1.ListActiveWorkersResult, error) {
	args := m.Called(ctx, tenantId, workerId)
	return args.Get(0).(*v1.ListActiveWorkersResult), args.Error(1)
}

func (m *mockLeaseRepo) GetConcurrencyStrategy(ctx context.Context, tenantId uuid.UUID, id int64) (*sqlcv1.V1StepConcurrency, error) {
	args := m.Called(ctx, tenantId, id)
	return args.Get(0).(*sqlcv1.V1StepConcurrency), args.Error(1)
}

func TestLeaseManager_AcquireWorkerLeases(t *testing.T) {
	l := zerolog.Nop()
	tenantId := uuid.UUID{}
	mockLeaseRepo := &mockLeaseRepo{}
	leaseManager := &LeaseManager{
		lr:       mockLeaseRepo,
		conf:     &sharedConfig{l: &l},
		tenantId: tenantId,
	}

	mockWorkers := []*v1.ListActiveWorkersResult{
		{ID: uuid.New(), Labels: nil},
		{ID: uuid.New(), Labels: nil},
	}
	mockLeases := []*sqlcv1.Lease{
		{ID: 1, ResourceId: "worker-1"},
		{ID: 2, ResourceId: "worker-2"},
	}

	mockLeaseRepo.On("ListActiveWorkers", mock.Anything, tenantId).Return(mockWorkers, nil)
	mockLeaseRepo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindWORKER, mock.Anything, mock.Anything).Return(mockLeases, nil)

	err := leaseManager.acquireWorkerLeases(context.Background())
	assert.NoError(t, err)
	assert.Len(t, leaseManager.workerLeases, 2)
}

func TestLeaseManager_AcquireQueueLeases(t *testing.T) {
	l := zerolog.Nop()
	tenantId := uuid.UUID{}
	mockLeaseRepo := &mockLeaseRepo{}
	leaseManager := &LeaseManager{
		lr:       mockLeaseRepo,
		conf:     &sharedConfig{l: &l},
		tenantId: tenantId,
	}

	mockQueues := []*sqlcv1.V1Queue{
		{Name: "queue-1"},
		{Name: "queue-2"},
	}
	mockLeases := []*sqlcv1.Lease{
		{ID: 1, ResourceId: "queue-1"},
		{ID: 2, ResourceId: "queue-2"},
	}

	mockLeaseRepo.On("ListQueues", mock.Anything, tenantId).Return(mockQueues, nil)
	mockLeaseRepo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindQUEUE, mock.Anything, mock.Anything).Return(mockLeases, nil)

	err := leaseManager.acquireQueueLeases(context.Background())
	assert.NoError(t, err)
	assert.Len(t, leaseManager.queueLeases, 2)
}

func TestLeaseManager_SendWorkerIds(t *testing.T) {
	tenantId := uuid.UUID{}
	// buffered so the non-blocking send in sendWorkerIds cannot race the
	// receiver and drop the message
	workersCh := make(notifierCh[*v1.ListActiveWorkersResult], 1)
	leaseManager := &LeaseManager{
		tenantId:  tenantId,
		workersCh: workersCh,
	}

	mockWorkers := []*v1.ListActiveWorkersResult{
		{ID: uuid.New(), Labels: nil},
	}

	leaseManager.sendWorkerIds(mockWorkers, false)

	result := <-workersCh
	assert.Equal(t, mockWorkers, result.items)
}

func TestLeaseManager_SendQueues(t *testing.T) {
	tenantId := uuid.UUID{}
	// buffered so the non-blocking send in sendQueues cannot race the
	// receiver and drop the message
	queuesCh := make(notifierCh[string], 1)
	leaseManager := &LeaseManager{
		tenantId: tenantId,
		queuesCh: queuesCh,
	}

	mockQueues := []string{"queue-1", "queue-2"}

	leaseManager.sendQueues(mockQueues, false)

	result := <-queuesCh
	assert.Equal(t, mockQueues, result.items)
}

func TestLeaseManager_AcquireWorkersBeforeListenerReady(t *testing.T) {
	tenantId := uuid.UUID{}
	workersCh := make(notifierCh[*v1.ListActiveWorkersResult])
	leaseManager := &LeaseManager{
		tenantId:  tenantId,
		workersCh: workersCh,
	}

	mockWorkers1 := []*v1.ListActiveWorkersResult{
		{ID: uuid.New(), Labels: nil},
	}
	mockWorkers2 := []*v1.ListActiveWorkersResult{
		{ID: uuid.New(), Labels: nil},
		{ID: uuid.New(), Labels: nil},
	}

	// Send workers before listener is ready
	go leaseManager.sendWorkerIds(mockWorkers1, false)
	time.Sleep(100 * time.Millisecond)
	resultCh := make(chan []*v1.ListActiveWorkersResult)
	go func() {
		msg := <-workersCh
		resultCh <- msg.items
	}()
	time.Sleep(100 * time.Millisecond)
	go leaseManager.sendWorkerIds(mockWorkers2, false)
	time.Sleep(100 * time.Millisecond)

	// Ensure only the latest workers are sent over the channel
	result := <-resultCh
	assert.Equal(t, mockWorkers2, result)
	assert.Len(t, workersCh, 0) // Ensure no additional workers are left in the channel
}

func TestAcquireBatchLeasesUsesStepLevelResourceIds(t *testing.T) {
	tenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	stepID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	otherStepID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

	resources := []*sqlcv1.ListDistinctBatchResourcesRow{
		{StepID: stepID, BatchKey: "key-1"},
		{StepID: stepID, BatchKey: "key-2"},
		{StepID: otherStepID, BatchKey: "key-3"},
	}

	leaseRepo := &fakeLeaseRepo{}
	repo := &fakeSchedulerRepo{
		leaseRepo: leaseRepo,
		batchRepo: &fakeBatchQueueFactory{resources: resources},
	}

	logger := zerolog.New(io.Discard)
	lm, _, _, _, batchesCh := newLeaseManager(&sharedConfig{repo: repo, l: &logger}, tenantID)

	received := make(chan []*sqlcv1.ListDistinctBatchResourcesRow, 1)
	ready := make(chan struct{})
	go func() {
		close(ready)
		if rows, ok := <-batchesCh; ok {
			received <- rows
		}
	}()

	<-ready

	require.NoError(t, lm.acquireBatchLeases(context.Background()))

	select {
	case rows := <-received:
		require.Len(t, rows, 3, "all batch keys for leased steps should propagate")
	case <-time.After(1 * time.Second):
		t.Fatal("expected batch resources to be sent")
	}

	require.Len(t, leaseRepo.resourceSets, 1)
	acquired := leaseRepo.resourceSets[0]
	require.ElementsMatch(t, []string{
		stepID.String(),
		otherStepID.String(),
	}, acquired, "leases should be requested per step_id only")
}

// leaseTestHarness wires a LeaseManager over a mock lease repository. Its context bounds every
// wait, the poll release channel is closed on every exit so a failing assertion cannot strand a
// poll that is parked inside a mock, and every goroutine started through spawn is joined during
// cleanup on success and failure paths alike.
type leaseTestHarness struct {
	t           *testing.T
	ctx         context.Context
	lm          *LeaseManager
	repo        *mockLeaseRepo
	release     chan struct{}
	releaseOnce sync.Once
	goroutines  sync.WaitGroup
}

func newLeaseTestHarness(t *testing.T, tenantId uuid.UUID) *leaseTestHarness {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	h := &leaseTestHarness{
		t:       t,
		ctx:     ctx,
		repo:    &mockLeaseRepo{},
		release: make(chan struct{}),
	}

	// cleanups run in reverse registration order: first release and cancel everything, then join
	// every spawned goroutine with a wait that does not depend on the already cancelled context
	t.Cleanup(func() {
		joined := make(chan struct{})

		go func() {
			h.goroutines.Wait()
			close(joined)
		}()

		select {
		case <-joined:
		case <-time.After(5 * time.Second):
			t.Errorf("spawned goroutines did not finish after release and cancellation")
		}
	})

	t.Cleanup(func() {
		h.releasePoll()
		cancel()
	})

	l := zerolog.Nop()

	h.lm = &LeaseManager{
		lr:       h.repo,
		conf:     &sharedConfig{l: &l, repo: &fakeSchedulerRepo{batchRepo: &fakeBatchQueueFactory{}}},
		l:        l,
		tenantId: tenantId,
		// buffered so the non-blocking sends from the poll and the drain are both kept
		workersCh:           make(notifierCh[*v1.ListActiveWorkersResult], 4),
		queuesCh:            make(notifierCh[string], 4),
		concurrencyLeasesCh: make(notifierCh[*sqlcv1.V1StepConcurrency], 4),
		batchesCh:           make(chan []*sqlcv1.ListDistinctBatchResourcesRow, 1),
	}

	return h
}

func (h *leaseTestHarness) releasePoll() {
	h.releaseOnce.Do(func() { close(h.release) })
}

// holdPoll is a mock hook for a list call: it reports that the poll holds its lease mutex and then
// parks until the poll is released or the test context ends.
func (h *leaseTestHarness) holdPoll(polling chan<- struct{}) func(mock.Arguments) {
	return func(mock.Arguments) {
		select {
		case polling <- struct{}{}:
		case <-h.ctx.Done():
			return
		}

		select {
		case <-h.release:
		case <-h.ctx.Done():
		}
	}
}

// spawn runs f on a goroutine that the harness joins during cleanup and returns a channel that
// closes when f returns.
func (h *leaseTestHarness) spawn(f func()) <-chan struct{} {
	done := make(chan struct{})

	h.goroutines.Add(1)

	go func() {
		defer h.goroutines.Done()
		defer close(done)
		f()
	}()

	return done
}

// startPoll runs acquireAllLeases on a spawned goroutine and returns a channel that closes when it
// finishes.
func (h *leaseTestHarness) startPoll() <-chan struct{} {
	return h.spawn(func() { h.lm.acquireAllLeases(h.ctx) })
}

func (h *leaseTestHarness) waitSignal(ch <-chan struct{}, what string) {
	h.t.Helper()

	select {
	case <-ch:
	case <-h.ctx.Done():
		h.t.Fatalf("timed out waiting for %s", what)
	}
}

func recvLeaseMsg[T any](h *leaseTestHarness, ch notifierCh[T], what string) notifierMsg[T] {
	h.t.Helper()

	select {
	case msg := <-ch:
		return msg
	case <-h.ctx.Done():
		h.t.Fatalf("timed out waiting for %s", what)
		return notifierMsg[T]{}
	}
}

func expectEmptyLists(h *leaseTestHarness, except string) {
	for _, list := range []struct {
		method string
		empty  any
	}{
		{"ListActiveWorkers", []*v1.ListActiveWorkersResult{}},
		{"ListQueues", []*sqlcv1.V1Queue{}},
		{"ListConcurrencyStrategies", []*sqlcv1.V1StepConcurrency{}},
	} {
		if list.method != except {
			h.repo.On(list.method, mock.Anything, h.lm.tenantId).Return(list.empty, nil)
		}
	}
}

// TestLeaseManager_NotifyDuringPollIsLeasedAfterPoll checks that a notification which arrives
// while the periodic poll holds the lease mutex for its kind is leased once that poll finishes,
// without waiting for another notification or poll. The poll runs the real acquireAllLeases with
// a mock whose list call parks while the mutex is held.
func TestLeaseManager_NotifyDuringPollIsLeasedAfterPoll(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()
	worker := &v1.ListActiveWorkersResult{ID: workerId}
	strategy := &sqlcv1.V1StepConcurrency{ID: 42}

	cases := []struct {
		name   string
		list   string
		setup  func(h *leaseTestHarness)
		notify func(h *leaseTestHarness) error
		verify func(t *testing.T, h *leaseTestHarness)
	}{
		{
			name: "concurrency strategy",
			list: "ListConcurrencyStrategies",
			setup: func(h *leaseTestHarness) {
				h.repo.On("GetConcurrencyStrategy", mock.Anything, tenantId, strategy.ID).Return(strategy, nil)
				h.repo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindCONCURRENCYSTRATEGY, []string{"42"}, mock.Anything).
					Return([]*sqlcv1.Lease{{ID: 1, ResourceId: "42"}}, nil)
			},
			notify: func(h *leaseTestHarness) error {
				return h.lm.notifyNewConcurrencyStrategy(h.ctx, strategy.ID)
			},
			verify: func(t *testing.T, h *leaseTestHarness) {
				require.Len(t, h.lm.concurrencyLeases, 1)
				assert.Equal(t, "42", h.lm.concurrencyLeases[0].ResourceId)

				// the poll's full refresh is sent while it holds the mutex, so it lands first
				full := recvLeaseMsg(h, h.lm.concurrencyLeasesCh, "full refresh")
				assert.False(t, full.isIncremental)
				assert.Empty(t, full.items)

				incr := recvLeaseMsg(h, h.lm.concurrencyLeasesCh, "incremental message")
				assert.True(t, incr.isIncremental)
				assert.Equal(t, []*sqlcv1.V1StepConcurrency{strategy}, incr.items)
			},
		},
		{
			name: "queue",
			list: "ListQueues",
			setup: func(h *leaseTestHarness) {
				h.repo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindQUEUE, []string{"queue-1"}, mock.Anything).
					Return([]*sqlcv1.Lease{{ID: 1, ResourceId: "queue-1"}}, nil)
			},
			notify: func(h *leaseTestHarness) error {
				return h.lm.notifyNewQueue(h.ctx, "queue-1")
			},
			verify: func(t *testing.T, h *leaseTestHarness) {
				require.Len(t, h.lm.queueLeases, 1)
				assert.Equal(t, "queue-1", h.lm.queueLeases[0].ResourceId)

				full := recvLeaseMsg(h, h.lm.queuesCh, "full refresh")
				assert.False(t, full.isIncremental)
				assert.Empty(t, full.items)

				incr := recvLeaseMsg(h, h.lm.queuesCh, "incremental message")
				assert.True(t, incr.isIncremental)
				assert.Equal(t, []string{"queue-1"}, incr.items)
			},
		},
		{
			name: "worker",
			list: "ListActiveWorkers",
			setup: func(h *leaseTestHarness) {
				h.repo.On("GetActiveWorker", mock.Anything, tenantId, workerId).Return(worker, nil)
				h.repo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindWORKER, []string{workerId.String()}, mock.Anything).
					Return([]*sqlcv1.Lease{{ID: 1, ResourceId: workerId.String()}}, nil)
			},
			notify: func(h *leaseTestHarness) error {
				return h.lm.notifyNewWorker(h.ctx, workerId)
			},
			verify: func(t *testing.T, h *leaseTestHarness) {
				require.Len(t, h.lm.workerLeases, 1)
				assert.Equal(t, workerId.String(), h.lm.workerLeases[0].ResourceId)

				full := recvLeaseMsg(h, h.lm.workersCh, "full refresh")
				assert.False(t, full.isIncremental)
				assert.Empty(t, full.items)

				incr := recvLeaseMsg(h, h.lm.workersCh, "incremental message")
				assert.True(t, incr.isIncremental)
				assert.Equal(t, []*v1.ListActiveWorkersResult{worker}, incr.items)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newLeaseTestHarness(t, tenantId)
			polling := make(chan struct{})

			var listEmpty any

			switch tc.list {
			case "ListActiveWorkers":
				listEmpty = []*v1.ListActiveWorkersResult{}
			case "ListQueues":
				listEmpty = []*sqlcv1.V1Queue{}
			default:
				listEmpty = []*sqlcv1.V1StepConcurrency{}
			}

			h.repo.On(tc.list, mock.Anything, tenantId).Run(h.holdPoll(polling)).Return(listEmpty, nil)
			expectEmptyLists(h, tc.list)
			tc.setup(h)

			pollDone := h.startPoll()
			h.waitSignal(polling, "the poll to hold its lease mutex")

			// the notification returns at once and nothing is leased while the poll holds the mutex
			require.NoError(t, tc.notify(h))
			h.repo.AssertNotCalled(t, "AcquireOrExtendLeases", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

			h.releasePoll()
			h.waitSignal(pollDone, "the poll to finish")

			tc.verify(t, h)
			h.repo.AssertExpectations(t)
		})
	}
}

// TestLeaseManager_DuplicateNotifiesDuringPollLeaseOnce checks that a burst of notifications for
// one cold strategy while the poll holds the mutex collapses to a single pending entry, one lookup
// and one lease acquisition, and that later notifications for a leased strategy touch nothing.
func TestLeaseManager_DuplicateNotifiesDuringPollLeaseOnce(t *testing.T) {
	tenantId := uuid.New()
	strategy := &sqlcv1.V1StepConcurrency{ID: 42}

	h := newLeaseTestHarness(t, tenantId)
	polling := make(chan struct{})

	h.repo.On("ListConcurrencyStrategies", mock.Anything, tenantId).Run(h.holdPoll(polling)).Return([]*sqlcv1.V1StepConcurrency{}, nil)
	expectEmptyLists(h, "ListConcurrencyStrategies")
	h.repo.On("GetConcurrencyStrategy", mock.Anything, tenantId, strategy.ID).Return(strategy, nil).Once()
	h.repo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindCONCURRENCYSTRATEGY, []string{"42"}, mock.Anything).
		Return([]*sqlcv1.Lease{{ID: 1, ResourceId: "42"}}, nil).Once()

	pollDone := h.startPoll()
	h.waitSignal(polling, "the poll to hold its lease mutex")

	const burst = 8

	for i := 0; i < burst; i++ {
		require.NoError(t, h.lm.notifyNewConcurrencyStrategy(h.ctx, strategy.ID))
	}

	assert.Equal(t, 1, h.lm.pendingConcurrency.size(), "duplicate notifications must collapse to one pending entry")

	h.releasePoll()
	h.waitSignal(pollDone, "the poll to finish")

	assert.Equal(t, 0, h.lm.pendingConcurrency.size())
	require.Len(t, h.lm.concurrencyLeases, 1)
	h.repo.AssertNumberOfCalls(t, "GetConcurrencyStrategy", 1)
	h.repo.AssertNumberOfCalls(t, "AcquireOrExtendLeases", 1)

	full := recvLeaseMsg(h, h.lm.concurrencyLeasesCh, "full refresh")
	assert.False(t, full.isIncremental)

	incr := recvLeaseMsg(h, h.lm.concurrencyLeasesCh, "incremental message")
	assert.True(t, incr.isIncremental)
	assert.Equal(t, []*sqlcv1.V1StepConcurrency{strategy}, incr.items)

	// the strategy is leased, so further notifications short-circuit without a database call
	require.NoError(t, h.lm.notifyNewConcurrencyStrategy(h.ctx, strategy.ID))
	h.repo.AssertNumberOfCalls(t, "GetConcurrencyStrategy", 1)
	h.repo.AssertNumberOfCalls(t, "AcquireOrExtendLeases", 1)
	assert.Empty(t, h.lm.concurrencyLeasesCh)

	h.repo.AssertExpectations(t)
}

// TestLeaseManager_PendingNotifiesAreBounded checks that the pending set holds at most
// pendingLeaseCapacity distinct resources, that a further one is dropped without an error, and
// that the retained ones are leased with one acquisition once the poll releases the mutex.
func TestLeaseManager_PendingNotifiesAreBounded(t *testing.T) {
	tenantId := uuid.New()

	h := newLeaseTestHarness(t, tenantId)
	polling := make(chan struct{})

	leases := make([]*sqlcv1.Lease, 0, pendingLeaseCapacity)

	for i := 1; i <= pendingLeaseCapacity; i++ {
		leases = append(leases, &sqlcv1.Lease{ID: int64(i), ResourceId: fmt.Sprintf("%d", i)})
	}

	h.repo.On("ListConcurrencyStrategies", mock.Anything, tenantId).Run(h.holdPoll(polling)).Return([]*sqlcv1.V1StepConcurrency{}, nil)
	expectEmptyLists(h, "ListConcurrencyStrategies")
	h.repo.On("GetConcurrencyStrategy", mock.Anything, tenantId, mock.Anything).Return(&sqlcv1.V1StepConcurrency{}, nil)
	h.repo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindCONCURRENCYSTRATEGY, mock.MatchedBy(func(ids []string) bool {
		return len(ids) == pendingLeaseCapacity
	}), mock.Anything).Return(leases, nil).Once()

	pollDone := h.startPoll()
	h.waitSignal(polling, "the poll to hold its lease mutex")

	for i := 1; i <= pendingLeaseCapacity; i++ {
		require.NoError(t, h.lm.notifyNewConcurrencyStrategy(h.ctx, int64(i)))
	}

	require.Equal(t, pendingLeaseCapacity, h.lm.pendingConcurrency.size())

	overflow := int64(pendingLeaseCapacity + 1)
	require.NoError(t, h.lm.notifyNewConcurrencyStrategy(h.ctx, overflow))
	assert.Equal(t, pendingLeaseCapacity, h.lm.pendingConcurrency.size(), "a hint beyond capacity is dropped")

	h.releasePoll()
	h.waitSignal(pollDone, "the poll to finish")

	assert.Equal(t, 0, h.lm.pendingConcurrency.size())
	assert.Len(t, h.lm.concurrencyLeases, pendingLeaseCapacity)
	h.repo.AssertNumberOfCalls(t, "GetConcurrencyStrategy", pendingLeaseCapacity)
	h.repo.AssertNotCalled(t, "GetConcurrencyStrategy", mock.Anything, tenantId, overflow)
	h.repo.AssertNumberOfCalls(t, "AcquireOrExtendLeases", 1)

	full := recvLeaseMsg(h, h.lm.concurrencyLeasesCh, "full refresh")
	assert.False(t, full.isIncremental)

	incr := recvLeaseMsg(h, h.lm.concurrencyLeasesCh, "incremental message")
	assert.True(t, incr.isIncremental)
	assert.Len(t, incr.items, pendingLeaseCapacity)

	h.repo.AssertExpectations(t)
}

// TestLeaseManager_PendingBacklogDoesNotDelayPollOrCleanup checks that a backlog of pending
// notifications whose lookups exhaust their database budget holds the lease mutex for at most one
// drain pass, so a following poll and cleanup both complete inside their own five second budgets.
func TestLeaseManager_PendingBacklogDoesNotDelayPollOrCleanup(t *testing.T) {
	tenantId := uuid.New()

	h := newLeaseTestHarness(t, tenantId)
	polling := make(chan struct{})
	lookupStarted := make(chan struct{})

	var lookupOnce sync.Once

	h.repo.On("ListConcurrencyStrategies", mock.Anything, tenantId).Run(h.holdPoll(polling)).Return([]*sqlcv1.V1StepConcurrency{}, nil).Once()
	h.repo.On("ListConcurrencyStrategies", mock.Anything, tenantId).Return([]*sqlcv1.V1StepConcurrency{}, nil)
	expectEmptyLists(h, "ListConcurrencyStrategies")
	// every lookup burns the whole pass budget
	h.repo.On("GetConcurrencyStrategy", mock.Anything, tenantId, mock.Anything).
		Run(func(args mock.Arguments) {
			lookupOnce.Do(func() { close(lookupStarted) })

			select {
			case <-args.Get(0).(context.Context).Done():
			case <-h.ctx.Done():
			}
		}).
		Return((*sqlcv1.V1StepConcurrency)(nil), context.DeadlineExceeded)
	h.repo.On("ReleaseLeases", mock.Anything, mock.Anything).Return(nil)

	pollDone := h.startPoll()
	h.waitSignal(polling, "the poll to hold its lease mutex")

	const backlog = 8

	for i := 1; i <= backlog; i++ {
		require.NoError(t, h.lm.notifyNewConcurrencyStrategy(h.ctx, int64(i)))
	}

	require.Equal(t, backlog, h.lm.pendingConcurrency.size())

	h.releasePoll()
	h.waitSignal(lookupStarted, "the drain pass to start its lookups")

	// the next poll contends with the drain pass and must still renew inside its budget
	renewCtx, cancelRenew := context.WithTimeout(h.ctx, 5*time.Second)
	defer cancelRenew()

	renewStart := time.Now()
	renewErr := make(chan error, 1)

	h.spawn(func() { renewErr <- h.lm.acquireConcurrencyLeases(renewCtx) })

	select {
	case err := <-renewErr:
		require.NoError(t, err, "renewal ran with an expired context")
	case <-h.ctx.Done():
		t.Fatal("timed out waiting for renewal")
	}

	renewElapsed := time.Since(renewStart)
	assert.Less(t, renewElapsed, 3*time.Second, "renewal waited longer than the pass in progress")

	h.waitSignal(pollDone, "the poll and its drain to finish")
	assert.Equal(t, 0, h.lm.pendingConcurrency.size(), "a failed pass drops its hints instead of retrying them")

	cleanupCtx, cancelCleanup := context.WithTimeout(h.ctx, 5*time.Second)
	defer cancelCleanup()

	cleanupStart := time.Now()
	require.NoError(t, h.lm.cleanup(cleanupCtx))
	assert.Less(t, time.Since(cleanupStart), 3*time.Second)

	// nothing is serviced after cleanup
	require.NoError(t, h.lm.notifyNewConcurrencyStrategy(h.ctx, 99))
	h.repo.AssertNotCalled(t, "GetConcurrencyStrategy", mock.Anything, tenantId, int64(99))

	t.Logf("renewal completed in %s behind a backlog of %d pending lookups", renewElapsed, backlog)
}

// TestLeaseManager_EmptyRefreshClearsLeaseCache checks that a poll which finds no resources drops
// the leases it just released from the local cache, so a later notification for a resource that
// became active again acquires a fresh lease instead of short-circuiting on a stale entry.
func TestLeaseManager_EmptyRefreshClearsLeaseCache(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()
	worker := &v1.ListActiveWorkersResult{ID: workerId}
	strategy := &sqlcv1.V1StepConcurrency{ID: 42}

	cases := []struct {
		name   string
		setup  func(h *leaseTestHarness) *sqlcv1.Lease
		poll   func(h *leaseTestHarness) error
		notify func(h *leaseTestHarness) error
		verify func(t *testing.T, h *leaseTestHarness)
	}{
		{
			name: "concurrency strategy",
			setup: func(h *leaseTestHarness) *sqlcv1.Lease {
				lease := &sqlcv1.Lease{ID: 1, ResourceId: "42"}
				h.lm.concurrencyLeases = []*sqlcv1.Lease{lease}
				h.repo.On("ListConcurrencyStrategies", mock.Anything, tenantId).Return([]*sqlcv1.V1StepConcurrency{}, nil)
				h.repo.On("GetConcurrencyStrategy", mock.Anything, tenantId, strategy.ID).Return(strategy, nil).Once()
				h.repo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindCONCURRENCYSTRATEGY, []string{"42"}, mock.Anything).
					Return([]*sqlcv1.Lease{{ID: 2, ResourceId: "42"}}, nil).Once()
				return lease
			},
			poll: func(h *leaseTestHarness) error { return h.lm.acquireConcurrencyLeases(h.ctx) },
			notify: func(h *leaseTestHarness) error {
				return h.lm.notifyNewConcurrencyStrategy(h.ctx, strategy.ID)
			},
			verify: func(t *testing.T, h *leaseTestHarness) {
				require.Len(t, h.lm.concurrencyLeases, 1)
				assert.Equal(t, int64(2), h.lm.concurrencyLeases[0].ID)

				full := recvLeaseMsg(h, h.lm.concurrencyLeasesCh, "full refresh")
				assert.False(t, full.isIncremental)
				assert.Empty(t, full.items)

				incr := recvLeaseMsg(h, h.lm.concurrencyLeasesCh, "incremental message")
				assert.True(t, incr.isIncremental)
				assert.Equal(t, []*sqlcv1.V1StepConcurrency{strategy}, incr.items)
			},
		},
		{
			name: "queue",
			setup: func(h *leaseTestHarness) *sqlcv1.Lease {
				lease := &sqlcv1.Lease{ID: 1, ResourceId: "queue-1"}
				h.lm.queueLeases = []*sqlcv1.Lease{lease}
				h.repo.On("ListQueues", mock.Anything, tenantId).Return([]*sqlcv1.V1Queue{}, nil)
				h.repo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindQUEUE, []string{"queue-1"}, mock.Anything).
					Return([]*sqlcv1.Lease{{ID: 2, ResourceId: "queue-1"}}, nil).Once()
				return lease
			},
			poll:   func(h *leaseTestHarness) error { return h.lm.acquireQueueLeases(h.ctx) },
			notify: func(h *leaseTestHarness) error { return h.lm.notifyNewQueue(h.ctx, "queue-1") },
			verify: func(t *testing.T, h *leaseTestHarness) {
				require.Len(t, h.lm.queueLeases, 1)
				assert.Equal(t, int64(2), h.lm.queueLeases[0].ID)

				full := recvLeaseMsg(h, h.lm.queuesCh, "full refresh")
				assert.False(t, full.isIncremental)
				assert.Empty(t, full.items)

				incr := recvLeaseMsg(h, h.lm.queuesCh, "incremental message")
				assert.True(t, incr.isIncremental)
				assert.Equal(t, []string{"queue-1"}, incr.items)
			},
		},
		{
			name: "worker",
			setup: func(h *leaseTestHarness) *sqlcv1.Lease {
				lease := &sqlcv1.Lease{ID: 1, ResourceId: workerId.String()}
				h.lm.workerLeases = []*sqlcv1.Lease{lease}
				h.repo.On("ListActiveWorkers", mock.Anything, tenantId).Return([]*v1.ListActiveWorkersResult{}, nil)
				h.repo.On("GetActiveWorker", mock.Anything, tenantId, workerId).Return(worker, nil).Once()
				h.repo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindWORKER, []string{workerId.String()}, mock.Anything).
					Return([]*sqlcv1.Lease{{ID: 2, ResourceId: workerId.String()}}, nil).Once()
				return lease
			},
			poll:   func(h *leaseTestHarness) error { return h.lm.acquireWorkerLeases(h.ctx) },
			notify: func(h *leaseTestHarness) error { return h.lm.notifyNewWorker(h.ctx, workerId) },
			verify: func(t *testing.T, h *leaseTestHarness) {
				require.Len(t, h.lm.workerLeases, 1)
				assert.Equal(t, int64(2), h.lm.workerLeases[0].ID)

				full := recvLeaseMsg(h, h.lm.workersCh, "full refresh")
				assert.False(t, full.isIncremental)
				assert.Empty(t, full.items)

				incr := recvLeaseMsg(h, h.lm.workersCh, "incremental message")
				assert.True(t, incr.isIncremental)
				assert.Equal(t, []*v1.ListActiveWorkersResult{worker}, incr.items)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newLeaseTestHarness(t, tenantId)
			lease := tc.setup(h)
			h.repo.On("ReleaseLeases", mock.Anything, []*sqlcv1.Lease{lease}).Return(nil).Once()

			// the resource went inactive: the poll releases its lease and sends an empty refresh
			require.NoError(t, tc.poll(h))
			require.Empty(t, h.lm.concurrencyLeases)
			require.Empty(t, h.lm.queueLeases)
			require.Empty(t, h.lm.workerLeases)

			// the resource is active again: the notification must acquire a fresh lease
			require.NoError(t, tc.notify(h))

			tc.verify(t, h)
			h.repo.AssertExpectations(t)
		})
	}
}

// TestLeaseManager_WaitingPollPreemptsDrainPasses checks that a poll which starts waiting for the
// lease mutex while a drain pass is in progress waits for that pass only: no further pass starts
// ahead of it even though entries are pending, the poll then runs with its fresh context, and the
// poll's own drain services what is still pending, so nothing is dropped.
func TestLeaseManager_WaitingPollPreemptsDrainPasses(t *testing.T) {
	tenantId := uuid.New()

	h := newLeaseTestHarness(t, tenantId)
	polling := make(chan struct{})
	acquireStarted := make(chan struct{})
	continueAcquire := make(chan struct{})

	var (
		callsMu sync.Mutex
		calls   []string
	)

	record := func(name string) {
		callsMu.Lock()
		defer callsMu.Unlock()

		calls = append(calls, name)
	}

	twoQueues := func(ids []string) bool { return len(ids) == 2 }
	noExisting := func(existing []*sqlcv1.Lease) bool { return len(existing) == 0 }
	twoExisting := func(existing []*sqlcv1.Lease) bool { return len(existing) == 2 }
	firstLeases := []*sqlcv1.Lease{{ID: 1, ResourceId: "queue-1"}, {ID: 2, ResourceId: "queue-2"}}

	h.repo.On("ListQueues", mock.Anything, tenantId).Run(h.holdPoll(polling)).Return([]*sqlcv1.V1Queue{}, nil).Once()
	h.repo.On("ListQueues", mock.Anything, tenantId).Return([]*sqlcv1.V1Queue{{Name: "queue-1"}, {Name: "queue-2"}}, nil)
	expectEmptyLists(h, "ListQueues")

	// the pass in progress leases queue-1 and queue-2 and parks until the test lets it finish
	h.repo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindQUEUE, mock.MatchedBy(twoQueues), mock.MatchedBy(noExisting)).
		Run(func(mock.Arguments) {
			record("pass in progress")
			close(acquireStarted)

			select {
			case <-continueAcquire:
			case <-h.ctx.Done():
			}
		}).
		Return(firstLeases, nil).Once()
	// the waiting poll extends both leases
	h.repo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindQUEUE, mock.MatchedBy(twoQueues), mock.MatchedBy(twoExisting)).
		Run(func(mock.Arguments) { record("renewal poll") }).
		Return(firstLeases, nil).Once()
	// the poll's own drain services the entry that arrived during the pass in progress
	h.repo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindQUEUE, []string{"queue-3"}, mock.Anything).
		Run(func(mock.Arguments) { record("drain after the poll") }).
		Return([]*sqlcv1.Lease{{ID: 3, ResourceId: "queue-3"}}, nil).Once()

	pollDone := h.startPoll()
	h.waitSignal(polling, "the poll to hold its lease mutex")

	require.NoError(t, h.lm.notifyNewQueue(h.ctx, "queue-1"))
	require.NoError(t, h.lm.notifyNewQueue(h.ctx, "queue-2"))

	h.releasePoll()
	h.waitSignal(acquireStarted, "the drain pass to start its acquisition")

	// an entry arrives during the pass in progress and a renewal poll starts waiting for the mutex
	require.NoError(t, h.lm.notifyNewQueue(h.ctx, "queue-3"))
	require.Equal(t, 1, h.lm.pendingQueues.size())

	renewCtx, cancelRenew := context.WithTimeout(h.ctx, 5*time.Second)
	defer cancelRenew()

	renewStart := time.Now()
	renewErr := make(chan error, 1)

	// the same sequence acquireAllLeases runs for this kind
	h.spawn(func() {
		err := h.lm.acquireQueueLeases(renewCtx)

		if err == nil {
			err = h.lm.drainPendingQueues(renewCtx)
		}

		renewErr <- err
	})

	require.Eventually(t, h.lm.pendingQueues.pollWaiting.Load, 5*time.Second, time.Millisecond, "the poll did not announce that it is waiting")

	close(continueAcquire)

	select {
	case err := <-renewErr:
		require.NoError(t, err)
	case <-h.ctx.Done():
		t.Fatal("timed out waiting for renewal")
	}

	assert.Less(t, time.Since(renewStart), 2*time.Second, "renewal waited for more than the pass in progress")

	h.waitSignal(pollDone, "the first poll and its drain to finish")

	callsMu.Lock()
	assert.Equal(t, []string{"pass in progress", "renewal poll", "drain after the poll"}, calls, "no drain pass may run ahead of the waiting poll")
	callsMu.Unlock()

	assert.Equal(t, 0, h.lm.pendingQueues.size())
	assert.Len(t, h.lm.queueLeases, 3)
	assert.False(t, h.lm.pendingQueues.pollWaiting.Load())
	h.repo.AssertExpectations(t)
}

// TestLeaseManager_CleanupDuringDrainPass checks that cleanup requested while a drain pass holds
// the lease mutex waits for that pass only, releases the lease the pass acquired, closes the lease
// channel, and that no notification is serviced afterwards.
func TestLeaseManager_CleanupDuringDrainPass(t *testing.T) {
	tenantId := uuid.New()
	strategy := &sqlcv1.V1StepConcurrency{ID: 42}
	lease := &sqlcv1.Lease{ID: 1, ResourceId: "42"}

	h := newLeaseTestHarness(t, tenantId)
	polling := make(chan struct{})
	lookupStarted := make(chan struct{})
	continueLookup := make(chan struct{})

	var (
		releasedMu sync.Mutex
		released   []*sqlcv1.Lease
	)

	h.repo.On("ListConcurrencyStrategies", mock.Anything, tenantId).Run(h.holdPoll(polling)).Return([]*sqlcv1.V1StepConcurrency{}, nil).Once()
	h.repo.On("GetConcurrencyStrategy", mock.Anything, tenantId, strategy.ID).
		Run(func(mock.Arguments) {
			close(lookupStarted)

			select {
			case <-continueLookup:
			case <-h.ctx.Done():
			}
		}).
		Return(strategy, nil).Once()
	h.repo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindCONCURRENCYSTRATEGY, []string{"42"}, mock.Anything).
		Return([]*sqlcv1.Lease{lease}, nil).Once()
	h.repo.On("ReleaseLeases", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			releasedMu.Lock()
			defer releasedMu.Unlock()

			released = append(released, args.Get(1).([]*sqlcv1.Lease)...)
		}).
		Return(nil)

	// the concurrency kind's poll followed by its drain, as acquireAllLeases runs them; the batch
	// kind is left out because acquireBatchLeases runs outside processMu and races cleanup on its
	// own, independently of the pending sets
	pollDone := h.spawn(func() {
		if err := h.lm.acquireConcurrencyLeases(h.ctx); err != nil {
			t.Errorf("poll: %v", err)
			return
		}

		if err := h.lm.drainPendingConcurrencyStrategies(h.ctx); err != nil {
			t.Errorf("drain: %v", err)
		}
	})

	h.waitSignal(polling, "the poll to hold its lease mutex")
	require.NoError(t, h.lm.notifyNewConcurrencyStrategy(h.ctx, strategy.ID))
	h.releasePoll()
	h.waitSignal(lookupStarted, "the drain pass to start its lookup")

	// cleanup arrives while the pass holds the mutex and processMu, so it can only run once the
	// pass is done
	cleanupCtx, cancelCleanup := context.WithTimeout(h.ctx, 5*time.Second)
	defer cancelCleanup()

	cleanupStart := time.Now()
	cleanupErr := make(chan error, 1)
	cleanupDone := h.spawn(func() { cleanupErr <- h.lm.cleanup(cleanupCtx) })

	// cleanup must be waiting for processMu's write side behind the pass, which holds its read
	// side: TryRLock fails only while a writer is pending or holding
	require.Eventually(t, func() bool {
		if h.lm.processMu.TryRLock() {
			h.lm.processMu.RUnlock()
			return false
		}

		return true
	}, 5*time.Second, time.Millisecond, "cleanup did not start waiting while the drain pass was in progress")

	close(continueLookup)

	h.waitSignal(cleanupDone, "cleanup to finish")
	require.NoError(t, <-cleanupErr)
	assert.Less(t, time.Since(cleanupStart), 3*time.Second, "cleanup waited for more than the pass in progress")

	h.waitSignal(pollDone, "the poll and its drain to finish")

	// the pass that was in progress completed, and cleanup released the lease it acquired
	h.repo.AssertNumberOfCalls(t, "AcquireOrExtendLeases", 1)

	releasedMu.Lock()
	assert.Equal(t, []*sqlcv1.Lease{lease}, released)
	releasedMu.Unlock()

	full := recvLeaseMsg(h, h.lm.concurrencyLeasesCh, "full refresh")
	assert.False(t, full.isIncremental)

	incr := recvLeaseMsg(h, h.lm.concurrencyLeasesCh, "incremental message")
	assert.True(t, incr.isIncremental)
	assert.Equal(t, []*sqlcv1.V1StepConcurrency{strategy}, incr.items)

	select {
	case _, open := <-h.lm.concurrencyLeasesCh:
		assert.False(t, open, "cleanup closes the lease channel")
	case <-h.ctx.Done():
		t.Fatal("timed out waiting for the lease channel to close")
	}

	// nothing is serviced after cleanup
	require.NoError(t, h.lm.notifyNewConcurrencyStrategy(h.ctx, 43))
	h.repo.AssertNotCalled(t, "GetConcurrencyStrategy", mock.Anything, tenantId, int64(43))
	h.repo.AssertNumberOfCalls(t, "AcquireOrExtendLeases", 1)
	h.repo.AssertExpectations(t)
}

// TestLeaseManager_DrainerYieldsToPollAnnouncedAfterItsCheck covers a drainer that passed its
// first pollWaiting check before the poll announced itself and then wins the mutex once the
// previous holder released it. drainPass is what runs after that first check: it must give the
// mutex back without taking entries, and the poll's own drain must service them afterwards.
func TestLeaseManager_DrainerYieldsToPollAnnouncedAfterItsCheck(t *testing.T) {
	tenantId := uuid.New()

	h := newLeaseTestHarness(t, tenantId)

	h.repo.On("ListQueues", mock.Anything, tenantId).Return([]*sqlcv1.V1Queue{}, nil)
	h.repo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindQUEUE, []string{"queue-1"}, mock.Anything).
		Return([]*sqlcv1.Lease{{ID: 1, ResourceId: "queue-1"}}, nil).Once()

	// queue-1 is pending and the poll announced itself after this drainer's first check
	require.True(t, h.lm.pendingQueues.add("queue-1"))
	h.lm.pendingQueues.pollWaiting.Store(true)

	// the drainer wins the free mutex
	ran, err := drainPass(h.ctx, h.lm, &h.lm.pendingQueues, &h.lm.queueLeasesMu, h.lm.leasePendingQueues)
	require.NoError(t, err)
	assert.False(t, ran, "a drainer that finds the poll waiting must not run a pass")
	assert.Equal(t, 1, h.lm.pendingQueues.size(), "the entry stays for the poll's drain")
	h.repo.AssertNotCalled(t, "AcquireOrExtendLeases", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	require.True(t, h.lm.queueLeasesMu.TryLock(), "the drainer must release the lease mutex for the poll")
	h.lm.queueLeasesMu.Unlock()
	require.True(t, h.lm.processMu.TryLock(), "the drainer must release processMu")
	h.lm.processMu.Unlock()

	// the poll runs, clears the flag once it holds the mutex, and its drain services the entry
	require.NoError(t, h.lm.acquireQueueLeases(h.ctx))
	assert.False(t, h.lm.pendingQueues.pollWaiting.Load())
	require.NoError(t, h.lm.drainPendingQueues(h.ctx))

	assert.Equal(t, 0, h.lm.pendingQueues.size())
	require.Len(t, h.lm.queueLeases, 1)
	assert.Equal(t, "queue-1", h.lm.queueLeases[0].ResourceId)
	h.repo.AssertNumberOfCalls(t, "AcquireOrExtendLeases", 1)

	full := recvLeaseMsg(h, h.lm.queuesCh, "full refresh")
	assert.False(t, full.isIncremental)

	incr := recvLeaseMsg(h, h.lm.queuesCh, "incremental message")
	assert.True(t, incr.isIncremental)
	assert.Equal(t, []string{"queue-1"}, incr.items)

	h.repo.AssertExpectations(t)
}
