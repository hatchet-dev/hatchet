package v1

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

const MAX_RATE_LIMIT_UPDATE_FREQUENCY = 500 * time.Millisecond // avoid boundary conditions on 1 second polls

type rateLimit struct {
	key          string
	val          int
	limitValue   int32
	window       string
	nextRefillAt *time.Time
	pausedUntil  *time.Time
}

type rateLimitSet map[string]*rateLimit

type rateLimiter struct {
	rateLimitRepo v1.RateLimitRepository

	tenantId uuid.UUID

	nextRefillAt   *time.Time
	nextRefillAtMu sync.RWMutex

	l *zerolog.Logger

	// unacked is a map of taskId to rateLimitSet
	unacked   map[int64]rateLimitSet
	unackedMu sync.RWMutex

	unflushedMu sync.RWMutex
	unflushed   rateLimitSet

	dbRateLimitsMu sync.RWMutex
	dbRateLimits   rateLimitSet

	// lastFlushAt is when dbRateLimits was read, guarded by dbRateLimitsMu
	lastFlushAt time.Time

	// dynamic rate limit definitions read by the queuers, written on the next flush
	pendingDefinitionsMu sync.Mutex
	pendingDefinitions   map[string]v1.RateLimitDefinition

	cleanup func()
}

func newRateLimiter(conf *sharedConfig, tenantId uuid.UUID) *rateLimiter {
	l := conf.l.With().Str("tenant_id", tenantId.String()).Logger()

	rl := &rateLimiter{
		rateLimitRepo:      conf.repo.RateLimit(),
		tenantId:           tenantId,
		l:                  &l,
		unacked:            make(map[int64]rateLimitSet),
		unflushed:          make(rateLimitSet),
		dbRateLimits:       make(rateLimitSet),
		pendingDefinitions: make(map[string]v1.RateLimitDefinition),
	}

	ctx, cancel := context.WithCancel(context.Background())
	rl.cleanup = cancel

	go rl.loopFlush(ctx)

	return rl
}

func (r *rateLimiter) Cleanup() {
	r.cleanup()
}

func (r *rateLimiter) loopFlush(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := r.flushToDatabase(ctx)

			if err != nil {
				logger.ShutdownAware(ctx, r.l, err, zerolog.ErrorLevel).Ctx(ctx).Err(err).Msg("error flushing rate limits to database")
			}
		}
	}
}

type rateLimitResult struct {
	succeeded bool
	taskId    int64

	ack  func()
	nack func()

	exceededKey   string
	exceededUnits int32
	exceededVal   int32
	nextRefillAt  *time.Time

	// pausedUntil is set when a paused key blocked the task, and is also its nextRefillAt
	pausedUntil *time.Time
}

// use returns true if the rate limits are not exceeded, false otherwise
func (r *rateLimiter) use(ctx context.Context, taskId int64, rls map[string]int32) (res rateLimitResult) {
	res.taskId = taskId

	// start with the db rate limits as the source of truth
	// if we don't have all rate limits in memory, check the database to determine if it exists
	if !r.rateLimitsExist(rls) {
		err := r.flushToDatabase(ctx)

		if err != nil {
			logger.ShutdownAware(ctx, r.l, err, zerolog.ErrorLevel).Ctx(ctx).Err(err).Msg("error flushing rate limits to database")

			return res
		}

		if !r.rateLimitsExist(rls) {
			return res
		}
	} else if r.shouldRefill() {
		err := r.flushToDatabase(ctx)

		if err != nil {
			logger.ShutdownAware(ctx, r.l, err, zerolog.ErrorLevel).Ctx(ctx).Err(err).Msg("error flushing rate limits to database")

			return res
		}
	}

	// a pause that ended after the snapshot was read may have been extended since, and acting on the stale
	// deadline would dispatch every run that was parked behind it
	if expiredAt, ok := r.pauseExpiredSinceFlush(rls); ok {
		err := r.refresh(ctx, expiredAt)

		if err != nil {
			logger.ShutdownAware(ctx, r.l, err, zerolog.ErrorLevel).Ctx(ctx).Err(err).Msg("error refreshing paused rate limits")

			return res
		}
	}

	currRls := r.copyDbRateLimits()

	// a pause wins over an exhausted window, since the window may refill long before the pause ends
	now := time.Now().UTC()

	for k, v := range rls {
		if currRl, ok := currRls[k]; ok && currRl.pausedUntil != nil && currRl.pausedUntil.After(now) {
			if res.pausedUntil == nil || currRl.pausedUntil.After(*res.pausedUntil) {
				res.exceededKey = k
				res.exceededUnits = v
				res.pausedUntil = currRl.pausedUntil
				res.nextRefillAt = currRl.pausedUntil
			}
		}
	}

	if res.pausedUntil != nil {
		return res
	}

	// we need to subtract any relevant unacked and unflushed rate limits for updates
	r.subtractUnacked(rls, currRls)
	r.subtractUnflushed(rls, currRls)

	// determine if we can use all the rate limits in the set
	for k, v := range rls {
		if currRl, ok := currRls[k]; ok {
			if currRl.val < int(v) {
				res.exceededKey = k
				res.exceededUnits = v
				res.exceededVal = int32(currRl.val) // nolint: gosec
				res.nextRefillAt = currRl.nextRefillAt

				return res
			}
		}
	}

	// if we can use all the rate limits, add them to the unacked set
	r.addToUnacked(taskId, rls)

	return rateLimitResult{
		succeeded: true,
		ack: func() {
			r.ack(taskId)
		},
		nack: func() {
			r.nack(taskId)
		},
	}
}

