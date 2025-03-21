package v1

import (
	"context"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

func (d *DispatcherServiceImpl) ListenForDurableEvent(contracts.V1Dispatcher_ListenForDurableEventServer) error {
	panic("implement me")
}

func (d *DispatcherServiceImpl) RegisterDurableEvent(context.Context, *contracts.RegisterDurableEventRequest) (*contracts.RegisterDurableEventResponse, error) {
	panic("implement me")
}
