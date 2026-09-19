package operatorsvc

import (
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
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

func (a dispatcherAdapter) AddOperatorStreamSession(workerId uuid.UUID, sessionId uuid.UUID, stream grpc.ServerStream, wrap func(*contracts.AssignedAction) proto.Message) StreamSession {
	return a.DispatcherImpl.AddOperatorStreamSession(workerId, sessionId, stream, wrap)
}

func (a dispatcherAdapter) AddOperatorSession(workerId uuid.UUID, sessionId uuid.UUID, handler ActionHandler) HandlerSession {
	return a.DispatcherImpl.AddOperatorSession(workerId, sessionId, handler)
}
