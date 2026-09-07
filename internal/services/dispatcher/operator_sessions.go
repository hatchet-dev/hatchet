package dispatcher

import (
	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// AddOperatorStreamSession registers a gRPC stream owned by an out-of-process operator as a live
// session for workerId, so the dispatcher fans assigned actions out to it like any SDK worker.
// The stream's server message type must be AssignedAction: the dispatcher encodes each action
// directly onto it. The caller chooses sessionId so the dispatcher's session key is the same id
// it records on the worker row as the listener session fence.
//
// The dispatcher signals fin when it wants the stream hung up: the shutdown drain in Start's
// cleanup does a blocking send on it for every stream-backed worker, so the caller must select
// on fin for as long as the session is registered. The caller must defer release, which removes
// the session from the dispatcher; the stream must outlive the session, since the dispatcher
// may be mid-send on it until release returns.
func (d *DispatcherImpl) AddOperatorStreamSession(
	workerId uuid.UUID,
	sessionId uuid.UUID,
	stream grpc.ServerStream,
) (fin <-chan bool, release func()) {
	finCh := make(chan bool)

	d.workers.Add(
		workerId,
		sessionId,
		newGRPCSubscribedWorker(stream, finCh, workerId, d.defaultMaxWorkerLockAcquisitionTime, d.pubBuffer),
	)

	return finCh, func() {
		d.workers.DeleteForSession(workerId, sessionId)
	}
}

// AddOperatorSession registers an in-process operator as a live session for workerId, so the
// dispatcher fans assigned actions out to op.HandleAction like it does for the operator
// manager's operators. Operator-backed sessions have no stream: nothing selects on fin, the
// shutdown drain in Start's cleanup skips them, and the owner tears them down by calling
// release, which removes the session from the dispatcher. The owner is responsible for the
// worker row (activation, heartbeats, deactivation, and the listener session id that fences
// them); the dispatcher only routes.
func (d *DispatcherImpl) AddOperatorSession(workerId uuid.UUID, op operator.Operator) (release func()) {
	sessionId := uuid.New()

	d.workers.Add(
		workerId,
		sessionId,
		newOperatorSubscribedWorker(workerId, d.pubBuffer, op),
	)

	return func() {
		d.workers.DeleteForSession(workerId, sessionId)
	}
}
