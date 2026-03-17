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
	l := NewTimeoutLock(5 * time.Millisecond)
	numberAcquired := 0
	wg := sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			if l.Acquire() {
				defer l.Release()
				numberAcquired++
				time.Sleep(time.Millisecond)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, numberAcquired, 5)
}

func TestLockAcquisitionOrdering(t *testing.T) {
	l := NewTimeoutLock(time.Second)
	orderedAcquired := make([]int, 0)
	wg := sync.WaitGroup{}
	wg.Add(10)
	goroutineNumber := 0
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			if l.Acquire() {
				defer l.Release()
				orderedAcquired = append(orderedAcquired, goroutineNumber)
				goroutineNumber++
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, 0, orderedAcquired[0])
	assert.Equal(t, 1, orderedAcquired[1])
	assert.Equal(t, 9, orderedAcquired[9])
}
