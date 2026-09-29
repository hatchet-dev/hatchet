package v1

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const concurrencyLeasesChBuffer = 1024

const (
	// pendingLeaseCapacity bounds each pending set. Hints beyond it are dropped and the periodic
	// poll discovers those resources instead. The set only fills with distinct resources notified
	// while the lease mutex is held, and one drain pass looks up every pending resource under a
	// single pendingLeaseTimeout, so the cap also bounds the work of one pass.
	pendingLeaseCapacity = 256

	// pendingLeaseTimeout is the database budget for one drain pass: the lookups for every pending
	// resource of one kind plus the single lease acquisition for all of them. It is a context
	// deadline on those calls, not a wall-clock bound on the pass.
	pendingLeaseTimeout = 1 * time.Second

	// maxPendingLeasePasses caps how many consecutive passes one drainer runs. A pass repeats only
	// when a hint arrived while the previous pass held the lease mutex, so a steady stream of hints
	// for resources this scheduler cannot lease cannot pin the draining goroutine (which may be
	// the poll). Anything left over is serviced by the next notification's drain or the next poll.
	maxPendingLeasePasses = 4
)

// pendingLeases is a bounded, deduplicated set of resources that were notified while the lease
// mutex for their kind was busy. A notification adds itself before it tries the mutex, so an entry
// is serviced by whichever pass runs next: the holder's post-unlock pass, the notification's own
// drain once the mutex is free, or the drain after the next poll. Entries left behind by the pass
// cap or by a failed pass wait for the next notification or poll. Nobody ever waits for the mutex.
type pendingLeases[K comparable] struct {
	mu  sync.Mutex
	set map[K]struct{}

	// pollWaiting is set while the kind's poll waits for the lease mutex. A drain checks it before
	// competing for the mutex and again once it holds the mutex, so a drainer that sees it set
	// releases the mutex without a pass. The poll therefore waits for at most the pass that was
	// already in progress when it announced itself, and then services the set itself.
	pollWaiting atomic.Bool
}

// add records k and reports whether it is pending. It returns false when the set is full.
func (p *pendingLeases[K]) add(k K) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.set[k]; ok {
		return true
	}

	if len(p.set) >= pendingLeaseCapacity {
		return false
	}

	if p.set == nil {
		p.set = make(map[K]struct{})
	}

	p.set[k] = struct{}{}

	return true
}

// take removes and returns every pending key.
func (p *pendingLeases[K]) take() []K {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.set) == 0 {
		return nil
	}

	keys := make([]K, 0, len(p.set))

	for k := range p.set {
		keys = append(keys, k)
	}

	clear(p.set)

	return keys
}

func (p *pendingLeases[K]) size() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.set)
}

func (p *pendingLeases[K]) reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.set = nil
}

// LeaseManager is responsible for leases on multiple queues and multiplexing
// queue results to callers. It is still tenant-scoped.
//
// Each lease kind has a mutex that the periodic poll holds for its whole body and a pending set
// for on-demand notifications that arrive while it is held. Notifications never block on the
// mutex: they record the resource and try to drain, and whoever releases the mutex drains what
// accumulated in passes of at most maxPendingLeasePasses, each with a pendingLeaseTimeout
// database budget. A waiting poll stops further passes from starting (a drainer that wins the
// mutex after the poll announced itself releases it without a pass), so the poll waits for at most
// the pass that was already in progress. Cleanup waits for the readers of processMu that are
// already admitted, which are at most one poll body per kind and one drain pass per kind.
type LeaseManager struct {
	lr v1.LeaseRepository

	conf *sharedConfig
	l    zerolog.Logger

	tenantId uuid.UUID

	workerLeasesMu sync.Mutex
	workerLeases   []*sqlcv1.Lease
	workersCh      notifierCh[*v1.ListActiveWorkersResult]
	pendingWorkers pendingLeases[uuid.UUID]

	queueLeasesMu sync.Mutex
	queueLeases   []*sqlcv1.Lease
	queuesCh      notifierCh[string]
	pendingQueues pendingLeases[string]

	concurrencyLeasesMu sync.Mutex
	concurrencyLeases   []*sqlcv1.Lease
	concurrencyLeasesCh notifierCh[*sqlcv1.V1StepConcurrency]
	pendingConcurrency  pendingLeases[int64]

	batchLeases []*sqlcv1.Lease
	batchesCh   chan []*sqlcv1.ListDistinctBatchResourcesRow

	cleanedUp bool
	processMu sync.RWMutex
}

