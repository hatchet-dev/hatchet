package scheduling

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type QueueItemWithOrder struct {
	*dbsqlc.QueueItem

	Order int
}

func GeneratePlan(
	slots []*dbsqlc.ListSemaphoreSlotsToAssignRow,
	uniqueActionsArr []string,
	queueItems []*QueueItemWithOrder,
	stepRateUnits map[string]map[string]int32,
	currRateLimits map[string]*dbsqlc.ListRateLimitsForTenantRow,
	workerLabels map[string][]*dbsqlc.GetWorkerLabelsRow,
	stepDesiredLabels map[string][]*dbsqlc.GetDesiredLabelsRow,
) (SchedulePlan, error) {

	plan := SchedulePlan{
		StepRunIds:             make([]pgtype.UUID, 0),
		StepRunTimeouts:        make([]string, 0),
		SlotIds:                make([]pgtype.UUID, 0),
		WorkerIds:              make([]pgtype.UUID, 0),
		UnassignedStepRunIds:   make([]pgtype.UUID, 0),
		RateLimitedStepRuns:    make([]pgtype.UUID, 0),
		QueuedStepRuns:         make([]repository.QueuedStepRun, 0),
		TimedOutStepRuns:       make([]pgtype.UUID, 0),
		QueuedItems:            make([]int64, 0),
		MinQueuedIds:           make(map[string]int64),
		ShouldContinue:         false,
		RateLimitUnitsConsumed: make(map[string]int32),
		RateLimitedQueues:      make(map[string][]time.Time),
	}

	workers := make(map[string]*WorkerState)
	workerStepWeights := make(map[string][]WorkerWithWeight, 0)

	// initialize worker states
	for _, slot := range slots {
		workerId := sqlchelpers.UUIDToStr(slot.WorkerId)

		if _, ok := workers[workerId]; !ok {
			workers[workerId] = NewWorkerState(
				workerId,
				workerLabels[workerId],
			)
		}
		workers[sqlchelpers.UUIDToStr(slot.WorkerId)].AddSlot(slot)
	}

	// compute affinity weights
	for stepId, desired := range stepDesiredLabels {
		for workerId, worker := range workers {
			weight := ComputeWeight(desired, worker.labels)

			// skip workers that are not a match (i.e. required)
			if weight < 0 {
				continue
			}

			workerStepWeights[stepId] = append(workerStepWeights[stepId], WorkerWithWeight{
				WorkerId: workerId,
				Weight:   weight,
			})
		}
	}

	// sort the weights
	for _, weights := range workerStepWeights {
		SortWorkerWeights(weights)
	}

	rateLimits := make(map[string]*RateLimit)

	for key, rl := range currRateLimits {
		rateLimits[key] = NewRateLimit(key, rl)
	}

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
		if len(workers) == 0 {
			plan.HandleNoSlots(qi)
			continue
		}

		var labeledWorkers []WorkerWithWeight

		// desired labels
		if weightedWorkers, ok := workerStepWeights[stepId]; ok {
			labeledWorkers = weightedWorkers

			if len(labeledWorkers) == 0 {
				plan.HandleNoSlots(qi)
				continue
			}
		}

		stepRunId := sqlchelpers.UUIDToStr(qi.StepRunId)
		isRateLimited := false

		// check if we're rate limited
		if srRateLimitUnits, ok := stepRateUnits[stepId]; ok {
			for key, units := range srRateLimitUnits {
				if rateLimit, ok := rateLimits[key]; ok {
					// add the step run id to the rate limit. if this returns false, then we're rate limited
					if !rateLimit.AddStepRunId(stepRunId, units) {
						isRateLimited = true

						if _, ok := plan.RateLimitedQueues[key]; !ok {
							plan.RateLimitedQueues[key] = make([]time.Time, 0)
						}

						plan.RateLimitedQueues[qi.Queue] = append(plan.RateLimitedQueues[qi.Queue], rateLimit.NextRefill())
					}
				}
			}
		}

		// pick a worker to assign the slot to
		assigned := false

		if !isRateLimited {

			// TODO hack
			workerPool := make([]*WorkerState, 0, len(workers))

			if qi.Sticky.Valid {
				if worker, ok := workers[sqlchelpers.UUIDToStr(qi.DesiredWorkerId)]; ok {
					workerPool = append(workerPool, worker)
				}

				if qi.Sticky.StickyStrategy == dbsqlc.StickyStrategyHARD && len(workerPool) == 0 {
					plan.HandleNoSlots(qi)
					continue
				}
			}

			// skip finding alternative workers if we have a HARD sticky worker
			if qi.Sticky.StickyStrategy != dbsqlc.StickyStrategyHARD {
				// desired label workers
				if labeledWorkers != nil {
					// TODO hack
					for _, worker := range labeledWorkers {
						if _, ok := workers[worker.WorkerId]; ok {
							workerPool = append(workerPool, workers[worker.WorkerId])
						}
					}
				} else {
					for _, worker := range workers {
						workerPool = append(workerPool, worker)
					}
				}
			}

			for _, worker := range workerPool {
				slot, isEmpty := worker.AssignSlot(qi, stepDesiredLabels[sqlchelpers.UUIDToStr(qi.StepId)])

				if slot == nil {
					// if we can't assign the slot to the worker then continue
					continue
				}

				// add the slot to the plan
				plan.AssignQiToSlot(qi, slot)

				// cleanup the worker if it's empty
				if isEmpty {
					delete(workers, worker.workerId)
					// TODO
				}

				assigned = true

				// break out of the loop
				break
			}
		}

		// if we couldn't assign the slot to any worker then mark as rate limited
		if isRateLimited {
			plan.HandleRateLimited(qi)

			// if we're rate limited then call rollback on the rate limits (this can happen if we've succeeded on one rate limit
			// but failed on another)
			for key := range stepRateUnits[stepId] {
				if rateLimit, ok := rateLimits[key]; ok {
					rateLimit.Rollback(stepRunId)
				}
			}
		} else if !assigned {
			plan.HandleUnassigned(qi)

			// if we can't assign the slot to any worker then we rollback the rate limit
			for key := range stepRateUnits[stepId] {
				if rateLimit, ok := rateLimits[key]; ok {
					rateLimit.Rollback(stepRunId)
				}
			}
		}
	}

	for key, rateLimit := range rateLimits {
		plan.RateLimitUnitsConsumed[key] = rateLimit.UnitsConsumed()
	}

	// if we have any worker slots left and we have unassigned steps then we should continue
	// TODO revise this with a more optimal solution using maps
	if len(workers) > 0 && len(plan.UnassignedStepRunIds) > 0 {
		for _, qi := range queueItems {
			for _, worker := range workers {
				if worker.CanAssign(qi, stepDesiredLabels[sqlchelpers.UUIDToStr(qi.StepId)]) {
					plan.ShouldContinue = true
					break
				}
			}
			if plan.ShouldContinue {
				break
			}
		}
	}

	return plan, nil
}
