//go:build integration

package rabbitmq_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/msgqueue/rabbitmq"
	"github.com/hatchet-dev/hatchet/pkg/random"
)

func TestMessageQueueIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(2) // we wait for 2 messages here

	url := "amqp://user:password@localhost:5672/"

	// Initialize the task queue implementation
	cleanup, tq := rabbitmq.New(
		rabbitmq.WithURL(url),
		rabbitmq.WithQos(100),
	)
	defer cleanup() // nolint: errcheck

	require.NotNil(t, tq, "task queue implementation should not be nil")

	id, _ := random.Generate(8) // nolint: errcheck

	// Test adding a task to a static queue
	staticQueue := msgqueue.EVENT_PROCESSING_QUEUE
	task := &msgqueue.Message{
		ID:         id,
		Payload:    map[string]interface{}{"key": "value"},
		Metadata:   map[string]interface{}{"tenant_id": "test-tenant"},
		Retries:    1,
		RetryDelay: 5,
	}

	err := tq.AddMessage(ctx, staticQueue, task)
	assert.NoError(t, err, "adding task to static queue should not error")

	preAck := func(receivedMessage *msgqueue.Message) error {
		defer wg.Done()
		assert.Equal(t, task.ID, receivedMessage.ID, "received task ID should match sent task ID")
		return nil
	}

	// Test subscription to the static queue
	cleanupQueue, err := tq.Subscribe(staticQueue, preAck, msgqueue.NoOpHook)
	require.NoError(t, err, "subscribing to static queue should not error")

	// Test tenant registration and queue creation
	tenantId := "test-tenant"
	err = tq.RegisterTenant(ctx, tenantId)
	assert.NoError(t, err, "registering tenant should not error")

	// Assuming there's a mechanism to retrieve a tenant-specific queue, e.g., by tenant ID
	tenantQueue := msgqueue.TenantEventConsumerQueue(tenantId)

	if err != nil {
		t.Fatalf("error creating tenant-specific queue: %v", err)
	}

	tqAck := func(receivedMessage *msgqueue.Message) error {
		defer wg.Done()
		assert.Equal(t, task.ID, receivedMessage.ID, "received tenant task ID should match sent task ID")
		return nil
	}

	// Test subscription to the tenant-specific queue
	cleanupTenantQueue, err := tq.Subscribe(tenantQueue, tqAck, msgqueue.NoOpHook)
	require.NoError(t, err, "subscribing to tenant-specific queue should not error")

	// send task to tenant-specific queue after 1 second to give time for subscriber
	go func() {
		time.Sleep(1 * time.Second)
		err = tq.AddMessage(ctx, tenantQueue, task)
		assert.NoError(t, err, "adding task to tenant-specific queue should not error")
	}()

	wg.Wait()

	if err := cleanupQueue(); err != nil {
		t.Fatalf("error cleaning up queue: %v", err)
	}
	if err := cleanupTenantQueue(); err != nil {
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
	cleanup, tq := rabbitmq.New(
		rabbitmq.WithURL(url),
		rabbitmq.WithQos(100),
	)
	defer cleanup() // nolint: errcheck

	require.NotNil(t, tq, "task queue implementation should not be nil")

	id, _ := random.Generate(8) // nolint: errcheck

	// Test adding a task to a static queue
	staticQueue := msgqueue.EVENT_PROCESSING_QUEUE
	task := &msgqueue.Message{
		ID:         id,
		Payload:    map[string]interface{}{"key": "value"},
		Metadata:   map[string]interface{}{"tenant_id": "test-tenant"},
		Retries:    2, // Allow up to 2 retries
		RetryDelay: 5, // Delay between retries in seconds
	}

	err := tq.AddMessage(ctx, staticQueue, task)
	assert.NoError(t, err, "adding task to static queue should not error")

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

	wg.Wait()

	if err := cleanupQueue(); err != nil {
		t.Fatalf("error cleaning up queue: %v", err)
	}
}

func TestDeadLetteringExceedRetriesFailure(t *testing.T) {
	// this falls under time threshold but over time in the DLQ
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var attempts int

	url := "amqp://user:password@localhost:5672/"

	// Initialize the task queue implementation
	cleanup, tq := rabbitmq.New(
		rabbitmq.WithURL(url),
		rabbitmq.WithQos(100),
	)
	defer cleanup() // nolint: errcheck

	require.NotNil(t, tq, "task queue implementation should not be nil")

	id, _ := random.Generate(8) // nolint: errcheck

	// Test adding a task to a static queue
	staticQueue := msgqueue.EVENT_PROCESSING_QUEUE
	task := &msgqueue.Message{
		ID:         id,
		Payload:    map[string]interface{}{"key": "value"},
		Metadata:   map[string]interface{}{"tenant_id": "test-tenant"},
		Retries:    2, // Allow up to 2 retries
		RetryDelay: 5, // Delay between retries in seconds
	}

	err := tq.AddMessage(ctx, staticQueue, task)
	assert.NoError(t, err, "adding task to static queue should not error")

	preAck := func(receivedMessage *msgqueue.Message) error {
		// only process messages which match the id
		if receivedMessage.ID != task.ID {
			return nil
		}

		if attempts > 2 {
			assert.Fail(t, "message exceeded maximum retry count")
			cancel()
			return nil // Stop retrying as it exceeds the limit
		}

		attempts++

		return fmt.Errorf("intentional error on attempt %d", attempts)
	}

	// Test subscription to the static queue
	cleanupQueue, err := tq.Subscribe(staticQueue, preAck, msgqueue.NoOpHook)
	require.NoError(t, err, "subscribing to static queue should not error")

	<-ctx.Done()

	if err := cleanupQueue(); err != nil {
		t.Fatalf("error cleaning up queue: %v", err)
	}
}
