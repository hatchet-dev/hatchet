//go:build integration

package rabbitmq

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/random"
)

type testMessagePayload struct {
	Key string `json:"key"`
}

var testTenantUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func TestMessageQueueIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(1) // we wait for 1 message here

	url := "amqp://user:password@localhost:5672/"

	// Initialize the task queue implementation
	cleanup, tq, err := New(
		WithURL(url),
		WithQos(100),
		WithDeadLetterBackoff(5*time.Second),
		WithMessageRejection(false, 10), // Disable message rejection for this test
	)

	require.Nil(t, err, "error should be nil")

	require.NotNil(t, tq, "task queue implementation should not be nil")

	id, _ := random.Generate(8) // nolint: errcheck

	// Test adding a task to a static queue
	staticQueue := msgqueue.NewRandomStaticQueue()

	defer func() {
		if err := tq.deleteQueue(staticQueue); err != nil {
			t.Fatalf("error deleting queue: %v", err)
		}

		if err := cleanup(); err != nil {
			t.Fatalf("error cleaning up queue: %v", err)
		}
	}()

	task, err := msgqueue.NewTenantMessage(testTenantUUID, id, false, true, map[string]interface{}{"key": "value"})

	if err != nil {
		t.Fatalf("error creating task: %v", err)
	}

	preAck := func(receivedMessage *msgqueue.Message) error {
		defer wg.Done()
		assert.Equal(t, task.ID, receivedMessage.ID, "received task ID should match sent task ID")
		return nil
	}

	// Test subscription to the static queue
	cleanupQueue, err := tq.Subscribe(staticQueue, preAck, msgqueue.NoOpHook)
	require.NoError(t, err, "subscribing to static queue should not error")

	// send task to queue after 1 second to give time for subscriber
	go func() {
		time.Sleep(1 * time.Second)
		err = tq.SendMessage(ctx, staticQueue, task)
		assert.NoError(t, err, "adding task to queue should not error")
	}()

	wg.Wait()

	if err := cleanupQueue(); err != nil {
		t.Fatalf("error cleaning up queue: %v", err)
	}
}

func TestBufferedSubMessageQueueIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(10) // we wait for 10 messages here

	url := "amqp://user:password@localhost:5672/"

	// Initialize the task queue implementation
	cleanup, tq, err := New(
		WithURL(url),
		WithQos(100),
		WithDeadLetterBackoff(5*time.Second),
	)

	require.Nil(t, err, "error should be nil")

	require.NotNil(t, tq, "task queue implementation should not be nil")

	id, _ := random.Generate(8) // nolint: errcheck

	// Test adding a task to a static queue
	staticQueue := msgqueue.NewRandomStaticQueue()

	defer func() {
		if err := tq.deleteQueue(staticQueue); err != nil {
			t.Fatalf("error deleting queue: %v", err)
		}

		if err := cleanup(); err != nil {
			t.Fatalf("error cleaning up queue: %v", err)
		}
	}()

	mqBuffer := msgqueue.NewMQSubBuffer(staticQueue, tq, func(tenantId uuid.UUID, msgId string, payloads [][]byte) error {
		msgs := msgqueue.JSONConvert[testMessagePayload](payloads)

		for _, msg := range msgs {
			assert.Equal(t, "value", msg.Key, "received task payload should match sent task payload")
			wg.Done()
		}

		return nil
	})

	cleanupQueue, err := mqBuffer.Start()

	if err != nil {
		t.Fatalf("error starting buffer: %v", err)
	}

	task, err := msgqueue.NewTenantMessage(testTenantUUID, id, false, true, &testMessagePayload{
		Key: "value",
	})

	if err != nil {
		t.Fatalf("error creating task: %v", err)
	}

	// send tasks to queue
	for i := 0; i < 10; i++ {
		err = tq.SendMessage(ctx, staticQueue, task)

		if err != nil {
			t.Fatalf("error sending task: %v", err)
		}
	}

	wg.Wait()

	if err := cleanupQueue(); err != nil {
		t.Fatalf("error cleaning up queue: %v", err)
	}
}

func TestBufferedPubMessageQueueIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(10)

	url := "amqp://user:password@localhost:5672/"

	// Initialize the task queue implementation
	cleanup, tq, err := New(
		WithURL(url),
		WithQos(100),
		WithDeadLetterBackoff(5*time.Second),
	)

	require.Nil(t, err, "error should be nil")

	require.NotNil(t, tq, "task queue implementation should not be nil")

	id, _ := random.Generate(8) // nolint: errcheck

	// Test adding a task to a static queue
	staticQueue := msgqueue.NewRandomStaticQueue()

	defer func() {
		if err := tq.deleteQueue(staticQueue); err != nil {
			t.Fatalf("error deleting queue: %v", err)
		}

		if err := cleanup(); err != nil {
			t.Fatalf("error cleaning up queue: %v", err)
		}
	}()

	cleanupQueue, err := tq.Subscribe(staticQueue, func(receivedMessage *msgqueue.Message) error {
		for range receivedMessage.Payloads {
			wg.Done()
		}

		return nil
	}, msgqueue.NoOpHook)

	if err != nil {
		t.Fatalf("error subscribing to queue: %v", err)
	}

	pub := msgqueue.NewMQPubBuffer(tq)
	defer pub.Stop()

	task, err := msgqueue.NewTenantMessage(testTenantUUID, id, false, true, &testMessagePayload{
		Key: "value",
	})

	if err != nil {
		t.Fatalf("error creating task: %v", err)
	}

	// send tasks to queue
	for i := 0; i < 10; i++ {
		err := pub.Pub(ctx, staticQueue, task, false)

		if err != nil {
			t.Fatalf("error sending task: %v", err)
		}
	}

	wg.Wait()

	if err := cleanupQueue(); err != nil {
		t.Fatalf("error cleaning up queue: %v", err)
	}
}

func TestDeadLetteringSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var attempts int
	wg := &sync.WaitGroup{}
	wg.Add(1) // we wait for the message to be processed successfully

	url := "amqp://user:password@localhost:5672/"

	// Initialize the task queue implementation
	cleanup, tq, err := New(
		WithURL(url),
		WithQos(100),
		WithDeadLetterBackoff(5*time.Second),
		WithMessageRejection(false, 10), // Disable message rejection for this test
	)

	require.Nil(t, err, "error should be nil")

	require.NotNil(t, tq, "task queue implementation should not be nil")

	id, _ := random.Generate(8) // nolint: errcheck

	// Test adding a task to a static queue
	staticQueue := msgqueue.NewRandomStaticQueue()

	defer func() {
		if err := tq.deleteQueue(staticQueue); err != nil {
			t.Fatalf("error deleting queue: %v", err)
		}

		if err := cleanup(); err != nil {
			t.Fatalf("error cleaning up queue: %v", err)
		}
	}()

	task, err := msgqueue.NewTenantMessage(testTenantUUID, id, false, true, &testMessagePayload{
		Key: "value",
	})

	if err != nil {
		t.Fatalf("error creating task: %v", err)
	}

	task.Retries = 2

	preAck := func(receivedMessage *msgqueue.Message) error {
		if receivedMessage.ID != task.ID {
			return nil
		}

		attempts++
		if attempts <= 2 {
			return fmt.Errorf("intentional error on attempt %d", attempts)
		}

		assert.Equal(t, task.ID, receivedMessage.ID, "received task ID should match sent task ID")
		wg.Done()
		return nil
	}

	// Test subscription to the static queue
	cleanupQueue, err := tq.Subscribe(staticQueue, preAck, msgqueue.NoOpHook)
	require.NoError(t, err, "subscribing to static queue should not error")

	err = tq.SendMessage(ctx, staticQueue, task)
	assert.NoError(t, err, "adding task to static queue should not error")

	wg.Wait()

	if err := cleanupQueue(); err != nil {
		t.Fatalf("error cleaning up queue: %v", err)
	}
}

