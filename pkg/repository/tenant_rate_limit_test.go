package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/limits"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func newTestRateLimitRepository(taskRunLimit, eventLimit int) *tenantRateLimitRepository {
	l := zerolog.Nop()
	shared := &sharedRepository{
		l: &l,
	}

	config := limits.LimitConfigFile{
		DefaultTaskRunRateLimit: taskRunLimit,
		DefaultEventRateLimit:   eventLimit,
	}

	return newTenantRateLimitRepository(shared, config, true, nil)
}

func TestTenantRateLimitRepository_BasicRateLimit(t *testing.T) {
	repo := newTestRateLimitRepository(10, 5)
	defer repo.Stop()

	ctx := context.Background()
	tenantId := "test-tenant-1"

	canCreate, _, err := repo.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
	require.NoError(t, err)
	assert.True(t, canCreate)

	precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
	err = precommit()
	require.NoError(t, err)
	postcommit()

	canCreate, _, err = repo.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
	require.NoError(t, err)
	assert.True(t, canCreate)
}

func TestTenantRateLimitRepository_RateLimitExceeded(t *testing.T) {
	repo := newTestRateLimitRepository(5, 5)
	defer repo.Stop()

	ctx := context.Background()
	tenantId := "test-tenant-2"

	for i := 0; i < 5; i++ {
		precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
		err := precommit()
		require.NoError(t, err)
		postcommit()
	}

	canCreate, percent, err := repo.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
	require.NoError(t, err)
	assert.False(t, canCreate)
	assert.Equal(t, 100, percent)

	precommit, _ := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
	err = precommit()
	assert.ErrorIs(t, err, ErrResourceExhausted)
}

func TestTenantRateLimitRepository_SeparateResources(t *testing.T) {
	repo := newTestRateLimitRepository(10, 5)
	defer repo.Stop()

	ctx := context.Background()
	tenantId := "test-tenant-3"

	for i := 0; i < 5; i++ {
		precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceEVENT, tenantId, 1)
		err := precommit()
		require.NoError(t, err)
		postcommit()
	}

	canCreate, _, err := repo.CanCreate(ctx, sqlcv1.LimitResourceEVENT, tenantId, 1)
	require.NoError(t, err)
	assert.False(t, canCreate)

	canCreate, _, err = repo.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
	require.NoError(t, err)
	assert.True(t, canCreate)
}

func TestTenantRateLimitRepository_SeparateTenants(t *testing.T) {
	repo := newTestRateLimitRepository(5, 5)
	defer repo.Stop()

	ctx := context.Background()
	tenant1 := "test-tenant-4a"
	tenant2 := "test-tenant-4b"

	for i := 0; i < 5; i++ {
		precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenant1, 1)
		err := precommit()
		require.NoError(t, err)
		postcommit()
	}

	canCreate, _, err := repo.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, tenant1, 1)
	require.NoError(t, err)
	assert.False(t, canCreate)

	canCreate, _, err = repo.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, tenant2, 1)
	require.NoError(t, err)
	assert.True(t, canCreate)
}

func TestTenantRateLimitRepository_NonRateLimitedResources(t *testing.T) {
	repo := newTestRateLimitRepository(5, 5)
	defer repo.Stop()

	ctx := context.Background()
	tenantId := "test-tenant-5"

	canCreate, _, err := repo.CanCreate(ctx, sqlcv1.LimitResourceWORKER, tenantId, 100)
	require.NoError(t, err)
	assert.True(t, canCreate)
}

func TestTenantRateLimitRepository_ConcurrentAccess(t *testing.T) {
	repo := newTestRateLimitRepository(100, 100)
	defer repo.Stop()

	ctx := context.Background()
	tenantId := "test-tenant-6"

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
			err := precommit()
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
				postcommit()
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, 50, successCount)

	canCreate, _, err := repo.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
	require.NoError(t, err)
	assert.True(t, canCreate)
}

func TestTenantRateLimitRepository_TenantGetter(t *testing.T) {
	repo := newTestRateLimitRepository(10, 10)
	defer repo.Stop()

	ctx := context.Background()
	tenant1 := "partition-tenant-1"
	tenant2 := "non-partition-tenant-2"

	precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenant1, 1)
	require.NoError(t, precommit())
	postcommit()

	precommit, postcommit = repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenant2, 1)
	require.NoError(t, precommit())
	postcommit()

	repo.SetTenantGetter(func() []string {
		return []string{tenant1}
	})

	repo.cleanupOldEntries()

	_, key1 := repo.checkRateLimit(tenant1, sqlcv1.LimitResourceTASKRUN, 0, false)
	assert.Equal(t, int32(1), key1)

	_, key2 := repo.checkRateLimit(tenant2, sqlcv1.LimitResourceTASKRUN, 0, false)
	assert.Equal(t, int32(0), key2)
}

