package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/runlog"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// Archive-then-drop of core day partitions (run log on).
//
// A day D whose partitions end before now - retention is a candidate. It is dropped only after:
//
//  1. Guard: nothing in flight needs D (runLogPins, runLogNonTerminalTasks). A pinned day stays whole.
//  2. Archive: every task of D is written to the run log, one tenant-day at a time, and v1_run_log_day
//     records the day as ARCHIVED with the watermark the day had before the archive started.
//  3. Recheck right before the drop: the guard holds and the watermark is unchanged. Otherwise the record is
//     deleted and the day is archived again once it is eligible.
//  4. Drop: the record moves to DROPPING before the first DETACH, so an interrupted drop is finished on the
//     next pass and never re-archived from a half-dropped day, then to DROPPED.
//
// v1_payload and v1_event of D go only once D+1 is droppable too: a payload reference id carries its own
// timestamp, so a task created just after midnight reads a payload row stored the day before.
// v1_lookup_table (monthly) goes once every day of its month is droppable.

const runLogArchivePageSize = 500

const (
	runLogStateArchived = "ARCHIVED"
	runLogStateDropping = "DROPPING"
	runLogStateDropped  = "DROPPED"
)

// runLogDayFamilies hold rows keyed by the day of their task (or DAG) and go with the day.
var runLogDayFamilies = map[string]bool{
	"v1_task":                           true,
	"v1_dag":                            true,
	"v1_task_event":                     true,
	"v1_log_line":                       true,
	"v1_durable_event_log_file":         true,
	"v1_durable_event_log_entry":        true,
	"v1_durable_event_log_branch_point": true,
	"v1_dag_to_task":                    true,
	"v1_dag_data":                       true,
	"v1_task_expression_eval":           true,
}

// runLogLaggedFamilies go one day after their day.
var runLogLaggedFamilies = map[string]bool{
	"v1_payload": true,
	"v1_event":   true,
}

const runLogMonthlyFamily = "v1_lookup_table"

type runLogWatermark struct {
	TaskCount  int64
	EventCount int64
	MaxEventID int64
}

type runLogDayRecord struct {
	State string
	runLogWatermark
}

type runLogDayArchive struct {
	TenantDays       int
	Stats            runlog.WriteStats
	PayloadsFromLog  int64
	PayloadsNotFound int64
}

type runLogPassStats struct {
	DaysCandidate int
	DaysArchived  int
	DaysDropped   int
	runLogDayArchive
	ArchiveDuration time.Duration
	Pinned          map[string]map[string]int64
	Invalidated     map[string]string
}