func newLeaseManager(conf *sharedConfig, tenantId uuid.UUID) (*LeaseManager, notifierCh[*v1.ListActiveWorkersResult], notifierCh[string], notifierCh[*sqlcv1.V1StepConcurrency], chan []*sqlcv1.ListDistinctBatchResourcesRow) {
	workersCh := make(notifierCh[*v1.ListActiveWorkersResult])
	queuesCh := make(notifierCh[string])
	concurrencyLeasesCh := make(notifierCh[*sqlcv1.V1StepConcurrency], concurrencyLeasesChBuffer)
	batchesCh := make(chan []*sqlcv1.ListDistinctBatchResourcesRow)

	return &LeaseManager{
		lr:                  conf.repo.Lease(),
		conf:                conf,
		l:                   conf.l.With().Str("tenant_id", tenantId.String()).Logger(),
		tenantId:            tenantId,
		workersCh:           workersCh,
		queuesCh:            queuesCh,
		concurrencyLeasesCh: concurrencyLeasesCh,
		batchesCh:           batchesCh,
	}, workersCh, queuesCh, concurrencyLeasesCh, batchesCh
}

func (l *LeaseManager) sendWorkerIds(workerIds []*v1.ListActiveWorkersResult, isIncremental bool) {
	defer func() {
		if r := recover(); r != nil {
			l.l.Error().Interface("recovered", r).Msg("recovered from panic")
		}
	}()

	select {
	case l.workersCh <- notifierMsg[*v1.ListActiveWorkersResult]{
		items:         workerIds,
		isIncremental: isIncremental,
	}:
	default:
	}
}

func (l *LeaseManager) sendQueues(queues []string, isIncremental bool) {
	defer func() {
		if r := recover(); r != nil {
			l.l.Error().Interface("recovered", r).Msg("recovered from panic")
		}
	}()

	select {
	case l.queuesCh <- notifierMsg[string]{
		items:         queues,
		isIncremental: isIncremental,
	}:
	default:
	}
}

func (l *LeaseManager) sendConcurrencyLeases(concurrencyLeases []*sqlcv1.V1StepConcurrency, isIncremental bool) {
	defer func() {
		if r := recover(); r != nil {
			l.l.Error().Interface("recovered", r).Msg("recovered from panic")
		}
	}()

	// a full refresh supersedes anything still queued, so drain stale messages first
	if !isIncremental {
		l.drainConcurrencyLeasesCh()
	}

	select {
	case l.concurrencyLeasesCh <- notifierMsg[*sqlcv1.V1StepConcurrency]{
		items:         concurrencyLeases,
		isIncremental: isIncremental,
	}:
	default:
	}
}

func (l *LeaseManager) drainConcurrencyLeasesCh() {
	for {
		select {
		case <-l.concurrencyLeasesCh:
		default:
			return
		}
	}
}

func (l *LeaseManager) sendBatches(batches []*sqlcv1.ListDistinctBatchResourcesRow) {
	defer func() {
		if r := recover(); r != nil {
			l.conf.l.Error().Interface("recovered", r).Msg("recovered from panic")
		}
	}()

	if l.cleanedUp {
		return
	}

	select {
	case l.batchesCh <- batches:
	default:
	}
}

// drainPendingLeases services the pending set for one lease kind. Each pass takes every pending
// resource under the kind's lease mutex and leases them in one batch, and it repeats while entries
// arrived during the pass, up to maxPendingLeasePasses. It returns without a pass when the mutex is
// busy (the holder runs a pass after it unlocks) or when the kind's poll is waiting for the mutex
// (the poll runs the drain after its own body). A failed pass returns its error; entries added
// during that pass stay pending for the next notification or poll.
func drainPendingLeases[K comparable](
	ctx context.Context,
	l *LeaseManager,
	pending *pendingLeases[K],
	leaseMu *sync.Mutex,
	lease func(ctx context.Context, keys []K) error,
) error {
	for pass := 0; pass < maxPendingLeasePasses; pass++ {
		if pending.size() == 0 || pending.pollWaiting.Load() {
			return nil
		}

		ran, err := drainPass(ctx, l, pending, leaseMu, lease)

		if err != nil || !ran {
			return err
		}
	}

	return nil
}

