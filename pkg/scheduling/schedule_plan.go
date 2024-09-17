package scheduling

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type SchedulePlan struct {
	StepRunIds             []int64
	StepRunTimeouts        []string
	SlotIds                []string
	WorkerIds              []pgtype.UUID
	UnassignedStepRunIds   []int64
	QueuedStepRuns         []repository.QueuedStepRun
	TimedOutStepRuns       []int64
	RateLimitedStepRuns    []int64
	RateLimitedQueues      map[string][]time.Time
	QueuedItems            []int64
	ShouldContinue         bool
	MinQueuedIds           map[string]int64
	RateLimitUnitsConsumed map[string]int32
	unassignedActions      map[string]struct{}
}

func (sp *SchedulePlan) UpdateMinQueuedIds(qi *QueueItemWithOrder) []repository.QueuedStepRun {
	if qi.Priority == 1 {
		if currMinQueued, ok := sp.MinQueuedIds[qi.Queue]; !ok {
			sp.MinQueuedIds[qi.Queue] = qi.ID
		} else if qi.ID < currMinQueued {
			sp.MinQueuedIds[qi.Queue] = qi.ID
		}
	}

	return sp.QueuedStepRuns
}

func (plan *SchedulePlan) HandleTimedOut(qi *QueueItemWithOrder) {
	plan.TimedOutStepRuns = append(plan.TimedOutStepRuns, qi.StepRunId.Int64)
	// mark as queued so that we don't requeue
	plan.QueuedItems = append(plan.QueuedItems, qi.ID)
}

func (plan *SchedulePlan) HandleNoSlots(qi *QueueItemWithOrder) {
	plan.UnassignedStepRunIds = append(plan.UnassignedStepRunIds, qi.StepRunId.Int64)

	plan.unassignedActions[qi.ActionId.String] = struct{}{}
}

func (plan *SchedulePlan) HandleUnassigned(qi *QueueItemWithOrder) {
	plan.UnassignedStepRunIds = append(plan.UnassignedStepRunIds, qi.StepRunId.Int64)

	plan.unassignedActions[qi.ActionId.String] = struct{}{}
}

func (plan *SchedulePlan) HandleRateLimited(qi *QueueItemWithOrder) {
	plan.RateLimitedStepRuns = append(plan.RateLimitedStepRuns, qi.StepRunId.Int64)
}

func (plan *SchedulePlan) AssignQiToSlot(qi *QueueItemWithOrder, slot *Slot) {
	plan.StepRunIds = append(plan.StepRunIds, qi.StepRunId.Int64)
	plan.StepRunTimeouts = append(plan.StepRunTimeouts, qi.StepTimeout.String)
	plan.SlotIds = append(plan.SlotIds, slot.ID)
	plan.WorkerIds = append(plan.WorkerIds, sqlchelpers.UUIDFromStr(slot.WorkerId))
	plan.QueuedItems = append(plan.QueuedItems, qi.ID)

	plan.QueuedStepRuns = append(plan.QueuedStepRuns, repository.QueuedStepRun{
		StepRunId:    qi.StepRunId.Int64,
		WorkerId:     slot.WorkerId,
		DispatcherId: slot.DispatcherId,
	})
}
