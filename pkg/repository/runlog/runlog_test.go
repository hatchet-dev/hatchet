//go:build !e2e && !load && !rampup && !integration

package runlog

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

var testDay = time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)

func testBundle(tenant, run uuid.UUID, i int, payloadRef uuid.UUID) *Bundle {
	insertedAt := pgtype.Timestamptz{Time: testDay.Add(time.Duration(i) * time.Minute), Valid: true}
	external := uuid.New()

	return &Bundle{
		TenantID: tenant,
		Task: &sqlcv1.V1Task{
			ID:            int64(i + 1),
			InsertedAt:    insertedAt,
			TenantID:      tenant,
			ExternalID:    external,
			WorkflowRunID: run,
			DisplayName:   fmt.Sprintf("task-%d", i),
			Input:         []byte(`{}`),
		},
		Events: []*sqlcv1.V1TaskEvent{{
			ID:             int64(100 + i),
			TaskID:         int64(i + 1),
			TaskInsertedAt: insertedAt,
			EventType:      sqlcv1.V1TaskEventTypeCOMPLETED,
			ExternalID:     uuid.New(),
		}},
		Payloads: []Payload{
			{Type: sqlcv1.V1PayloadTypeTASKINPUT, ID: int64(i + 1), InsertedAt: insertedAt, ExternalID: payloadRef, Shared: true, Value: json.RawMessage(`{"shared":true}`)},
			{Type: sqlcv1.V1PayloadTypeTASKEVENTDATA, ID: int64(100 + i), InsertedAt: insertedAt, ExternalID: uuid.New(), Value: json.RawMessage(fmt.Sprintf(`{"output":%d}`, i))},
		},
	}
}

