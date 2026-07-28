//go:build integration

package outbox_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/pgoutbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/msgqueue/outbox"
	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

var testTenantUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type sentMessage struct {
	queue msgqueue.Queue
	msg   *msgqueue.Message
}

type recorderMQ struct {
	mu   sync.Mutex
	sent []sentMessage
}

func (f *recorderMQ) Clone() (func() error, msgqueue.MessageQueue, error) {
	return func() error { return nil }, f, nil
}

func (f *recorderMQ) SetQOS(prefetchCount int) {}

func (f *recorderMQ) SendMessage(ctx context.Context, queue msgqueue.Queue, msg *msgqueue.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sent = append(f.sent, sentMessage{queue: queue, msg: msg})
	return nil
}

func (f *recorderMQ) Subscribe(queue msgqueue.Queue, preAck msgqueue.MsgHandler, postAck msgqueue.MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}

func (f *recorderMQ) RegisterTenant(ctx context.Context, tenantId uuid.UUID) error {
	return nil
}

func (f *recorderMQ) IsReady() bool {
	return true
}

func (f *recorderMQ) sentMessages() []sentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]sentMessage{}, f.sent...)
}

func stagedMessageCount(t *testing.T, ctx context.Context, conf *database.Layer, topic string) int {
	t.Helper()

	var count int
	err := conf.Pool.QueryRow(ctx, "SELECT count(*) FROM outbox.messages WHERE topic = $1", topic).Scan(&count)
	require.NoError(t, err)

	return count
}

func TestOLAPOutboxRelayIntegration(t *testing.T) {
	// `internal/testutils.Prepare` constructs a server config and requires a RabbitMQ URL.
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// the outbox tables are created by migrations, so auto-migration is disabled
		ob, err := pgoutbox.NewOutbox(t.Context(), conf.Pool, pgoutbox.WithAutoMigrate(false))
		require.NoError(t, err)

		mq := &recorderMQ{}

		conf.V1.SetMessagePublisher(nil, ob, nil)

		// staging via the repository's OLAP outbox writes durable rows instead of
		// publishing directly
		err = conf.V1.OLAPOutbox().MonitoringEvents(ctx, testTenantUUID, v1.CreateMonitoringEventPayload{
			TaskId:         1,
			EventType:      sqlcv1.V1EventTypeOlapQUEUED,
			EventTimestamp: time.Now(),
			EventPayload:   "test",
		})
		require.NoError(t, err)

		err = conf.V1.OLAPOutbox().CELEvaluationFailures(ctx, testTenantUUID, v1.CELEvaluationFailure{
			Source:       sqlcv1.V1CelEvaluationFailureSourceIDEMPOTENCYKEY,
			ErrorMessage: "test failure",
		})
		require.NoError(t, err)

		// internal task events are staged under the task processing queue's topic on
		// a caller transaction (as the repository's signaler does)
		internalEventMsg, err := v1.NewInternalEventMessage(testTenantUUID, v1.InternalTaskEvent{
			TenantID:  testTenantUUID,
			TaskID:    1,
			EventType: sqlcv1.V1TaskEventTypeFAILED,
		})
		require.NoError(t, err)

		internalEventBody, err := json.Marshal(internalEventMsg)
		require.NoError(t, err)

		tx, err := conf.Pool.Begin(ctx)
		require.NoError(t, err)

		require.NoError(t, ob.AddMessages(ctx, tx, outbox.Topic(msgqueue.TASK_PROCESSING_QUEUE), []pgoutbox.MessageOpts{{Payload: internalEventBody}}))
		require.NoError(t, tx.Commit(ctx))

		assert.Equal(t, 2, stagedMessageCount(t, ctx, conf, outbox.Topic(msgqueue.OLAP_QUEUE)))
		assert.Equal(t, 1, stagedMessageCount(t, ctx, conf, outbox.Topic(msgqueue.TASK_PROCESSING_QUEUE)))
		assert.Empty(t, mq.sentMessages())

		// the relay drains staged messages to the mq queue
		relay := outbox.NewRelay(
			mq,
			ob,
			outbox.WithQueue(msgqueue.OLAP_QUEUE),
			outbox.WithQueue(msgqueue.TASK_PROCESSING_QUEUE),
			outbox.WithSubscribeConfig(100, 500*time.Millisecond),
		)

		cleanupRelay, err := relay.Start()
		require.NoError(t, err)

		defer func() {
			if err := cleanupRelay(); err != nil {
				t.Fatalf("error cleaning up relay: %v", err)
			}
		}()

		require.Eventually(t, func() bool {
			return len(mq.sentMessages()) == 3 &&
				stagedMessageCount(t, ctx, conf, outbox.Topic(msgqueue.OLAP_QUEUE)) == 0 &&
				stagedMessageCount(t, ctx, conf, outbox.Topic(msgqueue.TASK_PROCESSING_QUEUE)) == 0
		}, 15*time.Second, 100*time.Millisecond, "relay should republish staged messages to the mq")

		queueNamesByMsgId := make(map[string]string)

		for _, s := range mq.sentMessages() {
			assert.Equal(t, testTenantUUID, s.msg.TenantID)
			queueNamesByMsgId[s.msg.ID] = s.queue.Name()
		}

		assert.Equal(t, map[string]string{
			msgqueue.MsgIDCreateMonitoringEvent: msgqueue.OLAP_QUEUE.Name(),
			msgqueue.MsgIDCELEvaluationFailure:  msgqueue.OLAP_QUEUE.Name(),
			msgqueue.MsgIDInternalEvent:         msgqueue.TASK_PROCESSING_QUEUE.Name(),
		}, queueNamesByMsgId)

		return nil
	})
}
