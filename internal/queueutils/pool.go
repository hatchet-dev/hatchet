package queueutils

import (
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	"github.com/rs/zerolog"
)

type OperationPool[T ID] struct {
	ops         sync.Map
	timeout     time.Duration
	description string
	method      OpMethod[T]
	ql          *zerolog.Logger
	maxJitter   time.Duration
}

func NewOperationPool[T ID](ql *zerolog.Logger, timeout time.Duration, description string, method OpMethod[T]) *OperationPool[T] {
	return &OperationPool[T]{
		timeout:     timeout,
		description: description,
		method:      method,
		ql:          ql,
		maxJitter:   0, // Default to no jitter
	}
}

func (p *OperationPool[T]) WithJitter(maxJitter time.Duration) *OperationPool[T] {
	p.maxJitter = maxJitter
	return p
}

func (p *OperationPool[T]) SetTenants(tenants []*sqlcv1.Tenant) {
	tenantMap := make(map[string]bool)

	for _, t := range tenants {
		tenantMap[t.ID.String()] = true
	}

	// delete tenants that are not in the list
	p.ops.Range(func(key, value interface{}) bool {
		if _, ok := tenantMap[key.(string)]; !ok {
			p.ops.Delete(key)
		}

		return true
	})
}

func (p *OperationPool[T]) SetPartitions(partitions []int64) {
	partitionMap := make(map[int64]bool)

	for _, partitionId := range partitions {
		partitionMap[partitionId] = true
	}

	p.ops.Range(func(key, value interface{}) bool {
		if _, ok := partitionMap[key.(int64)]; !ok {
			p.ops.Delete(key)
		}

		return true
	})
}

func (p *OperationPool[T]) RunOrContinue(id T) {
	p.GetOperation(id).RunOrContinue(p.ql)
}

func (p *OperationPool[T]) GetOperation(id T) *SerialOperation[T] {
	op, ok := p.ops.Load(id)

	if !ok {
		op = &SerialOperation[T]{
			id:          id,
			lastRun:     time.Now(),
			description: p.description,
			timeout:     p.timeout,
			method:      p.method,
			maxJitter:   p.maxJitter,
		}

		p.ops.Store(id, op)
	}

	return op.(*SerialOperation[T])
}