func TestDayWriterAndReader(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	log := New(store, "pfx")

	tenant := uuid.New()
	runA, runB := uuid.New(), uuid.New()
	ref := uuid.Must(uuid.NewV7())

	w := log.NewDayWriter(tenant, testDay.Add(13*time.Hour))

	var written []*Bundle

	for i := range 5 {
		run := runA
		if i >= 3 {
			run = runB
		}

		b := testBundle(tenant, run, i, ref)
		written = append(written, b)
		require.NoError(t, w.Add(ctx, b))
	}

	w.AddRunDays(runA, testDay.AddDate(0, 0, 1), testDay)

	// nothing is readable before Close writes the index and the marker
	_, err := log.GetTask(ctx, tenant, testDay, written[0].Task.ExternalID)
	require.ErrorIs(t, err, ErrNotFound)

	stats, err := w.Close(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(5), stats.Bundles)
	assert.Equal(t, int64(1), stats.Files)
	assert.Positive(t, stats.BytesCompressed)
	assert.Positive(t, stats.BytesUncompressed)

	prefix := "pfx/v1/" + tenant.String() + "/2026/03/04/"
	assert.Equal(t, []string{prefix + "_COMPLETE", prefix + "data-00000.jsonl.gz", prefix + "index.json.gz"}, store.Keys(prefix))

	t.Run("task", func(t *testing.T) {
		for _, want := range written {
			got, err := log.GetTask(ctx, tenant, testDay, want.Task.ExternalID)
			require.NoError(t, err)
			assertSameJSON(t, want, got)
		}

		_, err := log.GetTask(ctx, tenant, testDay, uuid.New())
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("ranged read of one bundle", func(t *testing.T) {
		idx, err := log.Index(ctx, tenant, testDay)
		require.NoError(t, err)

		loc := idx.Tasks[written[2].Task.ExternalID]
		assert.Positive(t, loc.Offset)

		raw, err := store.GetRange(ctx, prefix+"data-00000.jsonl.gz", loc.Offset, loc.Length)
		require.NoError(t, err)

		// the range is exactly one gzip member
		zr, err := gzip.NewReader(bytes.NewReader(raw))
		require.NoError(t, err)
		body, err := io.ReadAll(zr)
		require.NoError(t, err)

		var got Bundle
		require.NoError(t, json.Unmarshal(body, &got))
		assertSameJSON(t, written[2], &got)
	})

	t.Run("run", func(t *testing.T) {
		bundles, err := log.GetRun(ctx, tenant, testDay, runA)
		require.NoError(t, err)
		require.Len(t, bundles, 3)

		idx, err := log.Index(ctx, tenant, testDay)
		require.NoError(t, err)
		assert.Equal(t, []string{"2026-03-05"}, idx.Runs[runA].OtherDays)

		bundles, err = log.GetRun(ctx, tenant, testDay, runB)
		require.NoError(t, err)
		require.Len(t, bundles, 2)

		_, err = log.GetRun(ctx, tenant, testDay, uuid.New())
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("payload", func(t *testing.T) {
		p, err := log.GetPayload(ctx, tenant, testDay, written[4].Payloads[1].ExternalID)
		require.NoError(t, err)
		assert.JSONEq(t, `{"output":4}`, string(p.Value))

		_, err = log.GetPayload(ctx, tenant, testDay, uuid.New())
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("other tenant", func(t *testing.T) {
		_, err := log.GetTask(ctx, uuid.New(), testDay, written[0].Task.ExternalID)
		require.ErrorIs(t, err, ErrNotFound)
	})
}

func TestGetPayloadByReferenceTriesNextDay(t *testing.T) {
	ctx := context.Background()
	log := New(NewMemoryStore(), "")
	tenant := uuid.New()

	// a reference id minted just before midnight, first used by a task of the next day
	ref := uuidV7At(t, testDay.Add(24*time.Hour-time.Second))
	nextDay := testDay.AddDate(0, 0, 1)

	w := log.NewDayWriter(tenant, nextDay)
	require.NoError(t, w.Add(ctx, testBundle(tenant, uuid.New(), 0, ref)))
	_, err := w.Close(ctx)
	require.NoError(t, err)

	p, err := log.GetPayloadByReference(ctx, tenant, ref)
	require.NoError(t, err)
	assert.JSONEq(t, `{"shared":true}`, string(p.Value))

	_, err = log.GetPayloadByReference(ctx, tenant, uuidV7At(t, testDay))
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDayWriterSplitsDataFiles(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	log := New(store, "")
	tenant := uuid.New()

	defer func(prev int) { maxDataFileBytes = prev }(maxDataFileBytes)
	maxDataFileBytes = 1

	w := log.NewDayWriter(tenant, testDay)

	var written []*Bundle

	for i := range 3 {
		b := testBundle(tenant, uuid.New(), i, uuid.Must(uuid.NewV7()))
		written = append(written, b)
		require.NoError(t, w.Add(ctx, b))
	}

	stats, err := w.Close(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.Files)

	for i, want := range written {
		idx, err := log.Index(ctx, tenant, testDay)
		require.NoError(t, err)
		assert.Equal(t, i, idx.Tasks[want.Task.ExternalID].File)

		got, err := log.GetTask(ctx, tenant, testDay, want.Task.ExternalID)
		require.NoError(t, err)
		assertSameJSON(t, want, got)
	}
}

func TestIncompleteDayIsAbsent(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	log := New(store, "")
	tenant := uuid.New()

	w := log.NewDayWriter(tenant, testDay)
	b := testBundle(tenant, uuid.New(), 0, uuid.Must(uuid.NewV7()))
	require.NoError(t, w.Add(ctx, b))
	_, err := w.Close(ctx)
	require.NoError(t, err)

	// a crash between the index and the marker of a rewrite leaves the index without the marker
	marker := log.completeKey(tenant, testDay)
	delete(store.objects, marker)

	_, err = New(store, "").GetTask(ctx, tenant, testDay, b.Task.ExternalID)
	require.ErrorIs(t, err, ErrNotFound)
}

func uuidV7At(t *testing.T, at time.Time) uuid.UUID {
	id := uuid.Must(uuid.NewV7())
	ms := uint64(at.UnixMilli()) // nolint:gosec

	for i := range 6 {
		id[i] = byte(ms >> (40 - 8*i))
	}

	sec, _ := id.Time().UnixTime()
	require.Equal(t, at.Unix(), sec)

	return id
}

func assertSameJSON(t *testing.T, want, got any) {
	t.Helper()

	wantJSON, err := json.Marshal(want)
	require.NoError(t, err)
	gotJSON, err := json.Marshal(got)
	require.NoError(t, err)

	assert.JSONEq(t, string(wantJSON), string(gotJSON))
}
