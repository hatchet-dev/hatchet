package msgqueue

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatcherPubSubDeliveryRoutesOnlyDispatcherQueues(t *testing.T) {
	var queued []string
	mq := &mockMessageQueue{sendMessageFn: func(_ context.Context, q Queue, _ *Message) error {
		queued = append(queued, q.Name())
		return nil
	}}
	ps := &recordingPubSub{}

	q := NewDispatcherPubSubDeliveryQueue(mq, ps, 0)
	dispatcherId := uuid.New()
	msg := &Message{ID: MsgIDTaskAssignedBulk}

	require.NoError(t, q.SendMessage(context.Background(), QueueTypeFromDispatcherID(dispatcherId), msg))
	require.NoError(t, q.SendMessage(context.Background(), TASK_PROCESSING_QUEUE, msg))

	assert.Equal(t, []string{DispatcherTopic(dispatcherId).Name() + "/" + MsgIDTaskAssignedBulk}, ps.pubs)
	assert.Equal(t, []string{TASK_PROCESSING_QUEUE.Name()}, queued)
}

func TestDispatcherPubSubDeliveryDropsEverythingAtFullDropPercent(t *testing.T) {
	ps := &recordingPubSub{}
	q := NewDispatcherPubSubDeliveryQueue(&mockMessageQueue{}, ps, 100)

	require.NoError(t, q.SendMessage(context.Background(), QueueTypeFromDispatcherID(uuid.New()), &Message{ID: MsgIDTaskAssignedBulk}))
	assert.Empty(t, ps.pubs)
}

func TestDispatcherPubSubDeliveryClonePreservesRouting(t *testing.T) {
	ps := &recordingPubSub{}
	_, cloned, err := NewDispatcherPubSubDeliveryQueue(&mockMessageQueue{}, ps, 0).Clone()
	require.NoError(t, err)

	require.NoError(t, cloned.SendMessage(context.Background(), QueueTypeFromDispatcherID(uuid.New()), &Message{ID: MsgIDDurableCallbackCompleted}))
	assert.Len(t, ps.pubs, 1)
}

func TestDispatcherPubSubDeliveryKeepsUnrecoverableMessagesDurable(t *testing.T) {
	var queued []string
	mq := &mockMessageQueue{sendMessageFn: func(_ context.Context, _ Queue, msg *Message) error {
		queued = append(queued, msg.ID)
		return nil
	}}
	ps := &recordingPubSub{}
	q := NewDispatcherPubSubDeliveryQueue(mq, ps, 100)

	for _, id := range []string{MsgIDTaskCancelled, MsgIDBatchStart} {
		require.NoError(t, q.SendMessage(context.Background(), QueueTypeFromDispatcherID(uuid.New()), &Message{ID: id}))
	}

	assert.Empty(t, ps.pubs)
	assert.Equal(t, []string{MsgIDTaskCancelled, MsgIDBatchStart}, queued, "the drop percent must not apply to durable messages either")
}
