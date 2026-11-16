package timeout_lock

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAcquireContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lock := NewTimeoutLock(time.Second)
		if !lock.Acquire() {
			t.Fatal("could not acquire free lock")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		if err := lock.AcquireContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("want deadline, got %v", err)
		}
		lock.Release()
		if err := lock.AcquireContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expired context acquired free lock: %v", err)
		}
		if !lock.Acquire() {
			t.Fatal("expired context leaked lock ownership")
		}
		if err := lock.AcquireContext(context.Background()); !errors.Is(err, ErrTimeout) {
			t.Fatalf("want lock timeout, got %v", err)
		}
		lock.Release()
	})
}

func BenchmarkLockAcquisition(b *testing.B) {
	l := NewTimeoutLock(time.Second)
	for i := 0; i < b.N; i++ {
		go func() {
			if l.Acquire() {
				l.Release()
			}
		}()
	}
}

func TestLockAcquisitionTimeout(t *testing.T) {
	// Not all the locks should be acquired because of the timeout
	l := NewTimeoutLock(5 * time.Millisecond)
	numberAcquired := 0
	wg := sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			if l.Acquire() {
				defer l.Release()
				time.Sleep(time.Millisecond)
				numberAcquired++
			}
		}()
	}
	wg.Wait()
	assert.NotEqual(t, 10, numberAcquired)
}
