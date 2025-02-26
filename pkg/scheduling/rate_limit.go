package scheduling

import (
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type RateLimit struct {
	key               string
	currUnitsConsumed int32
	maxUnits          int32
	nextRefill        time.Time
	stepRunIdsToUnits map[string]int32
	mu                sync.Mutex
}

func NewRateLimit(key string, rl *dbsqlc.ListRateLimitsForTenantWithMutateRow) *RateLimit {
	return &RateLimit{
		key:               key,
		maxUnits:          rl.Value,
		nextRefill:        rl.NextRefillAt.Time,
		stepRunIdsToUnits: make(map[string]int32),
	}
}

func (rl *RateLimit) Key() string {
	return rl.key
}

func (rl *RateLimit) UnitsConsumed() int32 {
	return rl.currUnitsConsumed
}

func (rl *RateLimit) NextRefill() time.Time {
	return rl.nextRefill
}

func (rl *RateLimit) AddStepRunId(stepRunId string, units int32) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.stepRunIdsToUnits[stepRunId] = units
	rl.currUnitsConsumed += units

	if rl.currUnitsConsumed > rl.maxUnits {
		rl.rollback(stepRunId)
		return false
	}

	return true
}

func (rl *RateLimit) Rollback(stepRunId string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.rollback(stepRunId)
}

func (rl *RateLimit) rollback(stepRunId string) {
	if units, ok := rl.stepRunIdsToUnits[stepRunId]; ok {
		rl.currUnitsConsumed -= units
		delete(rl.stepRunIdsToUnits, stepRunId)
	}
}
