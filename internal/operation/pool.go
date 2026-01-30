package operation

import (
	"context"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/internal/services/partition"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	"github.com/rs/zerolog"
)

type OperationPool struct {
	ops         sync.Map
	timeout     time.Duration
	description string
	method      OpMethod
	ql          *zerolog.Logger
	operationId string
	cancel      context.CancelFunc

	hasInterval             bool
	intervalRepo            v1.IntervalSettingsRepository
	intervalMaxJitter       time.Duration
	intervalStartInterval   time.Duration
	intervalMaxInterval     time.Duration
	intervalIncBackoffCount int
	intervalGauge           IntervalGauge
}

func WithPoolInterval(
	repo v1.IntervalSettingsRepository,
	maxJitter, startInterval, maxInterval time.Duration,
	incBackoffCount int,
	gauge IntervalGauge,
) func(*OperationPool) {
	return func(p *OperationPool) {
		p.hasInterval = true
		p.intervalRepo = repo
		p.intervalMaxJitter = maxJitter
		p.intervalStartInterval = startInterval
		p.intervalMaxInterval = maxInterval
		p.intervalIncBackoffCount = incBackoffCount
		p.intervalGauge = gauge
	}
}

func NewOperationPool(p *partition.Partition, ql *zerolog.Logger, operationId string, timeout time.Duration, description string, method OpMethod, fs ...func(*OperationPool)) *OperationPool {
	pool := &OperationPool{
		operationId: operationId,
		timeout:     timeout,
		description: description,
		method:      method,
		ql:          ql,
	}

	for _, f := range fs {
		f(pool)
	}

	outerCtx, outerCancel := context.WithCancel(context.Background())

	// start a goroutine to continuously set tenants
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()

		for {
			select {
			case <-outerCtx.Done():
				return
			case <-t.C:

				// list all tenants
				innerCtx, innerCancel := context.WithTimeout(outerCtx, 5*time.Second)

				tenants, err := p.ListTenantsForController(innerCtx, sqlcv1.TenantMajorEngineVersionV1)

				if err != nil {
					innerCancel()
					ql.Error().Err(err).Msg("could not list tenants")
					continue
				}

				innerCancel()

				pool.setTenants(tenants)
			}
		}
	}()

	pool.cancel = outerCancel

	return pool
}

func (p *OperationPool) Cleanup() {
	p.cancel()

	// stop all operations
	p.ops.Range(func(key, value any) bool {
		if op, ok := value.(*SerialOperation); ok {
			op.Stop()
		}
		p.ops.Delete(key)
		return true
	})
}

func (p *OperationPool) setTenants(tenants []*sqlcv1.Tenant) {
	tenantMap := make(map[string]bool)

	for _, t := range tenants {
		tenantMap[t.ID.String()] = true
	}

	// init ops for new tenants
	for tenantId := range tenantMap {
		p.getOperation(tenantId)
	}

	// delete tenants that are not in the list
	p.ops.Range(func(key, value any) bool {
		if _, ok := tenantMap[key.(string)]; !ok {
			if op, ok := value.(*SerialOperation); ok {
				op.Stop()
			}
			p.ops.Delete(key)
		}

		return true
	})
}

func (p *OperationPool) RunOrContinue(id string) {
	p.getOperation(id).RunOrContinue(p.ql)
}

func (p *OperationPool) getOperation(id string) *SerialOperation {
	op, ok := p.ops.Load(id)

	if !ok {
		var fs []func(*SerialOperation)

		fs = append(fs, WithDescription(p.description))
		fs = append(fs, WithTimeout(p.timeout))

		if p.hasInterval {
			fs = append(fs, WithInterval(
				p.ql,
				p.intervalRepo,
				p.intervalMaxJitter,
				p.intervalStartInterval,
				p.intervalMaxInterval,
				p.intervalIncBackoffCount,
				p.intervalGauge,
			))
		}

		op = NewSerialOperation(
			p.ql,
			id,
			p.operationId,
			p.method,
			fs...,
		)

		p.ops.Store(id, op)
	}

	return op.(*SerialOperation)
}