// drainPass runs one pass under processMu's read side and the kind's lease mutex. It reports
// false without taking entries when the manager is cleaned up, when the mutex is busy, or when
// the kind's poll announced itself after the caller's pollWaiting check: the poll must not wait
// behind another pass, so the entries are left to the drain the poll runs after its body.
func drainPass[K comparable](
	ctx context.Context,
	l *LeaseManager,
	pending *pendingLeases[K],
	leaseMu *sync.Mutex,
	lease func(ctx context.Context, keys []K) error,
) (bool, error) {
	l.processMu.RLock()
	defer l.processMu.RUnlock()

	if l.cleanedUp {
		return false, nil
	}

	if !leaseMu.TryLock() {
		return false, nil
	}

	defer leaseMu.Unlock()

	if pending.pollWaiting.Load() {
		return false, nil
	}

	keys := pending.take()

	if len(keys) == 0 {
		return true, nil
	}

	passCtx, cancel := context.WithTimeout(ctx, pendingLeaseTimeout)
	defer cancel()

	return true, lease(passCtx, keys)
}

func (l *LeaseManager) acquireWorkerLeases(ctx context.Context) error {
	l.processMu.RLock()
	defer l.processMu.RUnlock()

	if l.cleanedUp {
		return nil
	}

	l.pendingWorkers.pollWaiting.Store(true)
	l.workerLeasesMu.Lock()
	l.pendingWorkers.pollWaiting.Store(false)
	defer l.workerLeasesMu.Unlock()

	activeWorkers, err := l.lr.ListActiveWorkers(ctx, l.tenantId)

	if err != nil {
		return err
	}

	currResourceIdsToLease := make(map[string]*sqlcv1.Lease, len(l.workerLeases))

	for _, lease := range l.workerLeases {
		currResourceIdsToLease[lease.ResourceId] = lease
	}

	workerIdsStr := make([]string, len(activeWorkers))
	activeWorkerIdsToResults := make(map[string]*v1.ListActiveWorkersResult, len(activeWorkers))

	leasesToExtend := make([]*sqlcv1.Lease, 0, len(activeWorkers))
	leasesToRelease := make([]*sqlcv1.Lease, 0, len(currResourceIdsToLease))

	for i, activeWorker := range activeWorkers {
		aw := activeWorker
		workerIdsStr[i] = activeWorker.ID.String()
		activeWorkerIdsToResults[workerIdsStr[i]] = aw

		if lease, ok := currResourceIdsToLease[workerIdsStr[i]]; ok {
			leasesToExtend = append(leasesToExtend, lease)
			delete(currResourceIdsToLease, workerIdsStr[i])
		}
	}

	for _, lease := range currResourceIdsToLease {
		leasesToRelease = append(leasesToRelease, lease)
	}

	successfullyAcquiredWorkerIds := make([]*v1.ListActiveWorkersResult, 0)

	if len(workerIdsStr) != 0 {
		workerLeases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindWORKER, workerIdsStr, leasesToExtend)

		if err != nil {
			return err
		}

		l.workerLeases = workerLeases

		for _, lease := range workerLeases {
			successfullyAcquiredWorkerIds = append(successfullyAcquiredWorkerIds, activeWorkerIdsToResults[lease.ResourceId])
		}
	} else {
		// every previously held lease is released below, so the cache must not keep any of them
		l.workerLeases = nil
	}

	l.sendWorkerIds(successfullyAcquiredWorkerIds, false)

	if len(leasesToRelease) != 0 {
		if err := l.lr.ReleaseLeases(ctx, l.tenantId, leasesToRelease); err != nil {
			return err
		}
	}

	return nil
}

// notifyNewWorker leases a worker on-demand. It never waits for workerLeasesMu: when the mutex is
// busy the worker is recorded as pending and leased by whoever releases the mutex.
func (l *LeaseManager) notifyNewWorker(ctx context.Context, workerId uuid.UUID) error {
	if !l.pendingWorkers.add(workerId) {
		l.l.Debug().Str("worker_id", workerId.String()).Msg("pending worker leases are at capacity, leaving the worker to the periodic poll")
		return nil
	}

	return l.drainPendingWorkers(ctx)
}

