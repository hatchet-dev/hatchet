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
}

func NewOperationPool(ql *zerolog.Logger, timeout time.Duration, description string, method OpMethod) *OperationPool {
	return &OperationPool{
		timeout:     timeout,
		description: description,
		method:      method,
		ql:          ql,
	}
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
		}

		p.ops.Store(id, op)
	}

	return op.(*SerialOperation)
}
