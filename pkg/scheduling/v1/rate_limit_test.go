//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRateLimitRepo struct {
	mock.Mock
}

func (m *mockRateLimitRepo) UpdateRateLimits(ctx context.Context, tenantId uuid.UUID, updates map[string]int, definitions map[string]v1.RateLimitDefinition) ([]*sqlcv1.ListRateLimitsForTenantWithMutateRow, *time.Time, error) {
	args := m.Called(ctx, tenantId, updates, definitions)
	return args.Get(0).([]*sqlcv1.ListRateLimitsForTenantWithMutateRow), args.Get(1).(*time.Time), args.Error(2)
}

func (m *mockRateLimitRepo) UpsertRateLimit(ctx context.Context, tenantId uuid.UUID, key string, opts *v1.UpsertRateLimitOpts) (*sqlcv1.RateLimit, error) {
	panic("not implemented")
}

func (m *mockRateLimitRepo) ListRateLimits(ctx context.Context, tenantId uuid.UUID, opts *v1.ListRateLimitOpts) (*v1.ListRateLimitsResult, error) {
	panic("not implemented")
}

func (m *mockRateLimitRepo) DeleteRateLimits(ctx context.Context, tenantId uuid.UUID, key string) error {
	panic("not implemented")
}

func TestRateLimiter_Use(t *testing.T) {
	l := zerolog.Nop()

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRows := []*sqlcv1.ListRateLimitsForTenantWithMutateRow{
		{Key: "key1", Value: 10},
		{Key: "key2", Value: 5},
		{Key: "key3", Value: 7},
	}
	nextRefill := time.Now().Add(2 * time.Second)
	mockRateLimitRepo.On("UpdateRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(mockRows, &nextRefill, nil)

	rateLimiter := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10},
			"key2": {key: "key2", val: 5},
			"key3": {key: "key3", val: 7},
		},
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	// Test simple rate limit usage
	res := rateLimiter.use(context.Background(), 1, map[string]int32{"key1": 5})
	assert.True(t, res.succeeded)
	res = rateLimiter.use(context.Background(), 2, map[string]int32{"key1": 6})
	assert.False(t, res.succeeded)

	// Test multiple keys
	res = rateLimiter.use(context.Background(), 3, map[string]int32{"key2": 3, "key3": 4})
	assert.True(t, res.succeeded)
	res = rateLimiter.use(context.Background(), 4, map[string]int32{"key2": 3, "key3": 4})
	assert.False(t, res.succeeded)
}

func TestRateLimiter_Ack(t *testing.T) {
	l := zerolog.Nop()

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRows := []*sqlcv1.ListRateLimitsForTenantWithMutateRow{
		{Key: "key1", Value: 10},
		{Key: "key2", Value: 5},
	}
	nextRefill := time.Now().Add(2 * time.Second)
	mockRateLimitRepo.On("UpdateRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(mockRows, &nextRefill, nil)

	rateLimiter := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10},
			"key2": {key: "key2", val: 5},
		},
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	rateLimiter.use(context.Background(), 1, map[string]int32{"key1": 5})
	rateLimiter.ack(1)

	// Verify unacked is empty and unflushed contains step1 rate limits
	assert.Empty(t, rateLimiter.unacked)
	assert.Equal(t, 5, rateLimiter.unflushed["key1"].val)
}

func TestRateLimiter_Nack(t *testing.T) {
	l := zerolog.Nop()

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRows := []*sqlcv1.ListRateLimitsForTenantWithMutateRow{
		{Key: "key1", Value: 10},
		{Key: "key2", Value: 5},
	}
	nextRefill := time.Now().Add(2 * time.Second)
	mockRateLimitRepo.On("UpdateRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(mockRows, &nextRefill, nil)

	rateLimiter := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10},
			"key2": {key: "key2", val: 5},
		},
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	rateLimiter.use(context.Background(), 1, map[string]int32{"key1": 5})
	rateLimiter.nack(1)

	// Verify unacked is empty and unflushed doesn't contain step1 rate limits
	assert.Empty(t, rateLimiter.unacked)
	assert.NotContains(t, rateLimiter.unflushed, "key1")
}

