package trigger

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAcquireSlotFailFast(t *testing.T) {
	tw := &TriggerWriter{semaphore: make(chan struct{}, 1)}

	release, err := tw.acquireSlot(context.Background(), false)
	if err != nil {
		t.Fatalf("expected first acquire to succeed, got %v", err)
	}

	_, err = tw.acquireSlot(context.Background(), false)
	if !errors.Is(err, ErrNoTriggerSlots) {
		t.Fatalf("expected ErrNoTriggerSlots, got %v", err)
	}

	release()

	release, err = tw.acquireSlot(context.Background(), false)
	if err != nil {
		t.Fatalf("expected acquire after release to succeed, got %v", err)
	}
	release()
}

func TestAcquireSlotWaitingTimesOut(t *testing.T) {
	tw := &TriggerWriter{semaphore: make(chan struct{}, 1)}

	held, err := tw.acquireSlot(context.Background(), true)
	if err != nil {
		t.Fatalf("expected first acquire to succeed, got %v", err)
	}
	defer held()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err = tw.acquireSlot(ctx, true)
	if !errors.Is(err, ErrNoTriggerSlots) {
		t.Fatalf("expected ErrNoTriggerSlots, got %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestAcquireSlotWaitingUnblocks(t *testing.T) {
	tw := &TriggerWriter{semaphore: make(chan struct{}, 1)}

	held, err := tw.acquireSlot(context.Background(), true)
	if err != nil {
		t.Fatalf("expected first acquire to succeed, got %v", err)
	}

	got := make(chan error, 1)
	go func() {
		release, err := tw.acquireSlot(context.Background(), true)
		if err == nil {
			release()
		}
		got <- err
	}()

	select {
	case err := <-got:
		t.Fatalf("waiter returned before slot was released: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	held()

	select {
	case err := <-got:
		if err != nil {
			t.Fatalf("expected waiting acquire to succeed after release, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for slot to be acquired")
	}
}
