package v1

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"

	"github.com/hatchet-dev/hatchet/internal/services/shared/timeout_lock"

	"github.com/hatchet-dev/hatchet/pkg/randomticker"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type ConcurrencyResults struct {
	*v1.RunConcurrencyResult

	TenantId uuid.UUID
}

type ConcurrencyManager struct {
	l *zerolog.Logger

	strategy *sqlcv1.V1StepConcurrency

	tenantId uuid.UUID

	repo v1.ConcurrencyRepository

	notifyConcurrencyCh chan map[string]string
	notifyMu            mutex

	resultsCh chan<- *ConcurrencyResults

	cleanup func()

	isCleanedUp bool

	rateLimiter *rate.Limiter

	minPollingInterval time.Duration

	maxPollingInterval time.Duration

	minCheckActiveInterval time.Duration

	maxCheckActiveInterval time.Duration

	advisoryLock       *timeout_lock.KeyedTimeoutLock[int64]
	advisoryParentLock *timeout_lock.KeyedTimeoutLock[int64]
}

func newConcurrencyManager(conf *sharedConfig, tenantId uuid.UUID, strategy *sqlcv1.V1StepConcurrency, resultsCh chan<- *ConcurrencyResults, advisoryLock *timeout_lock.KeyedTimeoutLock[int64], advisoryParentLock *timeout_lock.KeyedTimeoutLock[int64]) *ConcurrencyManager {
	repo := conf.repo.Concurrency()

	notifyConcurrencyCh := make(chan map[string]string, 2)

	l := conf.l.With().Str("tenant_id", tenantId.String()).Logger()

	c := &ConcurrencyManager{
		repo:                   repo,
		strategy:               strategy,
		tenantId:               tenantId,
		l:                      &l,
		notifyConcurrencyCh:    notifyConcurrencyCh,
		resultsCh:              resultsCh,
		notifyMu:               newMu(&l),
		rateLimiter:            newConcurrencyRateLimiter(conf.schedulerConcurrencyRateLimit),
		minPollingInterval:     conf.schedulerConcurrencyPollingMinInterval,
		maxPollingInterval:     conf.schedulerConcurrencyPollingMaxInterval,
		minCheckActiveInterval: conf.schedulerCheckActiveMinInterval,
		maxCheckActiveInterval: conf.schedulerCheckActiveMaxInterval,
		advisoryLock:           advisoryLock,
		advisoryParentLock:     advisoryParentLock,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cleanupMu := sync.Mutex{}
	c.cleanup = func() {
		cleanupMu.Lock()
		defer cleanupMu.Unlock()

		if c.isCleanedUp {
			return
		}

		c.isCleanedUp = true
		cancel()
	}

	go c.loopConcurrency(ctx)
	go c.loopCheckActive(ctx)

	return c
}

func (c *ConcurrencyManager) Cleanup() {
	c.cleanup()
}

func (c *ConcurrencyManager) notify(ctx context.Context) {
	ctx, span := telemetry.NewSpan(ctx, "notify-concurrency")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: c.tenantId.String()})

	// non-blocking write
	select {
	case c.notifyConcurrencyCh <- telemetry.GetCarrier(ctx):
	default:
	}
}

func (c *ConcurrencyManager) acquireStrategyLocks() bool {
	acquired := c.advisoryLock.Acquire(c.strategy.ID)
	if !acquired {
		return acquired
	}
	if c.strategy.ParentStrategyID.Valid {
		if !c.advisoryParentLock.Acquire(c.strategy.ParentStrategyID.Int64) {
			c.advisoryLock.Release(c.strategy.ID)
			return false
		}
	}
	return true

}

func (c *ConcurrencyManager) releaseStrategyLocks() {
	c.advisoryLock.Release(c.strategy.ID)
	if c.strategy.ParentStrategyID.Valid {
		c.advisoryParentLock.Release(c.strategy.ParentStrategyID.Int64)
	}
}