func TestRateLimiter_Concurrency(t *testing.T) {
	l := zerolog.Nop()

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRows := []*sqlcv1.ListRateLimitsForTenantWithMutateRow{
		{Key: "key1", Value: 100},
		{Key: "key2", Value: 100},
	}
	nextRefill := time.Now().Add(2 * time.Second)
	mockRateLimitRepo.On("UpdateRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(mockRows, &nextRefill, nil)

	rateLimiter := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 100},
			"key2": {key: "key2", val: 100},
		},
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	var wg sync.WaitGroup
	numUsers := 100
	useAmount := 1

	wg.Add(numUsers)
	for i := 0; i < numUsers; i++ {
		go func(taskId int64) {
			defer wg.Done()
			res := rateLimiter.use(context.Background(), taskId, map[string]int32{"key1": int32(useAmount), "key2": int32(useAmount)}) // nolint: gosec
			assert.True(t, res.succeeded)
			rateLimiter.ack(taskId)
		}(
			int64(i),
		)
	}

	wg.Wait()

	// After all usages, the total used amount should be numUsers * useAmount
	assert.Equal(t, numUsers*useAmount, rateLimiter.unflushed["key1"].val)
}

func TestRateLimiter_FlushToDatabase(t *testing.T) {
	l := zerolog.Nop()

	mockRateLimitRepo := &mockRateLimitRepo{} // Mock implementation of rateLimitRepo
	mockRows := []*sqlcv1.ListRateLimitsForTenantWithMutateRow{
		{Key: "key1", Value: 10},
		{Key: "key2", Value: 5},
	}
	nextRefill := time.Now().Add(2 * time.Second)
	mockRateLimitRepo.On("UpdateRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(mockRows, &nextRefill, nil)

	rateLimiter := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10},
			"key2": {key: "key2", val: 5},
		},
		unacked:       make(map[int64]rateLimitSet),
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
	mockRows := []*sqlcv1.ListRateLimitsForTenantWithMutateRow{
		{Key: "key1", Value: 1000},
		{Key: "key2", Value: 1000},
	}
	nextRefill := time.Now().Add(2 * time.Second)
	mockRateLimitRepo.On("UpdateRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(mockRows, &nextRefill, nil)

	r := rateLimiter{
		unacked:       make(map[int64]rateLimitSet),
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
		count := 0
		for pb.Next() {
			taskId := int64(count)
			requests := map[string]int32{
				"rate_limit_1": rand.Int31n(5), // nolint: gosec
				"rate_limit_2": rand.Int31n(5), // nolint: gosec
				"rate_limit_3": rand.Int31n(5), // nolint: gosec
			}

			r.use(context.Background(), taskId, requests)
			count++
		}
	})
}

func TestRateLimiter_ShouldRefill(t *testing.T) {
	l := zerolog.Nop()
	r := &rateLimiter{l: &l}

	// no refill deadline known yet
	assert.False(t, r.shouldRefill())

	// deadline still in the future: the cached window is current
	future := time.Now().UTC().Add(time.Minute)
	r.nextRefillAt = &future
	assert.False(t, r.shouldRefill())

	// deadline reached: limits have refilled in the database and must be re-read
	past := time.Now().UTC().Add(-time.Millisecond)
	r.nextRefillAt = &past
	assert.True(t, r.shouldRefill())
}