// archiveAndDropPartitions archives and drops the candidate days among partitions. It returns the partitions
// it does not manage, for the caller to drop as usual, and the date before which offloaded payload block
// index rows can go.
func (r *TaskRepositoryImpl) archiveAndDropPartitions(ctx context.Context, removeBefore time.Time, partitions []*sqlcv1.ListPartitionsBeforeDateRow) ([]*sqlcv1.ListPartitionsBeforeDateRow, time.Time, error) {
	start := time.Now()

	byDay := make(map[time.Time][]*sqlcv1.ListPartitionsBeforeDateRow)
	var monthly, others []*sqlcv1.ListPartitionsBeforeDateRow

	for _, p := range partitions {
		switch {
		case runLogDayFamilies[p.ParentTable] || runLogLaggedFamilies[p.ParentTable]:
			day, err := partitionDate(p.PartitionName)

			if err != nil {
				return nil, removeBefore, err
			}

			byDay[day] = append(byDay[day], p)
		case p.ParentTable == runLogMonthlyFamily:
			monthly = append(monthly, p)
		default:
			others = append(others, p)
		}
	}

	days := make([]time.Time, 0, len(byDay))

	for d := range byDay {
		days = append(days, d)
	}

	slices.SortFunc(days, func(a, b time.Time) int { return a.Compare(b) })

	stats := runLogPassStats{
		DaysCandidate: len(days),
		Pinned:        map[string]map[string]int64{},
		Invalidated:   map[string]string{},
	}

	records, err := r.runLogDayRecords(ctx)

	if err != nil {
		return nil, removeBefore, err
	}

	// phase 1: archive every unpinned candidate day that has no archive yet

	pins, err := r.runLogPins(ctx)

	if err != nil {
		return nil, removeBefore, err
	}

	archiveStart := time.Now()

	for _, day := range days {
		rec, hasRecord := records[day]

		if hasRecord && rec.State != runLogStateArchived {
			continue
		}

		dayPins, err := r.runLogDayPins(ctx, pins, day)

		if err != nil {
			return nil, removeBefore, err
		}

		if len(dayPins) > 0 {
			stats.Pinned[day.Format(time.DateOnly)] = dayPins
			continue
		}

		if hasRecord {
			// archived on an earlier pass; phase 2 checks that it still holds
			continue
		}

		if !slices.ContainsFunc(byDay[day], func(p *sqlcv1.ListPartitionsBeforeDateRow) bool { return runLogDayFamilies[p.ParentTable] }) {
			// only lagged partitions left, nothing to archive
			continue
		}

		watermark, err := r.runLogDayWatermark(ctx, day)

		if err != nil {
			return nil, removeBefore, err
		}

		dayStart := time.Now()

		archive, err := r.archiveRunLogDay(ctx, day)

		if err != nil {
			return nil, removeBefore, fmt.Errorf("could not archive day %s to the run log: %w", day.Format(time.DateOnly), err)
		}

		if err := r.markRunLogDayArchived(ctx, day, watermark, archive); err != nil {
			return nil, removeBefore, err
		}

		records[day] = runLogDayRecord{State: runLogStateArchived, runLogWatermark: watermark}

		stats.DaysArchived++
		stats.TenantDays += archive.TenantDays
		stats.Stats.Add(archive.Stats)
		stats.PayloadsFromLog += archive.PayloadsFromLog
		stats.PayloadsNotFound += archive.PayloadsNotFound

		// Warn, like the drop it precedes: it is the record of what left core
		r.l.Warn().Ctx(ctx).
			Str("day", day.Format(time.DateOnly)).
			Int("tenant_days", archive.TenantDays).
			Int64("bundles", archive.Stats.Bundles).
			Int64("data_files", archive.Stats.Files).
			Int64("bytes_compressed", archive.Stats.BytesCompressed).
			Int64("bytes_uncompressed", archive.Stats.BytesUncompressed).
			Int64("payloads_from_log", archive.PayloadsFromLog).
			Int64("payloads_not_found", archive.PayloadsNotFound).
			Int64("tasks", watermark.TaskCount).
			Int64("duration_ms", time.Since(dayStart).Milliseconds()).
			Msg("run log: archived day")
	}

	stats.ArchiveDuration = time.Since(archiveStart)

	// phase 2: recheck and drop

	pins, err = r.runLogPins(ctx)

	if err != nil {
		return nil, removeBefore, err
	}

	droppable := make(map[time.Time]bool, len(days))

	for _, day := range days {
		rec, hasRecord := records[day]

		if !hasRecord {
			// a day with only lagged partitions left that was never archived holds no tasks
			droppable[day] = !slices.ContainsFunc(byDay[day], func(p *sqlcv1.ListPartitionsBeforeDateRow) bool { return runLogDayFamilies[p.ParentTable] })
			continue
		}

		if rec.State == runLogStateArchived {
			reason, err := r.runLogArchiveStale(ctx, pins, day, rec.runLogWatermark)

			if err != nil {
				return nil, removeBefore, err
			}

			if reason != "" {
				stats.Invalidated[day.Format(time.DateOnly)] = reason

				r.l.Warn().Ctx(ctx).Str("day", day.Format(time.DateOnly)).Str("reason", reason).Msg("run log: archive of day is stale, keeping the day")

				if err := r.deleteRunLogDayRecord(ctx, day); err != nil {
					return nil, removeBefore, err
				}

				continue
			}

			if r.runLogBeforeDrop != nil {
				if err := r.runLogBeforeDrop(day); err != nil {
					return nil, removeBefore, err
				}
			}

			if err := r.setRunLogDayState(ctx, day, runLogStateDropping); err != nil {
				return nil, removeBefore, err
			}
		}

		for _, p := range byDay[day] {
			if !runLogDayFamilies[p.ParentTable] {
				continue
			}

			if err := r.detachAndDropPartition(ctx, p.ParentTable, p.PartitionName); err != nil {
				return nil, removeBefore, err
			}
		}

		if rec.State != runLogStateDropped {
			if err := r.setRunLogDayState(ctx, day, runLogStateDropped); err != nil {
				return nil, removeBefore, err
			}

			stats.DaysDropped++
		}

		droppable[day] = true
	}

	// lagged families: D goes when D and D+1 are both droppable
	blockIndexCutoff := removeBefore
	cutoff := runlog.Day(removeBefore)

	for _, day := range days {
		next := day.AddDate(0, 0, 1)
		nextDroppable := droppable[next]

		if _, isCandidate := byDay[next]; !isCandidate && !next.AddDate(0, 0, 1).After(cutoff) {
			// a day past retention without any partition left holds nothing that could read a payload of day
			nextDroppable = true
		}

		for _, p := range byDay[day] {
			if !runLogLaggedFamilies[p.ParentTable] {
				continue
			}

			if !droppable[day] || !nextDroppable {
				if p.ParentTable == "v1_payload" && day.Before(blockIndexCutoff) {
					blockIndexCutoff = day
				}

				continue
			}

			if err := r.detachAndDropPartition(ctx, p.ParentTable, p.PartitionName); err != nil {
				return nil, removeBefore, err
			}
		}
	}

	for _, p := range monthly {
		month, err := partitionDate(p.PartitionName)

		if err != nil {
			return nil, removeBefore, err
		}

		keep := false

		for _, day := range days {
			if day.Year() == month.Year() && day.Month() == month.Month() && !droppable[day] {
				keep = true
			}
		}

		if keep {
			continue
		}

		if err := r.detachAndDropPartition(ctx, p.ParentTable, p.PartitionName); err != nil {
			return nil, removeBefore, err
		}
	}

	invalidated := make([]string, 0, len(stats.Invalidated))

	for d, reason := range stats.Invalidated {
		invalidated = append(invalidated, d+":"+reason)
	}

	sort.Strings(invalidated)

	ev := r.l.Debug()

	if stats.DaysCandidate > 0 {
		ev = r.l.Warn()
	}

	ev.Ctx(ctx).
		Int("days_candidate", stats.DaysCandidate).
		Int("days_archived", stats.DaysArchived).
		Int("days_dropped", stats.DaysDropped).
		Int("days_pinned", len(stats.Pinned)).
		Interface("pinned", stats.Pinned).
		Strs("invalidated", invalidated).
		Int("tenant_days", stats.TenantDays).
		Int64("bundles", stats.Stats.Bundles).
		Int64("data_files", stats.Stats.Files).
		Int64("bytes_compressed", stats.Stats.BytesCompressed).
		Int64("bytes_uncompressed", stats.Stats.BytesUncompressed).
		Int64("payloads_from_log", stats.PayloadsFromLog).
		Int64("payloads_not_found", stats.PayloadsNotFound).
		Int64("archive_duration_ms", stats.ArchiveDuration.Milliseconds()).
		Int64("duration_ms", time.Since(start).Milliseconds()).
		Msg("run log: maintenance pass")

	return others, blockIndexCutoff, nil
}

