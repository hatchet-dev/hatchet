package scheduling

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type RateLimitedResult struct {
	StepRuns []pgtype.UUID
	Keys     []string
	Units    []int32
}

type SchedulePlan struct {
	StepRunIds             []pgtype.UUID
	StepRunTimeouts        []string
	SlotIds                []string
	WorkerIds              []pgtype.UUID
	UnassignedStepRunIds   []pgtype.UUID
	QueuedStepRuns         []repository.QueuedStepRun
	TimedOutStepRuns       []pgtype.UUID
	RateLimitedStepRuns    RateLimitedResult
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
	plan.TimedOutStepRuns = append(plan.TimedOutStepRuns, qi.StepRunId)
	// mark as queued so that we don't requeue
	plan.QueuedItems = append(plan.QueuedItems, qi.ID)
}

func (plan *SchedulePlan) HandleNoSlots(qi *QueueItemWithOrder) {
	plan.UnassignedStepRunIds = append(plan.UnassignedStepRunIds, qi.StepRunId)

	plan.unassignedActions[qi.ActionId.String] = struct{}{}
}

func (plan *SchedulePlan) HandleUnassigned(qi *QueueItemWithOrder) {
	plan.UnassignedStepRunIds = append(plan.UnassignedStepRunIds, qi.StepRunId)

	plan.unassignedActions[qi.ActionId.String] = struct{}{}
}

func (plan *SchedulePlan) HandleRateLimited(qi *QueueItemWithOrder, key string, units int32) {
	plan.RateLimitedStepRuns.StepRuns = append(plan.RateLimitedStepRuns.StepRuns, qi.StepRunId)
	plan.RateLimitedStepRuns.Keys = append(plan.RateLimitedStepRuns.Keys, key)
	plan.RateLimitedStepRuns.Units = append(plan.RateLimitedStepRuns.Units, units)
}

func (plan *SchedulePlan) AssignQiToSlot(qi *QueueItemWithOrder, slot *Slot) {
	plan.StepRunIds = append(plan.StepRunIds, qi.StepRunId)
	plan.StepRunTimeouts = append(plan.StepRunTimeouts, qi.StepTimeout.String)
	plan.SlotIds = append(plan.SlotIds, slot.ID)
	plan.WorkerIds = append(plan.WorkerIds, sqlchelpers.UUIDFromStr(slot.WorkerId))
	plan.QueuedItems = append(plan.QueuedItems, qi.ID)

	plan.QueuedStepRuns = append(plan.QueuedStepRuns, repository.QueuedStepRun{
		StepRunId:    sqlchelpers.UUIDToStr(qi.StepRunId),
		WorkerId:     slot.WorkerId,
		DispatcherId: slot.DispatcherId,
	})
}
