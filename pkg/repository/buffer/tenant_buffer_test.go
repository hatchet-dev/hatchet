package buffer

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type testMockEvent struct {
	ID    int
	Value string
	Size  int
}

type testMockResult struct {
	ID int
}

func testMockOutputFunc(ctx context.Context, items []testMockEvent) ([]testMockResult, error) {
	var results []testMockResult
	//nolint:gosec
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond) // Simulate output function execution time with some randomness

	for _, item := range items {
		results = append(results, testMockResult{ID: item.ID})
	}
	return results, nil
}

func testMockSizeFunc(item testMockEvent) int {
	return item.Size
}

func TestNewTenantBufManager(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.Disabled)

	opts := TenantBufManagerOpts[testMockEvent, testMockResult]{
		OutputFunc: testMockOutputFunc,
		SizeFunc:   testMockSizeFunc,
		L:          &logger,
		V:          validator.NewDefaultValidator(),
	}

	manager, err := NewTenantBufManager(opts)
	require.NoError(t, err)
	assert.NotNil(t, manager)
}

func TestTenantBufferManager_BuffItem(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.Disabled)

	opts := TenantBufManagerOpts[testMockEvent, testMockResult]{
		OutputFunc: testMockOutputFunc,
		SizeFunc:   testMockSizeFunc,
		L:          &logger,
		V:          validator.NewDefaultValidator(),
	}

	manager, err := NewTenantBufManager(opts)
	require.NoError(t, err)

	// Buff some items for a tenant
	tenantKey := "tenant_1"
	event := testMockEvent{ID: 1, Value: "test_event", Size: 10}

	respChan, err := manager.BuffItem(tenantKey, event)
	require.NoError(t, err)

	// Collect the result after the item is flushed
	resp := <-respChan
	require.NoError(t, resp.Err)
	assert.Equal(t, event.ID, resp.Result.ID)
}

func generateTestCases(numTenants int) []struct {
	tenantKey string
	event     testMockEvent
} {
	testCases := make([]struct {
		tenantKey string
		event     testMockEvent
	}, numTenants)

	for i := 0; i < numTenants; i++ {
		tenantKey := fmt.Sprintf("tenant_%d", i+1)
		testCases[i] = struct {
			tenantKey string
			event     testMockEvent
		}{
			tenantKey: tenantKey,
			event: testMockEvent{
				ID:    i + 1,
				Value: tenantKey,
				Size:  10 + i,
			},
		}
	}

	return testCases
}

func TestTenantBufferManager_CreateMultipleBuffers(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.Disabled)

	opts := TenantBufManagerOpts[testMockEvent, testMockResult]{
		OutputFunc: testMockOutputFunc,
		SizeFunc:   testMockSizeFunc,
		L:          &logger,
		V:          validator.NewDefaultValidator(),
	}

	manager, err := NewTenantBufManager(opts)
	require.NoError(t, err)

	// Generate an arbitrary number of test cases (e.g., 100)
	testCases := generateTestCases(100)
	testCases = append(testCases, generateTestCases(10)...)   // Add a duplicate test case
	testCases = append(testCases, generateTestCases(1000)...) // Add a duplicate test case

	// Create a wait group to synchronize goroutines
	var wg sync.WaitGroup

	for _, tc := range testCases {
		wg.Add(1)

		// Launch a goroutine for each test case
		go func(tc struct {
			tenantKey string
			event     testMockEvent
		}) {
			defer wg.Done()

			// Buff events for the given tenant
			respChan, err := manager.BuffItem(tc.tenantKey, tc.event)
			require.NoError(t, err)

			resp := <-respChan
			require.NoError(t, resp.Err)
			assert.Equal(t, tc.event.ID, resp.Result.ID)
		}(tc)
	}

	wg.Wait()

}

func TestTenantBufferManager_OrderPreservation(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.Disabled)

	opts := TenantBufManagerOpts[testMockEvent, testMockResult]{
		OutputFunc: testMockOutputFunc,
		SizeFunc:   testMockSizeFunc,
		L:          &logger,
		V:          validator.NewDefaultValidator(),
	}

	manager, err := NewTenantBufManager(opts)
	require.NoError(t, err)

	tenantKey := "tenant_order"
	var wg sync.WaitGroup
	expectedOrder := []int{1011, 20200, 33020, 4010221, 51}

	rand.Shuffle(len(expectedOrder), func(i, j int) {
		expectedOrder[i], expectedOrder[j] = expectedOrder[j], expectedOrder[i]
	})

	// Buff multiple events and track the order
	for _, id := range expectedOrder {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := testMockEvent{ID: id, Value: fmt.Sprintf("test-%d", id), Size: 10}
			respChan, err := manager.BuffItem(tenantKey, event)
			require.NoError(t, err)

			// Collect the result after the item is flushed
			resp := <-respChan
			require.NoError(t, resp.Err)
			assert.Equal(t, id, resp.Result.ID)
		}(id)
	}

	wg.Wait()

}

func TestTenantBufferManager_Cleanup(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.Disabled)

	opts := TenantBufManagerOpts[testMockEvent, testMockResult]{
		OutputFunc: testMockOutputFunc,
		SizeFunc:   testMockSizeFunc,
		L:          &logger,
		V:          validator.NewDefaultValidator(),
	}

	manager, err := NewTenantBufManager(opts)
	require.NoError(t, err)

	tenantKey := "tenant_cleanup"
	event := testMockEvent{ID: 1, Value: "cleanup_event", Size: 10}

	_, err = manager.BuffItem(tenantKey, event)
	require.NoError(t, err)

	// Ensure buffers are cleaned up
	err = manager.Cleanup()
	require.NoError(t, err)

	// Try to buff an item after cleanup, should return an error
	_, err = manager.BuffItem(tenantKey, event)
	require.Error(t, err)
}