// partitionDate parses the date suffix of a partition name (<table>_YYYYMMDD).
func partitionDate(partitionName string) (time.Time, error) {
	i := strings.LastIndex(partitionName, "_")

	if i < 0 {
		return time.Time{}, fmt.Errorf("partition %s has no date suffix", partitionName)
	}

	return time.Parse("20060102", partitionName[i+1:])
}

// runLogPinsQuery returns, per day and reason, how many pointers to in-flight work point into the day.
//
// A run tree is tied together like this:
//   - a legacy DAG has a v1_dag row; its tasks carry dag_id/dag_inserted_at and are listed in v1_dag_to_task,
//     including steps created on later days. A step not created yet is a v1_match with trigger_dag_*.
//   - an operator DAG is a durable orchestrator task (is_dag_orchestrator); its steps are tasks whose
//     parent_task_* points at the orchestrator and whose workflow_run_id is the orchestrator's external id.
//     The orchestrator's durable event log entries list the steps it spawned (child_task_external_id).
//   - a child workflow is a separate run whose root task's parent_task_* points at the spawning task.
//
// From every task a hot row or a match points at, the walk goes up parent_task_* to the root (a live child
// keeps its ancestors' days), picks up the DAG of every task on the way (all its tasks' days) and the steps of
// every orchestrator on the way (all their days). Finished child workflows of a live parent are not pinned:
// the parent holds the child's result in its own SIGNAL_COMPLETED event or durable log entry.
const runLogPinsQuery = `
WITH RECURSIVE ptr AS (
    SELECT task_id AS id, task_inserted_at AS inserted_at, 'queue_item' AS reason FROM v1_queue_item
    UNION ALL SELECT task_id, task_inserted_at, 'batched_queue_item' FROM v1_batched_queue_item
    UNION ALL SELECT task_id, task_inserted_at, 'rate_limited_queue_item' FROM v1_rate_limited_queue_items
    UNION ALL SELECT task_id, task_inserted_at, 'paused_queue_item' FROM v1_paused_workflow_queue_item
    UNION ALL SELECT task_id, task_inserted_at, 'retry_queue_item' FROM v1_retry_queue_item
    UNION ALL SELECT task_id, task_inserted_at, CASE WHEN evicted_at IS NULL THEN 'runtime' ELSE 'evicted_runtime' END FROM v1_task_runtime
    UNION ALL SELECT task_id, task_inserted_at, 'concurrency_slot' FROM v1_concurrency_slot
    UNION ALL SELECT signal_task_id, signal_task_inserted_at, 'match' FROM v1_match WHERE signal_task_id IS NOT NULL
    UNION ALL SELECT trigger_parent_task_id, trigger_parent_task_inserted_at, 'match' FROM v1_match WHERE trigger_parent_task_id IS NOT NULL
    UNION ALL SELECT trigger_existing_task_id, trigger_existing_task_inserted_at, 'match' FROM v1_match WHERE trigger_existing_task_id IS NOT NULL
), walk AS (
    SELECT t.id, t.inserted_at, t.parent_task_id, t.parent_task_inserted_at, t.dag_id, t.dag_inserted_at, t.is_dag_orchestrator
    FROM (SELECT DISTINCT id, inserted_at FROM ptr) p
    JOIN v1_task t ON t.id = p.id AND t.inserted_at = p.inserted_at

    UNION

    SELECT t.id, t.inserted_at, t.parent_task_id, t.parent_task_inserted_at, t.dag_id, t.dag_inserted_at, t.is_dag_orchestrator
    FROM walk w
    JOIN v1_task t ON t.id = w.parent_task_id AND t.inserted_at = w.parent_task_inserted_at
), live_dag AS (
    SELECT dag_id AS id, dag_inserted_at AS inserted_at FROM walk WHERE dag_id IS NOT NULL
    UNION
    SELECT trigger_dag_id, trigger_dag_inserted_at FROM v1_match WHERE trigger_dag_id IS NOT NULL
), pins AS (
    SELECT inserted_at, reason FROM ptr
    UNION ALL SELECT trigger_dag_inserted_at, 'match' FROM v1_match WHERE trigger_dag_id IS NOT NULL
    UNION ALL SELECT inserted_at, 'run_tree' FROM walk
    UNION ALL SELECT inserted_at, 'dag' FROM live_dag
    UNION ALL
    SELECT dt.task_inserted_at, 'dag_task'
    FROM live_dag d
    JOIN v1_dag_to_task dt ON dt.dag_id = d.id AND dt.dag_inserted_at = d.inserted_at
    UNION ALL
    SELECT lt.inserted_at, 'orchestrator_step'
    FROM walk w
    JOIN v1_durable_event_log_entry e ON e.durable_task_id = w.id AND e.durable_task_inserted_at = w.inserted_at
    JOIN v1_lookup_table lt ON lt.external_id = e.child_task_external_id
    WHERE w.is_dag_orchestrator AND e.child_task_external_id IS NOT NULL
)
SELECT (inserted_at AT TIME ZONE 'UTC')::date AS day, reason, count(*) AS n
FROM pins
GROUP BY 1, 2
`