func TestRateLimiter_FlushDefinitions(t *testing.T) {
	minute := v1.RateLimitDefinition{LimitValue: 10, Window: "1 MINUTE"}

	tests := []struct {
		name            string
		dbRateLimits    rateLimitSet
		nextRefillAt    time.Time
		pending         map[string]v1.RateLimitDefinition
		wantFlush       bool
		wantDefinitions map[string]v1.RateLimitDefinition
	}{
		{
			// use() needs the key before the next refill, or the task can't be assigned
			name:            "new key bypasses refill guard",
			dbRateLimits:    rateLimitSet{},
			nextRefillAt:    time.Now().Add(time.Minute),
			pending:         map[string]v1.RateLimitDefinition{"new": minute},
			wantFlush:       true,
			wantDefinitions: map[string]v1.RateLimitDefinition{"new": minute},
		},
		{
			name:         "changed definition waits for refill guard",
			dbRateLimits: rateLimitSet{"k": {key: "k", limitValue: 10, window: "1 MINUTE"}},
			nextRefillAt: time.Now().Add(time.Minute),
			pending:      map[string]v1.RateLimitDefinition{"k": {LimitValue: 5, Window: "1 MINUTE"}},
			wantFlush:    false,
		},
		{
			name:            "changed definition is written once guard passes",
			dbRateLimits:    rateLimitSet{"k": {key: "k", limitValue: 10, window: "1 MINUTE"}},
			nextRefillAt:    time.Now().Add(-time.Second),
			pending:         map[string]v1.RateLimitDefinition{"k": {LimitValue: 5, Window: "1 MINUTE"}},
			wantFlush:       true,
			wantDefinitions: map[string]v1.RateLimitDefinition{"k": {LimitValue: 5, Window: "1 MINUTE"}},
		},
		{
			name:            "unchanged definition is not rewritten",
			dbRateLimits:    rateLimitSet{"k": {key: "k", limitValue: 10, window: "1 MINUTE"}},
			nextRefillAt:    time.Now().Add(-time.Second),
			pending:         map[string]v1.RateLimitDefinition{"k": minute},
			wantFlush:       true,
			wantDefinitions: map[string]v1.RateLimitDefinition{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := zerolog.Nop()
			repo := &mockRateLimitRepo{}
			nextRefill := time.Now().Add(2 * time.Second)

			if tt.wantFlush {
				repo.On("UpdateRateLimits", context.Background(), mock.Anything, mock.Anything, tt.wantDefinitions).
					Return([]*sqlcv1.ListRateLimitsForTenantWithMutateRow{}, &nextRefill, nil)
			}

			r := &rateLimiter{
				dbRateLimits:  tt.dbRateLimits,
				nextRefillAt:  &tt.nextRefillAt,
				unacked:       make(map[int64]rateLimitSet),
				unflushed:     make(rateLimitSet),
				l:             &l,
				rateLimitRepo: repo,
			}

			r.addDefinitions(tt.pending)

			require.NoError(t, r.flushToDatabase(context.Background()))
			repo.AssertExpectations(t)

			if tt.wantFlush {
				assert.Empty(t, r.pendingDefinitions)
			} else {
				repo.AssertNotCalled(t, "UpdateRateLimits", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				assert.Equal(t, tt.pending, r.pendingDefinitions, "pending definitions are kept for the next flush")
			}
		})
	}
}

func TestRateLimiter_FailedFlushDropsDefinitions(t *testing.T) {
	l := zerolog.Nop()
	repo := &mockRateLimitRepo{}
	nextRefill := time.Now().Add(2 * time.Second)
	bad := map[string]v1.RateLimitDefinition{"bad": {LimitValue: 10, Window: "1 MINUTE"}}

	repo.On("UpdateRateLimits", context.Background(), mock.Anything, map[string]int{"k": 3}, bad).
		Return([]*sqlcv1.ListRateLimitsForTenantWithMutateRow(nil), (*time.Time)(nil), fmt.Errorf("upsert failed")).Once()
	repo.On("UpdateRateLimits", context.Background(), mock.Anything, map[string]int{"k": 3}, map[string]v1.RateLimitDefinition{}).
		Return([]*sqlcv1.ListRateLimitsForTenantWithMutateRow{{Key: "k", Value: 7}}, &nextRefill, nil).Once()

	r := &rateLimiter{
		dbRateLimits:  rateLimitSet{},
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     rateLimitSet{"k": {key: "k", val: 3}},
		l:             &l,
		rateLimitRepo: repo,
	}

	r.addDefinitions(bad)

	require.Error(t, r.flushToDatabase(context.Background()))
	assert.Empty(t, r.pendingDefinitions)
	assert.Equal(t, 3, r.unflushed["k"].val, "usage is kept for the next flush")

	// usage still flushes once the bad definition is gone
	require.NoError(t, r.flushToDatabase(context.Background()))
	assert.Empty(t, r.unflushed)
	assert.Equal(t, 7, r.dbRateLimits["k"].val)

	repo.AssertExpectations(t)
}
