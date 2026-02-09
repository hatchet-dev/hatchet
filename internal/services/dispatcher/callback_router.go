package dispatcher

// DurableCallbackHandler is implemented by the gRPC DispatcherServiceImpl to deliver
// callback completion results to the correct DurableTask stream.
type DurableCallbackHandler interface {
	DeliverCallbackCompletion(taskExternalId string, nodeId int64, invocationCount int64, payload []byte) error
}