type runLogPinSet map[time.Time]map[string]int64

func (r *TaskRepositoryImpl) runLogPins(ctx context.Context) (runLogPinSet, error) {
	rows, err := r.ddlPool.Query(ctx, runLogPinsQuery)

	if err != nil {
		return nil, fmt.Errorf("could not list run log pins: %w", err)
	}

	defer rows.Close()

	pins := runLogPinSet{}

	for rows.Next() {
		var (
			day    time.Time
			reason string
			n      int64
		)

		if err := rows.Scan(&day, &reason, &n); err != nil {
			return nil, err
		}

		day = runlog.Day(day)

		if pins[day] == nil {
			pins[day] = map[string]int64{}
		}

		pins[day][reason] += n
	}

	return pins, rows.Err()
}

// runLogDayPins returns why day is pinned, if it is: in-flight pointers into it and tasks of it that are not
// terminal.
func (r *TaskRepositoryImpl) runLogDayPins(ctx context.Context, pins runLogPinSet, day time.Time) (map[string]int64, error) {
	reasons := map[string]int64{}

	for reason, n := range pins[day] {
		reasons[reason] = n
	}

	nonTerminal, err := r.runLogNonTerminalTasks(ctx, day)

	if err != nil {
		return nil, err
	}

	if nonTerminal > 0 {
		reasons["non_terminal_task"] = nonTerminal
	}

	return reasons, nil
}

// runLogNonTerminalTasks counts (up to 1000) tasks of day born QUEUED that have no terminal event at their
// current retry count: in flight, or lost.
func (r *TaskRepositoryImpl) runLogNonTerminalTasks(ctx context.Context, day time.Time) (int64, error) {
	var n int64

	err := r.ddlPool.QueryRow(ctx, `
		SELECT count(*) FROM (
			SELECT 1
			FROM v1_task t
			WHERE t.inserted_at >= $1 AND t.inserted_at < $2
				AND t.initial_state = 'QUEUED'
				AND NOT EXISTS (
					SELECT 1
					FROM v1_task_event e
					WHERE e.task_id = t.id
						AND e.task_inserted_at = t.inserted_at
						AND e.retry_count = t.retry_count
						AND e.event_type IN ('COMPLETED', 'FAILED', 'CANCELLED')
				)
			LIMIT 1000
		) x
	`, day, day.AddDate(0, 0, 1)).Scan(&n)

	return n, err
}

