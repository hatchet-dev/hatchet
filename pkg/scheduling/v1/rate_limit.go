package v2

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
)

type rateLimit struct {
	key string
	val int
}

type rateLimitSet map[string]*rateLimit

type rateLimiter struct {
	rateLimitRepo v1.RateLimitRepository

	tenantId pgtype.UUID

	l *zerolog.Logger

	// unacked is a map of taskId to rateLimitSet
	unacked   map[int64]rateLimitSet
	unackedMu sync.RWMutex

	unflushedMu sync.RWMutex
	unflushed   rateLimitSet

	dbRateLimitsMu sync.RWMutex
	dbRateLimits   rateLimitSet

	cleanup func()
}

func newRateLimiter(conf *sharedConfig, tenantId pgtype.UUID) *rateLimiter {
	rl := &rateLimiter{
		rateLimitRepo: conf.repo.RateLimit(),
		tenantId:      tenantId,
		l:             conf.l,
		unacked:       make(map[int64]rateLimitSet),
		unflushed:     make(rateLimitSet),
		dbRateLimits:  make(rateLimitSet),
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

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := r.flushToDatabase(ctx)

			if err != nil {
				r.l.Error().Err(err).Msg("error flushing rate limits to database")
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
}

// use returns true if the rate limits are not exceeded, false otherwise
func (r *rateLimiter) use(ctx context.Context, taskId int64, rls map[string]int32) (res rateLimitResult) {
	res.taskId = taskId

	// start with the db rate limits as the source of truth
	// if we don't have all rate limits in memory, check the database to determine if it exists
	if !r.rateLimitsExist(rls) {
		err := r.flushToDatabase(ctx)

		if err != nil {
			r.l.Error().Err(err).Msg("error flushing rate limits to database")
			return res
		}

		if !r.rateLimitsExist(rls) {
			return res
		}
	}

	currRls := r.copyDbRateLimits()

	// we need to subtract any relevant unacked and unflushed rate limits for updates
	r.subtractUnacked(rls, currRls)
	r.subtractUnflushed(rls, currRls)

	// determine if we can use all the rate limits in the set
	for k, v := range rls {
		if currRls[k].val < int(v) {
			res.exceededKey = k
			res.exceededUnits = v
			res.exceededVal = int32(currRls[k].val) // nolint: gosec

			return res
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

func (r *rateLimiter) copyDbRateLimits() rateLimitSet {
	r.dbRateLimitsMu.RLock()
	defer r.dbRateLimitsMu.RUnlock()

	rls := make(rateLimitSet)

	for k, v := range r.dbRateLimits {
		rls[k] = &rateLimit{
			key: k,
			val: v.val,
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

// flushToDatabase involves writing the rate limits and reading new rate limits from the
// database
func (r *rateLimiter) flushToDatabase(ctx context.Context) error {
	r.unflushedMu.Lock()
	defer r.unflushedMu.Unlock()

	r.dbRateLimitsMu.Lock()
	defer r.dbRateLimitsMu.Unlock()

	// copy the unflushed rate limits to a new map
	updates := make(map[string]int)

	for k, v := range r.unflushed {
		updates[k] = v.val
	}

	newRateLimits, err := r.rateLimitRepo.UpdateRateLimits(ctx, r.tenantId, updates)

	if err != nil {
		return err
	}

	r.dbRateLimits = make(rateLimitSet)

	// update the db rate limits
	for key, newVal := range newRateLimits {
		r.dbRateLimits[key] = &rateLimit{
			key: key,
			val: newVal,
		}
	}

	// clear the unflushed rate limits
	r.unflushed = make(rateLimitSet)

	return nil
}
