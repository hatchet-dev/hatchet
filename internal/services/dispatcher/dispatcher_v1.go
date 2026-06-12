package dispatcher

import (
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/task/trigger"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/streams"
	"github.com/hatchet-dev/hatchet/internal/syncx"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type DispatcherServiceImpl struct {
	contracts.UnimplementedV1DispatcherServer
	repo               v1.Repository
	mq                 msgqueue.MessageQueue
	v                  validator.Validator
	analytics          analytics.Analytics
	triggerWriter      *trigger.TriggerWriter
	pubBuffer          *msgqueue.MQPubBuffer
	streamSessions     *streams.Registry
	l                  *zerolog.Logger
	durableInvocations syncx.Map[uuid.UUID, *durableTaskInvocation]
	workerInvocations  syncx.Map[uuid.UUID, *durableTaskInvocation]
	dispatcherId       uuid.UUID
}

// CancelStreamSessions hangs up all registered long-lived streams (durable event
// and durable task listeners). It is called during shutdown before GracefulStop,
// which would otherwise block on them until the process is killed.
func (d *DispatcherServiceImpl) CancelStreamSessions() {
	d.streamSessions.CancelAll()
}

func newDispatcherService(repo v1.Repository, mq msgqueue.MessageQueue, v validator.Validator, l *zerolog.Logger, dispatcherId uuid.UUID, a analytics.Analytics) *DispatcherServiceImpl {
	pubBuffer := msgqueue.NewMQPubBuffer(mq)
	tw := trigger.NewTriggerWriter(mq, repo, l, pubBuffer, 0)

	return &DispatcherServiceImpl{
		repo:          repo,
		mq:            mq,
		v:             v,
		l:             l,
		triggerWriter: tw,
		pubBuffer:     pubBuffer,
		dispatcherId:  dispatcherId,
		analytics:     a,

		streamSessions: streams.NewRegistry(),
	}
}