func TestTenantRateLimitRepository_Disabled(t *testing.T) {
	l := zerolog.Nop()
	shared := &sharedRepository{
		l: &l,
	}

	config := limits.LimitConfigFile{
		DefaultTaskRunRateLimit: 1,
		DefaultEventRateLimit:   1,
	}

	repo := newTenantRateLimitRepository(shared, config, false, nil)
	defer repo.Stop()

	ctx := context.Background()
	tenantId := "test-tenant-disabled"

	canCreate, _, err := repo.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 100)
	require.NoError(t, err)
	assert.True(t, canCreate)

	precommit, _ := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 100)
	err = precommit()
	require.NoError(t, err)
}

func TestTenantRateLimitRepository_EnforceLimitsFunc(t *testing.T) {
	l := zerolog.Nop()
	shared := &sharedRepository{
		l: &l,
	}

	config := limits.LimitConfigFile{
		DefaultTaskRunRateLimit: 1,
		DefaultEventRateLimit:   1,
	}

	enforceLimitsFunc := func(ctx context.Context, tenantId string) (bool, error) {
		return tenantId == "enforced-tenant", nil
	}

	repo := newTenantRateLimitRepository(shared, config, true, enforceLimitsFunc)
	defer repo.Stop()

	ctx := context.Background()

	precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, "enforced-tenant", 1)
	require.NoError(t, precommit())
	postcommit()

	precommit, _ = repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, "enforced-tenant", 1)
	err := precommit()
	assert.ErrorIs(t, err, ErrResourceExhausted)

	precommit, postcommit = repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, "non-enforced-tenant", 100)
	require.NoError(t, precommit())
	postcommit()
}

func TestTenantRateLimitRepository_BulkRequest(t *testing.T) {
	repo := newTestRateLimitRepository(10, 10)
	defer repo.Stop()

	ctx := context.Background()
	tenantId := "test-tenant-bulk"

	canCreate, _, err := repo.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 5)
	require.NoError(t, err)
	assert.True(t, canCreate)

	precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 5)
	require.NoError(t, precommit())
	postcommit()

	canCreate, _, err = repo.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 6)
	require.NoError(t, err)
	assert.False(t, canCreate)

	precommit, postcommit = repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 5)
	require.NoError(t, precommit())
	postcommit()

	canCreate, percent, err := repo.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
	require.NoError(t, err)
	assert.False(t, canCreate)
	assert.Equal(t, 100, percent)
}

func TestTenantRateLimitRepository_SlidingWindowBuckets(t *testing.T) {
	repo := newTestRateLimitRepository(100, 100)
	defer repo.Stop()

	ctx := context.Background()
	tenantId := "test-tenant-sliding"

	window := repo.getWindow(tenantId, sqlcv1.LimitResourceTASKRUN)

	window.mu.Lock()
	for _, bucket := range window.buckets {
		assert.Equal(t, int32(0), bucket.count)
	}
	window.mu.Unlock()

	precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 10)
	require.NoError(t, precommit())
	postcommit()

	window.mu.Lock()
	totalCount := int32(0)
	for _, bucket := range window.buckets {
		totalCount += bucket.count
	}
	window.mu.Unlock()

	assert.Equal(t, int32(10), totalCount)
}

func TestTenantRateLimitRepository_DefaultLimits(t *testing.T) {
	repo := newTestRateLimitRepository(1000, 500)
	defer repo.Stop()

	defaults := repo.DefaultLimits()

	assert.Len(t, defaults, 2)

	var taskRunLimit *Limit
	var eventLimit *Limit
	for i := range defaults {
		if defaults[i].Resource == sqlcv1.LimitResourceTASKRUN {
			taskRunLimit = &defaults[i]
		}
		if defaults[i].Resource == sqlcv1.LimitResourceEVENT {
			eventLimit = &defaults[i]
		}
	}

	require.NotNil(t, taskRunLimit)
	assert.Equal(t, int32(1000), taskRunLimit.Limit)
	assert.Equal(t, int32(800), taskRunLimit.Alarm)

	require.NotNil(t, eventLimit)
	assert.Equal(t, int32(500), eventLimit.Limit)
	assert.Equal(t, int32(400), eventLimit.Alarm)
}

func TestTenantRateLimitRepository_OnSuccessMeterCallback(t *testing.T) {
	repo := newTestRateLimitRepository(100, 100)
	defer repo.Stop()

	ctx := context.Background()
	tenantId := "test-tenant-callback"

	var callbackCalled bool
	var callbackResource sqlcv1.LimitResource
	var callbackTenantId string
	var callbackUsage int64

	repo.SetOnSuccessMeterCallback(func(resource sqlcv1.LimitResource, tid string, usage int64) {
		callbackCalled = true
		callbackResource = resource
		callbackTenantId = tid
		callbackUsage = usage
	})

	precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 10)
	require.NoError(t, precommit())
	postcommit()

	time.Sleep(10 * time.Millisecond)

	assert.True(t, callbackCalled)
	assert.Equal(t, sqlcv1.LimitResourceTASKRUN, callbackResource)
	assert.Equal(t, tenantId, callbackTenantId)
	assert.Equal(t, int64(10), callbackUsage)
}

