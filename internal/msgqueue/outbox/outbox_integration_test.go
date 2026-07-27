//go:build integration

package outbox_test

import (
	"context"
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

func stagedMessageCount(t *testing.T, ctx context.Context, conf *database.Layer) int {
	t.Helper()

	var count int
	err := conf.Pool.QueryRow(ctx, "SELECT count(*) FROM outbox.messages WHERE topic = $1", outbox.Topic(msgqueue.OLAP_QUEUE)).Scan(&count)
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

		delegate := &recorderMQ{}

		conf.V1.SetMessagePublisher(delegate, nil, ob, nil)

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

		assert.Equal(t, 2, stagedMessageCount(t, ctx, conf))
		assert.Empty(t, delegate.sentMessages())

		// the relay drains staged messages to the delegate queue
		relay := outbox.NewRelay(
			delegate,
			ob,
			outbox.WithQueue(msgqueue.OLAP_QUEUE),
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
			return len(delegate.sentMessages()) == 2 && stagedMessageCount(t, ctx, conf) == 0
		}, 15*time.Second, 100*time.Millisecond, "relay should republish staged messages to the delegate")

		msgIds := make([]string, 0)

		for _, s := range delegate.sentMessages() {
			assert.Equal(t, msgqueue.OLAP_QUEUE.Name(), s.queue.Name())
			assert.Equal(t, testTenantUUID, s.msg.TenantID)
			msgIds = append(msgIds, s.msg.ID)
		}

		assert.ElementsMatch(t, []string{msgqueue.MsgIDCreateMonitoringEvent, msgqueue.MsgIDCELEvaluationFailure}, msgIds)

		return nil
	})
}
