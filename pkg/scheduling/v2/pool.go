package v2

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type sharedConfig struct {
	queries          *dbsqlc.Queries
	pool             *pgxpool.Pool
	l                *zerolog.Logger
	singleQueueLimit int
}

// SchedulingPool is responsible for managing a pool of tenantManagers.
type SchedulingPool struct {
	tenants sync.Map

	cf *sharedConfig

	resultsCh chan *QueueResults
}

func NewSchedulingPool(l *zerolog.Logger, p *pgxpool.Pool, singleQueueLimit int) (*SchedulingPool, func() error) {
	resultsCh := make(chan *QueueResults)

	s := &SchedulingPool{
		cf: &sharedConfig{
			queries:          dbsqlc.New(),
			pool:             p,
			l:                l,
			singleQueueLimit: singleQueueLimit,
		},
		resultsCh: resultsCh,
	}

	return s, func() error {
		s.cleanup()
		return nil
	}
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
	tenantMap := make(map[string]bool)

	for _, t := range tenants {
		tenantMap[sqlchelpers.UUIDToStr(t.ID)] = true
	}

	toCleanup := make([]*tenantManager, 0)

	// delete tenants that are not in the list
	p.tenants.Range(func(key, value interface{}) bool {
		if _, ok := tenantMap[key.(string)]; !ok {
			toCleanup = append(toCleanup, value.(*tenantManager))
			p.tenants.Delete(key)
		}

		return true
	})

	go func() {
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
	tm := p.getTenantManager(tenantId)

	tm.refreshAll(ctx)
}

func (p *SchedulingPool) Replenish(ctx context.Context, tenantId string) {
	tm := p.getTenantManager(tenantId)

	tm.replenish(ctx)
}

func (p *SchedulingPool) Queue(ctx context.Context, tenantId string, queueName string) {
	tm := p.getTenantManager(tenantId)

	tm.queue(ctx, queueName)
}

func (p *SchedulingPool) getTenantManager(tenantId string) *tenantManager {
	tm, ok := p.tenants.Load(tenantId)

	if !ok {
		tm = newTenantManager(p.cf, tenantId, p.resultsCh)
		p.tenants.Store(tenantId, tm)
	}

	return tm.(*tenantManager)
}