func TestRetiredQueueDeletedOnceUnused(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	url := "amqp://user:password@localhost:5672/"

	start := func() *MessageQueueImpl {
		cleanup, mq, err := New(WithURL(url), WithQos(100))
		require.NoError(t, err, "a retired queue must never fail startup")

		t.Cleanup(func() {
			assert.NoError(t, cleanup())
		})

		return mq
	}

	conn, err := amqp.Dial(url)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, conn.Close())
	})

	// each check gets its own channel: a passive declare of a missing queue closes it
	inspect := func(name string) (amqp.Queue, error) {
		ch, err := conn.Channel()
		require.NoError(t, err)

		defer ch.Close() // nolint: errcheck

		return ch.QueueDeclarePassive(name, true, false, false, false, nil)
	}

	q := msgqueue.NewRandomStaticQueue()
	names := []string{q.Name(), getTmpDLQName(q.DLQ().Name()), getProcDLQName(q.DLQ().Name())}

	retired := msgqueue.RetiredQueues
	msgqueue.RetiredQueues = []msgqueue.Queue{q}

	t.Cleanup(func() {
		msgqueue.RetiredQueues = retired
	})

	// an older version still reads the queue
	older := start()
	received := make(chan struct{}, 1)

	cleanupSub, err := older.Subscribe(q, func(*msgqueue.Message) error {
		received <- struct{}{}
		return nil
	}, msgqueue.NoOpHook)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		main, err := inspect(q.Name())
		return err == nil && main.Consumers > 0
	}, 10*time.Second, 100*time.Millisecond)

	newer := start()

	for _, name := range names {
		_, err := inspect(name)
		assert.NoError(t, err, "%s should be kept while the queue is read", name)
	}

	msg, err := msgqueue.NewTenantMessage(testTenantUUID, "test", false, true, map[string]interface{}{"key": "value"})
	require.NoError(t, err)
	require.NoError(t, newer.SendMessage(ctx, q, msg))

	select {
	case <-received:
	case <-ctx.Done():
		t.Fatal("the older version stopped receiving")
	}

	require.NoError(t, cleanupSub())

	require.Eventually(t, func() bool {
		main, mainErr := inspect(q.Name())
		proc, procErr := inspect(names[2])

		return mainErr == nil && procErr == nil && main.Consumers == 0 && proc.Consumers == 0
	}, 10*time.Second, 100*time.Millisecond)

	newest := start()

	for _, name := range names {
		_, err := inspect(name)
		assert.Error(t, err, "%s should be deleted", name)
	}

	require.NoError(t, newest.deleteRetiredQueue(ctx, q), "deleting queues that are already gone should succeed")
}

// deleteQueue is a helper function for removing durable queues which are used for tests.
func (t *MessageQueueImpl) deleteQueue(q msgqueue.Queue) error {
	ctx := context.Background()
	poolCh, err := t.subChannels.Acquire(ctx)

	if err != nil {
		t.l.Error().Ctx(ctx).Msgf("[deleteQueue] cannot acquire channel for deleting queue: %v", err)
		return err
	}

	ch := poolCh.Value()

	if ch.IsClosed() {
		poolCh.Destroy()
		return fmt.Errorf("channel is closed")
	}

	defer poolCh.Release()

	_, err = ch.QueueDelete(q.Name(), true, true, false)

	if err != nil {
		t.l.Error().Ctx(ctx).Msgf("cannot delete queue: %q, %v", q.Name(), err)
		return err
	}

	if q.DLQ().Name() != "" {
		dlq1 := getTmpDLQName(q.DLQ().Name())
		dlq2 := getProcDLQName(q.DLQ().Name())

		_, err = ch.QueueDelete(dlq1, true, true, false)

		if err != nil {
			t.l.Error().Ctx(ctx).Msgf("cannot delete dead letter queue: %q, %v", dlq1, err)
			return err
		}

		_, err = ch.QueueDelete(dlq2, true, true, false)

		if err != nil {
			t.l.Error().Ctx(ctx).Msgf("cannot delete dead letter queue: %q, %v", dlq2, err)
			return err
		}
	}

	return nil
}
