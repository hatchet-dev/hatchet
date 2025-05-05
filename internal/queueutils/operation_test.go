//go:build !e2e && !load && !rampup && !integration

package queueutils

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

var l = zerolog.Nop()

func TestSerialOperation_RunOrContinue_NoConcurrentExecution(t *testing.T) {
	var runCount int
	var mu sync.Mutex

	mockMethod := func(ctx context.Context, id string) (bool, error) {
		if !mu.TryLock() {
			panic("Concurrent execution detected")
		}

		runCount++
		mu.Unlock()

		time.Sleep(100 * time.Millisecond) // Simulate method execution time
		return false, nil
	}

	operation := &SerialOperation{
		id:          "1234",
		description: "Test operation",
		timeout:     2 * time.Second,
		method:      mockMethod,
	}

	// First run
	operation.RunOrContinue(&l)

	// Try to trigger a set of runs concurrently, it should not start until the first finishes
	time.Sleep(10 * time.Millisecond)
	operation.RunOrContinue(&l)
	operation.RunOrContinue(&l)
	operation.RunOrContinue(&l)
	operation.RunOrContinue(&l)

	// Wait for both to finish
	time.Sleep(250 * time.Millisecond)

	mu.Lock()
	assert.Equal(t, 2, runCount, "The method should not run concurrently")
	mu.Unlock()
}

func TestSerialOperation_RunOrContinue_ShouldRunAfterCompletion(t *testing.T) {
	var runCount int
	var mu sync.Mutex

	mockMethod := func(ctx context.Context, id string) (bool, error) {
		mu.Lock()
		runCount++
		mu.Unlock()

		time.Sleep(50 * time.Millisecond) // Simulate method execution time
		return false, nil
	}

	operation := &SerialOperation{
		id:          "1234",
		description: "Test operation",
		timeout:     2 * time.Second,
		method:      mockMethod,
	}

	// First run
	operation.RunOrContinue(&l)
	time.Sleep(110 * time.Millisecond)

	// Second run after first finishes
	operation.RunOrContinue(&l)
	time.Sleep(110 * time.Millisecond)

	mu.Lock()
	assert.Equal(t, 2, runCount, "The method should run twice after completion")
	mu.Unlock()
}

func TestSerialOperation_RunOrContinue_ShouldRunOnContinues(t *testing.T) {
	var runCount int
	var mu sync.Mutex

	mockMethod := func(ctx context.Context, id string) (bool, error) {
		mu.Lock()
		runCount++
		mu.Unlock()

		time.Sleep(25 * time.Millisecond) // Simulate method execution time
		return runCount < 5, nil
	}

	operation := &SerialOperation{
		id:          "1234",
		description: "Test operation",
		timeout:     2 * time.Second,
		method:      mockMethod,
	}

	// First run
	operation.RunOrContinue(&l)
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	assert.Equal(t, 5, runCount, "The method should run five times on continue")
	mu.Unlock()
}
