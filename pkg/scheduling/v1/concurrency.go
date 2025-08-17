package v1

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/scheduling/v0/randomticker"
)

type ConcurrencyResults struct {
	*v1.RunConcurrencyResult

	TenantId pgtype.UUID
}

type ConcurrencyManager struct {
	l *zerolog.Logger

	strategy *sqlcv1.V1StepConcurrency

	tenantId pgtype.UUID

	repo v1.ConcurrencyRepository

	notifyConcurrencyCh chan map[string]string
	notifyMu            mutex

	resultsCh chan<- *ConcurrencyResults

	cleanup func()

	isCleanedUp bool

	rateLimiter *rate.Limiter
}

func newConcurrencyManager(conf *sharedConfig, tenantId pgtype.UUID, strategy *sqlcv1.V1StepConcurrency, resultsCh chan<- *ConcurrencyResults) *ConcurrencyManager {
	repo := conf.repo.Concurrency()

	notifyConcurrencyCh := make(chan map[string]string, 2)

	c := &ConcurrencyManager{
		repo:                repo,
		strategy:            strategy,
		tenantId:            tenantId,
		l:                   conf.l,
		notifyConcurrencyCh: notifyConcurrencyCh,
		resultsCh:           resultsCh,
		notifyMu:            newMu(conf.l),
		rateLimiter:         newConcurrencyRateLimiter(conf.schedulerConcurrencyRateLimit),
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

	// non-blocking write
	select {
	case c.notifyConcurrencyCh <- telemetry.GetCarrier(ctx):
	default:
	}
}

func (c *ConcurrencyManager) loopConcurrency(ctx context.Context) {
	ticker := randomticker.NewRandomTicker(500*time.Millisecond, 5*time.Second)
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

		telemetry.WithAttributes(span, telemetry.AttributeKV{
			Key:   "strategy_id",
			Value: c.strategy.ID,
		})

		if !c.rateLimiter.Allow() {
			span.End()
			c.l.Debug().Msgf("rate limit exceeded for strategy %d", c.strategy.ID)
			continue
		}

		start := time.Now()

		results, err := c.repo.RunConcurrencyStrategy(ctx, c.tenantId, c.strategy)

		if err != nil {
			span.End()
			c.l.Error().Err(err).Msg("error running concurrency strategy")
			continue
		}

		if time.Since(start) > 100*time.Millisecond {
			c.l.Warn().
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
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		ctx, span := telemetry.NewSpan(ctx, "concurrency-check-active")

		telemetry.WithAttributes(span, telemetry.AttributeKV{
			Key:   "strategy_id",
			Value: c.strategy.ID,
		})

		start := time.Now()

		err := c.repo.UpdateConcurrencyStrategyIsActive(ctx, c.tenantId, c.strategy)

		if err != nil {
			span.End()
			c.l.Error().Err(err).Msg("error updating concurrency strategy is_active")
			continue
		}

		if time.Since(start) > 100*time.Millisecond {
			c.l.Warn().
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
