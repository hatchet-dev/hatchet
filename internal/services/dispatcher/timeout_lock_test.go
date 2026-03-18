package dispatcher

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
