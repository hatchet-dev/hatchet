package msgqueue

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTopicNames(t *testing.T) {
	tenantId := uuid.MustParse("707d0855-80ab-4e1f-a156-f1c4546cbf52")

	tenantTopic := TenantTopic(tenantId)
	assert.Equal(t, "707d0855-80ab-4e1f-a156-f1c4546cbf52_v1", tenantTopic.Name())
	assert.True(t, tenantTopic.IsTenantStream())

	schedulerTopic := SchedulerPartitionTopic("mypartition")
	assert.Equal(t, "mypartition_scheduler_v1", schedulerTopic.Name())
	assert.False(t, schedulerTopic.IsTenantStream())
}

type recordingPubSub struct {
	pubs []string
}

func (r *recordingPubSub) Pub(ctx context.Context, topic Topic, msg *Message) error {
	r.pubs = append(r.pubs, topic.Name()+"/"+msg.ID)
	return nil
}

func (r *recordingPubSub) Sub(topic Topic, handler AckHook) (func() error, error) {
	return func() error { return nil }, nil
}

func (r *recordingPubSub) IsReady() bool {
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
