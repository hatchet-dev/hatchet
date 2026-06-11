//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"

	repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// ---- benchMocks: minimal stubs for the interfaces the scheduler needs ----

type benchQueueFactoryRepo struct{}

func (m *benchQueueFactoryRepo) NewQueue(tenantId uuid.UUID, queueName string) repo.QueueRepository {
	return &benchQueueRepo{}
}

type benchQueueRepo struct{}

func (m *benchQueueRepo) ListQueueItems(ctx context.Context, limit int) ([]*sqlcv1.V1QueueItem, error) {
	return nil, nil
}
func (m *benchQueueRepo) MarkQueueItemsProcessed(ctx context.Context, r *repo.AssignResults) (succeeded []*repo.AssignedItem, failed []*repo.AssignedItem, err error) {
	return nil, nil, nil
}
func (m *benchQueueRepo) GetTaskRateLimits(ctx context.Context, tx *repo.OptimisticTx, queueItems []*sqlcv1.V1QueueItem) (map[int64]map[string]int32, error) {
	return nil, nil
}
func (m *benchQueueRepo) RequeueRateLimitedItems(ctx context.Context, tenantId uuid.UUID, queueName string) ([]*sqlcv1.RequeueRateLimitedQueueItemsRow, error) {
	return nil, nil
}
func (m *benchQueueRepo) GetDesiredLabels(ctx context.Context, tx *repo.OptimisticTx, stepIds []uuid.UUID) (map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow, error) {
	return nil, nil
}
func (m *benchQueueRepo) GetStepSlotRequests(ctx context.Context, tx *repo.OptimisticTx, stepIds []uuid.UUID) (map[uuid.UUID]map[string]int32, error) {
	return nil, nil
}
func (m *benchQueueRepo) Cleanup() {}

type benchConcurrencyRepo struct{}

func (m *benchConcurrencyRepo) RunConcurrencyStrategy(ctx context.Context, tenantId uuid.UUID, strategy *sqlcv1.V1StepConcurrency) (*repo.RunConcurrencyResult, error) {
	return &repo.RunConcurrencyResult{}, nil
}
func (m *benchConcurrencyRepo) UpdateConcurrencyStrategyIsActive(ctx context.Context, tenantId uuid.UUID, strategy *sqlcv1.V1StepConcurrency) error {
	return nil
}
func (m *benchConcurrencyRepo) DeactivateStaleStepConcurrency(ctx context.Context, tenantId uuid.UUID) error {
	return nil
}
func (m *benchConcurrencyRepo) ListTenantsWithManyStepConcurrencies(ctx context.Context, threshold int64) ([]*sqlcv1.ListTenantsWithManyStepConcurrenciesRow, error) {
	return nil, nil
}

// ---- Extended mock scheduler repo that wraps mockSchedulerRepo ----

type benchMockSchedulerRepo struct {
	*mockSchedulerRepo
	rateLimitRepo    repo.RateLimitRepository
	leaseRepo        repo.LeaseRepository
	queueFactoryRepo repo.QueueFactoryRepository
	concurrencyRepo  repo.ConcurrencyRepository
}

func (m *benchMockSchedulerRepo) RateLimit() repo.RateLimitRepository {
	if m.rateLimitRepo != nil {
		return m.rateLimitRepo
	}
	panic("rateLimitRepo not set")
}

func (m *benchMockSchedulerRepo) Lease() repo.LeaseRepository {
	if m.leaseRepo != nil {
		return m.leaseRepo
	}
	panic("leaseRepo not set")
}

func (m *benchMockSchedulerRepo) QueueFactory() repo.QueueFactoryRepository {
	if m.queueFactoryRepo != nil {
		return m.queueFactoryRepo
	}
	panic("queueFactoryRepo not set")
}

func (m *benchMockSchedulerRepo) Concurrency() repo.ConcurrencyRepository {
	if m.concurrencyRepo != nil {
		return m.concurrencyRepo
	}
	panic("concurrencyRepo not set")
}

