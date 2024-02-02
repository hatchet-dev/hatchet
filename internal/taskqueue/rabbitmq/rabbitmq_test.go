//go:build integration

package rabbitmq_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/taskqueue"
	"github.com/hatchet-dev/hatchet/internal/taskqueue/rabbitmq"
)

func TestTaskQueueIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := "amqp://user:password@localhost:5672/"

	// Initialize the task queue implementation
	tq := rabbitmq.New(ctx,
		rabbitmq.WithURL(url),
	)

	require.NotNil(t, tq, "task queue implementation should not be nil")

	// Test adding a task to a static queue
	staticQueue := taskqueue.EVENT_PROCESSING_QUEUE
	task := &taskqueue.Task{
		ID:         "test-task-id",
		Payload:    map[string]interface{}{"key": "value"},
		Metadata:   map[string]interface{}{"tenant_id": "test-tenant"},
		Retries:    1,
		RetryDelay: 5,
	}

	err := tq.AddTask(ctx, staticQueue, task)
	assert.NoError(t, err, "adding task to static queue should not error")

	// Test subscription to the static queue
	taskChan, err := tq.Subscribe(ctx, staticQueue)
	require.NoError(t, err, "subscribing to static queue should not error")

	select {
	case receivedTask := <-taskChan:
		assert.Equal(t, task.ID, receivedTask.ID, "received task ID should match sent task ID")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for task from static queue")
	}

	// Test tenant registration and queue creation
	tenantId := "test-tenant"
	err = tq.RegisterTenant(ctx, tenantId)
	assert.NoError(t, err, "registering tenant should not error")

	// Assuming there's a mechanism to retrieve a tenant-specific queue, e.g., by tenant ID
	tenantQueue, err := taskqueue.TenantEventConsumerQueue(tenantId)

	if err != nil {
		t.Fatalf("error creating tenant-specific queue: %v", err)
	}

	// Test subscription to the tenant-specific queue
	tenantTaskChan, err := tq.Subscribe(ctx, tenantQueue)
	require.NoError(t, err, "subscribing to tenant-specific queue should not error")

	// send task to tenant-specific queue after 1 second to give time for subscriber
	go func() {
		time.Sleep(1 * time.Second)
		err = tq.AddTask(ctx, tenantQueue, task)
		assert.NoError(t, err, "adding task to tenant-specific queue should not error")
	}()

	select {
	case receivedTask := <-tenantTaskChan:
		assert.Equal(t, task.ID, receivedTask.ID, "received tenant task ID should match sent task ID")
		break
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for task from tenant-specific queue")
		break
	}
}
