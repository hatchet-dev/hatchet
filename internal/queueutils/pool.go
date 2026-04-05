package queueutils

import (
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/syncx"

	"github.com/rs/zerolog"
)

type OperationPool struct {
	method      OpMethod
	ql          *zerolog.Logger
	ops         syncx.Map[string, *SerialOperation]
	description string
	timeout     time.Duration
	maxJitter   time.Duration
}

func NewOperationPool(ql *zerolog.Logger, timeout time.Duration, description string, method OpMethod) *OperationPool {
	return &OperationPool{
		timeout:     timeout,
		description: description,
		method:      method,
		ql:          ql,
		maxJitter:   0, // Default to no jitter
	}
}

func (p *OperationPool) WithJitter(maxJitter time.Duration) *OperationPool {
	p.maxJitter = maxJitter
	return p
}

func (p *OperationPool) SetTenants(tenants []uuid.UUID) {
	tenantMap := make(map[string]bool)

	for _, t := range tenants {
		tenantMap[t.String()] = true
	}

	// delete tenants that are not in the list
	p.ops.Range(func(key string, value *SerialOperation) bool {
		if _, ok := tenantMap[key]; !ok {
			p.ops.Delete(key)
		}

		return true
	})
}

func (p *OperationPool) RunOrContinue(id string) {
	p.GetOperation(id).RunOrContinue(p.ql)
}

func (p *OperationPool) GetOperation(id string) *SerialOperation {
	op, ok := p.ops.Load(id)

	if !ok {
		op = &SerialOperation{
			id:          id,
			lastRun:     time.Now(),
			description: p.description,
			timeout:     p.timeout,
			method:      p.method,
			maxJitter:   p.maxJitter,
		}

		p.ops.Store(id, op)
	}

	return op
}