func TestTenantRateLimitRepository_Load10000Tenants(t *testing.T) {
	repo := newTestRateLimitRepository(1000, 1000)
	defer repo.Stop()

	ctx := context.Background()
	numTenants := 10000
	requestsPerTenant := 10

	tenantIds := make([]string, numTenants)
	for i := 0; i < numTenants; i++ {
		tenantIds[i] = fmt.Sprintf("load-test-tenant-%d", i)
	}

	start := time.Now()

	var wg sync.WaitGroup
	successCount := int64(0)
	var mu sync.Mutex

	for _, tenantId := range tenantIds {
		wg.Add(1)
		go func(tid string) {
			defer wg.Done()
			for j := 0; j < requestsPerTenant; j++ {
				precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tid, 1)
				if err := precommit(); err == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
					postcommit()
				}
			}
		}(tenantId)
	}

	wg.Wait()
	elapsed := time.Since(start)

	totalRequests := int64(numTenants * requestsPerTenant)
	assert.Equal(t, totalRequests, successCount)

	opsPerSecond := float64(totalRequests) / elapsed.Seconds()
	t.Logf("Load test results:")
	t.Logf("  Tenants: %d", numTenants)
	t.Logf("  Requests per tenant: %d", requestsPerTenant)
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Total time: %v", elapsed)
	t.Logf("  Operations/second: %.2f", opsPerSecond)

	assert.Greater(t, opsPerSecond, float64(100000), "Should handle at least 100k ops/sec")
}

func BenchmarkTenantRateLimitRepository_Meter(b *testing.B) {
	repo := newTestRateLimitRepository(1000000, 1000000)
	defer repo.Stop()

	ctx := context.Background()
	tenantId := "benchmark-tenant"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
		_ = precommit()
		postcommit()
	}
}

func BenchmarkTenantRateLimitRepository_MeterParallel(b *testing.B) {
	repo := newTestRateLimitRepository(100000000, 100000000)
	defer repo.Stop()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		tenantId := fmt.Sprintf("benchmark-tenant-%d", time.Now().UnixNano())
		for pb.Next() {
			precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
			_ = precommit()
			postcommit()
		}
	})
}

func BenchmarkTenantRateLimitRepository_CanCreate(b *testing.B) {
	repo := newTestRateLimitRepository(1000000, 1000000)
	defer repo.Stop()

	ctx := context.Background()
	tenantId := "benchmark-tenant"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = repo.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
	}
}

func BenchmarkTenantRateLimitRepository_10000Tenants(b *testing.B) {
	repo := newTestRateLimitRepository(1000000, 1000000)
	defer repo.Stop()

	ctx := context.Background()
	numTenants := 10000
	tenantIds := make([]string, numTenants)
	for i := 0; i < numTenants; i++ {
		tenantIds[i] = fmt.Sprintf("bench-tenant-%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tenantId := tenantIds[i%numTenants]
		precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
		_ = precommit()
		postcommit()
	}
}

func BenchmarkTenantRateLimitRepository_100000Tenants(b *testing.B) {
	repo := newTestRateLimitRepository(1000000, 1000000)
	defer repo.Stop()

	ctx := context.Background()
	numTenants := 100000
	tenantIds := make([]string, numTenants)
	for i := 0; i < numTenants; i++ {
		tenantIds[i] = fmt.Sprintf("bench-tenant-%d", i)
	}

	for i := 0; i < numTenants; i++ {
		precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantIds[i], 1)
		_ = precommit()
		postcommit()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tenantId := tenantIds[i%numTenants]
		precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
		_ = precommit()
		postcommit()
	}
}

func BenchmarkTenantRateLimitRepository_100000TenantsParallel(b *testing.B) {
	repo := newTestRateLimitRepository(1000000, 1000000)
	defer repo.Stop()

	ctx := context.Background()
	numTenants := 100000
	tenantIds := make([]string, numTenants)
	for i := 0; i < numTenants; i++ {
		tenantIds[i] = fmt.Sprintf("bench-tenant-%d", i)
	}

	for i := 0; i < numTenants; i++ {
		precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantIds[i], 1)
		_ = precommit()
		postcommit()
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			tenantId := tenantIds[i%numTenants]
			precommit, postcommit := repo.Meter(ctx, sqlcv1.LimitResourceTASKRUN, tenantId, 1)
			_ = precommit()
			postcommit()
			i++
		}
	})
}