func (l *LeaseManager) drainPendingWorkers(ctx context.Context) error {
	return drainPendingLeases(ctx, l, &l.pendingWorkers, &l.workerLeasesMu, l.leasePendingWorkers)
}

// leasePendingWorkers acquires leases for the given workers in one query and hands the acquired
// ones to the worker channel. The caller holds workerLeasesMu.
func (l *LeaseManager) leasePendingWorkers(ctx context.Context, workerIds []uuid.UUID) error {
	held := make(map[string]struct{}, len(l.workerLeases))

	for _, lease := range l.workerLeases {
		held[lease.ResourceId] = struct{}{}
	}

	resourceIds := make([]string, 0, len(workerIds))
	workers := make(map[string]*v1.ListActiveWorkersResult, len(workerIds))

	for _, workerId := range workerIds {
		resourceId := workerId.String()

		if _, ok := held[resourceId]; ok {
			continue
		}

		worker, err := l.lr.GetActiveWorker(ctx, l.tenantId, workerId)

		if err != nil {
			// a worker that is not active yet is discovered by the periodic poll once it is
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}

			return err
		}

		resourceIds = append(resourceIds, resourceId)
		workers[resourceId] = worker
	}

	if len(resourceIds) == 0 {
		return nil
	}

	leases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindWORKER, resourceIds, []*sqlcv1.Lease{})

	if err != nil {
		return err
	}

	// leases that were not returned are owned by another scheduler, which manages those workers
	acquired := make([]*v1.ListActiveWorkersResult, 0, len(leases))

	for _, lease := range leases {
		worker, ok := workers[lease.ResourceId]

		if !ok {
			continue
		}

		l.workerLeases = append(l.workerLeases, lease)
		acquired = append(acquired, worker)
	}

	if len(acquired) != 0 {
		l.sendWorkerIds(acquired, true)
	}

	return nil
}

func (l *LeaseManager) acquireQueueLeases(ctx context.Context) error {
	l.processMu.RLock()
	defer l.processMu.RUnlock()

	if l.cleanedUp {
		return nil
	}

	l.pendingQueues.pollWaiting.Store(true)
	l.queueLeasesMu.Lock()
	l.pendingQueues.pollWaiting.Store(false)
	defer l.queueLeasesMu.Unlock()

	queues, err := l.lr.ListQueues(ctx, l.tenantId)

	if err != nil {
		return err
	}

	currResourceIdsToLease := make(map[string]*sqlcv1.Lease, len(l.queueLeases))

	for _, lease := range l.queueLeases {
		currResourceIdsToLease[lease.ResourceId] = lease
	}

	queueIdsStr := make([]string, len(queues))
	leasesToExtend := make([]*sqlcv1.Lease, 0, len(queues))
	leasesToRelease := make([]*sqlcv1.Lease, 0, len(currResourceIdsToLease))

	for i, q := range queues {
		queueIdsStr[i] = q.Name

		if lease, ok := currResourceIdsToLease[queueIdsStr[i]]; ok {
			leasesToExtend = append(leasesToExtend, lease)
			delete(currResourceIdsToLease, queueIdsStr[i])
		}
	}

	for _, lease := range currResourceIdsToLease {
		leasesToRelease = append(leasesToRelease, lease)
	}

	successfullyAcquiredQueues := []string{}

	if len(queueIdsStr) != 0 {

		queueLeases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindQUEUE, queueIdsStr, leasesToExtend)

		if err != nil {
			return err
		}

		l.queueLeases = queueLeases

		for _, lease := range queueLeases {
			successfullyAcquiredQueues = append(successfullyAcquiredQueues, lease.ResourceId)
		}
	} else {
		// every previously held lease is released below, so the cache must not keep any of them
		l.queueLeases = nil
	}

	l.sendQueues(successfullyAcquiredQueues, false)

	if len(leasesToRelease) != 0 {
		if err := l.lr.ReleaseLeases(ctx, l.tenantId, leasesToRelease); err != nil {
			return err
		}
	}

	return nil
}

