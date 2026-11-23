package msgqueue

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatcherDoorbellRoutesOnlyDispatcherQueues(t *testing.T) {
	var queued []string
	mq := &mockMessageQueue{sendMessageFn: func(_ context.Context, q Queue, _ *Message) error {
		queued = append(queued, q.Name())
		return nil
	}}
	ps := &recordingPubSub{}

	q := NewDispatcherDoorbellQueue(mq, ps, 0)
	dispatcherId := uuid.New()
	msg := &Message{ID: MsgIDTaskAssignedBulk}

	require.NoError(t, q.SendMessage(context.Background(), QueueTypeFromDispatcherID(dispatcherId), msg))
	require.NoError(t, q.SendMessage(context.Background(), TASK_PROCESSING_QUEUE, msg))

	assert.Equal(t, []string{DispatcherTopic(dispatcherId).Name() + "/" + MsgIDTaskAssignedBulk}, ps.pubs)
	assert.Equal(t, []string{TASK_PROCESSING_QUEUE.Name()}, queued)
}

func TestDispatcherDoorbellDropsEverythingAtFullDropPercent(t *testing.T) {
	ps := &recordingPubSub{}
	q := NewDispatcherDoorbellQueue(&mockMessageQueue{}, ps, 100)

	require.NoError(t, q.SendMessage(context.Background(), QueueTypeFromDispatcherID(uuid.New()), &Message{ID: MsgIDTaskAssignedBulk}))
	assert.Empty(t, ps.pubs)
}

func TestDispatcherDoorbellClonePreservesRouting(t *testing.T) {
	ps := &recordingPubSub{}
	_, cloned, err := NewDispatcherDoorbellQueue(&mockMessageQueue{}, ps, 0).Clone()
	require.NoError(t, err)

	require.NoError(t, cloned.SendMessage(context.Background(), QueueTypeFromDispatcherID(uuid.New()), &Message{ID: MsgIDTaskCancelled}))
	assert.Len(t, ps.pubs, 1)
}
