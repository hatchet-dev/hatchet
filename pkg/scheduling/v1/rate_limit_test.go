//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRateLimitRepo struct {
	mock.Mock
}

func (m *mockRateLimitRepo) FlushRateLimits(ctx context.Context, tenantId uuid.UUID, usage map[string]int, definitions map[string]v1.RateLimitDefinition) ([]*sqlcv1.ListRateLimitsForTenantWithMutateRow, *time.Time, error) {
	args := m.Called(ctx, tenantId, usage, definitions)
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

func (m *mockRateLimitRepo) PauseRateLimit(ctx context.Context, tenantId uuid.UUID, key string, until time.Time) (*sqlcv1.V1RateLimitPause, error) {
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
	mockRateLimitRepo.On("FlushRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(mockRows, &nextRefill, nil)

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
	mockRateLimitRepo.On("FlushRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(mockRows, &nextRefill, nil)

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
	mockRateLimitRepo.On("FlushRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(mockRows, &nextRefill, nil)

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
	mockRateLimitRepo.On("FlushRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(mockRows, &nextRefill, nil)

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
	mockRateLimitRepo.On("FlushRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(mockRows, &nextRefill, nil)

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

func TestRateLimiter_NewDefinitionFlushesBeforeRefill(t *testing.T) {
	l := zerolog.Nop()

	def := v1.RateLimitDefinition{LimitValue: 10, Window: "1 HOUR"}

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRows := []*sqlcv1.ListRateLimitsForTenantWithMutateRow{
		{Key: "new", Value: 10, LimitValue: def.LimitValue, Window: def.Window},
	}
	nextRefill := time.Now().Add(2 * time.Second)
	mockRateLimitRepo.On("FlushRateLimits", context.Background(), mock.Anything, mock.Anything, map[string]v1.RateLimitDefinition{"new": def}).Return(mockRows, &nextRefill, nil).Once()

	// a refill deadline in the future would otherwise make flushToDatabase return early
	future := time.Now().UTC().Add(time.Minute)

	r := &rateLimiter{
		dbRateLimits:  make(rateLimitSet),
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     make(rateLimitSet),
		nextRefillAt:  &future,
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	r.addDefinitions(map[string]v1.RateLimitDefinition{"new": def})

	res := r.use(context.Background(), 1, map[string]int32{"new": 1})
	assert.True(t, res.succeeded)

	mockRateLimitRepo.AssertExpectations(t)
}

func TestRateLimiter_UnchangedDefinitionSkipsFlush(t *testing.T) {
	l := zerolog.Nop()

	def := v1.RateLimitDefinition{LimitValue: 10, Window: "1 HOUR"}

	mockRateLimitRepo := &mockRateLimitRepo{}

	future := time.Now().UTC().Add(time.Minute)

	r := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10, limitValue: def.LimitValue, window: def.Window},
		},
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     make(rateLimitSet),
		nextRefillAt:  &future,
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	r.addDefinitions(map[string]v1.RateLimitDefinition{"key1": def})

	assert.NoError(t, r.flushToDatabase(context.Background()))
	mockRateLimitRepo.AssertNotCalled(t, "FlushRateLimits", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestRateLimiter_FailedFlushKeepsDefinitions(t *testing.T) {
	l := zerolog.Nop()

	stricter := v1.RateLimitDefinition{LimitValue: 5, Window: "1 HOUR"}
	strictest := v1.RateLimitDefinition{LimitValue: 3, Window: "1 HOUR"}

	future := time.Now().UTC().Add(time.Minute)

	r := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10, limitValue: 10, window: "1 HOUR"},
		},
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     make(rateLimitSet),
		nextRefillAt:  &future,
		l:             &l,
		rateLimitRepo: nil,
	}

	mockRateLimitRepo := &mockRateLimitRepo{}

	// a queue pass adds a newer definition while the failing flush is in flight
	mockRateLimitRepo.On("FlushRateLimits", context.Background(), mock.Anything, mock.Anything, map[string]v1.RateLimitDefinition{"key1": stricter}).
		Run(func(mock.Arguments) {
			r.addDefinitions(map[string]v1.RateLimitDefinition{"key1": strictest})
		}).
		Return(([]*sqlcv1.ListRateLimitsForTenantWithMutateRow)(nil), (*time.Time)(nil), errors.New("flush failed")).Once()

	nextRefill := time.Now().Add(2 * time.Second)
	mockRateLimitRepo.On("FlushRateLimits", context.Background(), mock.Anything, mock.Anything, map[string]v1.RateLimitDefinition{"key1": strictest}).
		Return([]*sqlcv1.ListRateLimitsForTenantWithMutateRow{
			{Key: "key1", Value: 3, LimitValue: strictest.LimitValue, Window: strictest.Window},
		}, &nextRefill, nil).Once()

	r.rateLimitRepo = mockRateLimitRepo

	r.addDefinitions(map[string]v1.RateLimitDefinition{"key1": stricter})

	assert.Error(t, r.flushToDatabase(context.Background()))
	assert.NoError(t, r.flushToDatabase(context.Background()))

	assert.Equal(t, 3, r.dbRateLimits["key1"].val)
	mockRateLimitRepo.AssertExpectations(t)
}

func BenchmarkRateLimiter(b *testing.B) {
	l := zerolog.Nop()

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRows := []*sqlcv1.ListRateLimitsForTenantWithMutateRow{
		{Key: "key1", Value: 1000},
		{Key: "key2", Value: 1000},
	}
	nextRefill := time.Now().Add(2 * time.Second)
	mockRateLimitRepo.On("FlushRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(mockRows, &nextRefill, nil)

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

func TestRateLimiter_PausedKeyBlocksWithoutReservingUnits(t *testing.T) {
	l := zerolog.Nop()

	pausedUntil := time.Now().UTC().Add(time.Hour)

	r := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"paused": {key: "paused", val: 10, pausedUntil: &pausedUntil},
		},
		unacked:   make(map[int64]rateLimitSet),
		unflushed: make(rateLimitSet),
		l:         &l,
	}

	res := r.use(context.Background(), 1, map[string]int32{"paused": 1})
	assert.False(t, res.succeeded)
	assert.Equal(t, "paused", res.exceededKey)
	assert.Equal(t, int32(1), res.exceededUnits)
	assert.Equal(t, &pausedUntil, res.pausedUntil)
	assert.Equal(t, &pausedUntil, res.nextRefillAt, "the item should be requeued when the pause ends")
	assert.Empty(t, r.unacked, "a pause must not consume units")

	scheduleRes := &scheduleRateLimitResult{rateLimitResult: &res}
	assert.True(t, scheduleRes.shouldRemoveFromQueue())
}

func TestRateLimiter_PauseWinsOverExhaustedWindow(t *testing.T) {
	l := zerolog.Nop()

	windowRefillAt := time.Now().UTC().Add(time.Minute)
	earlierPause := time.Now().UTC().Add(30 * time.Minute)
	laterPause := time.Now().UTC().Add(time.Hour)

	r := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"exhausted": {key: "exhausted", val: 0, nextRefillAt: &windowRefillAt},
			"paused-a":  {key: "paused-a", val: 10, pausedUntil: &earlierPause},
			"paused-b":  {key: "paused-b", val: 10, pausedUntil: &laterPause},
		},
		unacked:   make(map[int64]rateLimitSet),
		unflushed: make(rateLimitSet),
		l:         &l,
	}

	// requeuing at the window refill would only find the key still paused
	res := r.use(context.Background(), 1, map[string]int32{"exhausted": 1, "paused-a": 1, "paused-b": 1})
	assert.False(t, res.succeeded)
	assert.Equal(t, "paused-b", res.exceededKey)
	assert.Equal(t, &laterPause, res.pausedUntil)
	assert.Equal(t, &laterPause, res.nextRefillAt)
}

func TestRateLimiter_ExpiredPauseRefreshesAndSeesExtension(t *testing.T) {
	l := zerolog.Nop()

	expired := time.Now().UTC().Add(-time.Second)
	extended := time.Now().UTC().Add(time.Hour)
	nextRefill := time.Now().Add(time.Minute)

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRateLimitRepo.On("FlushRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return([]*sqlcv1.ListRateLimitsForTenantWithMutateRow{
		{Key: "key1", Value: 10, PausedUntil: pgtype.Timestamptz{Time: extended, Valid: true}},
	}, &nextRefill, nil).Once()

	// the refill deadline is in the future, so only the expired pause can trigger this read
	future := time.Now().UTC().Add(time.Minute)

	r := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10, pausedUntil: &expired},
		},
		lastFlushAt:   time.Now().UTC().Add(-time.Minute),
		nextRefillAt:  &future,
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	res := r.use(context.Background(), 1, map[string]int32{"key1": 1})
	assert.False(t, res.succeeded, "a pause extended after the snapshot was read must keep blocking")

	if assert.NotNil(t, res.pausedUntil) {
		assert.True(t, extended.Equal(*res.pausedUntil))
	}

	// the refreshed snapshot postdates the old deadline, so later calls don't read again
	r.use(context.Background(), 2, map[string]int32{"key1": 1})

	mockRateLimitRepo.AssertExpectations(t)
}

