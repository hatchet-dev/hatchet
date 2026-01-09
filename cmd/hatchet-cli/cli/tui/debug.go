package tui

import (
	"fmt"
	"sync"
	"time"
)

// DebugLog represents a single debug log entry
type DebugLog struct {
	Timestamp time.Time
	Message   string
}

// DebugLogger is a fixed-size ring buffer for debug logs
type DebugLogger struct {
	mu       sync.RWMutex
	logs     []DebugLog
	capacity int
	index    int
	size     int
}

// NewDebugLogger creates a new debug logger with the specified capacity
func NewDebugLogger(capacity int) *DebugLogger {
	return &DebugLogger{
		logs:     make([]DebugLog, capacity),
		capacity: capacity,
		index:    0,
		size:     0,
	}
}

// Log adds a new log entry to the ring buffer
func (d *DebugLogger) Log(format string, args ...interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logs[d.index] = DebugLog{
		Timestamp: time.Now(),
		Message:   fmt.Sprintf(format, args...),
	}

	d.index = (d.index + 1) % d.capacity
	if d.size < d.capacity {
		d.size++
	}
}

// GetLogs returns all logs in chronological order
func (d *DebugLogger) GetLogs() []DebugLog {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.size == 0 {
		return []DebugLog{}
	}

	result := make([]DebugLog, d.size)

	if d.size < d.capacity {
		// Buffer not full yet, logs are from 0 to index-1
		copy(result, d.logs[:d.size])
	} else {
		// Buffer is full, logs wrap around
		// Copy from index to end (older logs)
		n := copy(result, d.logs[d.index:])
		// Copy from start to index (newer logs)
		copy(result[n:], d.logs[:d.index])
	}

	return result
}

// Clear removes all logs
func (d *DebugLogger) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.index = 0
	d.size = 0
}

// Size returns the current number of logs
func (d *DebugLogger) Size() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.size
}

// Capacity returns the maximum capacity
func (d *DebugLogger) Capacity() int {
	return d.capacity
}
