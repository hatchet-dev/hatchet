package v1

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type sharedConfig struct {
	repo v1.SchedulerRepository

	l *zerolog.Logger

	singleQueueLimit int

	schedulerConcurrencyRateLimit int

	schedulerConcurrencyPollingMinInterval time.Duration

	schedulerConcurrencyPollingMaxInterval time.Duration
}

// SchedulingPool is responsible for managing a pool of tenantManagers.
type SchedulingPool struct {
	Extensions *Extensions

	tenants sync.Map
	setMu   mutex

	cf *sharedConfig

	resultsCh chan *QueueResults

	concurrencyResultsCh chan *ConcurrencyResults

	optimisticSchedulingEnabled bool
	optimisticSemaphore         chan struct{}
}

func NewSchedulingPool(
	repo v1.SchedulerRepository,
	l *zerolog.Logger,
	singleQueueLimit int,
	schedulerConcurrencyRateLimit int,
	schedulerConcurrencyPollingMinInterval time.Duration,
	schedulerConcurrencyPollingMaxInterval time.Duration,
	optimisticSchedulingEnabled bool,
	optimisticSlots int,
) (*SchedulingPool, func() error, error) {
	resultsCh := make(chan *QueueResults, 1000)
	concurrencyResultsCh := make(chan *ConcurrencyResults, 1000)
	semaphore := make(chan struct{}, optimisticSlots)

	s := &SchedulingPool{
		Extensions: &Extensions{},
		cf: &sharedConfig{
			repo:                                   repo,
			l:                                      l,
			singleQueueLimit:                       singleQueueLimit,
			schedulerConcurrencyRateLimit:          schedulerConcurrencyRateLimit,
			schedulerConcurrencyPollingMinInterval: schedulerConcurrencyPollingMinInterval,
			schedulerConcurrencyPollingMaxInterval: schedulerConcurrencyPollingMaxInterval,
		},
		resultsCh:                   resultsCh,
		concurrencyResultsCh:        concurrencyResultsCh,
		setMu:                       newMu(l),
		optimisticSchedulingEnabled: optimisticSchedulingEnabled,
		optimisticSemaphore:         semaphore,
	}

	return s, func() error {
		s.cleanup()
		return nil
	}, nil
}

func (p *SchedulingPool) GetResultsCh() chan *QueueResults {
	return p.resultsCh
}

func (p *SchedulingPool) GetConcurrencyResultsCh() chan *ConcurrencyResults {
	return p.concurrencyResultsCh
}

func (p *SchedulingPool) cleanup() {
	toCleanup := make([]*tenantManager, 0)

	p.tenants.Range(func(key, value interface{}) bool {
		toCleanup = append(toCleanup, value.(*tenantManager))

		return true
	})

	p.cleanupTenants(toCleanup)

	err := p.Extensions.Cleanup()

	if err != nil {
		p.cf.l.Error().Err(err).Msg("failed to cleanup extensions")
	}
}

func (p *SchedulingPool) SetTenants(tenants []*sqlcv1.Tenant) {
	if ok := p.setMu.TryLock(); !ok {
		return
	}

	defer p.setMu.Unlock()

	tenantMap := make(map[uuid.UUID]bool)

	for _, t := range tenants {
		tenantId := t.ID
		tenantMap[tenantId] = true
		p.getTenantManager(tenantId, true) // nolint: ineffassign
	}

	toCleanup := make([]*tenantManager, 0)

	// delete tenants that are not in the list
	p.tenants.Range(func(key, value interface{}) bool {
		tenantId := key.(uuid.UUID)

		if _, ok := tenantMap[tenantId]; !ok {
			toCleanup = append(toCleanup, value.(*tenantManager))
		}

		return true
	})

	// delete each tenant from the map
	for _, tm := range toCleanup {
		tenantId := tm.tenantId.String()
		p.tenants.Delete(tenantId)
	}

	go func() {
		// it is safe to cleanup tenants in a separate goroutine because we no longer have pointers to
		// any cleaned up tenants in the map
		p.cleanupTenants(toCleanup)
	}()

	go p.Extensions.SetTenants(tenants)
}

