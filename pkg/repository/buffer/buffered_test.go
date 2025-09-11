//go:build !e2e && !load && !rampup && !integration

package buffer

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	// "github.com/hatchet-dev/hatchet/pkg/validator"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type mockItem struct {
	ID    int
	Size  int
	Value string
}

type mockResult struct {
	ID int
}

func mockOutputFunc(ctx context.Context, items []mockItem) ([]*mockResult, error) {
	var results []*mockResult
	for _, item := range items {
		results = append(results, &mockResult{ID: item.ID})
	}
	return results, nil
}

func mockSizeFunc(item mockItem) int {
	return item.Size
}

func TestIngestBufInitialization(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.Disabled)

	opts := IngestBufOpts[mockItem, mockResult]{
		Name:               "test",
		MaxCapacity:        5,
		FlushPeriod:        1 * time.Second,
		MaxDataSizeInQueue: 100,
		OutputFunc:         mockOutputFunc,
		SizeFunc:           mockSizeFunc,
		L:                  &logger,
		FlushStrategy:      Dynamic,
	}

	// Initialize the buffer
	buf := NewIngestBuffer(opts)
	assert.Equal(t, 5, buf.maxCapacity)
	assert.Equal(t, 1*time.Second, buf.flushPeriod)
	assert.Equal(t, 100, buf.maxDataSizeInQueue)
	assert.NotNil(t, buf.inputChan)
	assert.Equal(t, 0, buf.safeFetchSizeOfData())
	assert.Equal(t, initialized, buf.state)
	v := validator.NewDefaultValidator()
	assert.NoError(t, v.Validate(opts))

}

func TestIngestBufValidation(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.Disabled)

	opts := IngestBufOpts[mockItem, mockResult]{
		Name:               "test",
		MaxCapacity:        0,
		FlushPeriod:        -1 * time.Second,
		MaxDataSizeInQueue: -1,
		OutputFunc:         nil,
		SizeFunc:           nil,
		L:                  &logger,
	}

	v := validator.NewDefaultValidator()
	err := v.Validate(opts)

	require.Error(t, err)

}

func TestIngestBufBuffering(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.Disabled)

	opts := IngestBufOpts[mockItem, mockResult]{
		Name:               "test",
		MaxCapacity:        2,
		FlushPeriod:        1 * time.Second,
		MaxDataSizeInQueue: 100,
		OutputFunc:         mockOutputFunc,
		SizeFunc:           mockSizeFunc,
		L:                  &logger,
	}

	buf := NewIngestBuffer(opts)
	_, err := buf.Start()

	require.NoError(t, err)

	item := mockItem{ID: 1, Size: 10, Value: "test"}
	resp, err := buf.FireAndWait(context.Background(), item)
	assert.NoError(t, err)
	assert.Equal(t, 1, resp.ID)

	assert.Equal(t, 0, buf.safeFetchSizeOfData())
	assert.Equal(t, 0, buf.safeCheckSizeOfBuffer())
}

func TestIngestBufAutoFlushOnCapacity(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.Disabled)

	opts := IngestBufOpts[mockItem, mockResult]{
		Name:               "test",
		MaxCapacity:        2,
		FlushPeriod:        5 * time.Second,
		MaxDataSizeInQueue: 100,
		OutputFunc:         mockOutputFunc,
		SizeFunc:           mockSizeFunc,
		L:                  &logger,
	}

	buf := NewIngestBuffer(opts)
	_, err := buf.Start()

	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)

	for i := 1; i <= 2; i++ {
		go func(i int) {
			defer wg.Done()
			item := mockItem{ID: i, Size: 10, Value: "test"}
			resp, err := buf.FireAndWait(context.Background(), item)
			assert.NoError(t, err)
			assert.Equal(t, i, resp.ID)
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 0, buf.safeFetchSizeOfData())
	assert.Equal(t, 0, buf.safeCheckSizeOfBuffer())

}

func TestIngestBufAutoFlushOnSize(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.Disabled)

	opts := IngestBufOpts[mockItem, mockResult]{
		Name:               "test",
		MaxCapacity:        10,
		FlushPeriod:        5 * time.Second,
		MaxDataSizeInQueue: 20, // Flush on size
		OutputFunc:         mockOutputFunc,
		SizeFunc:           mockSizeFunc,
		L:                  &logger,
	}

	buf := NewIngestBuffer(opts)
	_, err := buf.Start()

	require.NoError(t, err)

	item := mockItem{ID: 1, Size: 25, Value: "test"}
	resp, err := buf.FireAndWait(context.Background(), item)
	assert.NoError(t, err)
	assert.Equal(t, 1, resp.ID)

	assert.Equal(t, 0, buf.safeFetchSizeOfData())
	assert.Equal(t, 0, buf.safeCheckSizeOfBuffer())

}

func TestIngestBufTimeoutFlush(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.Disabled)

	opts := IngestBufOpts[mockItem, mockResult]{
		Name:               "test",
		MaxCapacity:        10,
		FlushPeriod:        100 * time.Millisecond,
		MaxDataSizeInQueue: 100,
		OutputFunc:         mockOutputFunc,
		SizeFunc:           mockSizeFunc,
		L:                  &logger,
	}

	buf := NewIngestBuffer(opts)
	_, err := buf.Start()

	require.NoError(t, err)

	item := mockItem{ID: 1, Size: 1, Value: "test"}
	doneChan := make(chan *mockResult)

	go func() {
		resp, err := buf.FireAndWait(context.Background(), item)
		assert.NoError(t, err)
		doneChan <- resp
	}()

	select {
	case resp := <-doneChan:
		assert.Equal(t, 1, resp.ID)
	case <-time.After(500 * time.Millisecond):
		t.Error("Flush should have been triggered by timeout")
	}

	assert.Equal(t, 0, buf.safeFetchSizeOfData())
	assert.Equal(t, 0, buf.safeCheckSizeOfBuffer())
}

func TestIngestBufOrderPreservation(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.Disabled)

	opts := IngestBufOpts[mockItem, mockResult]{
		Name:               "test",
		MaxCapacity:        5,
		FlushPeriod:        5 * time.Second,
		MaxDataSizeInQueue: 100,
		OutputFunc: func(ctx context.Context, items []mockItem) ([]*mockResult, error) {
			var results []*mockResult
			for _, item := range items {
				results = append(results, &mockResult{ID: item.ID})
			}
			return results, nil
		},
		SizeFunc: mockSizeFunc,
		L:        &logger,
	}

	buf := NewIngestBuffer(opts)
	_, err := buf.Start()

	require.NoError(t, err)

	var wg sync.WaitGroup
	expectedOrder := []int{1011, 20200, 33020, 4010221, 51}

	rand.Shuffle(len(expectedOrder), func(i, j int) {
		expectedOrder[i], expectedOrder[j] = expectedOrder[j], expectedOrder[i]
	})

	for _, id := range expectedOrder {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			item := mockItem{ID: id, Size: 10, Value: fmt.Sprintf("test-%d", id)}
			resp, err := buf.FireAndWait(context.Background(), item)
			require.NoError(t, err)
			assert.Equal(t, id, resp.ID)
		}(id)
	}

	wg.Wait()

}
