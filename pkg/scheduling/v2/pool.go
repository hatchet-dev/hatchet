package v2

import (
	"context"
	"sync"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type sharedConfig struct {
	repo repository.SchedulerRepository

	l *zerolog.Logger

	singleQueueLimit int
}

// SchedulingPool is responsible for managing a pool of tenantManagers.
type SchedulingPool struct {
	tenants sync.Map
	setMu   mutex

	cf *sharedConfig

	resultsCh chan *QueueResults
}

func NewSchedulingPool(repo repository.SchedulerRepository, l *zerolog.Logger, singleQueueLimit int) (*SchedulingPool, func() error, error) {
	resultsCh := make(chan *QueueResults, 1000)

	s := &SchedulingPool{
		cf: &sharedConfig{
			repo:             repo,
			l:                l,
			singleQueueLimit: singleQueueLimit,
		},
		resultsCh: resultsCh,
		setMu:     newMu(l),
	}

	return s, func() error {
		s.cleanup()
		return nil
	}, nil
}

func (p *SchedulingPool) GetResultsCh() chan *QueueResults {
	return p.resultsCh
}

func (p *SchedulingPool) cleanup() {
	toCleanup := make([]*tenantManager, 0)

	p.tenants.Range(func(key, value interface{}) bool {
		toCleanup = append(toCleanup, value.(*tenantManager))

		return true
	})

	p.cleanupTenants(toCleanup)
}

func (p *SchedulingPool) SetTenants(tenants []*dbsqlc.Tenant) {
	if ok := p.setMu.TryLock(); !ok {
		return
	}

	defer p.setMu.Unlock()

	tenantMap := make(map[string]bool)

	for _, t := range tenants {
		tenantId := sqlchelpers.UUIDToStr(t.ID)
		tenantMap[tenantId] = true
		p.getTenantManager(tenantId, true) // nolint: ineffassign
	}

	toCleanup := make([]*tenantManager, 0)

	// delete tenants that are not in the list
	p.tenants.Range(func(key, value interface{}) bool {
		tenantId := key.(string)

		if _, ok := tenantMap[tenantId]; !ok {
			toCleanup = append(toCleanup, value.(*tenantManager))
		}

		return true
	})

	// delete each tenant from the map
	for _, tm := range toCleanup {
		tenantId := sqlchelpers.UUIDToStr(tm.tenantId)
		p.tenants.Delete(tenantId)
	}

	go func() {
		// it is safe to cleanup tenants in a separate goroutine because we no longer have pointers to
		// any cleaned up tenants in the map
		p.cleanupTenants(toCleanup)
	}()
}

func (p *SchedulingPool) cleanupTenants(toCleanup []*tenantManager) {
	wg := sync.WaitGroup{}

	for _, tm := range toCleanup {
		wg.Add(1)

		go func(tm *tenantManager) {
			defer wg.Done()

			err := tm.Cleanup()

			if err != nil {
				p.cf.l.Error().Err(err).Msgf("failed to cleanup tenant manager for tenant %s", sqlchelpers.UUIDToStr(tm.tenantId))
			}
		}(tm)
	}

	wg.Wait()
}

func (p *SchedulingPool) RefreshAll(ctx context.Context, tenantId string) {
	if tm := p.getTenantManager(tenantId, false); tm != nil {
		tm.refreshAll(ctx)
	}
}

func (p *SchedulingPool) Replenish(ctx context.Context, tenantId string) {
	if tm := p.getTenantManager(tenantId, false); tm != nil {
		tm.replenish(ctx)
	}
}

func (p *SchedulingPool) Queue(ctx context.Context, tenantId string, queueName string) {
	if tm := p.getTenantManager(tenantId, false); tm != nil {
		tm.queue(ctx, queueName)
	}
}

func (p *SchedulingPool) getTenantManager(tenantId string, storeIfNotFound bool) *tenantManager {
	tm, ok := p.tenants.Load(tenantId)

	if !ok {
		if storeIfNotFound {
			tm = newTenantManager(p.cf, tenantId, p.resultsCh)
			p.tenants.Store(tenantId, tm)
		} else {
			return nil
		}
	}

	return tm.(*tenantManager)
}
