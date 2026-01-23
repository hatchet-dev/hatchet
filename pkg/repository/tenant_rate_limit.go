package repository

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/config/limits"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const (
	// rateLimitWindowSize is the total window size for rate limiting (1 minute)
	rateLimitWindowSize = time.Minute
	// rateLimitBucketCount is the number of buckets in the sliding window
	rateLimitBucketCount = 6
	// rateLimitBucketSize is the size of each bucket (10 seconds)
	rateLimitBucketSize = rateLimitWindowSize / rateLimitBucketCount
	// cleanupInterval is how often we clean up stale windows
	cleanupInterval = time.Minute
)

type tenantResourceKey struct {
	tenantId string
	resource sqlcv1.LimitResource
}

type slidingWindowBucket struct {
	timestamp time.Time
	count     int32
}

type slidingWindow struct {
	buckets [rateLimitBucketCount]slidingWindowBucket
	mu      sync.Mutex
}

type tenantRateLimitRepository struct {
	*sharedRepository
	windows           map[tenantResourceKey]*slidingWindow
	planRateLimits    *PlanRateLimitMap
	getTenants        TenantGetter
	stopChan          chan struct{}
	enforceLimitsFunc func(ctx context.Context, tenantId string) (bool, error)
	onSuccessMeterCb  func(resource sqlcv1.LimitResource, tenantId string, currentUsage int64)
	l                 *zerolog.Logger
	config            limits.LimitConfigFile
	windowsMu         sync.RWMutex
	tenantsMu         sync.RWMutex
	enabled           bool
}

func newTenantRateLimitRepository(
	shared *sharedRepository,
	config limits.LimitConfigFile,
	enabled bool,
	enforceLimitsFunc func(ctx context.Context, tenantId string) (bool, error),
) *tenantRateLimitRepository {
	r := &tenantRateLimitRepository{
		sharedRepository:  shared,
		windows:           make(map[tenantResourceKey]*slidingWindow),
		config:            config,
		stopChan:          make(chan struct{}),
		enabled:           enabled,
		enforceLimitsFunc: enforceLimitsFunc,
		l:                 shared.l,
	}

	go r.cleanupLoop()

	return r
}

// SetTenantGetter sets the function that returns tenant IDs for the current partition
func (r *tenantRateLimitRepository) SetTenantGetter(getter TenantGetter) {
	r.tenantsMu.Lock()
	defer r.tenantsMu.Unlock()
	r.getTenants = getter
}

// SetPlanRateLimitMap sets the plan rate limit map
func (r *tenantRateLimitRepository) SetPlanRateLimitMap(planRateLimitMap PlanRateLimitMap) error {
	r.planRateLimits = &planRateLimitMap
	return nil
}

func (r *tenantRateLimitRepository) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.cleanupOldEntries()
		}
	}
}

func (r *tenantRateLimitRepository) cleanupOldEntries() {
	r.tenantsMu.RLock()
	getTenants := r.getTenants
	r.tenantsMu.RUnlock()

	currentTenants := make(map[string]struct{})
	if getTenants != nil {
		for _, t := range getTenants() {
			currentTenants[t] = struct{}{}
		}
	}

	r.windowsMu.Lock()
	defer r.windowsMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rateLimitWindowSize)

	for key, window := range r.windows {
		// Remove if tenant not in current partition (only if we have a tenant getter)
		if getTenants != nil {
			if _, ok := currentTenants[key.tenantId]; !ok {
				delete(r.windows, key)
				continue
			}
		}

		// Check if all buckets are expired
		window.mu.Lock()
		allExpired := true
		for _, bucket := range window.buckets {
			if bucket.timestamp.After(cutoff) && bucket.count > 0 {
				allExpired = false
				break
			}
		}
		window.mu.Unlock()

		if allExpired {
			delete(r.windows, key)
		}
	}
}

func (r *tenantRateLimitRepository) getWindow(tenantId string, resource sqlcv1.LimitResource) *slidingWindow {
	key := tenantResourceKey{tenantId: tenantId, resource: resource}

	r.windowsMu.RLock()
	window, ok := r.windows[key]
	r.windowsMu.RUnlock()

	if ok {
		return window
	}

	r.windowsMu.Lock()
	defer r.windowsMu.Unlock()

	if window, ok := r.windows[key]; ok {
		return window
	}

	window = &slidingWindow{}
	r.windows[key] = window
	return window
}

func (r *tenantRateLimitRepository) getRateLimit(tenantId string, resource sqlcv1.LimitResource) int32 {
	// TODO(mnafees): Look up tenant's plan and get rate limit from plan
	// For now, use default limits
	switch resource {
	case sqlcv1.LimitResourceTASKRUN:
		return int32(r.config.DefaultTaskRunRateLimit) // nolint: gosec
	case sqlcv1.LimitResourceEVENT:
		return int32(r.config.DefaultEventRateLimit) // nolint: gosec
	default:
		return 0
	}
}