func (r *rateLimiter) rateLimitsExist(rls map[string]int32) bool {
	r.dbRateLimitsMu.RLock()
	defer r.dbRateLimitsMu.RUnlock()

	for k := range rls {
		if _, ok := r.dbRateLimits[k]; !ok {
			return false
		}
	}

	return true
}

// pauseExpiredSinceFlush returns the latest pause deadline among the keys that has passed since the snapshot was read
func (r *rateLimiter) pauseExpiredSinceFlush(rls map[string]int32) (time.Time, bool) {
	r.dbRateLimitsMu.RLock()
	defer r.dbRateLimitsMu.RUnlock()

	now := time.Now().UTC()

	var expiredAt time.Time
	found := false

	for k := range rls {
		rl, ok := r.dbRateLimits[k]

		if !ok || rl.pausedUntil == nil || rl.pausedUntil.After(now) || !rl.pausedUntil.After(r.lastFlushAt) {
			continue
		}

		if !found || rl.pausedUntil.After(expiredAt) {
			expiredAt = *rl.pausedUntil
			found = true
		}
	}

	return expiredAt, found
}

func (r *rateLimiter) shouldRefill() bool {
	r.nextRefillAtMu.Lock()
	defer r.nextRefillAtMu.Unlock()

	if r.nextRefillAt == nil {
		return false
	}

	// refill once the deadline has been reached, so use() sees refilled limits
	// immediately instead of waiting for the next background flush tick. The
	// early-return guard in flushToDatabase bounds how often this hits the
	// database.
	return !r.nextRefillAt.After(time.Now().UTC())
}

func (r *rateLimiter) copyDbRateLimits() rateLimitSet {
	r.dbRateLimitsMu.RLock()
	defer r.dbRateLimitsMu.RUnlock()

	rls := make(rateLimitSet)

	for k, v := range r.dbRateLimits {
		rls[k] = &rateLimit{
			key:          k,
			val:          v.val,
			nextRefillAt: v.nextRefillAt,
			pausedUntil:  v.pausedUntil,
		}
	}

	return rls
}

func (r *rateLimiter) subtractUnacked(candidateRls map[string]int32, currRls rateLimitSet) {
	r.unackedMu.RLock()
	defer r.unackedMu.RUnlock()

	for _, set := range r.unacked {
		for k, v := range set {
			if _, ok := candidateRls[k]; ok {
				if _, ok := currRls[k]; ok {
					unackedRl := v
					currRls[k].val -= unackedRl.val
				}
			}
		}
	}
}

func (r *rateLimiter) subtractUnflushed(candidateRls map[string]int32, currRls rateLimitSet) {
	r.unflushedMu.RLock()
	defer r.unflushedMu.RUnlock()

	for k, v := range r.unflushed {
		if _, ok := candidateRls[k]; ok {
			if _, ok := currRls[k]; ok {
				currRls[k].val -= v.val
			}
		}
	}
}

func (r *rateLimiter) addToUnacked(taskId int64, rls map[string]int32) {
	r.unackedMu.Lock()
	defer r.unackedMu.Unlock()

	for k, v := range rls {
		if _, ok := r.unacked[taskId]; !ok {
			r.unacked[taskId] = make(rateLimitSet)
		}

		r.unacked[taskId][k] = &rateLimit{
			key: k,
			val: int(v),
		}
	}
}

func (r *rateLimiter) ack(taskId int64) {
	// remove the rate limits from the unacked set and add them to the unflushed set
	r.unackedMu.Lock()
	defer r.unackedMu.Unlock()

	r.unflushedMu.Lock()
	defer r.unflushedMu.Unlock()

	if _, ok := r.unacked[taskId]; ok {
		for k, v := range r.unacked[taskId] {
			if _, ok := r.unflushed[k]; !ok {
				r.unflushed[k] = &rateLimit{
					key: k,
					val: 0,
				}
			}

			r.unflushed[k].val += v.val
		}

		delete(r.unacked, taskId)
	}
}

func (r *rateLimiter) nack(taskId int64) {
	// remove the rate limits from the unacked set
	r.unackedMu.Lock()
	defer r.unackedMu.Unlock()

	delete(r.unacked, taskId)
}