// notifyNewQueue leases a queue on-demand. It never waits for queueLeasesMu: when the mutex is
// busy the queue is recorded as pending and leased by whoever releases the mutex.
func (l *LeaseManager) notifyNewQueue(ctx context.Context, queueName string) error {
	if !l.pendingQueues.add(queueName) {
		l.l.Debug().Str("queue_name", queueName).Msg("pending queue leases are at capacity, leaving the queue to the periodic poll")
		return nil
	}

	return l.drainPendingQueues(ctx)
}

func (l *LeaseManager) drainPendingQueues(ctx context.Context) error {
	return drainPendingLeases(ctx, l, &l.pendingQueues, &l.queueLeasesMu, l.leasePendingQueues)
}

// leasePendingQueues acquires leases for the given queues in one query and hands the acquired ones
// to the queue channel. The caller holds queueLeasesMu.
func (l *LeaseManager) leasePendingQueues(ctx context.Context, queueNames []string) error {
	held := make(map[string]struct{}, len(l.queueLeases))

	for _, lease := range l.queueLeases {
		held[lease.ResourceId] = struct{}{}
	}

	resourceIds := make([]string, 0, len(queueNames))

	for _, queueName := range queueNames {
		if _, ok := held[queueName]; ok {
			continue
		}

		resourceIds = append(resourceIds, queueName)
	}

	if len(resourceIds) == 0 {
		return nil
	}

	leases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindQUEUE, resourceIds, []*sqlcv1.Lease{})

	if err != nil {
		return err
	}

	// leases that were not returned are owned by another scheduler, which manages those queues
	acquired := make([]string, 0, len(leases))

	for _, lease := range leases {
		if lease.ResourceId == "" {
			continue
		}

		l.queueLeases = append(l.queueLeases, lease)
		acquired = append(acquired, lease.ResourceId)
	}

	if len(acquired) != 0 {
		l.sendQueues(acquired, true)
	}

	return nil
}

func (l *LeaseManager) acquireConcurrencyLeases(ctx context.Context) error {
	l.processMu.RLock()
	defer l.processMu.RUnlock()

	if l.cleanedUp {
		return nil
	}

	l.pendingConcurrency.pollWaiting.Store(true)
	l.concurrencyLeasesMu.Lock()
	l.pendingConcurrency.pollWaiting.Store(false)
	defer l.concurrencyLeasesMu.Unlock()

	strats, err := l.lr.ListConcurrencyStrategies(ctx, l.tenantId)

	if err != nil {
		return err
	}

	currResourceIdsToLease := make(map[string]*sqlcv1.Lease, len(l.concurrencyLeases))

	for _, lease := range l.concurrencyLeases {
		currResourceIdsToLease[lease.ResourceId] = lease
	}

	strategyIdsStr := make([]string, len(strats))
	activeStratIdsToStrategies := make(map[string]*sqlcv1.V1StepConcurrency, len(strats))

	leasesToExtend := make([]*sqlcv1.Lease, 0, len(strats))
	leasesToRelease := make([]*sqlcv1.Lease, 0, len(currResourceIdsToLease))

	for i, s := range strats {
		strategyIdsStr[i] = fmt.Sprintf("%d", s.ID)

		if lease, ok := currResourceIdsToLease[strategyIdsStr[i]]; ok {
			leasesToExtend = append(leasesToExtend, lease)
			delete(currResourceIdsToLease, strategyIdsStr[i])
		}

		activeStratIdsToStrategies[strategyIdsStr[i]] = s
	}

	for _, lease := range currResourceIdsToLease {
		leasesToRelease = append(leasesToRelease, lease)
	}

	successfullyAcquiredStrats := []*sqlcv1.V1StepConcurrency{}

	if len(strategyIdsStr) != 0 {

		concurrencyLeases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindCONCURRENCYSTRATEGY, strategyIdsStr, leasesToExtend)

		if err != nil {
			return err
		}

		l.concurrencyLeases = concurrencyLeases

		for _, lease := range concurrencyLeases {
			successfullyAcquiredStrats = append(successfullyAcquiredStrats, activeStratIdsToStrategies[lease.ResourceId])
		}
	} else {
		// every previously held lease is released below, so the cache must not keep any of them
		l.concurrencyLeases = nil
	}

	l.sendConcurrencyLeases(successfullyAcquiredStrats, false)

	if len(leasesToRelease) != 0 {
		if err := l.lr.ReleaseLeases(ctx, l.tenantId, leasesToRelease); err != nil {
			return err
		}
	}

	return nil
}