func (r *tenantRateLimitRepository) checkRateLimit(
	tenantId string,
	resource sqlcv1.LimitResource,
	count int32,
	increment bool,
) (allowed bool, currentUsage int32) {
	limit := r.getRateLimit(tenantId, resource)
	if limit <= 0 {
		return true, 0
	}

	window := r.getWindow(tenantId, resource)
	window.mu.Lock()
	defer window.mu.Unlock()

	now := time.Now()
	currentBucketIdx := int(now.Unix()/int64(rateLimitBucketSize.Seconds())) % rateLimitBucketCount

	var totalCount int32
	cutoff := now.Add(-rateLimitWindowSize)

	for i, bucket := range window.buckets {
		if bucket.timestamp.After(cutoff) {
			if i == currentBucketIdx {
				// Current bucket - check if it's from the current time window
				bucketTime := now.Truncate(rateLimitBucketSize)
				if bucket.timestamp.Before(bucketTime) {
					// Bucket is from previous cycle, don't count it
					continue
				}
			}
			totalCount += bucket.count
		}
	}

	if totalCount+count > limit {
		return false, totalCount
	}

	if increment {
		bucketTime := now.Truncate(rateLimitBucketSize)
		if window.buckets[currentBucketIdx].timestamp.Before(bucketTime) {
			// New bucket - reset count
			window.buckets[currentBucketIdx] = slidingWindowBucket{
				count:     count,
				timestamp: bucketTime,
			}
		} else {
			// Existing bucket - add to count
			window.buckets[currentBucketIdx].count += count
		}
	}

	return true, totalCount + count
}

func (r *tenantRateLimitRepository) GetLimits(ctx context.Context, tenantId string) ([]*sqlcv1.TenantResourceLimit, error) {
	return []*sqlcv1.TenantResourceLimit{}, nil
}

func (r *tenantRateLimitRepository) CanCreate(
	ctx context.Context,
	resource sqlcv1.LimitResource,
	tenantId string,
	numberOfResources int32,
) (bool, int, error) {
	if !r.enabled {
		return true, 0, nil
	}

	// Check per-tenant override
	if r.enforceLimitsFunc != nil {
		enforce, err := r.enforceLimitsFunc(ctx, tenantId)
		if err != nil {
			return false, 0, err
		}
		if !enforce {
			return true, 0, nil
		}
	}

	// Only rate limit TASK_RUN and EVENT resources
	if resource != sqlcv1.LimitResourceTASKRUN && resource != sqlcv1.LimitResourceEVENT {
		return true, 0, nil
	}

	allowed, usage := r.checkRateLimit(tenantId, resource, numberOfResources, false) // Check without incrementing
	limit := r.getRateLimit(tenantId, resource)

	percent := 0
	if limit > 0 {
		percent = int((float64(usage) / float64(limit)) * 100)
	}

	return allowed, percent, nil
}

func (r *tenantRateLimitRepository) SelectOrInsertTenantLimits(ctx context.Context, tenantId string, plan *string) error {
	// Rate limit repository doesn't use DB for limits, no-op
	return nil
}

func (r *tenantRateLimitRepository) UpsertTenantLimits(ctx context.Context, tenantId string, plan *string) error {
	// Rate limit repository doesn't use DB for limits, no-op
	return nil
}

func (r *tenantRateLimitRepository) ResolveAllTenantResourceLimits(ctx context.Context) error {
	// Rate limit repository doesn't use DB for limits, no-op
	return nil
}

func (r *tenantRateLimitRepository) SetPlanLimitMap(planLimitMap PlanLimitMap) error {
	// This is for DB-based limits, convert to rate limits if needed
	// For now, we use separate PlanRateLimitMap
	return nil
}

func (r *tenantRateLimitRepository) DefaultLimits() []Limit {
	return []Limit{
		{
			Resource:         sqlcv1.LimitResourceTASKRUN,
			Limit:            int32(r.config.DefaultTaskRunRateLimit),            // nolint: gosec
			Alarm:            int32(r.config.DefaultTaskRunRateLimit * 80 / 100), // nolint: gosec
			Window:           nil,                                                // In-memory, no window-based reset
			CustomValueMeter: false,
		},
		{
			Resource:         sqlcv1.LimitResourceEVENT,
			Limit:            int32(r.config.DefaultEventRateLimit),            // nolint: gosec
			Alarm:            int32(r.config.DefaultEventRateLimit * 80 / 100), // nolint: gosec
			Window:           nil,
			CustomValueMeter: false,
		},
	}
}

func (r *tenantRateLimitRepository) Meter(
	ctx context.Context,
	resource sqlcv1.LimitResource,
	tenantId string,
	numberOfResources int32,
) (precommit func() error, postcommit func()) {
	return func() error {
			if !r.enabled {
				return nil
			}

			// Check per-tenant override
			if r.enforceLimitsFunc != nil {
				enforce, err := r.enforceLimitsFunc(ctx, tenantId)
				if err != nil {
					return err
				}
				if !enforce {
					return nil
				}
			}

			// Only rate limit TASK_RUN and EVENT resources
			if resource != sqlcv1.LimitResourceTASKRUN && resource != sqlcv1.LimitResourceEVENT {
				return nil
			}

			allowed, _ := r.checkRateLimit(tenantId, resource, numberOfResources, true) // Check and increment
			if !allowed {
				return ErrResourceExhausted
			}
			return nil
		}, func() {
			// Callback for successful metering (already incremented in precommit)
			if r.onSuccessMeterCb != nil {
				_, usage := r.checkRateLimit(tenantId, resource, 0, false)
				r.onSuccessMeterCb(resource, tenantId, int64(usage))
			}
		}
}

func (r *tenantRateLimitRepository) SetOnSuccessMeterCallback(cb func(resource sqlcv1.LimitResource, tenantId string, currentUsage int64)) {
	r.onSuccessMeterCb = cb
}

func (r *tenantRateLimitRepository) Stop() {
	close(r.stopChan)
}
