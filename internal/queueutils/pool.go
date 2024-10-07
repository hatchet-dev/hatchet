package queueutils

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type OperationPool struct {
	ops         sync.Map
	timeout     time.Duration
	description string
	method      OpMethod
	ql          *zerolog.Logger
	// setTenantsMu sync.RWMutex
}

func NewOperationPool(ql *zerolog.Logger, timeout time.Duration, description string, method OpMethod) *OperationPool {
	return &OperationPool{
		timeout:     timeout,
		description: description,
		method:      method,
		ql:          ql,
	}
}

// func (p *OperationPool) SetTenants(tenants []*dbsqlc.Tenant) {
// 	p.setTenantsMu.Lock()
// 	defer p.setTenantsMu.Unlock()

// 	tenantMap := make(map[string]bool)

// 	for _, t := range tenants {
// 		tenantMap[sqlchelpers.UUIDToStr(t.ID)] = true
// 	}

// 	// delete tenants that are not in the list
// 	p.ops.Range(func(key, value interface{}) bool {
// 		if _, ok := tenantMap[key.(string)]; !ok {
// 			p.ops.Delete(key)
// 		}

// 		return true
// 	})
// }

func (p *OperationPool) RunOrContinue(id string) {
	// p.setTenantsMu.RLock()
	// defer p.setTenantsMu.RUnlock()

	p.GetOperation(id).RunOrContinue(p.ql)
}

func (p *OperationPool) GetOperation(id string) *SerialOperation {
	// p.setTenantsMu.RLock()
	// defer p.setTenantsMu.RUnlock()

	op, ok := p.ops.Load(id)

	if !ok {
		op = &SerialOperation{
			id:          id,
			lastRun:     time.Now(),
			description: p.description,
			timeout:     p.timeout,
			method:      p.method,
		}

		p.ops.Store(id, op)
	}

	return op.(*SerialOperation)
}