func (r *rateLimiter) addDefinitions(definitions map[string]v1.RateLimitDefinition) {
	if r == nil || len(definitions) == 0 {
		return
	}

	r.pendingDefinitionsMu.Lock()
	defer r.pendingDefinitionsMu.Unlock()

	if r.pendingDefinitions == nil {
		r.pendingDefinitions = make(map[string]v1.RateLimitDefinition)
	}

	for k, def := range definitions {
		r.pendingDefinitions[k] = def
	}
}

// takeChangedDefinitions drains the pending definitions, keeping only those that differ from the database. Callers must hold dbRateLimitsMu.
func (r *rateLimiter) takeChangedDefinitions() map[string]v1.RateLimitDefinition {
	r.pendingDefinitionsMu.Lock()
	defer r.pendingDefinitionsMu.Unlock()

	changed := make(map[string]v1.RateLimitDefinition)

	for k, def := range r.pendingDefinitions {
		if curr, ok := r.dbRateLimits[k]; ok && curr.limitValue == def.LimitValue && curr.window == def.Window {
			continue
		}

		changed[k] = def
	}

	r.pendingDefinitions = make(map[string]v1.RateLimitDefinition)

	return changed
}

// restoreDefinitions requeues definitions from a failed flush; definitions added since the flush started are newer and win.
func (r *rateLimiter) restoreDefinitions(definitions map[string]v1.RateLimitDefinition) {
	r.pendingDefinitionsMu.Lock()
	defer r.pendingDefinitionsMu.Unlock()

	if r.pendingDefinitions == nil {
		r.pendingDefinitions = make(map[string]v1.RateLimitDefinition)
	}

	for k, def := range definitions {
		if _, ok := r.pendingDefinitions[k]; !ok {
			r.pendingDefinitions[k] = def
		}
	}
}

// flushToDatabase involves writing the rate limits and reading new rate limits from the
// database
func (r *rateLimiter) flushToDatabase(ctx context.Context) error {
	return r.flush(ctx, nil)
}

// refresh reads the rate limits regardless of the refill deadline, unless a flush has already read them after staleBefore
func (r *rateLimiter) refresh(ctx context.Context, staleBefore time.Time) error {
	return r.flush(ctx, &staleBefore)
}

func (r *rateLimiter) flush(ctx context.Context, staleBefore *time.Time) error {
	r.unflushedMu.Lock()
	defer r.unflushedMu.Unlock()

	r.dbRateLimitsMu.Lock()
	defer r.dbRateLimitsMu.Unlock()

	r.nextRefillAtMu.Lock()
	defer r.nextRefillAtMu.Unlock()

	// concurrent callers that saw the same stale snapshot only need one of them to read
	if staleBefore != nil && !r.lastFlushAt.Before(*staleBefore) {
		return nil
	}

	definitions := r.takeChangedDefinitions()

	// check the refill time to avoid unnecessary database calls; new or changed definitions can't wait,
	// since use() treats a key with no row as unavailable
	if staleBefore == nil && r.nextRefillAt != nil && len(definitions) == 0 {
		if r.nextRefillAt.After(time.Now().UTC()) {
			return nil
		}
	}

	readAt := time.Now().UTC()

	// copy the unflushed rate limits to a new map
	usage := make(map[string]int)

	for k, v := range r.unflushed {
		usage[k] = v.val
	}

	newRateLimits, nextRefillAt, err := r.rateLimitRepo.FlushRateLimits(ctx, r.tenantId, usage, definitions)

	if err != nil {
		r.restoreDefinitions(definitions)
		return err
	}

	// don't update more than one time per second
	if nextRefillAt != nil {
		if time.Until(*nextRefillAt) < MAX_RATE_LIMIT_UPDATE_FREQUENCY {
			minRefillAt := time.Now().Add(MAX_RATE_LIMIT_UPDATE_FREQUENCY)
			nextRefillAt = &minRefillAt
		}

		// update the next refill time; we already have the lock
		r.nextRefillAt = nextRefillAt
	}

	r.dbRateLimits = make(rateLimitSet)

	// update the db rate limits
	for _, newVal := range newRateLimits {
		key := newVal.Key
		next := newVal.NextRefillAt.Time

		var pausedUntil *time.Time

		if newVal.PausedUntil.Valid {
			t := newVal.PausedUntil.Time
			pausedUntil = &t
		}

		r.dbRateLimits[key] = &rateLimit{
			key:          key,
			val:          int(newVal.Value),
			limitValue:   newVal.LimitValue,
			window:       newVal.Window,
			nextRefillAt: &next,
			pausedUntil:  pausedUntil,
		}
	}

	r.lastFlushAt = readAt

	// clear the unflushed rate limits
	r.unflushed = make(rateLimitSet)

	return nil
}
