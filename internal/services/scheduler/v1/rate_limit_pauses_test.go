package scheduler

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

type recordingMQ struct {
	msgqueue.MessageQueue
	sent []string
}

func (m *recordingMQ) SendMessage(_ context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	m.sent = append(m.sent, q.Name()+"/"+msg.ID)
	return nil
}

type recordingPubSub struct {
	msgqueue.PubSub
	pubs []string
}

func (p *recordingPubSub) Pub(_ context.Context, topic msgqueue.Topic, msg *msgqueue.Message) error {
	p.pubs = append(p.pubs, topic.Name()+"/"+msg.ID)
	return nil
}

func TestNotifyRateLimitPausesEndedPublishesWake(t *testing.T) {
	l := zerolog.Nop()
	mq := &recordingMQ{}
	ps := &recordingPubSub{}

	s := &Scheduler{mq: mq, pubsub: ps, l: &l}
	s.notifyRateLimitPausesEnded(context.Background(), uuid.New())

	assert.Empty(t, mq.sent)
	assert.Equal(t, []string{"task_controller_wake_v1/rate-limit-pauses-changed"}, ps.pubs)
}
