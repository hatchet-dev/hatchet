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

type SchedulePlan struct {
	StepRunIds           []pgtype.UUID
	StepRunTimeouts      []string
	SlotIds              []pgtype.UUID
	WorkerIds            []pgtype.UUID
	UnassignedStepRunIds []pgtype.UUID
	QueuedStepRuns       []repository.QueuedStepRun
	TimedOutStepRuns     []pgtype.UUID
	QueuedItems          []int64
	ShouldContinue       bool
	MinQueuedIds         map[string]int64
}

func popRandMapValue(m map[string]*dbsqlc.ListSemaphoreSlotsToAssignRow) *dbsqlc.ListSemaphoreSlotsToAssignRow {
	for k, v := range m {
		delete(m, k)
		return v
	}

	return nil
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

	// NOTE(abelanger5): this is a version of the assignment problem. There is a more optimal solution i.e. optimal
	// matching which can run in polynomial time. This is a naive approach which assigns the first steps which were
	// queued to the first slots which are seen.
	actionsToSlots := make(map[string]map[string]*dbsqlc.ListSemaphoreSlotsToAssignRow)
	slotsToActions := make(map[string][]string)

	for _, slot := range slots {
		slotId := sqlchelpers.UUIDToStr(slot.ID)

		if _, ok := actionsToSlots[slot.ActionId]; !ok {
			actionsToSlots[slot.ActionId] = make(map[string]*dbsqlc.ListSemaphoreSlotsToAssignRow)
		}

		actionsToSlots[slot.ActionId][slotId] = slot

		if _, ok := slotsToActions[slotId]; !ok {
			slotsToActions[slotId] = make([]string, 0)
		}

		slotsToActions[slotId] = append(slotsToActions[slotId], slot.ActionId)
	}

	// assemble debug information
	startingSlotsPerAction := make(map[string]int)

	for action, slots := range actionsToSlots {
		startingSlotsPerAction[action] = len(slots)
	}

	allStepRunsWithActionAssigned := make(map[string]bool)

	for _, uniqueAction := range uniqueActionsArr {
		allStepRunsWithActionAssigned[uniqueAction] = true
	}

	for _, qi := range queueItems {
		if currMinQueued, ok := plan.MinQueuedIds[qi.Queue]; !ok {
			plan.MinQueuedIds[qi.Queue] = qi.ID
		} else if qi.ID < currMinQueued {
			plan.MinQueuedIds[qi.Queue] = qi.ID
		}

		if len(actionsToSlots[qi.ActionId.String]) == 0 {
			allStepRunsWithActionAssigned[qi.ActionId.String] = false
			plan.UnassignedStepRunIds = append(plan.UnassignedStepRunIds, qi.StepRunId)
			continue
		}

		// if the current time is after the scheduleTimeoutAt, then mark this as timed out
		now := time.Now().UTC().UTC()
		scheduleTimeoutAt := qi.ScheduleTimeoutAt.Time

		// timed out if the scheduleTimeoutAt is set and the current time is after the scheduleTimeoutAt
		isTimedOut := !scheduleTimeoutAt.IsZero() && scheduleTimeoutAt.Before(now)

		if isTimedOut {
			plan.TimedOutStepRuns = append(plan.TimedOutStepRuns, qi.StepRunId)
			// mark as queued so that we don't requeue
			plan.QueuedItems = append(plan.QueuedItems, qi.ID)
			continue
		}

		slot := popRandMapValue(actionsToSlots[qi.ActionId.String])

		// delete from all other actions
		for _, action := range slotsToActions[sqlchelpers.UUIDToStr(slot.ID)] {
			delete(actionsToSlots[action], sqlchelpers.UUIDToStr(slot.ID))
		}

		plan.StepRunIds = append(plan.StepRunIds, qi.StepRunId)
		plan.StepRunTimeouts = append(plan.StepRunTimeouts, qi.StepTimeout.String)
		plan.SlotIds = append(plan.SlotIds, slot.ID)
		plan.WorkerIds = append(plan.WorkerIds, slot.WorkerId)

		plan.QueuedStepRuns = append(plan.QueuedStepRuns, repository.QueuedStepRun{
			StepRunId:    sqlchelpers.UUIDToStr(qi.StepRunId),
			WorkerId:     sqlchelpers.UUIDToStr(slot.WorkerId),
			DispatcherId: sqlchelpers.UUIDToStr(slot.DispatcherId),
		})

		plan.QueuedItems = append(plan.QueuedItems, qi.ID)
	}

	// if at least one of the actions got all step runs assigned, and there are slots remaining, return true
	for _, action := range uniqueActionsArr {
		if _, ok := allStepRunsWithActionAssigned[action]; ok {
			// check if there are slots remaining
			if len(actionsToSlots[action]) > 0 {
				plan.ShouldContinue = true
				break
			}
		}
	}

	// TODO
	return plan, nil
}