func (c *ConcurrencyManager) loopConcurrency(ctx context.Context) {
	ticker := randomticker.NewRandomTicker(
		c.minPollingInterval,
		c.maxPollingInterval,
	)
	defer ticker.Stop()

	for {
		var carrier map[string]string

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case carrier = <-c.notifyConcurrencyCh:
		}

		ctx, span := telemetry.NewSpanWithCarrier(ctx, "concurrency-manager", carrier)

		telemetry.WithAttributes(span,
			telemetry.AttributeKV{Key: "concurrency.strategy.id", Value: c.strategy.ID},
			telemetry.AttributeKV{Key: "tenant.id", Value: c.tenantId.String()},
		)

		if !c.rateLimiter.Allow() {
			span.End()
			c.l.Debug().Ctx(ctx).Msgf("rate limit exceeded for strategy %d", c.strategy.ID)
			continue
		}

		// acquire in-memory queue lock before running strategy because failure to acquire database-level
		// locks will delay scheduling until next polling tick
		lockStart := time.Now()
		if acquired := c.acquireStrategyLocks(); !acquired {
			span.End()
			c.l.Error().Ctx(ctx).Msg(fmt.Sprintf("(concurrency loop) could not acquire in-memory advisory lock in %s for strategy id %d, tenant id %s", time.Since(lockStart), c.strategy.ID, c.strategy.TenantID))
			continue
		}
		start := time.Now()
		results, err := c.repo.RunConcurrencyStrategy(ctx, c.tenantId, c.strategy)
		c.releaseStrategyLocks()
		if err != nil {
			span.End()
			c.l.Error().Ctx(ctx).Err(err).Msg("error running concurrency strategy")
			continue
		}

		if time.Since(start) > 100*time.Millisecond {
			c.l.Warn().Ctx(ctx).
				Msgf("concurrency strategy %d took longer than 100ms (%s) to process %d items", c.strategy.ID, time.Since(start), len(results.Queued))
		}
		c.resultsCh <- &ConcurrencyResults{
			RunConcurrencyResult: results,
			TenantId:             c.tenantId,
		}

		span.End()
	}
}

func (c *ConcurrencyManager) loopCheckActive(ctx context.Context) {
	ticker := randomticker.NewRandomTicker(
		c.minCheckActiveInterval,
		c.maxCheckActiveInterval,
	)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		ctx, span := telemetry.NewSpan(ctx, "concurrency-check-active")

		telemetry.WithAttributes(span,
			telemetry.AttributeKV{Key: "concurrency.strategy.id", Value: c.strategy.ID},
			telemetry.AttributeKV{Key: "tenant.id", Value: c.tenantId.String()},
		)
		lockStart := time.Now()
		if acquired := c.acquireStrategyLocks(); !acquired {
			span.End()
			c.l.Error().Ctx(ctx).Msg(fmt.Sprintf("(check active loop) could not acquire in-memory advisory lock in %s for strategy id %d, tenant id %s", time.Since(lockStart), c.strategy.ID, c.strategy.TenantID))
			continue
		}
		start := time.Now()
		err := c.repo.UpdateConcurrencyStrategyIsActive(ctx, c.tenantId, c.strategy)
		c.releaseStrategyLocks()
		if err != nil {
			span.End()
			c.l.Error().Ctx(ctx).Err(err).Msg("error updating concurrency strategy is_active")
			continue
		}

		if time.Since(start) > 100*time.Millisecond {
			c.l.Warn().Ctx(ctx).
				Msgf("checking is_active on concurrency strategy %d took longer than 100ms (%s)", c.strategy.ID, time.Since(start))
		}

		span.End()
	}
}

func newConcurrencyRateLimiter(rateLimit int) *rate.Limiter {
	if rateLimit <= 0 {
		rateLimit = 20
	}

	return rate.NewLimiter(rate.Limit(rateLimit), rateLimit)
}