func (r *TaskRepositoryImpl) runLogDayWatermark(ctx context.Context, day time.Time) (runLogWatermark, error) {
	var w runLogWatermark

	err := r.ddlPool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM v1_task WHERE inserted_at >= $1 AND inserted_at < $2),
			count(*),
			coalesce(max(id), 0)
		FROM v1_task_event
		WHERE task_inserted_at >= $1 AND task_inserted_at < $2
	`, day, day.AddDate(0, 0, 1)).Scan(&w.TaskCount, &w.EventCount, &w.MaxEventID)

	return w, err
}

// runLogArchiveStale says why an archived day must not be dropped now, or "" when it can be.
func (r *TaskRepositoryImpl) runLogArchiveStale(ctx context.Context, pins runLogPinSet, day time.Time, archived runLogWatermark) (string, error) {
	dayPins, err := r.runLogDayPins(ctx, pins, day)

	if err != nil {
		return "", err
	}

	if len(dayPins) > 0 {
		reasons := make([]string, 0, len(dayPins))

		for reason := range dayPins {
			reasons = append(reasons, reason)
		}

		sort.Strings(reasons)

		return "pinned:" + strings.Join(reasons, ","), nil
	}

	current, err := r.runLogDayWatermark(ctx, day)

	if err != nil {
		return "", err
	}

	if current != archived {
		return fmt.Sprintf("changed:%+v->%+v", archived, current), nil
	}

	return "", nil
}

func (r *TaskRepositoryImpl) runLogDayRecords(ctx context.Context) (map[time.Time]runLogDayRecord, error) {
	rows, err := r.ddlPool.Query(ctx, `SELECT day, state, task_count, event_count, max_event_id FROM v1_run_log_day`)

	if err != nil {
		return nil, fmt.Errorf("could not list run log days: %w", err)
	}

	defer rows.Close()

	records := map[time.Time]runLogDayRecord{}

	for rows.Next() {
		var (
			day time.Time
			rec runLogDayRecord
		)

		if err := rows.Scan(&day, &rec.State, &rec.TaskCount, &rec.EventCount, &rec.MaxEventID); err != nil {
			return nil, err
		}

		records[runlog.Day(day)] = rec
	}

	return records, rows.Err()
}

func (r *TaskRepositoryImpl) markRunLogDayArchived(ctx context.Context, day time.Time, w runLogWatermark, a runLogDayArchive) error {
	_, err := r.ddlPool.Exec(ctx, `
		INSERT INTO v1_run_log_day (day, state, task_count, event_count, max_event_id, tenant_days, bundles, bytes_compressed, bytes_uncompressed, archived_at)
		VALUES ($1, 'ARCHIVED', $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (day) DO UPDATE SET
			state = 'ARCHIVED',
			task_count = EXCLUDED.task_count,
			event_count = EXCLUDED.event_count,
			max_event_id = EXCLUDED.max_event_id,
			tenant_days = EXCLUDED.tenant_days,
			bundles = EXCLUDED.bundles,
			bytes_compressed = EXCLUDED.bytes_compressed,
			bytes_uncompressed = EXCLUDED.bytes_uncompressed,
			archived_at = EXCLUDED.archived_at,
			updated_at = NOW()
	`, pgtype.Date{Time: day, Valid: true}, w.TaskCount, w.EventCount, w.MaxEventID, a.TenantDays, a.Stats.Bundles, a.Stats.BytesCompressed, a.Stats.BytesUncompressed)

	return err
}

func (r *TaskRepositoryImpl) setRunLogDayState(ctx context.Context, day time.Time, state string) error {
	_, err := r.ddlPool.Exec(ctx, `UPDATE v1_run_log_day SET state = $2, updated_at = NOW() WHERE day = $1`, pgtype.Date{Time: day, Valid: true}, state)

	return err
}

func (r *TaskRepositoryImpl) deleteRunLogDayRecord(ctx context.Context, day time.Time) error {
	_, err := r.ddlPool.Exec(ctx, `DELETE FROM v1_run_log_day WHERE day = $1 AND state = 'ARCHIVED'`, pgtype.Date{Time: day, Valid: true})

	return err
}

// oldestTaskPartitionDay is the day of the oldest v1_task partition, or nil when there is none.
func oldestTaskPartitionDay(ctx context.Context, db sqlcv1.DBTX) (*time.Time, error) {
	var day pgtype.Date

	err := db.QueryRow(ctx, `
		SELECT min(to_date(substring(c.relname FROM '_(\d{8})$'), 'YYYYMMDD'))
		FROM pg_inherits i
		JOIN pg_class c ON c.oid = i.inhrelid
		WHERE i.inhparent = 'v1_task'::regclass
	`).Scan(&day)

	if err != nil || !day.Valid {
		return nil, err
	}

	return &day.Time, nil
}

// archiveRunLogDay writes every task of day to the run log, one writer per tenant, and closes every
// tenant-day. Re-running it overwrites the same objects.
func (r *TaskRepositoryImpl) archiveRunLogDay(ctx context.Context, day time.Time) (runLogDayArchive, error) {
	var res runLogDayArchive

	lo, hi := day, day.AddDate(0, 0, 1)
	db := r.pool.ForShared()

	writers := map[uuid.UUID]*runlog.DayWriter{}
	dagsEmbedded := map[int64]bool{}

	var lastID int64

	for {
		rows, err := db.Query(ctx, `
			SELECT *
			FROM v1_task
			WHERE inserted_at >= $1 AND inserted_at < $2 AND id > $3
			ORDER BY id
			LIMIT $4
		`, lo, hi, lastID, runLogArchivePageSize)

		if err != nil {
			return res, err
		}

		tasks, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[sqlcv1.V1Task])

		if err != nil {
			return res, fmt.Errorf("could not list tasks: %w", err)
		}

		if len(tasks) == 0 {
			break
		}

		lastID = tasks[len(tasks)-1].ID

		page, err := r.buildRunLogBundles(ctx, db, lo, hi, tasks, dagsEmbedded)

		if err != nil {
			return res, err
		}

		res.PayloadsFromLog += page.payloadsFromLog
		res.PayloadsNotFound += page.payloadsNotFound

		for _, b := range page.bundles {
			w, ok := writers[b.TenantID]

			if !ok {
				w = r.runLog.NewDayWriter(b.TenantID, day)
				writers[b.TenantID] = w
			}

			if err := w.Add(ctx, b); err != nil {
				return res, err
			}
		}

		for _, rd := range page.runDays {
			writers[rd.tenant].AddRunDays(rd.run, rd.days...)
		}
	}

	for _, w := range writers {
		stats, err := w.Close(ctx)

		if err != nil {
			return res, err
		}

		res.TenantDays++
		res.Stats.Add(stats)
	}

	return res, nil
}

type runLogRunDays struct {
	tenant uuid.UUID
	run    uuid.UUID
	days   []time.Time
}

type runLogPage struct {
	bundles          []*runlog.Bundle
	runDays          []runLogRunDays
	payloadsFromLog  int64
	payloadsNotFound int64
}

// buildRunLogBundles reads everything the bundles of one page of tasks of day [lo, hi) hold.
func (r *TaskRepositoryImpl) buildRunLogBundles(ctx context.Context, db sqlcv1.DBTX, lo, hi time.Time, tasks []*sqlcv1.V1Task, dagsEmbedded map[int64]bool) (*runLogPage, error) {
	page := &runLogPage{}

	taskIds := make([]int64, len(tasks))
	var durableIds, dagIds []int64

	bundleByTaskId := make(map[int64]*runlog.Bundle, len(tasks))

	for i, t := range tasks {
		taskIds[i] = t.ID

		b := &runlog.Bundle{TenantID: t.TenantID, Task: t, Events: []*sqlcv1.V1TaskEvent{}, Payloads: []runlog.Payload{}}
		bundleByTaskId[t.ID] = b
		page.bundles = append(page.bundles, b)

		if t.IsDurable.Bool || t.IsDagOrchestrator {
			durableIds = append(durableIds, t.ID)
		}

		if t.DagID.Valid && !t.DagInsertedAt.Time.Before(lo) && t.DagInsertedAt.Time.Before(hi) && !dagsEmbedded[t.DagID.Int64] {
			dagsEmbedded[t.DagID.Int64] = true
			dagIds = append(dagIds, t.DagID.Int64)
			b.Dag = &runlog.Dag{Tasks: []*sqlcv1.V1DagToTask{}}
		}
	}

	events, err := collectRunLogRows[sqlcv1.V1TaskEvent](ctx, db, `
		SELECT * FROM v1_task_event
		WHERE task_inserted_at >= $1 AND task_inserted_at < $2 AND task_id = ANY($3::bigint[])
		ORDER BY task_id, id
	`, lo, hi, taskIds)

	if err != nil {
		return nil, err
	}

	for _, e := range events {
		b := bundleByTaskId[e.TaskID]
		b.Events = append(b.Events, e)
	}

	var entries []*sqlcv1.V1DurableEventLogEntry

	if len(durableIds) > 0 {
		files, err := collectRunLogRows[sqlcv1.V1DurableEventLogFile](ctx, db, `
			SELECT * FROM v1_durable_event_log_file
			WHERE durable_task_inserted_at >= $1 AND durable_task_inserted_at < $2 AND durable_task_id = ANY($3::bigint[])
		`, lo, hi, durableIds)

		if err != nil {
			return nil, err
		}

		entries, err = collectRunLogRows[sqlcv1.V1DurableEventLogEntry](ctx, db, `
			SELECT * FROM v1_durable_event_log_entry
			WHERE durable_task_inserted_at >= $1 AND durable_task_inserted_at < $2 AND durable_task_id = ANY($3::bigint[])
			ORDER BY durable_task_id, branch_id, node_id
		`, lo, hi, durableIds)

		if err != nil {
			return nil, err
		}

		branchPoints, err := collectRunLogRows[sqlcv1.V1DurableEventLogBranchPoint](ctx, db, `
			SELECT * FROM v1_durable_event_log_branch_point
			WHERE durable_task_inserted_at >= $1 AND durable_task_inserted_at < $2 AND durable_task_id = ANY($3::bigint[])
		`, lo, hi, durableIds)

		if err != nil {
			return nil, err
		}

		durableLog := func(taskId int64) *runlog.DurableLog {
			b := bundleByTaskId[taskId]

			if b.DurableLog == nil {
				b.DurableLog = &runlog.DurableLog{Entries: []*sqlcv1.V1DurableEventLogEntry{}, BranchPoints: []*sqlcv1.V1DurableEventLogBranchPoint{}}
			}

			return b.DurableLog
		}

		for _, f := range files {
			durableLog(f.DurableTaskID).File = f
		}

		for _, e := range entries {
			dl := durableLog(e.DurableTaskID)
			dl.Entries = append(dl.Entries, e)
		}

		for _, bp := range branchPoints {
			dl := durableLog(bp.DurableTaskID)
			dl.BranchPoints = append(dl.BranchPoints, bp)
		}
	}

	dagBundles := map[int64]*runlog.Bundle{}

	if len(dagIds) > 0 {
		for _, b := range page.bundles {
			if b.Dag != nil {
				dagBundles[b.Task.DagID.Int64] = b
			}
		}

		dags, err := collectRunLogRows[sqlcv1.V1Dag](ctx, db, `
			SELECT * FROM v1_dag WHERE inserted_at >= $1 AND inserted_at < $2 AND id = ANY($3::bigint[])
		`, lo, hi, dagIds)

		if err != nil {
			return nil, err
		}

		for _, d := range dags {
			dagBundles[d.ID].Dag.Dag = d
		}

		dagData, err := collectRunLogRows[sqlcv1.V1DagData](ctx, db, `
			SELECT * FROM v1_dag_data WHERE dag_inserted_at >= $1 AND dag_inserted_at < $2 AND dag_id = ANY($3::bigint[])
		`, lo, hi, dagIds)

		if err != nil {
			return nil, err
		}

		for _, d := range dagData {
			dagBundles[d.DagID].Dag.Data = d
		}

		dagTasks, err := collectRunLogRows[sqlcv1.V1DagToTask](ctx, db, `
			SELECT * FROM v1_dag_to_task WHERE dag_inserted_at >= $1 AND dag_inserted_at < $2 AND dag_id = ANY($3::bigint[])
			ORDER BY dag_id, task_inserted_at, task_id
		`, lo, hi, dagIds)

		if err != nil {
			return nil, err
		}

		for _, dt := range dagTasks {
			dag := dagBundles[dt.DagID].Dag
			dag.Tasks = append(dag.Tasks, dt)
		}

		for _, b := range dagBundles {
			if b.Dag.Dag == nil {
				return nil, fmt.Errorf("task %d points at dag %d of its own day, which does not exist", b.Task.ID, b.Task.DagID.Int64)
			}

			days := make([]time.Time, 0, len(b.Dag.Tasks))

			for _, dt := range b.Dag.Tasks {
				days = append(days, dt.TaskInsertedAt.Time)
			}

			page.runDays = append(page.runDays, runLogRunDays{tenant: b.TenantID, run: b.Dag.Dag.ExternalID, days: days})
		}
	}

	if err := r.addRunLogOrchestratorDays(ctx, db, page, entries, bundleByTaskId); err != nil {
		return nil, err
	}

	if err := r.addRunLogPayloads(ctx, db, page); err != nil {
		return nil, err
	}

	return page, nil
}

// addRunLogOrchestratorDays records the days of the steps each DAG orchestrator of the page spawned.
func (r *TaskRepositoryImpl) addRunLogOrchestratorDays(ctx context.Context, db sqlcv1.DBTX, page *runLogPage, entries []*sqlcv1.V1DurableEventLogEntry, bundleByTaskId map[int64]*runlog.Bundle) error {
	childIds := []uuid.UUID{}
	childToOrchestrator := map[uuid.UUID]*runlog.Bundle{}

	for _, e := range entries {
		b := bundleByTaskId[e.DurableTaskID]

		if b.Task.IsDagOrchestrator && e.ChildTaskExternalID != nil {
			childIds = append(childIds, *e.ChildTaskExternalID)
			childToOrchestrator[*e.ChildTaskExternalID] = b
		}
	}

	if len(childIds) == 0 {
		return nil
	}

	rows, err := db.Query(ctx, `SELECT external_id, inserted_at FROM v1_lookup_table WHERE external_id = ANY($1::uuid[])`, childIds)

	if err != nil {
		return err
	}

	defer rows.Close()

	daysByOrchestrator := map[*runlog.Bundle][]time.Time{}

	for rows.Next() {
		var (
			externalId uuid.UUID
			insertedAt time.Time
		)

		if err := rows.Scan(&externalId, &insertedAt); err != nil {
			return err
		}

		b := childToOrchestrator[externalId]
		daysByOrchestrator[b] = append(daysByOrchestrator[b], insertedAt)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	for b, days := range daysByOrchestrator {
		page.runDays = append(page.runDays, runLogRunDays{tenant: b.TenantID, run: b.Task.ExternalID, days: days})
	}

	return nil
}

// addRunLogPayloads resolves every payload the page's bundles reference through the payload store, so inline
// and offloaded payloads come out the same. A shared payload whose row is already dropped (a DAG step created
// days after the DAG, whose input row went with the DAG's day) is read from the run log, where the bundle of
// an earlier task of the run holds it.
func (r *TaskRepositoryImpl) addRunLogPayloads(ctx context.Context, db sqlcv1.DBTX, page *runLogPage) error {
	var (
		opts    []RetrievePayloadOpts
		targets []*runlog.Bundle
	)

	add := func(b *runlog.Bundle, opt RetrievePayloadOpts) {
		opts = append(opts, opt)
		targets = append(targets, b)
	}

	for _, b := range page.bundles {
		t := b.Task

		add(b, RetrievePayloadOpts{
			TenantId:           t.TenantID,
			Id:                 t.ID,
			InsertedAt:         t.InsertedAt,
			Type:               sqlcv1.V1PayloadTypeTASKINPUT,
			ExternalId:         t.ExternalID,
			PayloadReferenceId: t.PayloadReferenceID,
		})

		if b.Dag != nil {
			d := b.Dag.Dag

			add(b, RetrievePayloadOpts{
				TenantId:           d.TenantID,
				Id:                 d.ID,
				InsertedAt:         d.InsertedAt,
				Type:               sqlcv1.V1PayloadTypeDAGINPUT,
				ExternalId:         d.ExternalID,
				PayloadReferenceId: d.PayloadReferenceID,
			})
		}

		for _, e := range b.Events {
			add(b, taskEventDataStoreOpts(b.TenantID, e, nil).key)
		}

		if b.DurableLog != nil {
			for _, e := range b.DurableLog.Entries {
				add(b, RetrievePayloadOpts{
					TenantId:   e.TenantID,
					Id:         e.ID,
					InsertedAt: e.InsertedAt,
					Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
					ExternalId: e.ExternalID,
				})

				add(b, RetrievePayloadOpts{
					TenantId:   e.TenantID,
					Id:         e.ID,
					InsertedAt: e.InsertedAt,
					Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
					ExternalId: e.ResultPayloadExternalID,
				})
			}
		}
	}

	found, err := r.payloadStore.Retrieve(ctx, db, opts...)

	if err != nil {
		return fmt.Errorf("could not retrieve payloads: %w", err)
	}

	for i, opt := range opts {
		b := targets[i]

		value, ok := found[opt]

		if !ok && opt.PayloadReferenceId != nil {
			p, err := r.runLog.GetPayloadByReference(ctx, b.TenantID, *opt.PayloadReferenceId)

			switch {
			case err == nil:
				value, ok = p.Value, true
				page.payloadsFromLog++
			case !errors.Is(err, runlog.ErrNotFound):
				return err
			}
		}

		if !ok {
			// empty payloads are never stored, so most of these are empty bodies rather than lost ones
			page.payloadsNotFound++
			continue
		}

		if !json.Valid(value) {
			return fmt.Errorf("payload %s of type %s for %d is not JSON", opt.effectiveExternalId(), opt.Type, opt.Id)
		}

		b.Payloads = append(b.Payloads, runlog.Payload{
			Type:       opt.Type,
			ID:         opt.Id,
			InsertedAt: opt.InsertedAt,
			ExternalID: opt.effectiveExternalId(),
			Shared:     opt.PayloadReferenceId != nil,
			Value:      value,
		})
	}

	return nil
}

func collectRunLogRows[T any](ctx context.Context, db sqlcv1.DBTX, query string, args ...any) ([]*T, error) {
	rows, err := db.Query(ctx, query, args...)

	if err != nil {
		return nil, err
	}

	res, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[T])

	if err != nil {
		return nil, fmt.Errorf("could not read rows: %w", err)
	}

	return res, nil
}