func TestRateLimiter_ExpiredPauseRefreshesAndRuns(t *testing.T) {
	l := zerolog.Nop()

	expired := time.Now().UTC().Add(-time.Second)
	nextRefill := time.Now().Add(time.Minute)

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRateLimitRepo.On("FlushRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return([]*sqlcv1.ListRateLimitsForTenantWithMutateRow{
		{Key: "key1", Value: 10},
	}, &nextRefill, nil).Once()

	future := time.Now().UTC().Add(time.Minute)

	r := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10, pausedUntil: &expired},
		},
		lastFlushAt:   time.Now().UTC().Add(-time.Minute),
		nextRefillAt:  &future,
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	res := r.use(context.Background(), 1, map[string]int32{"key1": 1})
	assert.True(t, res.succeeded)
	assert.Nil(t, res.pausedUntil)

	mockRateLimitRepo.AssertExpectations(t)
}

func TestRateLimiter_ExpiredPauseReadAfterDeadlineSkipsRefresh(t *testing.T) {
	l := zerolog.Nop()

	// the database can still report a pause the engine's clock already sees as over; that read is trusted
	expired := time.Now().UTC().Add(-2 * time.Second)
	future := time.Now().UTC().Add(time.Minute)

	mockRateLimitRepo := &mockRateLimitRepo{}

	r := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10, pausedUntil: &expired},
		},
		lastFlushAt:   time.Now().UTC().Add(-time.Second),
		nextRefillAt:  &future,
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	res := r.use(context.Background(), 1, map[string]int32{"key1": 1})
	assert.True(t, res.succeeded)

	mockRateLimitRepo.AssertNotCalled(t, "FlushRateLimits", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestRateLimiter_RefreshBypassesRefillDeadlineOnce(t *testing.T) {
	l := zerolog.Nop()

	pausedUntil := time.Now().UTC().Add(time.Hour)
	nextRefill := time.Now().Add(time.Minute)

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRateLimitRepo.On("FlushRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return([]*sqlcv1.ListRateLimitsForTenantWithMutateRow{
		{Key: "key1", Value: 10, PausedUntil: pgtype.Timestamptz{Time: pausedUntil, Valid: true}},
	}, &nextRefill, nil).Once()

	future := time.Now().UTC().Add(time.Minute)

	r := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"key1": {key: "key1", val: 10},
		},
		nextRefillAt:  &future,
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	changedAt := time.Now().UTC()

	assert.NoError(t, r.flushToDatabase(context.Background()), "a plain flush waits for the refill deadline")
	assert.NoError(t, r.refresh(context.Background(), changedAt))

	// a second notification for the same change finds the snapshot already newer than it
	assert.NoError(t, r.refresh(context.Background(), changedAt))

	res := r.use(context.Background(), 1, map[string]int32{"key1": 1})
	assert.False(t, res.succeeded, "the pause written before the notification must apply without waiting for the refill deadline")

	mockRateLimitRepo.AssertExpectations(t)
}

