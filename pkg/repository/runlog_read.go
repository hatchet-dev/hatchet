package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/hatchet-dev/hatchet/internal/statusutils"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/runlog"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// Read side of the run log. Readers of finished work ask core first and come here only for what core no
// longer has. The log is consulted only for days whose partitions maintenance has started to drop
// (v1_run_log_day DROPPING or DROPPED): their archive is complete and never changes, so a miss for any other
// day is a real miss and costs no object store request.

// runLogDroppedDaysRefresh bounds how often a miss on a day not known to be dropped rereads v1_run_log_day.
const runLogDroppedDaysRefresh = time.Second

// runLogPayloadSearchDays is how many days before the day a payload row is stored under the reader looks
// for the bundle that holds it. A bundle sits on its task's day, while the task's output and event payloads
// are stored under the day they were written, which is later by up to the longest retry backoff.
const runLogPayloadSearchDays = 31

type runLogReader struct {
	*runlog.Log

	pool *fairpool.Pool

	mu            sync.Mutex
	droppedDays   map[time.Time]bool
	droppedDaysAt time.Time

	// absent holds the tenant-days of dropped days that have no archive (the tenant had no task that day)
	absent *lru.Cache[string, struct{}]
}

func newRunLogReader(l *runlog.Log, pool *fairpool.Pool) *runLogReader {
	absent, err := lru.New[string, struct{}](4096)

	if err != nil {
		panic(err)
	}

	return &runLogReader{Log: l, pool: pool, droppedDays: map[time.Time]bool{}, absent: absent}
}

