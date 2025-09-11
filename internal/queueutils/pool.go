package queueutils

import (
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"

	"github.com/rs/zerolog"
)

type OperationPool struct {
	ops         sync.Map
	timeout     time.Duration
	description string
	method      OpMethod
	ql          *zerolog.Logger
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

func (p *OperationPool) SetTenants(tenants []*dbsqlc.Tenant) {
	tenantMap := make(map[string]bool)

	for _, t := range tenants {
		tenantMap[sqlchelpers.UUIDToStr(t.ID)] = true
	}

	// delete tenants that are not in the list
	p.ops.Range(func(key, value interface{}) bool {
		if _, ok := tenantMap[key.(string)]; !ok {
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

	return op.(*SerialOperation)
}
