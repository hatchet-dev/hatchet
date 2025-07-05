//go:build !e2e && !load && !rampup && !integration

package postgres

import (
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
)

func TestMultiplexedListener_SubscribeUnsubscribe(t *testing.T) {
	logger := zerolog.Nop()
	m := &multiplexedListener{
		subscribers: make(map[string][]chan *repository.PubSubMessage),
		l:           &logger,
	}

	// Test subscribing to a queue
	queueName := "test-queue"
	ch := m.subscribe(queueName)

	if ch == nil {
		t.Fatal("Expected channel to be returned")
	}

	// Check that the subscriber was added
	m.subscribersMu.RLock()
	subscribers, exists := m.subscribers[queueName]
	m.subscribersMu.RUnlock()

	if !exists {
		t.Fatal("Expected queue to exist in subscribers map")
	}

	if len(subscribers) != 1 {
		t.Fatalf("Expected 1 subscriber, got %d", len(subscribers))
	}

	// Test unsubscribing
	m.unsubscribe(queueName, ch)

	m.subscribersMu.RLock()
	_, exists = m.subscribers[queueName]
	m.subscribersMu.RUnlock()

	if exists {
		t.Fatal("Expected queue to be removed from subscribers map after unsubscribe")
	}

	// Channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("Expected channel to be closed")
		}
	default:
		t.Fatal("Expected channel to be closed")
	}
}

func TestMultiplexedListener_MultipleSubscribers(t *testing.T) {
	logger := zerolog.Nop()
	m := &multiplexedListener{
		subscribers: make(map[string][]chan *repository.PubSubMessage),
		l:           &logger,
	}

	queueName := "test-queue"

	// Subscribe multiple channels to the same queue
	ch1 := m.subscribe(queueName)
	ch2 := m.subscribe(queueName)
	ch3 := m.subscribe(queueName)

	// Check that all subscribers were added
	m.subscribersMu.RLock()
	subscribers, exists := m.subscribers[queueName]
	m.subscribersMu.RUnlock()

	if !exists {
		t.Fatal("Expected queue to exist in subscribers map")
	}

	if len(subscribers) != 3 {
		t.Fatalf("Expected 3 subscribers, got %d", len(subscribers))
	}

	// Unsubscribe one channel
	m.unsubscribe(queueName, ch2)

	m.subscribersMu.RLock()
	subscribers, exists = m.subscribers[queueName]
	m.subscribersMu.RUnlock()

	if !exists {
		t.Fatal("Expected queue to still exist in subscribers map")
	}

	if len(subscribers) != 2 {
		t.Fatalf("Expected 2 subscribers after unsubscribe, got %d", len(subscribers))
	}

	// Clean up remaining channels
	m.unsubscribe(queueName, ch1)
	m.unsubscribe(queueName, ch3)

	m.subscribersMu.RLock()
	_, exists = m.subscribers[queueName]
	m.subscribersMu.RUnlock()

	if exists {
		t.Fatal("Expected queue to be removed from subscribers map after all unsubscribes")
	}
}

func TestMultiplexedListener_PublishToSubscribers(t *testing.T) {
	logger := zerolog.Nop()
	m := &multiplexedListener{
		subscribers: make(map[string][]chan *repository.PubSubMessage),
		l:           &logger,
	}

	queueName := "test-queue"
	testPayload := []byte("test-payload")

	// Subscribe to the queue
	ch1 := m.subscribe(queueName)
	ch2 := m.subscribe(queueName)

	// Create a test message
	msg := &repository.PubSubMessage{
		QueueName: queueName,
		Payload:   testPayload,
	}

	// Publish the message
	m.publishToSubscribers(msg)

	// Both subscribers should receive the message
	select {
	case receivedMsg := <-ch1:
		if receivedMsg.QueueName != queueName {
			t.Errorf("Expected queue name %s, got %s", queueName, receivedMsg.QueueName)
		}
		if string(receivedMsg.Payload) != string(testPayload) {
			t.Errorf("Expected payload %s, got %s", string(testPayload), string(receivedMsg.Payload))
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected to receive message on ch1")
	}

	select {
	case receivedMsg := <-ch2:
		if receivedMsg.QueueName != queueName {
			t.Errorf("Expected queue name %s, got %s", queueName, receivedMsg.QueueName)
		}
		if string(receivedMsg.Payload) != string(testPayload) {
			t.Errorf("Expected payload %s, got %s", string(testPayload), string(receivedMsg.Payload))
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected to receive message on ch2")
	}

	// Clean up
	m.unsubscribe(queueName, ch1)
	m.unsubscribe(queueName, ch2)
}

func TestMultiplexedListener_PublishToNonExistentQueue(t *testing.T) {
	logger := zerolog.Nop()
	m := &multiplexedListener{
		subscribers: make(map[string][]chan *repository.PubSubMessage),
		l:           &logger,
	}

	// Create a test message for a queue with no subscribers
	msg := &repository.PubSubMessage{
		QueueName: "non-existent-queue",
		Payload:   []byte("test-payload"),
	}

	// This should not panic or error
	m.publishToSubscribers(msg)
}

func TestMultiplexedListener_ConcurrentAccess(t *testing.T) {
	logger := zerolog.Nop()
	m := &multiplexedListener{
		subscribers: make(map[string][]chan *repository.PubSubMessage),
		l:           &logger,
	}

	queueName := "test-queue"
	numGoroutines := 10
	messagesPerGoroutine := 10 // Reduced for faster test

	var wg sync.WaitGroup
	var setupWg sync.WaitGroup
	receivedCount := int64(0)
	var mu sync.Mutex

	// Start multiple subscribers
	for i := range numGoroutines {
		_ = i // Use the variable to avoid unused variable warning
		wg.Add(1)
		setupWg.Add(1)
		go func() {
			defer wg.Done()
			ch := m.subscribe(queueName)
			defer m.unsubscribe(queueName, ch)

			// Signal that this subscriber is ready
			setupWg.Done()

			for range messagesPerGoroutine {
				select {
				case <-ch:
					mu.Lock()
					receivedCount++
					mu.Unlock()
				case <-time.After(1 * time.Second):
					t.Errorf("Timeout waiting for message")
					return
				}
			}
		}()
	}

	// Wait for all subscribers to be set up
	setupWg.Wait()

	// Start publisher
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range messagesPerGoroutine {
			msg := &repository.PubSubMessage{
				QueueName: queueName,
				Payload:   []byte("test-payload"),
			}
			m.publishToSubscribers(msg)
			// Small delay to allow message processing
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()

	// Each message should be received by all subscribers
	expectedCount := int64(numGoroutines * messagesPerGoroutine)
	mu.Lock()
	actualCount := receivedCount
	mu.Unlock()

	if actualCount != expectedCount {
		t.Errorf("Expected %d messages received, got %d", expectedCount, actualCount)
	}
}
