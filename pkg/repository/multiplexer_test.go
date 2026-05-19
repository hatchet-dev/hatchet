//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestMultiplexedListener_SubscribeUnsubscribe(t *testing.T) {
	logger := zerolog.Nop()
	m := &multiplexedListener{
		subscribers: make(map[string][]chan *PubSubMessage),
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
		subscribers: make(map[string][]chan *PubSubMessage),
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
		subscribers: make(map[string][]chan *PubSubMessage),
		l:           &logger,
	}

	queueName := "test-queue"
	testPayload := []byte("test-payload")

	// Subscribe to the queue
	ch1 := m.subscribe(queueName)
	ch2 := m.subscribe(queueName)

	// Create a test message
	msg := &PubSubMessage{
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
		subscribers: make(map[string][]chan *PubSubMessage),
		l:           &logger,
	}

	// Create a test message for a queue with no subscribers
	msg := &PubSubMessage{
		QueueName: "non-existent-queue",
		Payload:   []byte("test-payload"),
	}

	// This should not panic or error
	m.publishToSubscribers(msg)
}

func TestMultiplexedListener_ConcurrentAccess(t *testing.T) {
	logger := zerolog.Nop()
	m := &multiplexedListener{
		subscribers: make(map[string][]chan *PubSubMessage),
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
			msg := &PubSubMessage{
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

const testQueueName = "test-queue"

var byteOverhead []byte = []byte(`{"queue_name":"test-queue","payload":}`)
var byteOverheadSize = len(byteOverhead)

// TestPubSubMessageWrappedSize verifies that we correctly calculate the size
// of messages after wrapping in PubSubMessage, accounting for JSON marshaling
// overhead without double base64 encoding. PostgreSQL's pg_notify has an 8000 byte limit.
func TestPubSubMessageWrappedSize(t *testing.T) {
	tests := []struct {
		name             string
		queueName        string
		payloadSize      int  // size of the original JSON payload
		expectUnderLimit bool // expect to be under pg_notify's 8000 byte limit
	}{
		{
			name:             "small message stays under limit",
			queueName:        testQueueName,
			payloadSize:      1000,
			expectUnderLimit: true,
		},
		{
			name:             "7000 byte message should stay under limit",
			queueName:        testQueueName,
			payloadSize:      7000,
			expectUnderLimit: true,
		},
		{
			name:             "7999 byte message at the boundary",
			queueName:        testQueueName,
			payloadSize:      7999 - byteOverheadSize,
			expectUnderLimit: true, // 7950 + 38 byte PubSubMessage wrapper = 7988, just under 8000
		},
		{
			name:             "8001 byte message should exceed limit",
			queueName:        testQueueName,
			payloadSize:      8001 - byteOverheadSize,
			expectUnderLimit: false, // 7970 + 38 byte PubSubMessage wrapper = 8008, exceeds 8000
		},
	}

	logger := zerolog.Nop()
	m := &multiplexedListener{
		l: &logger,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a JSON payload of exactly the specified size
			payload := createJSONPayload(tt.payloadSize)

			// Use the actual wrapMessage method to wrap and marshal
			wrappedBytes, err := m.wrapMessage(tt.queueName, payload)
			if err != nil {
				t.Fatalf("failed to wrap message: %v", err)
			}

			wrappedSize := len(wrappedBytes)
			underLimit := wrappedSize <= 8000

			t.Logf("Overhead size: %d bytes", byteOverheadSize)

			t.Logf("Original payload size: %d, Wrapped size: %d", len(payload), wrappedSize)

			if underLimit != tt.expectUnderLimit {
				t.Errorf("Expected under 8000 bytes: %v, but got size: %d (under: %v)",
					tt.expectUnderLimit, wrappedSize, underLimit)
			}
		})
	}
}

// TestEmptyPayloadHandling verifies that empty payloads are handled correctly
func TestEmptyPayloadHandling(t *testing.T) {
	logger := zerolog.Nop()
	m := &multiplexedListener{
		l: &logger,
	}

	// Test wrapping an empty payload
	wrappedBytes, err := m.wrapMessage("test-queue", "")
	if err != nil {
		t.Fatalf("failed to wrap empty message: %v", err)
	}

	// Should produce valid JSON with null payload
	expected := `{"queue_name":"test-queue","payload":null}`
	actual := string(wrappedBytes)
	if actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}

	t.Logf("Empty payload wrapped as: %s", actual)
	t.Logf("Size: %d bytes", len(wrappedBytes))
}

// createJSONPayload creates a JSON string of exactly the specified size
func createJSONPayload(size int) string {
	// Create a simple JSON object with a large string field
	// The JSON structure will be: {"data":"<padding>"}
	// Structure overhead: { (1) + "data" (6) + : (1) + " (1) + " (1) + } (1) = 11 bytes
	const jsonStructureOverhead = 11
	paddingSize := size - jsonStructureOverhead
	if paddingSize < 0 {
		paddingSize = 0
	}
	padding := strings.Repeat("x", paddingSize)
	payload := map[string]string{
		"data": padding,
	}
	bytes, _ := json.Marshal(payload)
	return string(bytes)
}
