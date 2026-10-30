package dispatcher

import (
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/task/trigger"
	"github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1/v1connect"
	"github.com/hatchet-dev/hatchet/internal/services/shared/streams"
	"github.com/hatchet-dev/hatchet/internal/syncx"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type durableInvocationsKey struct {
	tenantId uuid.UUID
	taskId   uuid.UUID
}

var _ v1connect.V1DispatcherHandler = (*DispatcherServiceImpl)(nil)

type DispatcherServiceImpl struct {
	v1connect.UnimplementedV1DispatcherHandler
	repo               v1.Repository
	mq                 msgqueue.MessageQueue
	v                  validator.Validator
	analytics          analytics.Analytics
	alerter            *hatcheterrors.Wrapped
	triggerWriter      *trigger.TriggerWriter
	pubBuffer          *msgqueue.MQPubBuffer
	streamSessions     *streams.Registry
	tenantBudgets      tenantBudgets
	l                  *zerolog.Logger
	durableInvocations syncx.Map[durableInvocationsKey, *durableTaskInvocation]
	workerInvocations  syncx.Map[uuid.UUID, *durableTaskInvocation]
	dispatcherId       uuid.UUID
}

// CancelStreamSessions hangs up all registered long-lived streams (durable event
// and durable task listeners). It is called during shutdown before GracefulStop,
// which would otherwise block on them until the process is killed.
func (d *DispatcherServiceImpl) CancelStreamSessions() {
	d.streamSessions.CancelAll()
}

func newDispatcherService(repo v1.Repository, mq msgqueue.MessageQueue, pubsub msgqueue.PubSub, v validator.Validator, l *zerolog.Logger, dispatcherId uuid.UUID, a analytics.Analytics, alerter *hatcheterrors.Wrapped, promGate *prometheus.Gate) *DispatcherServiceImpl {
	pubBuffer := msgqueue.NewMQPubBuffer(mq)
	tw := trigger.NewTriggerWriter(mq, pubsub, repo, l, pubBuffer, 0, promGate)

	return &DispatcherServiceImpl{
		repo:          repo,
		mq:            mq,
		v:             v,
		l:             l,
		triggerWriter: tw,
		pubBuffer:     pubBuffer,
		dispatcherId:  dispatcherId,
		analytics:     a,
		alerter:       alerter,

		streamSessions: streams.NewRegistry(),
	}
}
