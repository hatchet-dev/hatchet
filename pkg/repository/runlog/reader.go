package runlog

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const indexCacheSize = 256

// Log reads and writes the run log in one store under one prefix.
type Log struct {
	store  Store
	prefix string

	// indexes caches decoded indexes of complete tenant-days. A complete tenant-day is only rewritten when the
	// archive of a day is invalidated before its partitions are dropped, and readers do not consult the log
	// for a day core still holds.
	indexes *lru.Cache[string, *Index]
}

func New(store Store, prefix string) *Log {
	indexes, err := lru.New[string, *Index](indexCacheSize)

	if err != nil {
		panic(err)
	}

	return &Log{store: store, prefix: prefix, indexes: indexes}
}

// Day truncates t to its UTC day.
func Day(t time.Time) time.Time {
	return t.UTC().Truncate(24 * time.Hour)
}

func (l *Log) dayPrefix(tenant uuid.UUID, day time.Time) string {
	p := fmt.Sprintf("v1/%s/%s/", tenant, Day(day).Format("2006/01/02"))

	if l.prefix == "" {
		return p
	}

	return l.prefix + "/" + p
}

func (l *Log) dataKey(tenant uuid.UUID, day time.Time, file int) string {
	return fmt.Sprintf("%sdata-%05d.jsonl.gz", l.dayPrefix(tenant, day), file)
}

func (l *Log) indexKey(tenant uuid.UUID, day time.Time) string {
	return l.dayPrefix(tenant, day) + "index.json.gz"
}

func (l *Log) completeKey(tenant uuid.UUID, day time.Time) string {
	return l.dayPrefix(tenant, day) + "_COMPLETE"
}

// DayPrefix is the key prefix of a tenant-day, for listing the store.
func (l *Log) DayPrefix(tenant uuid.UUID, day time.Time) string {
	return l.dayPrefix(tenant, day)
}

// IsComplete reports whether the tenant-day's archive is complete.
func (l *Log) IsComplete(ctx context.Context, tenant uuid.UUID, day time.Time) (bool, error) {
	return l.store.Exists(ctx, l.completeKey(tenant, day))
}

// Index returns the tenant-day's index, or ErrNotFound when the tenant-day is not completely archived.
func (l *Log) Index(ctx context.Context, tenant uuid.UUID, day time.Time) (*Index, error) {
	cacheKey := l.dayPrefix(tenant, day)

	if idx, ok := l.indexes.Get(cacheKey); ok {
		return idx, nil
	}

	complete, err := l.IsComplete(ctx, tenant, day)

	if err != nil {
		return nil, err
	}

	if !complete {
		return nil, ErrNotFound
	}

	raw, err := l.store.Get(ctx, l.indexKey(tenant, day))

	if err != nil {
		return nil, err
	}

	var idx Index

	if err := decodeGzipJSON(raw, &idx); err != nil {
		return nil, fmt.Errorf("could not decode index %s: %w", l.indexKey(tenant, day), err)
	}

	l.indexes.Add(cacheKey, &idx)

	return &idx, nil
}

// GetTask returns the bundle of one task.
func (l *Log) GetTask(ctx context.Context, tenant uuid.UUID, day time.Time, taskExternalID uuid.UUID) (*Bundle, error) {
	idx, err := l.Index(ctx, tenant, day)

	if err != nil {
		return nil, err
	}

	loc, ok := idx.Tasks[taskExternalID]

	if !ok {
		return nil, ErrNotFound
	}

	return l.readBundle(ctx, tenant, day, loc)
}

// GetRun returns the bundles of every task of a run: the ones on the given day and on the other days the run's
// index entries list. The run external id is the tasks' workflow_run_id: the DAG's external id for a legacy
// DAG, the orchestrator task's for an operator DAG, the task's own for a standalone task.
func (l *Log) GetRun(ctx context.Context, tenant uuid.UUID, day time.Time, runExternalID uuid.UUID) ([]*Bundle, error) {
	var bundles []*Bundle

	visited := map[string]bool{}
	pending := []string{Day(day).Format(time.DateOnly)}

	for len(pending) > 0 {
		dayStr := pending[0]
		pending = pending[1:]

		if visited[dayStr] {
			continue
		}

		visited[dayStr] = true

		d, err := time.Parse(time.DateOnly, dayStr)

		if err != nil {
			return nil, err
		}

		idx, err := l.Index(ctx, tenant, d)

		if errors.Is(err, ErrNotFound) {
			continue
		}

		if err != nil {
			return nil, err
		}

		run, ok := idx.Runs[runExternalID]

		if !ok {
			continue
		}

		for _, taskID := range run.Tasks {
			b, err := l.readBundle(ctx, tenant, d, idx.Tasks[taskID])

			if err != nil {
				return nil, err
			}

			bundles = append(bundles, b)
		}

		pending = append(pending, run.OtherDays...)
	}

	if len(bundles) == 0 {
		return nil, ErrNotFound
	}

	return bundles, nil
}

