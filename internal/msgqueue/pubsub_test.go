package msgqueue

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestTopicNames(t *testing.T) {
	tenantId := uuid.MustParse("707d0855-80ab-4e1f-a156-f1c4546cbf52")

	tenantTopic := TenantTopic(tenantId)
	assert.Equal(t, "707d0855-80ab-4e1f-a156-f1c4546cbf52_v1", tenantTopic.Name())
	assert.Equal(t, TopicKindTenantStream, tenantTopic.Kind())

	schedulerTopic := SchedulerPartitionTopic("mypartition")
	assert.Equal(t, "mypartition_scheduler_v1", schedulerTopic.Name())
	assert.Equal(t, TopicKindSchedulerPartition, schedulerTopic.Kind())
}

type recordingPubSub struct {
	pubs []string
}

func (r *recordingPubSub) Pub(ctx context.Context, topic Topic, msg *Message) error {
	r.pubs = append(r.pubs, topic.Name()+"/"+msg.ID)
	return nil
}

func (r *recordingPubSub) Sub(topic Topic, handler MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}

func (r *recordingPubSub) IsReady() bool {
	return true
}

func TestPubTenantMessage(t *testing.T) {
	tenantId := uuid.New()
	l := zerolog.Nop()

	t.Run("stream-consumed id is published", func(t *testing.T) {
		inner := &recordingPubSub{}

		err := PubTenantMessage(context.Background(), &l, nil, inner, nil, &Message{ID: MsgIDTaskCompleted, TenantID: tenantId})
		assert.NoError(t, err)
		assert.Len(t, inner.pubs, 1)
	})

	t.Run("non-stream id is not published", func(t *testing.T) {
		inner := &recordingPubSub{}

		err := PubTenantMessage(context.Background(), &l, nil, inner, nil, &Message{ID: MsgIDTaskTrigger, TenantID: tenantId})
		assert.NoError(t, err)
		assert.Empty(t, inner.pubs)
	})

	t.Run("nil tenant id is not published", func(t *testing.T) {
		inner := &recordingPubSub{}

		err := PubTenantMessage(context.Background(), &l, nil, inner, nil, &Message{ID: MsgIDTaskCompleted, TenantID: uuid.Nil})
		assert.NoError(t, err)
		assert.Empty(t, inner.pubs)
	})

	t.Run("durable send happens before stream publish", func(t *testing.T) {
		inner := &recordingPubSub{}
		var sent []string
		mq := &mockMessageQueue{sendMessageFn: func(ctx context.Context, q Queue, msg *Message) error {
			sent = append(sent, q.Name()+"/"+msg.ID)
			return nil
		}}

		err := PubTenantMessage(context.Background(), &l, mq, inner, TASK_PROCESSING_QUEUE, &Message{ID: MsgIDTaskFailed, TenantID: tenantId})
		assert.NoError(t, err)
		assert.Len(t, sent, 1)
		assert.Len(t, inner.pubs, 1)
	})

	t.Run("durable error is returned and skips the stream publish", func(t *testing.T) {
		inner := &recordingPubSub{}
		mq := &mockMessageQueue{sendMessageFn: func(ctx context.Context, q Queue, msg *Message) error {
			return assert.AnError
		}}

		err := PubTenantMessage(context.Background(), &l, mq, inner, TASK_PROCESSING_QUEUE, &Message{ID: MsgIDTaskFailed, TenantID: tenantId})
		assert.Error(t, err)
		assert.Empty(t, inner.pubs)
	})

	t.Run("stream publish error is swallowed when durably sent", func(t *testing.T) {
		inner := &erroringPubSub{}
		mq := &mockMessageQueue{}

		err := PubTenantMessage(context.Background(), &l, mq, inner, TASK_PROCESSING_QUEUE, &Message{ID: MsgIDTaskFailed, TenantID: tenantId})
		assert.NoError(t, err)
	})

	t.Run("stream publish error is returned when it is the only delivery path", func(t *testing.T) {
		inner := &erroringPubSub{}

		err := PubTenantMessage(context.Background(), &l, nil, inner, nil, &Message{ID: MsgIDTaskStreamEvent, TenantID: tenantId})
		assert.Error(t, err)
	})
}

type erroringPubSub struct{}

func (e *erroringPubSub) Pub(ctx context.Context, topic Topic, msg *Message) error {
	return assert.AnError
}

func (e *erroringPubSub) Sub(topic Topic, handler MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}

func (e *erroringPubSub) IsReady() bool {
	return true
}

func TestGatedPubSub(t *testing.T) {
	tenantId := uuid.New()

	cases := []struct {
		name      string
		disabled  bool
		topic     Topic
		msgId     string
		delivered bool
	}{
		{"enabled tenant other", false, TenantTopic(tenantId), MsgIDTaskCompleted, true},
		{"enabled tenant stream", false, TenantTopic(tenantId), MsgIDTaskStreamEvent, true},
		{"enabled scheduler", false, SchedulerPartitionTopic("p"), MsgIDCheckTenantQueue, true},
		{"disabled tenant other", true, TenantTopic(tenantId), MsgIDTaskCompleted, false},
		{"disabled tenant stream", true, TenantTopic(tenantId), MsgIDTaskStreamEvent, true},
		{"disabled scheduler", true, SchedulerPartitionTopic("p"), MsgIDCheckTenantQueue, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inner := &recordingPubSub{}
			ps := NewGatedPubSub(inner, tc.disabled)

			err := ps.Pub(context.Background(), tc.topic, &Message{ID: tc.msgId, TenantID: tenantId})
			assert.NoError(t, err)

			if tc.delivered {
				assert.Len(t, inner.pubs, 1)
			} else {
				assert.Empty(t, inner.pubs)
			}
		})
	}
}
