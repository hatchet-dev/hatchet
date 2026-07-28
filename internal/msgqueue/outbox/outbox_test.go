package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	outboxsqlc "github.com/hatchet-dev/pgoutbox/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

var testTenantUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func zerologNop() *zerolog.Logger {
	l := zerolog.Nop()
	return &l
}

type sentMessage struct {
	queue msgqueue.Queue
	msg   *msgqueue.Message
}

type fakeMQ struct {
	mu   sync.Mutex
	sent []sentMessage
	err  error
}

func (f *fakeMQ) Clone() (func() error, msgqueue.MessageQueue, error) {
	return func() error { return nil }, &fakeMQ{}, nil
}

func (f *fakeMQ) SetQOS(prefetchCount int) {}

func (f *fakeMQ) SendMessage(ctx context.Context, queue msgqueue.Queue, msg *msgqueue.Message) error {
	if f.err != nil {
		return f.err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.sent = append(f.sent, sentMessage{queue: queue, msg: msg})
	return nil
}

func (f *fakeMQ) Subscribe(queue msgqueue.Queue, preAck msgqueue.MsgHandler, postAck msgqueue.MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}

func (f *fakeMQ) RegisterTenant(ctx context.Context, tenantId uuid.UUID) error {
	return nil
}

func (f *fakeMQ) IsReady() bool {
	return true
}

func (f *fakeMQ) sentMessages() []sentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]sentMessage{}, f.sent...)
}

func newTestMessage(t *testing.T) *msgqueue.Message {
	t.Helper()

	msg, err := msgqueue.NewTenantMessage(testTenantUUID, "test-msg", false, true, map[string]any{"key": "value"})
	require.NoError(t, err)

	return msg
}

type fakeFlushContext struct {
	context.Context
}

func (f fakeFlushContext) Tx() pgx.Tx {
	return nil
}

func outboxMessage(t *testing.T, id int64, msg *msgqueue.Message) *outboxsqlc.Message {
	t.Helper()

	payload, err := json.Marshal(msg)
	require.NoError(t, err)

	return &outboxsqlc.Message{
		ID:      id,
		Topic:   Topic(msgqueue.OLAP_QUEUE),
		Payload: payload,
	}
}

func TestTopic(t *testing.T) {
	assert.Equal(t, "mq.olap_queue_v2", Topic(msgqueue.OLAP_QUEUE))
}

func TestRelayFlusherPublishesToMQ(t *testing.T) {
	mq := &fakeMQ{}

	f := &relayFlusher{
		mq: mq,
		queue:    msgqueue.OLAP_QUEUE,
		l:        zerologNop(),
	}

	msg := newTestMessage(t)

	err := f.Flush(fakeFlushContext{Context: context.Background()}, []*outboxsqlc.Message{
		outboxMessage(t, 1, msg),
		// unparseable payloads are dropped rather than wedging the topic
		{ID: 2, Topic: Topic(msgqueue.OLAP_QUEUE), Payload: []byte("not json")},
	})
	require.NoError(t, err)

	sent := mq.sentMessages()
	require.Len(t, sent, 1)
	assert.Equal(t, msgqueue.OLAP_QUEUE.Name(), sent[0].queue.Name())
	assert.Equal(t, msg.ID, sent[0].msg.ID)
	assert.Equal(t, msg.TenantID, sent[0].msg.TenantID)
}

func TestRelayFlusherReturnsMQError(t *testing.T) {
	mq := &fakeMQ{err: fmt.Errorf("publish failed")}

	f := &relayFlusher{
		mq: mq,
		queue:    msgqueue.OLAP_QUEUE,
		l:        zerologNop(),
	}

	err := f.Flush(fakeFlushContext{Context: context.Background()}, []*outboxsqlc.Message{
		outboxMessage(t, 1, newTestMessage(t)),
	})
	require.Error(t, err)
}