// dayDropped reports whether core has started dropping the day's partitions.
func (r *runLogReader) dayDropped(ctx context.Context, day time.Time) (bool, error) {
	day = runlog.Day(day)

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.droppedDays[day] || time.Since(r.droppedDaysAt) < runLogDroppedDaysRefresh {
		return r.droppedDays[day], nil
	}

	rows, err := r.pool.ForShared().Query(ctx, `SELECT day FROM v1_run_log_day WHERE state IN ('DROPPING', 'DROPPED')`)

	if err != nil {
		return false, fmt.Errorf("could not list dropped run log days: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var d time.Time

		if err := rows.Scan(&d); err != nil {
			return false, err
		}

		r.droppedDays[runlog.Day(d)] = true
	}

	if err := rows.Err(); err != nil {
		return false, err
	}

	r.droppedDaysAt = time.Now()

	return r.droppedDays[day], nil
}

// index returns the tenant-day's index, or nil when the day is not dropped or the tenant has no archive on it.
func (r *runLogReader) index(ctx context.Context, tenantId uuid.UUID, day time.Time) (*runlog.Index, error) {
	dropped, err := r.dayDropped(ctx, day)

	if err != nil || !dropped {
		return nil, err
	}

	key := r.DayPrefix(tenantId, day)

	if r.absent.Contains(key) {
		return nil, nil
	}

	idx, err := r.Index(ctx, tenantId, day)

	if errors.Is(err, runlog.ErrNotFound) {
		r.absent.Add(key, struct{}{})
		return nil, nil
	}

	return idx, err
}

// fillPayloads adds to found the payloads of opts that core did not have and the run log holds.
func (r *runLogReader) fillPayloads(ctx context.Context, opts []RetrievePayloadOpts, found map[RetrievePayloadOpts][]byte) error {
	for _, opt := range opts {
		if _, ok := found[opt]; ok {
			continue
		}

		externalId := opt.effectiveExternalId()

		if externalId == uuid.Nil {
			continue
		}

		value, err := r.payload(ctx, opt.TenantId, payloadStoredAt(opt), externalId)

		if err != nil {
			return err
		}

		if value != nil {
			found[opt] = value
		}
	}

	return nil
}

// payloadStoredAt is the time the payload row is stored under, as v1_payload_inserted_at computes it.
func payloadStoredAt(opt RetrievePayloadOpts) time.Time {
	if opt.PayloadReferenceId != nil && opt.PayloadReferenceId.Version() == 7 {
		sec, nsec := opt.PayloadReferenceId.Time().UnixTime()
		return time.Unix(sec, nsec)
	}

	return opt.InsertedAt.Time
}

// payload finds a payload by the external id of its v1_payload row. The row's own day comes first, then the
// next day (a run created just after midnight from an event just before), then earlier days, where the
// bundle of a task that wrote the payload later is.
func (r *runLogReader) payload(ctx context.Context, tenantId uuid.UUID, storedAt time.Time, externalId uuid.UUID) ([]byte, error) {
	stored := runlog.Day(storedAt)

	dropped, err := r.dayDropped(ctx, stored)

	if err != nil || !dropped {
		return nil, err
	}

	days := []time.Time{stored, stored.AddDate(0, 0, 1)}

	for i := 1; i <= runLogPayloadSearchDays; i++ {
		days = append(days, stored.AddDate(0, 0, -i))
	}

	for _, day := range days {
		idx, err := r.index(ctx, tenantId, day)

		if err != nil {
			return nil, err
		}

		if idx == nil {
			continue
		}

		if _, ok := idx.Payloads[externalId]; !ok {
			continue
		}

		p, err := r.GetPayload(ctx, tenantId, day, externalId)

		if err != nil {
			return nil, err
		}

		return p.Value, nil
	}

	return nil, nil
}

// locate finds the tenant-day archive holding the run or task with the given external id, or returns a nil
// index when core has not dropped its day or the run log does not hold it. The day comes from core's
// v1_lookup_table, so once the monthly lookup partition is dropped too the run cannot be found.
func (r *runLogReader) locate(ctx context.Context, tenantId, externalId uuid.UUID) (time.Time, *runlog.Index, error) {
	day, err := runlog.ResolveDay(ctx, r.pool.ForTenant(tenantId), tenantId, externalId)

	if errors.Is(err, runlog.ErrNotFound) {
		return day, nil, nil
	}

	if err != nil {
		return day, nil, err
	}

	idx, err := r.index(ctx, tenantId, day)

	if err != nil || idx == nil {
		return day, nil, err
	}

	_, isRun := idx.Runs[externalId]
	_, isTask := idx.Tasks[externalId]

	if !isRun && !isTask {
		return day, nil, nil
	}

	return day, idx, nil
}

// archivedRun returns the bundles of a run, or of a single task, known by its external id, when core has
// dropped its day and the run log holds it; nil otherwise.
func (r *runLogReader) archivedRun(ctx context.Context, tenantId, externalId uuid.UUID) ([]*runlog.Bundle, error) {
	day, idx, err := r.locate(ctx, tenantId, externalId)

	if err != nil || idx == nil {
		return nil, err
	}

	if _, ok := idx.Runs[externalId]; ok {
		return r.GetRun(ctx, tenantId, day, externalId)
	}

	b, err := r.GetTask(ctx, tenantId, day, externalId)

	if err != nil {
		return nil, err
	}

	return []*runlog.Bundle{b}, nil
}

// archivedOutputEvents are the task's terminal events at its current retry, decoded as core's
// listTaskOutputEvents decodes them.
func archivedOutputEvents(b *runlog.Bundle) ([]*TaskOutputEvent, error) {
	var res []*TaskOutputEvent

	for _, e := range b.Events {
		if e.RetryCount != -1 && e.RetryCount != b.Task.RetryCount {
			continue
		}

		if e.EventType != sqlcv1.V1TaskEventTypeCOMPLETED &&
			e.EventType != sqlcv1.V1TaskEventTypeFAILED &&
			e.EventType != sqlcv1.V1TaskEventTypeCANCELLED {
			continue
		}

		raw := e.Data

		if p := b.PayloadFor(sqlcv1.V1PayloadTypeTASKEVENTDATA, e.ID, e.InsertedAt); p != nil {
			raw = p.Value
		}

		o, err := DecodeTaskCompletionEventPayload(NewPayload[TaskCompletionEventShape](raw))

		if err != nil {
			return nil, fmt.Errorf("could not decode archived output event %d of task %s: %w", e.ID, b.Task.ExternalID, err)
		}

		res = append(res, o)
	}

	return res, nil
}

// runOutputBundles are the bundles whose events stand for the run, as core's FlattenExternalIds returns them
// for a run id: only the orchestrator of an operator DAG (its output holds every step's), every task otherwise.
func runOutputBundles(bundles []*runlog.Bundle) []*runlog.Bundle {
	for _, b := range bundles {
		if b.Task.IsDagOrchestrator {
			return []*runlog.Bundle{b}
		}
	}

	return bundles
}

// listArchivedFinalizedWorkflowRuns is ListFinalizedWorkflowRuns for runs core has dropped.
func (r *TaskRepositoryImpl) listArchivedFinalizedWorkflowRuns(ctx context.Context, tenantId uuid.UUID, rootExternalIds []uuid.UUID) ([]*ListFinalizedWorkflowRunsResponse, error) {
	res := make([]*ListFinalizedWorkflowRunsResponse, 0)

	for _, rootId := range rootExternalIds {
		bundles, err := r.runLog.archivedRun(ctx, tenantId, rootId)

		if err != nil {
			return nil, err
		}

		if len(bundles) == 0 {
			continue
		}

		run := &ListFinalizedWorkflowRunsResponse{WorkflowRunId: rootId}

		for _, b := range runOutputBundles(bundles) {
			events, err := archivedOutputEvents(b)

			if err != nil {
				return nil, err
			}

			run.OutputEvents = append(run.OutputEvents, events...)
		}

		res = append(res, run)
	}

	return res, nil
}

// archivedWorkflowRunResultDetails is GetWorkflowRunResultDetails for a run core has dropped, or nil when the
// run log does not hold it either.
func (r *TaskRepositoryImpl) archivedWorkflowRunResultDetails(ctx context.Context, tenantId, externalId uuid.UUID) (*WorkflowRunDetails, error) {
	bundles, err := r.runLog.archivedRun(ctx, tenantId, externalId)

	if err != nil || len(bundles) == 0 {
		return nil, err
	}

	first := bundles[0]

	for _, b := range bundles {
		if b.Task.ExternalID == externalId {
			first = b
		}
	}

	details := &WorkflowRunDetails{
		ReadableIdToDetails: map[StepReadableId]TaskRunDetails{},
		AdditionalMetadata:  first.Task.AdditionalMetadata,
	}

	if first.Task.DagID.Valid {
		for _, b := range bundles {
			if b.Dag != nil {
				if p := b.PayloadFor(sqlcv1.V1PayloadTypeDAGINPUT, b.Dag.Dag.ID, b.Dag.Dag.InsertedAt); p != nil {
					details.InputPayload = p.Value
				}
			}
		}
	} else if p := first.PayloadFor(sqlcv1.V1PayloadTypeTASKINPUT, first.Task.ID, first.Task.InsertedAt); p != nil {
		details.InputPayload, err = userInputFromTaskInputPayload(taskInputPayloadVersion(first.Task.TaskInputPayloadVersion), p.Value)

		if err != nil {
			return nil, fmt.Errorf("failed to read archived task input: %w", err)
		}
	}

	isOperatorDAG := first.Task.IsDagOrchestrator && first.Task.ExternalID == externalId

	for _, b := range bundles {
		events, err := archivedOutputEvents(b)

		if err != nil {
			return nil, err
		}

		for _, event := range events {
			status, err := statusutils.V1RunStatusFromEventType(event.EventType)

			if err != nil {
				return nil, err
			}

			if isOperatorDAG && b.Task.IsDagOrchestrator {
				details.OrchestratorStatus = status
				continue
			}

			details.ReadableIdToDetails[StepReadableId(event.StepReadableID)] = TaskRunDetails{
				OutputPayload: event.Output,
				Status:        *status,
				Error:         &event.ErrorMessage,
				ExternalId:    event.TaskExternalId,
			}
		}
	}

	return details, nil
}

// ListArchivedExternalIds returns those of the external ids (runs or tasks) that the run log holds for a day
// core has dropped.
func (r *TaskRepositoryImpl) ListArchivedExternalIds(ctx context.Context, tenantId uuid.UUID, externalIds []uuid.UUID) ([]uuid.UUID, error) {
	if r.runLog == nil {
		return nil, nil
	}

	var archived []uuid.UUID

	for _, id := range externalIds {
		_, idx, err := r.runLog.locate(ctx, tenantId, id)

		if err != nil {
			return nil, err
		}

		if idx != nil {
			archived = append(archived, id)
		}
	}

	return archived, nil
}
