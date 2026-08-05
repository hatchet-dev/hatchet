//go:build integration

package rabbitmq

import (
	"context"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

const testAMQPURL = "amqp://user:password@localhost:5672/"

func newTestPubSub(t *testing.T, fs ...PubSubOpt) *PubSub {
	t.Helper()

	opts := append([]PubSubOpt{WithPubSubURL(testAMQPURL)}, fs...)

	cleanup, ps, err := NewPubSub(opts...)
	require.NoError(t, err)
	require.NotNil(t, ps)

	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Errorf("error cleaning up pubsub: %v", err)
		}
	})

	return ps
}

func TestPubSubTenantFanout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps := newTestPubSub(t)

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	wg := &sync.WaitGroup{}
	wg.Add(2) // both subscribers must receive the message

	msg, err := msgqueue.NewTenantMessage(tenantId, "task-completed", true, false, map[string]interface{}{"key": "value"})
	require.NoError(t, err)

	handler := func(received *msgqueue.Message) error {
		defer wg.Done()
		assert.Equal(t, msg.ID, received.ID)
		return nil
	}

	cleanupSub1, err := ps.Sub(topic, handler)
	require.NoError(t, err)

	cleanupSub2, err := ps.Sub(topic, handler)
	require.NoError(t, err)

	// give the exclusive queues time to bind
	time.Sleep(1 * time.Second)

	require.NoError(t, ps.Pub(ctx, topic, msg))

	wg.Wait()

	require.NoError(t, cleanupSub1())
	require.NoError(t, cleanupSub2())
}

func TestPubSubAtMostOnce(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps := newTestPubSub(t)

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	msg, err := msgqueue.NewTenantMessage(tenantId, "task-completed", true, false, map[string]interface{}{"key": "value"})
	require.NoError(t, err)

	// publish with no subscriber: the message must be dropped
	require.NoError(t, ps.Pub(ctx, topic, msg))

	received := make(chan struct{}, 1)

	cleanupSub, err := ps.Sub(topic, func(received2 *msgqueue.Message) error {
		received <- struct{}{}
		return nil
	})
	require.NoError(t, err)

	select {
	case <-received:
		t.Fatal("message published before subscription should never be delivered")
	case <-time.After(3 * time.Second):
	}

	require.NoError(t, cleanupSub())
}

func TestPubSubSchedulerTopicRoundtrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps := newTestPubSub(t)

	topic := msgqueue.SchedulerPartitionTopic(uuid.NewString())

	wg := &sync.WaitGroup{}
	wg.Add(1)

	msg, err := msgqueue.NewTenantMessage(uuid.New(), "check-tenant-queue", true, false, map[string]interface{}{"key": "value"})
	require.NoError(t, err)

	cleanupSub, err := ps.Sub(topic, func(received *msgqueue.Message) error {
		defer wg.Done()
		assert.Equal(t, msg.ID, received.ID)
		return nil
	})
	require.NoError(t, err)

	// give the exclusive queue time to be declared
	time.Sleep(1 * time.Second)

	require.NoError(t, ps.Pub(ctx, topic, msg))

	wg.Wait()

	require.NoError(t, cleanupSub())
}

func TestPubSubCompressedRoundtrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// threshold of 1 byte forces compression
	ps := newTestPubSub(t, WithPubSubGzip(true, 1))

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	payload := make([]byte, 32*1024)
	_, err := rand.Read(payload)
	require.NoError(t, err)

	msg, err := msgqueue.NewTenantMessage(tenantId, "task-stream-event", true, false, map[string]interface{}{"data": payload})
	require.NoError(t, err)

	originalPayload := make([]byte, len(msg.Payloads[0]))
	copy(originalPayload, msg.Payloads[0])

	wg := &sync.WaitGroup{}
	wg.Add(1)

	cleanupSub, err := ps.Sub(topic, func(received *msgqueue.Message) error {
		defer wg.Done()
		assert.Equal(t, msg.ID, received.ID)
		require.Len(t, received.Payloads, 1)
		assert.Equal(t, originalPayload, received.Payloads[0], "payload should be transparently decompressed")
		return nil
	})
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	require.NoError(t, ps.Pub(ctx, topic, msg))

	wg.Wait()

	require.NoError(t, cleanupSub())
}

func TestPubSubReconnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ps := newTestPubSub(t)

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	receivedCh := make(chan string, 10)

	cleanupSub, err := ps.Sub(topic, func(received *msgqueue.Message) error {
		receivedCh <- received.ID
		return nil
	})
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	msg1, err := msgqueue.NewTenantMessage(tenantId, "task-completed", true, false, map[string]interface{}{"key": "one"})
	require.NoError(t, err)

	require.NoError(t, ps.Pub(ctx, topic, msg1))

	select {
	case id := <-receivedCh:
		assert.Equal(t, "task-completed", id)
	case <-ctx.Done():
		t.Fatal("timed out waiting for first message")
	}

	// force-close the subscriber connection; the channel pool reconnects and the
	// subscriber loop re-declares its queue and resubscribes
	require.NoError(t, ps.subChannels.getConnection().Close())

	// wait for the pool health check (1s tick) to redial and the subscriber to recover
	require.Eventually(t, func() bool {
		return ps.subChannels.hasActiveConnection()
	}, 15*time.Second, 250*time.Millisecond)

	time.Sleep(2 * time.Second)

	msg2, err := msgqueue.NewTenantMessage(tenantId, "task-failed", true, false, map[string]interface{}{"key": "two"})
	require.NoError(t, err)

	require.NoError(t, ps.Pub(ctx, topic, msg2))

	select {
	case id := <-receivedCh:
		assert.Equal(t, "task-failed", id)
	case <-ctx.Done():
		t.Fatal("timed out waiting for message after reconnect")
	}

	require.NoError(t, cleanupSub())
}