// BenchmarkIdleTenants measures resource usage (goroutines, heap, mallocs) when
// N tenants are active with 0 workers and the scheduling loops are running.
func BenchmarkIdleTenants(b *testing.B) {
	numTenants := 50
	if v := os.Getenv("NUM_TENANTS"); v != "" {
		fmt.Sscanf(v, "%d", &numTenants)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.N)*time.Millisecond)
	defer cancel()

	l := zerolog.Nop()

	rl := new(mockRateLimitRepo)
	rl.On("UpdateRateLimits", mock.Anything, mock.Anything, mock.Anything).
		Return([]*sqlcv1.ListRateLimitsForTenantWithMutateRow{}, (*time.Time)(nil), nil)

	ls := new(mockLeaseRepo)
	ls.On("ListQueues", mock.Anything, mock.Anything).Return([]*sqlcv1.V1Queue{}, nil)
	ls.On("ListActiveWorkers", mock.Anything, mock.Anything).Return([]*repo.ListActiveWorkersResult{}, nil)
	ls.On("ListConcurrencyStrategies", mock.Anything, mock.Anything).Return([]*sqlcv1.V1StepConcurrency{}, nil)
	ls.On("AcquireOrExtendLeases", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]*sqlcv1.Lease{}, nil)
	ls.On("ReleaseLeases", mock.Anything, mock.Anything).Return(nil)
	ls.On("GetActiveWorker", mock.Anything, mock.Anything, mock.Anything).
		Return(&repo.ListActiveWorkersResult{}, nil)
	ls.On("GetConcurrencyStrategy", mock.Anything, mock.Anything, mock.Anything).
		Return(&sqlcv1.V1StepConcurrency{}, nil)
	ls.On("RenewLeases", mock.Anything, mock.Anything).Return([]*sqlcv1.Lease{}, nil)

	am := new(mockAssignmentRepo)

	mr := &benchMockSchedulerRepo{
		mockSchedulerRepo: &mockSchedulerRepo{assignment: am},
		rateLimitRepo:     rl,
		leaseRepo:         ls,
		queueFactoryRepo:  &benchQueueFactoryRepo{},
		concurrencyRepo:   &benchConcurrencyRepo{},
	}

	cf := &sharedConfig{
		repo:                                   mr,
		l:                                      &l,
		singleQueueLimit:                       100,
		schedulerConcurrencyRateLimit:          20,
		schedulerConcurrencyPollingMinInterval: 500 * time.Millisecond,
		schedulerConcurrencyPollingMaxInterval: 5 * time.Second,
		schedulerCheckActiveMinInterval:        30 * time.Second,
		schedulerCheckActiveMaxInterval:        60 * time.Second,
		schedulerAdvisoryLockTimeout:           5 * time.Second,
	}

	resultsCh := make(chan *QueueResults, 1000)
	concurrencyResultsCh := make(chan *ConcurrencyResults, 1000)

	for i := 0; i < numTenants; i++ {
		_ = newTenantManager(cf, uuid.New(), resultsCh, concurrencyResultsCh, &Extensions{})
	}

	var drainWg sync.WaitGroup
	drainWg.Add(2)
	go func() {
		defer drainWg.Done()
		for {
			select {
			case <-resultsCh:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		defer drainWg.Done()
		for {
			select {
			case <-concurrencyResultsCh:
			case <-ctx.Done():
				return
			}
		}
	}()

	b.ResetTimer()

	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)

	<-ctx.Done()
	runtime.ReadMemStats(&m1)

	allocMB := float64(m1.TotalAlloc-m0.TotalAlloc) / 1024 / 1024
	heapMB := float64(m1.HeapInuse-m0.HeapInuse) / 1024 / 1024
	goroutines := runtime.NumGoroutine()
	gcCycles := m1.NumGC - m0.NumGC

	b.ReportMetric(allocMB, "alloc_mb")
	b.ReportMetric(heapMB, "heap_mb")
	b.ReportMetric(float64(goroutines), "goroutines_end")
	b.ReportMetric(float64(m1.Mallocs-m0.Mallocs)/1e6, "mallocs_million")
	b.ReportMetric(float64(gcCycles), "gc_cycles")
	b.ReportMetric(float64(numTenants), "tenants")

	if hp := os.Getenv("HEAP_PROFILE"); hp != "" {
		f, _ := os.Create(hp)
		if f != nil {
			pprof.WriteHeapProfile(f)
			f.Close()
		}
	}
}