func (p *SchedulingPool) cleanupTenants(toCleanup []*tenantManager) {
	wg := sync.WaitGroup{}

	for _, tm := range toCleanup {
		wg.Add(1)

		go func(tm *tenantManager) {
			defer wg.Done()

			err := tm.Cleanup()

			if err != nil {
				p.cf.l.Error().Err(err).Msgf("failed to cleanup tenant manager for tenant %s", tm.tenantId.String())
			}
		}(tm)
	}

	wg.Wait()
}

func (p *SchedulingPool) Replenish(ctx context.Context, tenantId uuid.UUID) {
	if tm := p.getTenantManager(tenantId, false); tm != nil {
		tm.replenish(ctx)
	}
}

func (p *SchedulingPool) NotifyQueues(ctx context.Context, tenantId uuid.UUID, queueNames []string) {
	if tm := p.getTenantManager(tenantId, false); tm != nil {
		tm.queue(ctx, queueNames)
	}
}

func (p *SchedulingPool) NotifyConcurrency(ctx context.Context, tenantId uuid.UUID, strategyIds []int64) {
	if tm := p.getTenantManager(tenantId, false); tm != nil {
		tm.notifyConcurrency(ctx, strategyIds)
	}
}

func (p *SchedulingPool) getTenantManager(tenantId uuid.UUID, storeIfNotFound bool) *tenantManager {
	tm, ok := p.tenants.Load(tenantId)

	if !ok {
		if storeIfNotFound {
			tm = newTenantManager(p.cf, tenantId, p.resultsCh, p.concurrencyResultsCh, p.Extensions)
			p.tenants.Store(tenantId, tm)
		} else {
			return nil
		}
	}

	return tm.(*tenantManager)
}

var ErrTenantNotFound = fmt.Errorf("tenant not found in pool")
var ErrNoOptimisticSlots = fmt.Errorf("no optimistic slots for scheduling")

func (p *SchedulingPool) RunOptimisticScheduling(ctx context.Context, tenantId uuid.UUID, opts []*v1.WorkflowNameTriggerOpts, localWorkerIds map[uuid.UUID]struct{}) (map[uuid.UUID][]*AssignedItemWithTask, []*v1.V1TaskWithPayload, []*v1.DAGWithData, error) {
	if !p.optimisticSchedulingEnabled {
		return nil, nil, nil, ErrNoOptimisticSlots
	}

	// attempt to acquire a slot in the semaphore
	select {
	case p.optimisticSemaphore <- struct{}{}:
		// acquired a slot
		defer func() {
			<-p.optimisticSemaphore
		}()
	default:
		// no slots available
		return nil, nil, nil, ErrNoOptimisticSlots
	}

	tm := p.getTenantManager(tenantId, false)

	if tm == nil {
		return nil, nil, nil, ErrTenantNotFound
	}

	return tm.runOptimisticScheduling(ctx, opts, localWorkerIds)
}

func (p *SchedulingPool) RunOptimisticSchedulingFromEvents(ctx context.Context, tenantId uuid.UUID, opts []v1.EventTriggerOpts, localWorkerIds map[uuid.UUID]struct{}) (map[uuid.UUID][]*AssignedItemWithTask, *v1.TriggerFromEventsResult, error) {
	if !p.optimisticSchedulingEnabled {
		return nil, nil, ErrNoOptimisticSlots
	}

	// attempt to acquire a slot in the semaphore
	select {
	case p.optimisticSemaphore <- struct{}{}:
		// acquired a slot
		defer func() {
			<-p.optimisticSemaphore
		}()
	default:
		// no slots available
		return nil, nil, ErrNoOptimisticSlots
	}

	tm := p.getTenantManager(tenantId, false)

	if tm == nil {
		return nil, nil, ErrTenantNotFound
	}

	return tm.runOptimisticSchedulingFromEvents(ctx, opts, localWorkerIds)
}