// notifyNewConcurrencyStrategy leases a strategy on-demand and hands it to the lease channel,
// which spins up its ConcurrencyManager. It never waits for concurrencyLeasesMu: when the mutex
// is busy the strategy is recorded as pending and leased by whoever releases the mutex.
func (l *LeaseManager) notifyNewConcurrencyStrategy(ctx context.Context, strategyId int64) error {
	if !l.pendingConcurrency.add(strategyId) {
		l.l.Debug().Int64("strategy_id", strategyId).Msg("pending concurrency strategy leases are at capacity, leaving the strategy to the periodic poll")
		return nil
	}

	return l.drainPendingConcurrencyStrategies(ctx)
}

func (l *LeaseManager) drainPendingConcurrencyStrategies(ctx context.Context) error {
	return drainPendingLeases(ctx, l, &l.pendingConcurrency, &l.concurrencyLeasesMu, l.leasePendingConcurrencyStrategies)
}

// leasePendingConcurrencyStrategies acquires leases for the given strategies in one query and
// hands the acquired ones to the lease channel. The caller holds concurrencyLeasesMu.
func (l *LeaseManager) leasePendingConcurrencyStrategies(ctx context.Context, strategyIds []int64) error {
	held := make(map[string]struct{}, len(l.concurrencyLeases))

	for _, lease := range l.concurrencyLeases {
		held[lease.ResourceId] = struct{}{}
	}

	resourceIds := make([]string, 0, len(strategyIds))
	strategies := make(map[string]*sqlcv1.V1StepConcurrency, len(strategyIds))

	for _, strategyId := range strategyIds {
		resourceId := fmt.Sprintf("%d", strategyId)

		if _, ok := held[resourceId]; ok {
			continue
		}

		strategy, err := l.lr.GetConcurrencyStrategy(ctx, l.tenantId, strategyId)

		if err != nil {
			// a strategy that no longer exists has nothing to lease
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}

			return err
		}

		resourceIds = append(resourceIds, resourceId)
		strategies[resourceId] = strategy
	}

	if len(resourceIds) == 0 {
		return nil
	}

	leases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindCONCURRENCYSTRATEGY, resourceIds, []*sqlcv1.Lease{})

	if err != nil {
		return err
	}

	// leases that were not returned are owned by another scheduler, which manages those strategies
	acquired := make([]*sqlcv1.V1StepConcurrency, 0, len(leases))

	for _, lease := range leases {
		strategy, ok := strategies[lease.ResourceId]

		if !ok {
			continue
		}

		l.concurrencyLeases = append(l.concurrencyLeases, lease)
		acquired = append(acquired, strategy)
	}

	if len(acquired) != 0 {
		l.sendConcurrencyLeases(acquired, true)
	}

	return nil
}

func (l *LeaseManager) acquireBatchLeases(ctx context.Context) error {
	batchRepo := l.conf.repo.BatchQueue().NewBatchQueue(l.tenantId)

	resources, err := batchRepo.ListBatchResources(ctx)
	if err != nil {
		return err
	}

	currResourceIdsToLease := make(map[string]*sqlcv1.Lease, len(l.batchLeases))

	for _, lease := range l.batchLeases {
		currResourceIdsToLease[lease.ResourceId] = lease
	}

	resourceIdToRows := make(map[string][]*sqlcv1.ListDistinctBatchResourcesRow)
	resourceIds := make([]string, 0, len(resources))
	leasesToExtend := make([]*sqlcv1.Lease, 0, len(resources))
	leasesToRelease := make([]*sqlcv1.Lease, 0, len(currResourceIdsToLease))

	for _, row := range resources {
		if row == nil || row.BatchKey == "" {
			continue
		}

		resourceId := row.StepID.String()
		resourceIdToRows[resourceId] = append(resourceIdToRows[resourceId], row)

		if len(resourceIdToRows[resourceId]) == 1 {
			resourceIds = append(resourceIds, resourceId)
		}

		if lease, ok := currResourceIdsToLease[resourceId]; ok {
			leasesToExtend = append(leasesToExtend, lease)
			delete(currResourceIdsToLease, resourceId)
		}
	}

	for _, lease := range currResourceIdsToLease {
		leasesToRelease = append(leasesToRelease, lease)
	}

	successfullyAcquired := make([]*sqlcv1.ListDistinctBatchResourcesRow, 0, len(resources))

	if len(resourceIds) != 0 {
		batchLeases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindBATCH, resourceIds, leasesToExtend)
		if err != nil {
			return err
		}

		l.batchLeases = batchLeases

		for _, lease := range batchLeases {
			if rows, ok := resourceIdToRows[lease.ResourceId]; ok {
				successfullyAcquired = append(successfullyAcquired, rows...)
			}
		}
	} else {
		l.batchLeases = nil
	}

	l.sendBatches(successfullyAcquired)

	if len(leasesToRelease) != 0 {
		if err := l.lr.ReleaseLeases(ctx, l.tenantId, leasesToRelease); err != nil {
			return err
		}
	}

	return nil
}

