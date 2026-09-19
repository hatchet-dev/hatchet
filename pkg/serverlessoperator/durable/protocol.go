// Package durable relays one durable task invocation between the engine, reached through an
// operator.DurableChannel, and a serverless endpoint, reached over an operator-dialed websocket.
// The socket is the invocation's single request: the core sends the assigned action as the
// first frame, forwards the endpoint's DurableTaskRequests to the engine and the engine's
// DurableTaskResponses back, and reads the outcome from the endpoint's final done frame.
// What an exit means depends on what the engine has already committed to (an acknowledged
// eviction, a reported error); outcome.go holds that table.
//
// Every frame is one protojson v1.ServerlessDurableFrame (api-contracts/v1/serverless.proto),
// encoded and decoded through the contract package.
package durable

// Close codes the relay sends. The 4xxx codes are application-defined; 1013 is the standard
// "try again later" code, used for backpressure.
const (
	// CloseEvicted follows a forwarded server_evict: the engine superseded the invocation.
	CloseEvicted = 4001
	// CloseShuttingDown means the operator is stopping; the engine re-delivers the task.
	CloseShuttingDown = 4002
	// CloseCancelled means the engine cancelled the task.
	CloseCancelled = 4003
	// CloseInvocationMismatch means a request carried another task id or invocation count.
	CloseInvocationMismatch = 4004
	// CloseForbiddenMessage means the endpoint sent a host-internal request
	// (register_worker, worker_status), a frame that is not a request or done, a done
	// frame whose output is not JSON, or a done frame with status evicted before the engine
	// acknowledged an eviction.
	CloseForbiddenMessage = 4005
	// CloseRequestInFlight means a second ack-bearing request was sent before the first
	// was acknowledged.
	CloseRequestInFlight = 4006
	// CloseTimeout means the endpoint's request timeout elapsed with no done frame.
	CloseTimeout = 4007
	// CloseUnresponsive means two consecutive pings went unanswered.
	CloseUnresponsive = 4008
	// CloseBackpressure means the endpoint fell more than sendQueueSize frames or
	// Params.MaxQueuedBytes behind.
	CloseBackpressure = 1013
	// CloseNormal is sent after a done frame.
	CloseNormal = 1000
	// CloseInternalError is sent when the engine side of the relay fails.
	CloseInternalError = 1011
)