// BenchmarkReplenishContention measures how replenish scales with many workers.
func BenchmarkReplenishContention(b *testing.B) {
	tenantId := uuid.New()

	workers := make([]*repo.ListActiveWorkersResult, 100)
	for i := range workers {
		workers[i] = &repo.ListActiveWorkersResult{
			ID:   uuid.New(),
			Name: fmt.Sprintf("w-%d", i),
		}
	}

	am := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tid uuid.UUID, wids []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			rows := make([]*sqlcv1.ListActionsForWorkersRow, 0, len(wids))
			for _, wid := range wids {
				rows = append(rows, &sqlcv1.ListActionsForWorkersRow{
					WorkerId: wid,
					ActionId: pgtypeText("process-order"),
				})
			}
			return rows, nil
		},
		listWorkerSlotConfigsFn: func(ctx context.Context, tid uuid.UUID, wids []uuid.UUID) ([]*sqlcv1.ListWorkerSlotConfigsRow, error) {
			rows := make([]*sqlcv1.ListWorkerSlotConfigsRow, 0, len(wids))
			for _, wid := range wids {
				rows = append(rows, &sqlcv1.ListWorkerSlotConfigsRow{
					WorkerID: wid,
					SlotType: repo.SlotTypeDefault,
				})
			}
			return rows, nil
		},
		listAvailableSlotsForWorkersAndTypesFn: func(ctx context.Context, tid uuid.UUID, p sqlcv1.ListAvailableSlotsForWorkersAndTypesParams) ([]*sqlcv1.ListAvailableSlotsForWorkersAndTypesRow, error) {
			rows := make([]*sqlcv1.ListAvailableSlotsForWorkersAndTypesRow, 0, len(p.Workerids))
			for _, wid := range p.Workerids {
				rows = append(rows, &sqlcv1.ListAvailableSlotsForWorkersAndTypesRow{
					ID:             wid,
					SlotType:       repo.SlotTypeDefault,
					AvailableSlots: 100,
				})
			}
			return rows, nil
		},
	}

	l := zerolog.Nop()
	mr := &mockSchedulerRepo{assignment: am}
	cf := &sharedConfig{repo: mr, l: &l}
	s := newScheduler(cf, tenantId, nil, &Extensions{})
	s.setWorkers(workers)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			if err := s.replenish(ctx, false); err != nil {
				b.Error(err)
			}
		}
	})
}

// BenchmarkTryAssignBatch measures the assignment hot path.
func BenchmarkTryAssignBatch(b *testing.B) {
	tenantId := uuid.New()
	l := zerolog.Nop()
	mr := &mockSchedulerRepo{assignment: &mockAssignmentRepo{}}
	cf := &sharedConfig{repo: mr, l: &l}
	s := newScheduler(cf, tenantId, nil, &Extensions{})

	s.actions["process-order"] = &action{
		actionId: "process-order",
		slotsByTypeAndWorkerId: map[string]map[uuid.UUID][]*slot{repo.SlotTypeDefault: {}},
	}

	for i := 0; i < 100; i++ {
		wid := uuid.New()
		w := &worker{ListActiveWorkersResult: &repo.ListActiveWorkersResult{ID: wid, Name: fmt.Sprintf("w-%d", i)}}
		for j := 0; j < 10; j++ {
			sl := newSlot(w, newSlotMeta([]string{"process-order"}, repo.SlotTypeDefault))
			s.actions["process-order"].slots = append(s.actions["process-order"].slots, sl)
			s.actions["process-order"].slotsByTypeAndWorkerId[repo.SlotTypeDefault][wid] =
				append(s.actions["process-order"].slotsByTypeAndWorkerId[repo.SlotTypeDefault][wid], sl)
		}
	}

	qis := make([]*sqlcv1.V1QueueItem, 50)
	for i := range qis {
		qis[i] = &sqlcv1.V1QueueItem{
			ID:       int64(i),
			TenantID: tenantId,
			ActionID: "process-order",
			TaskID:   int64(1000 + i),
			Queue:    "default",
			StepID:   uuid.New(),
		}
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := s.tryAssignBatch(ctx, "process-order", qis, 0, nil, nil, nil, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func pgtypeText(s string) struct {
	String string
	Valid  bool
} {
	return struct {
		String string
		Valid  bool
	}{String: s, Valid: true}
}