// GetPayload returns a payload by the external id of its v1_payload row (the payload reference id for a
// shared row), looking only at the given day.
func (l *Log) GetPayload(ctx context.Context, tenant uuid.UUID, day time.Time, payloadExternalID uuid.UUID) (*Payload, error) {
	idx, err := l.Index(ctx, tenant, day)

	if err != nil {
		return nil, err
	}

	loc, ok := idx.Payloads[payloadExternalID]

	if !ok {
		return nil, ErrNotFound
	}

	b, err := l.readBundle(ctx, tenant, day, loc)

	if err != nil {
		return nil, err
	}

	p := b.PayloadByExternalID(payloadExternalID)

	if p == nil {
		return nil, fmt.Errorf("index of %s points payload %s at a bundle without it", l.dayPrefix(tenant, day), payloadExternalID)
	}

	return p, nil
}

// GetPayloadByReference returns a shared payload by its version 7 reference id. The row's day is the
// timestamp in the id; the first bundle that holds it can be on the next day (an event just before midnight
// whose run was created just after), so both days are tried.
func (l *Log) GetPayloadByReference(ctx context.Context, tenant uuid.UUID, referenceID uuid.UUID) (*Payload, error) {
	if referenceID.Version() != 7 {
		return nil, fmt.Errorf("payload reference id %s is not a version 7 uuid", referenceID)
	}

	sec, nsec := referenceID.Time().UnixTime()
	day := Day(time.Unix(sec, nsec))

	for _, d := range []time.Time{day, day.AddDate(0, 0, 1)} {
		p, err := l.GetPayload(ctx, tenant, d, referenceID)

		if errors.Is(err, ErrNotFound) {
			continue
		}

		return p, err
	}

	return nil, ErrNotFound
}

func (l *Log) readBundle(ctx context.Context, tenant uuid.UUID, day time.Time, loc Location) (*Bundle, error) {
	key := l.dataKey(tenant, day, loc.File)

	raw, err := l.store.GetRange(ctx, key, loc.Offset, loc.Length)

	if err != nil {
		return nil, err
	}

	var b Bundle

	if err := decodeGzipJSON(raw, &b); err != nil {
		return nil, fmt.Errorf("could not decode bundle %s@%d: %w", key, loc.Offset, err)
	}

	return &b, nil
}

func decodeGzipJSON(raw []byte, v any) error {
	zr, err := gzip.NewReader(bytes.NewReader(raw))

	if err != nil {
		return err
	}

	// one bundle is one gzip member; reading past it would run into the next bundle
	zr.Multistream(false)

	body, err := io.ReadAll(zr)

	if err != nil {
		return err
	}

	return json.Unmarshal(body, v)
}

// ResolveDay returns the UTC day of a task or legacy DAG known only by its external id, from core's
// v1_lookup_table. Lookup partitions are monthly and dropped only when the whole month is past core
// retention, so the row outlives its task by up to a month. Once it is gone too, the day cannot be resolved
// from core and ResolveDay returns ErrNotFound.
func ResolveDay(ctx context.Context, db sqlcv1.DBTX, tenant, externalID uuid.UUID) (time.Time, error) {
	var insertedAt time.Time

	err := db.QueryRow(ctx, `
		SELECT inserted_at
		FROM v1_lookup_table
		WHERE external_id = $1 AND tenant_id = $2
		ORDER BY inserted_at DESC
		LIMIT 1
	`, externalID, tenant).Scan(&insertedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}

	if err != nil {
		return time.Time{}, err
	}

	return Day(insertedAt), nil
}
