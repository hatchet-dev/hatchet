package scheduling

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type QueueItemWithOrder struct {
	*dbsqlc.QueueItem

	Order int
}

type Slot struct {
	ID           string
	WorkerId     string
	DispatcherId string
	ActionId     string
}

func GeneratePlan(
	ctx context.Context,
	slots []*Slot,
	uniqueActionsArr []string,
	queueItems []*QueueItemWithOrder,
	stepRunRateUnits map[string]map[string]int32,
	currRateLimits map[string]*dbsqlc.ListRateLimitsForTenantWithMutateRow,
	workerLabels map[string][]*dbsqlc.GetWorkerLabelsRow,
	stepDesiredLabels map[string][]*dbsqlc.GetDesiredLabelsRow,
) (SchedulePlan, error) {
	_, span := telemetry.NewSpan(ctx, "generate-scheduling-plan")
	defer span.End()

	plan := SchedulePlan{
		StepRunIds:           make([]pgtype.UUID, 0),
		StepRunTimeouts:      make([]string, 0),
		SlotIds:              make([]string, 0),
		WorkerIds:            make([]pgtype.UUID, 0),
		UnassignedStepRunIds: make([]pgtype.UUID, 0),
		RateLimitedStepRuns: RateLimitedResult{
			StepRuns: make([]pgtype.UUID, 0),
			Keys:     make([]string, 0),
		},
		QueuedStepRuns:         make([]repository.QueuedStepRun, 0),
		TimedOutStepRuns:       make([]pgtype.UUID, 0),
		QueuedItems:            make([]int64, 0),
		MinQueuedIds:           make(map[string]int64),
		ShouldContinue:         false,
		RateLimitUnitsConsumed: make(map[string]int32),
		unassignedActions:      make(map[string]struct{}),
	}

	// initialize worker states
	workerManager := NewWorkerStateManager(slots, workerLabels, stepDesiredLabels)

	workers := workerManager.workers

	rateLimits := make(map[string]*RateLimit)

	for key, rl := range currRateLimits {
		rateLimits[key] = NewRateLimit(key, rl)
	}

	// collect the queue counts
	queueCounts := make(map[string]int64)

	for _, qi := range queueItems {

		if _, ok := plan.MinQueuedIds[qi.Queue]; !ok {
			plan.MinQueuedIds[qi.Queue] = 0
		}

		queueCounts[qi.Queue]++
	}

	drainedQueues := make(map[string]struct{})

	// NOTE(abelanger5): this is a version of the assignment problem. There is a more optimal solution i.e. optimal
	// matching which can run in polynomial time. This is a naive approach which assigns the first steps which were
	// queued to the first slots which are seen.
	for _, qi := range queueItems {

		plan.UpdateMinQueuedIds(qi)

		// if we're timed out then mark as timed out
		if IsTimedout(qi) {
			plan.HandleTimedOut(qi)
			continue
		}

		stepId := sqlchelpers.UUIDToStr(qi.StepId)

		// if we're out of slots then mark as unassigned
		if !workerManager.HasEligibleWorkers(stepId) {
			plan.HandleNoSlots(qi)
			continue
		}

		stepRunId := sqlchelpers.UUIDToStr(qi.StepRunId)
		rateLimitedOnKey := ""
		rateLimitedUnits := int32(0)

		// check if we're rate limited
		if srRateLimitUnits, ok := stepRunRateUnits[stepRunId]; ok {
			for key, units := range srRateLimitUnits {
				if rateLimit, ok := rateLimits[key]; ok {
					// add the step run id to the rate limit. if this returns false, then we're rate limited
					if !rateLimit.AddStepRunId(stepRunId, units) {
						rateLimitedOnKey = key
						rateLimitedUnits = units
					}
				}
			}
		}

		// pick a worker to assign the slot to
		assigned := false

		if rateLimitedOnKey == "" {
			slot := workerManager.AttemptAssignSlot(qi)

			// we assign the slot to the plan
			if slot != nil {
				// add the slot to the plan
				plan.AssignQiToSlot(qi, slot)

				queueCounts[qi.Queue]--

				if _, ok := drainedQueues[qi.Queue]; !ok && queueCounts[qi.Queue] == 0 {
					drainedQueues[qi.Queue] = struct{}{}
				}

				assigned = true
			}
		}

		// if we couldn't assign the slot to any worker then mark as rate limited
		if rateLimitedOnKey != "" {
			plan.HandleRateLimited(qi, rateLimitedOnKey, rateLimitedUnits)

			// if we're rate limited then call rollback on the rate limits (this can happen if we've succeeded on one rate limit
			// but failed on another)
			for key := range stepRunRateUnits[stepRunId] {
				if rateLimit, ok := rateLimits[key]; ok {
					rateLimit.Rollback(stepRunId)
				}
			}
		} else if !assigned {
			plan.HandleUnassigned(qi)

			// if we can't assign the slot to any worker then we rollback the rate limit
			for key := range stepRunRateUnits[stepRunId] {
				if rateLimit, ok := rateLimits[key]; ok {
					rateLimit.Rollback(stepRunId)
				}
			}
		}
	}

	for key, rateLimit := range rateLimits {
		plan.RateLimitUnitsConsumed[key] = rateLimit.UnitsConsumed()
	}

	// we're looking to see if there's a single queue where all items have been properly scheduled
	if len(drainedQueues) > 0 && len(workers) > 0 {
		for _, qi := range queueItems {
			if _, ok := drainedQueues[qi.Queue]; ok {
				// if the queue is drained then we check if we can assign the action to any worker
				for _, workers := range workers {
					// if we can assign the action to any worker then we should continue and return early
					if workers.CanAssign(qi.ActionId.String, nil) {
						plan.ShouldContinue = true
						return plan, nil
					}
				}
			}

		}
	}

	return plan, nil
}