func TestRateLimiter_PauseOnUnusedKeyDoesNotBlock(t *testing.T) {
	l := zerolog.Nop()

	pausedUntil := time.Now().UTC().Add(time.Hour)

	r := &rateLimiter{
		dbRateLimits: rateLimitSet{
			"paused": {key: "paused", val: 10, pausedUntil: &pausedUntil},
			"open":   {key: "open", val: 10},
		},
		unacked:   make(map[int64]rateLimitSet),
		unflushed: make(rateLimitSet),
		l:         &l,
	}

	res := r.use(context.Background(), 1, map[string]int32{"open": 1})
	assert.True(t, res.succeeded)
}

func TestRateLimiter_FlushLoadsPauses(t *testing.T) {
	l := zerolog.Nop()

	pausedUntil := time.Now().UTC().Add(time.Hour)
	nextRefill := time.Now().Add(2 * time.Second)

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRateLimitRepo.On("FlushRateLimits", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return([]*sqlcv1.ListRateLimitsForTenantWithMutateRow{
		{Key: "paused", Value: 10, PausedUntil: pgtype.Timestamptz{Time: pausedUntil, Valid: true}},
		{Key: "open", Value: 10},
	}, &nextRefill, nil)

	r := &rateLimiter{
		dbRateLimits:  make(rateLimitSet),
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     make(rateLimitSet),
		l:             &l,
		rateLimitRepo: mockRateLimitRepo,
	}

	assert.NoError(t, r.flushToDatabase(context.Background()))

	if assert.NotNil(t, r.dbRateLimits["paused"].pausedUntil) {
		assert.True(t, pausedUntil.Equal(*r.dbRateLimits["paused"].pausedUntil))
	}

	assert.Nil(t, r.dbRateLimits["open"].pausedUntil)

	res := r.use(context.Background(), 1, map[string]int32{"paused": 1})
	assert.False(t, res.succeeded)

	res = r.use(context.Background(), 2, map[string]int32{"open": 1})
	assert.True(t, res.succeeded)
}

func TestScheduleRateLimitResult_PausedItemsAlwaysMoveToDatabase(t *testing.T) {
	soon := time.Now().UTC().Add(500 * time.Millisecond)

	paused := &scheduleRateLimitResult{rateLimitResult: &rateLimitResult{nextRefillAt: &soon, pausedUntil: &soon}}
	assert.True(t, paused.shouldRemoveFromQueue(), "a short pause must still park the item so its schedule timeout is shifted")

	window := &scheduleRateLimitResult{rateLimitResult: &rateLimitResult{nextRefillAt: &soon}}
	assert.False(t, window.shouldRemoveFromQueue(), "a window refilling within the threshold stays in memory")
}

func TestTenantManager_NotifyRateLimitsChangedRefreshesLimiter(t *testing.T) {
	l := zerolog.Nop()

	pausedUntil := time.Now().UTC().Add(time.Hour)
	nextRefill := time.Now().Add(time.Minute)

	mockRateLimitRepo := &mockRateLimitRepo{}
	mockRateLimitRepo.On("FlushRateLimits", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*sqlcv1.ListRateLimitsForTenantWithMutateRow{
		{Key: "key1", Value: 10, PausedUntil: pgtype.Timestamptz{Time: pausedUntil, Valid: true}},
	}, &nextRefill, nil).Once()

	future := time.Now().UTC().Add(time.Minute)

	tm := &tenantManager{
		l: l,
		rl: &rateLimiter{
			dbRateLimits: rateLimitSet{
				"key1": {key: "key1", val: 10},
			},
			lastFlushAt:   time.Now().UTC().Add(-time.Second),
			nextRefillAt:  &future,
			unacked:       make(map[int64]rateLimitSet),
			unflushed:     make(rateLimitSet),
			l:             &l,
			rateLimitRepo: mockRateLimitRepo,
		},
	}

	tm.notifyRateLimitsChanged(context.Background())

	if assert.NotNil(t, tm.rl.dbRateLimits["key1"].pausedUntil) {
		assert.True(t, pausedUntil.Equal(*tm.rl.dbRateLimits["key1"].pausedUntil))
	}

	mockRateLimitRepo.AssertExpectations(t)
}
