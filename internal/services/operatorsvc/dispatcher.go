package operatorsvc

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
)

// dispatcherAdapter presents *dispatcher.DispatcherImpl as a DispatcherBackend. The two session
// hooks return concrete types, so each one is narrowed to the interface the service holds.
type dispatcherAdapter struct {
	*dispatcher.DispatcherImpl
}

// NewDispatcherBackend adapts the local dispatcher to what an operator session needs from it.
func NewDispatcherBackend(d *dispatcher.DispatcherImpl) DispatcherBackend {
	return dispatcherAdapter{d}
}

func (a dispatcherAdapter) AddOperatorStreamSession(ctx context.Context, workerId uuid.UUID, sessionId uuid.UUID, stream *rpcstream.Sender[v1contracts.OperatorListenResponse]) StreamSession {
	return a.DispatcherImpl.AddOperatorStreamSession(ctx, workerId, sessionId, stream)
}

func (a dispatcherAdapter) AddOperatorSession(workerId uuid.UUID, sessionId uuid.UUID, handler ActionHandler) HandlerSession {
	return a.DispatcherImpl.AddOperatorSession(workerId, sessionId, handler)
}
