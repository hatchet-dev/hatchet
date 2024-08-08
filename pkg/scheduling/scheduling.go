package scheduling

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type QueueItemWithOrder struct {
	*dbsqlc.QueueItem

	Order int
}

// Generate generates a random string of n bytes.
func GeneratePlan(
	slots []*dbsqlc.ListSemaphoreSlotsToAssignRow,
	uniqueActionsArr []string,
	queueItems []QueueItemWithOrder,
) (SchedulePlan, error) {

	plan := SchedulePlan{
		StepRunIds:           make([]pgtype.UUID, 0),
		StepRunTimeouts:      make([]string, 0),
		SlotIds:              make([]pgtype.UUID, 0),
		WorkerIds:            make([]pgtype.UUID, 0),
		UnassignedStepRunIds: make([]pgtype.UUID, 0),
		QueuedStepRuns:       make([]repository.QueuedStepRun, 0),
		TimedOutStepRuns:     make([]pgtype.UUID, 0),
		QueuedItems:          make([]int64, 0),
		MinQueuedIds:         make(map[string]int64),
		ShouldContinue:       false,
	}

	workers := make(map[string]*WorkerState)

	// initialize worker states
	for _, slot := range slots {
		if _, ok := workers[sqlchelpers.UUIDToStr(slot.WorkerId)]; !ok {
			workers[sqlchelpers.UUIDToStr(slot.WorkerId)] = NewWorkerState(
				sqlchelpers.UUIDToStr(slot.WorkerId),
			)
		}
		workers[sqlchelpers.UUIDToStr(slot.WorkerId)].AddSlot(slot)
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

		// if we're out of slots then mark as unassigned
		if len(workers) == 0 {
			plan.HandleNoSlots(qi)
			continue
		}

		// pick a worker to assign the slot to
		assigned := false
		for _, worker := range workers {
			slot, isEmpty := worker.AssignSlot(qi)

			if slot == nil {
				// if we can't assign the slot to the worker then continue
				continue
			}

			// add the slot to the plan
			plan.AssignQiToSlot(qi, slot)

			// cleanup the worker if it's empty
			if isEmpty {
				delete(workers, worker.workerId)
			}

			assigned = true

			// break out of the loop
			break
		}

		// if we couldn't assign the slot to any worker then mark as unassigned
		if !assigned {
			plan.HandleUnassigned(qi)
		}
	}

	// if we have any worker slots left and we have unassigned steps then we should continue
	// TODO revise this with a more optimal solution using maps
	if len(workers) > 0 && len(plan.UnassignedStepRunIds) > 0 {
		for _, qi := range queueItems {
			for _, worker := range workers {
				if worker.CanAssign(qi) {
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
