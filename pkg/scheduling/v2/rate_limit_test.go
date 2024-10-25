package v2

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRateLimitRepo struct {
	mock.Mock
}

func (m *mockRateLimitRepo) ListCandidateRateLimits(ctx context.Context, tenantId pgtype.UUID) ([]string, error) {
	args := m.Called(ctx, tenantId)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockRateLimitRepo) UpdateRateLimits(ctx context.Context, tenantId pgtype.UUID, updates map[string]int) (map[string]int, error) {
	args := m.Called(ctx, tenantId, updates)
	return args.Get(0).(map[string]int), args.Error(1)
}

func TestRateLimiter_Use(t *testing.T) {
	l := zerolog.Nop()

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRateLimitRepo.On("UpdateRateLimits", context.Background(), mock.Anything, mock.Anything).Return(map[string]int{"key1": 10, "key2": 5, "key3": 7}, nil)

	rateLimiter := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10},
			"key2": {key: "key2", val: 5},
			"key3": {key: "key3", val: 7},
		},
		unacked:       make(map[string]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	// Test simple rate limit usage
	res := rateLimiter.use(context.Background(), "step1", map[string]int32{"key1": 5})
	assert.True(t, res.succeeded)
	res = rateLimiter.use(context.Background(), "step2", map[string]int32{"key1": 6})
	assert.False(t, res.succeeded)

	// Test multiple keys
	res = rateLimiter.use(context.Background(), "step3", map[string]int32{"key2": 3, "key3": 4})
	assert.True(t, res.succeeded)
	res = rateLimiter.use(context.Background(), "step4", map[string]int32{"key2": 3, "key3": 4})
	assert.False(t, res.succeeded)
}

func TestRateLimiter_Ack(t *testing.T) {
	l := zerolog.Nop()

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRateLimitRepo.On("UpdateRateLimits", context.Background(), mock.Anything, mock.Anything).Return(map[string]int{"key1": 10, "key2": 5}, nil)

	rateLimiter := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10},
			"key2": {key: "key2", val: 5},
		},
		unacked:       make(map[string]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	rateLimiter.use(context.Background(), "step1", map[string]int32{"key1": 5})
	rateLimiter.ack("step1")

	// Verify unacked is empty and unflushed contains step1 rate limits
	assert.Empty(t, rateLimiter.unacked)
	assert.Equal(t, 5, rateLimiter.unflushed["key1"].val)
}

func TestRateLimiter_Nack(t *testing.T) {
	l := zerolog.Nop()

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRateLimitRepo.On("UpdateRateLimits", context.Background(), mock.Anything, mock.Anything).Return(map[string]int{"key1": 10, "key2": 5}, nil)

	rateLimiter := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10},
			"key2": {key: "key2", val: 5},
		},
		unacked:       make(map[string]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	rateLimiter.use(context.Background(), "step1", map[string]int32{"key1": 5})
	rateLimiter.nack("step1")

	// Verify unacked is empty and unflushed doesn't contain step1 rate limits
	assert.Empty(t, rateLimiter.unacked)
	assert.NotContains(t, rateLimiter.unflushed, "step1")
}

func TestRateLimiter_Concurrency(t *testing.T) {
	l := zerolog.Nop()

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRateLimitRepo.On("UpdateRateLimits", context.Background(), mock.Anything, mock.Anything).Return(map[string]int{"key1": 100, "key2": 100}, nil)

	rateLimiter := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 100},
			"key2": {key: "key2", val: 100},
		},
		unacked:       make(map[string]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	var wg sync.WaitGroup
	numUsers := 100
	useAmount := 1

	wg.Add(numUsers)
	for i := 0; i < numUsers; i++ {
		go func(stepRunId string) {
			defer wg.Done()
			res := rateLimiter.use(context.Background(), stepRunId, map[string]int32{"key1": int32(useAmount), "key2": int32(useAmount)}) // nolint: gosec
			assert.True(t, res.succeeded)
			rateLimiter.ack(stepRunId)
		}(
			"step" + strconv.Itoa(i),
		)
	}

	wg.Wait()

	// After all usages, the total used amount should be numUsers * useAmount
	assert.Equal(t, numUsers*useAmount, rateLimiter.unflushed["key1"].val)
}

func TestRateLimiter_FlushToDatabase(t *testing.T) {
	l := zerolog.Nop()

	mockRateLimitRepo := &mockRateLimitRepo{} // Mock implementation of rateLimitRepo
	mockRateLimitRepo.On("UpdateRateLimits", context.Background(), mock.Anything, mock.Anything).Return(map[string]int{"key1": 10, "key2": 5}, nil)

	rateLimiter := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10},
			"key2": {key: "key2", val: 5},
		},
		unacked:       make(map[string]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	// Add some rate limits to unflushed
	rateLimiter.unflushed["key1"] = &rateLimit{key: "key1", val: 5}
	rateLimiter.unflushed["key2"] = &rateLimit{key: "key2", val: 3}

	// Flush rate limits to database
	err := rateLimiter.flushToDatabase(context.Background())
	assert.NoError(t, err)

	// Verify that dbRateLimits contains the updated values
	assert.Equal(t, 10, rateLimiter.dbRateLimits["key1"].val)
	assert.Equal(t, 5, rateLimiter.dbRateLimits["key2"].val)

	// Verify that unflushed is empty
	assert.Empty(t, rateLimiter.unflushed)
}

func BenchmarkRateLimiter(b *testing.B) {
	l := zerolog.Nop()

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRateLimitRepo.On("UpdateRateLimits", context.Background(), mock.Anything, mock.Anything).Return(map[string]int{"key1": 1000, "key2": 1000}, nil)

	r := rateLimiter{
		unacked:       make(map[string]rateLimitSet),
		unflushed:     make(rateLimitSet),
		dbRateLimits:  make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	// Initialize dbRateLimits with some random rate limits
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("rate_limit_%d", i)
		value := rand.Intn(1000) // nolint: gosec
		r.dbRateLimits[key] = &rateLimit{
			key: key,
			val: value,
		}
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			stepRunId := uuid.New().String()
			requests := map[string]int32{
				"rate_limit_1": rand.Int31n(5), // nolint: gosec
				"rate_limit_2": rand.Int31n(5), // nolint: gosec
				"rate_limit_3": rand.Int31n(5), // nolint: gosec
			}

			r.use(context.Background(), stepRunId, requests)
		}
	})
}
