package dispatcher

import "github.com/google/uuid"

type DurableCallbackHandler interface {
	DeliverCallbackCompletion(taskExternalId uuid.UUID, nodeId int64, invocationCount int64, payload []byte) error
}