// acquireAllLeases runs one poll of every lease kind. After each kind's poll releases its mutex it
// drains the notifications that arrived while the poll held it.
func (l *LeaseManager) acquireAllLeases(ctx context.Context) {
	loopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	wg := sync.WaitGroup{}

	wg.Add(4)

	go func() {
		defer wg.Done()

		if err := l.acquireWorkerLeases(loopCtx); err != nil {
			l.l.Error().Err(err).Msg("error acquiring worker leases")
		}

		if err := l.drainPendingWorkers(loopCtx); err != nil {
			l.l.Error().Err(err).Msg("error acquiring pending worker leases")
		}
	}()

	go func() {
		defer wg.Done()

		if err := l.acquireQueueLeases(loopCtx); err != nil {
			l.l.Error().Err(err).Msg("error acquiring queue leases")
		}

		if err := l.drainPendingQueues(loopCtx); err != nil {
			l.l.Error().Err(err).Msg("error acquiring pending queue leases")
		}
	}()

	go func() {
		defer wg.Done()

		if err := l.acquireConcurrencyLeases(loopCtx); err != nil {
			l.l.Error().Err(err).Msg("error acquiring concurrency leases")
		}

		if err := l.drainPendingConcurrencyStrategies(loopCtx); err != nil {
			l.l.Error().Err(err).Msg("error acquiring pending concurrency leases")
		}
	}()

	go func() {
		defer wg.Done()

		if err := l.acquireBatchLeases(loopCtx); err != nil {
			l.conf.l.Error().Err(err).Msg("error acquiring batch leases")
		}
	}()
	wg.Wait()
}

// loopForLeases acquires new leases every 5 seconds for workers, queues, and concurrency strategies
func (l *LeaseManager) loopForLeases(ctx context.Context) {
	// Perform an initial lease acquisition immediately so that callers don't have to wait
	// for the first ticker interval before workers, queues, and concurrency strategies are discovered.
	l.acquireAllLeases(ctx)

	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.acquireAllLeases(ctx)
		}
	}
}

func (l *LeaseManager) cleanup(ctx context.Context) error {
	// we acquire a process locks here to prevent concurrent cleanup and lease acquisition
	l.processMu.Lock()
	defer l.processMu.Unlock()

	if l.cleanedUp {
		return nil
	}

	l.cleanedUp = true

	// drains check cleanedUp under processMu before touching a pending set, so nothing is
	// serviced after this point
	l.pendingWorkers.reset()
	l.pendingQueues.reset()
	l.pendingConcurrency.reset()

	eg := errgroup.Group{}

	eg.Go(func() error {
		return l.lr.ReleaseLeases(ctx, l.tenantId, l.workerLeases)
	})

	eg.Go(func() error {
		return l.lr.ReleaseLeases(ctx, l.tenantId, l.queueLeases)
	})

	eg.Go(func() error {
		return l.lr.ReleaseLeases(ctx, l.tenantId, l.concurrencyLeases)
	})

	eg.Go(func() error {
		return l.lr.ReleaseLeases(ctx, l.tenantId, l.batchLeases)
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	// close channels: this is safe to do because each channel is guarded by l.cleanedUp + the process lock
	close(l.workersCh)
	close(l.queuesCh)
	close(l.concurrencyLeasesCh)
	close(l.batchesCh)

	return nil
}

func (l *LeaseManager) start(ctx context.Context) {
	go l.loopForLeases(ctx)
}
