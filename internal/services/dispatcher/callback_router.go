package dispatcher

type DurableCallbackHandler interface {
	DeliverCallbackCompletion(taskExternalId string, nodeId int64, invocationCount int64, payload []byte) error
}
