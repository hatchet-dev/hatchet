//go:build !e2e && !load && !rampup && !integration

package operator

import (
	"testing"
	"time"
)

// TestRecordTaskDrainsOnCleanup verifies Cleanup blocks until every recorded task has been
// released, and that RecordTask is a no-op once shutdown has begun.
func TestRecordTaskDrainsOnCleanup(t *testing.T) {
	s := &SharedOperator[struct{}]{}

	release := s.RecordTask()

	cleanupDone := make(chan struct{})

	go func() {
		s.Cleanup()
		close(cleanupDone)
	}()

	// Cleanup must not return while the task is still in flight.
	select {
	case <-cleanupDone:
		t.Fatal("Cleanup returned before the in-flight task was released")
	case <-time.After(50 * time.Millisecond):
	}

	release()

	select {
	case <-cleanupDone:
	case <-time.After(time.Second):
		t.Fatal("Cleanup did not return after the task was released")
	}

	// After shutdown, RecordTask is a no-op and its release is safe to call.
	s.RecordTask()()
}

// TestReleaseIsIdempotent ensures calling release more than once does not over-decrement the
// task counter (which would panic the WaitGroup).
func TestReleaseIsIdempotent(t *testing.T) {
	s := &SharedOperator[struct{}]{}

	release := s.RecordTask()
	release()
	release()

	done := make(chan struct{})

	go func() {
		s.Cleanup()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Cleanup blocked despite all tasks being released")
	}
}
