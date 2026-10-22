//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// v7ReferenceIdAt builds a version 7 uuid whose embedded millisecond timestamp is t.
func v7ReferenceIdAt(t time.Time) uuid.UUID {
	id := uuid.New()

	var ms [8]byte
	binary.BigEndian.PutUint64(ms[:], uint64(t.UnixMilli())<<16)
	copy(id[:6], ms[:6])

	id[6] = (id[6] & 0x0f) | 0x70
	id[8] = (id[8] & 0x3f) | 0x80

	return id
}

func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// recordingExternalStore stands in for the S3 store hatchet-cloud installs through OverwriteExternalStore.
type recordingExternalStore struct {
	requested []RetrieveFromExternalOpts
	payloads  map[RetrieveFromExternalOpts][]byte
}

func (s *recordingExternalStore) Store(ctx context.Context, payloads ...OffloadToExternalStoreOpts) (*ExternalIndexFileLocationKey, error) {
	return nil, fmt.Errorf("not used by this test")
}

func (s *recordingExternalStore) Retrieve(ctx context.Context, opts ...RetrieveFromExternalOpts) (map[RetrieveFromExternalOpts][]byte, error) {
	s.requested = append(s.requested, opts...)

	result := make(map[RetrieveFromExternalOpts][]byte, len(opts))

	for _, opt := range opts {
		if data, ok := s.payloads[opt]; ok {
			result[opt] = data
		}
	}

	return result, nil
}

func TestReadPayloadsLocatesInlineRowsByOwnKeyAndByReferenceId(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	store := newBatchTestRepository(pool).sharedRepository.payloadStore

	now := time.Now().UTC()
	referrerInsertedAt := timestamptz(now)

	// The shared row's reference id was minted three days before the referrer was created, so the row
	// lives in a partition the referrer's own inserted_at does not point at.
	referenceTime := now.Add(-72 * time.Hour)
	_, err := pool.Exec(ctx, "SELECT create_v1_range_partition('v1_payload', $1::DATE)", partitionDateOf(referenceTime))
	require.NoError(t, err)

	v7Reference := v7ReferenceIdAt(referenceTime)
	require.Equal(t, uuid.Version(7), v7Reference.Version())

	v4Reference := uuid.New()
	require.NotEqual(t, uuid.Version(7), v4Reference.Version())

	v7Owner := PayloadRef[TaskInputShape]{TenantId: internalTenantId, Id: 1, InsertedAt: referrerInsertedAt, ExternalId: uuid.New(), PayloadReferenceId: &v7Reference}
	ownRow := PayloadRef[TaskInputShape]{TenantId: internalTenantId, Id: 2, InsertedAt: referrerInsertedAt, ExternalId: uuid.New()}
	v4Owner := PayloadRef[TaskInputShape]{TenantId: internalTenantId, Id: 3, InsertedAt: referrerInsertedAt, ExternalId: uuid.New(), PayloadReferenceId: &v4Reference}

	require.NoError(t, store.Store(ctx, pool,
		v7Owner.StoreOpts(NewPayload[TaskInputShape]([]byte(`{"shared":"v7"}`))),
		ownRow.StoreOpts(NewPayload[TaskInputShape]([]byte(`{"own":true}`))),
		v4Owner.StoreOpts(NewPayload[TaskInputShape]([]byte(`{"shared":"v4"}`))),
	))

	var storedAt time.Time
	require.NoError(t, pool.QueryRow(ctx, "SELECT inserted_at FROM v1_payload WHERE external_id = $1", v7Reference).Scan(&storedAt))
	assert.Equal(t, referenceTime.Truncate(time.Millisecond).UnixMicro(), storedAt.UnixMicro(), "the shared row is stored under the reference id's timestamp")

	v7Referrer := PayloadRef[TaskInputShape]{TenantId: internalTenantId, Id: 10, InsertedAt: referrerInsertedAt, ExternalId: uuid.New(), PayloadReferenceId: &v7Reference}
	v4Referrer := PayloadRef[TaskInputShape]{TenantId: internalTenantId, Id: 11, InsertedAt: referrerInsertedAt, ExternalId: uuid.New(), PayloadReferenceId: &v4Reference}
	missing := PayloadRef[TaskInputShape]{TenantId: internalTenantId, Id: 12, InsertedAt: referrerInsertedAt, ExternalId: uuid.New()}

	payloads, err := store.TaskInputs().Retrieve(ctx, pool, v7Referrer, ownRow, v4Referrer, missing)
	require.NoError(t, err)

	assert.JSONEq(t, `{"shared":"v7"}`, string(payloads[v7Referrer].Bytes()))
	assert.JSONEq(t, `{"own":true}`, string(payloads[ownRow].Bytes()))
	assert.JSONEq(t, `{"shared":"v4"}`, string(payloads[v4Referrer].Bytes()))

	_, found := payloads[missing]
	assert.False(t, found, "a key with no row is absent from the result")
	assert.Len(t, payloads, 3)
}

func TestReadPayloadsReturnsExternalLocationRowsIntact(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	queries := sqlcv1.New()

	now := time.Now().UTC()
	referenceTime := now.Add(-72 * time.Hour)
	_, err := pool.Exec(ctx, "SELECT create_v1_range_partition('v1_payload', $1::DATE)", partitionDateOf(referenceTime))
	require.NoError(t, err)

	ownExternalId := uuid.New()
	v7Reference := v7ReferenceIdAt(referenceTime)

	const insertExternalRow = `
		INSERT INTO v1_payload (tenant_id, id, inserted_at, external_id, type, location, external_location_key, inline_content)
		VALUES ($1, $2, $3, $4, 'TASK_INPUT', 'EXTERNAL', $5, NULL)`

	_, err = pool.Exec(ctx, insertExternalRow, internalTenantId, int64(42), timestamptz(now), ownExternalId, "blocks/own.idx")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, insertExternalRow, internalTenantId, int64(43), timestamptz(referenceTime.Truncate(time.Millisecond)), v7Reference, "blocks/shared.idx")
	require.NoError(t, err)

	rows, err := queries.ReadPayloads(ctx, pool, sqlcv1.ReadPayloadsParams{
		Tenantids:           []uuid.UUID{internalTenantId, internalTenantId},
		Ids:                 []int64{42, 99},
		Insertedats:         []pgtype.Timestamptz{timestamptz(now), timestamptz(now)},
		Types:               []string{"TASK_INPUT", "TASK_INPUT"},
		Payloadreferenceids: []uuid.UUID{uuid.Nil, v7Reference},
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byExternalId := make(map[uuid.UUID]*sqlcv1.V1Payload, len(rows))

	for _, row := range rows {
		byExternalId[row.ExternalID] = row
	}

	own := byExternalId[ownExternalId]
	require.NotNil(t, own)
	assert.Equal(t, sqlcv1.V1PayloadLocationEXTERNAL, own.Location)
	assert.Equal(t, "blocks/own.idx", own.ExternalLocationKey.String)
	assert.Nil(t, own.InlineContent)

	shared := byExternalId[v7Reference]
	require.NotNil(t, shared)
	assert.Equal(t, sqlcv1.V1PayloadLocationEXTERNAL, shared.Location)
	assert.Equal(t, "blocks/shared.idx", shared.ExternalLocationKey.String)
	assert.Nil(t, shared.InlineContent)
	assert.Equal(t, referenceTime.Truncate(time.Millisecond).UnixMicro(), shared.InsertedAt.Time.UnixMicro())
}

func TestPayloadMissingInlineIsResolvedThroughTheOffloadedBlockIndex(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	queries := sqlcv1.New()
	store := newBatchTestRepository(pool).sharedRepository.payloadStore

	external := &recordingExternalStore{payloads: make(map[RetrieveFromExternalOpts][]byte)}
	store.OverwriteExternalStore(external)

	now := time.Now().UTC()
	referenceTime := now.Add(-72 * time.Hour)
	v7Reference := v7ReferenceIdAt(referenceTime)
	ownExternalId := uuid.New()

	// Index blocks are keyed by the partition date the row was stored under: the reference id's day for
	// the shared row, the referrer's own day for the own row.
	for _, block := range []sqlcv1.CreateOffloadedPayloadIndexBlockParams{
		{Payloadinsertedatdate: partitionDateOf(referenceTime), Blocklowerexternalidbound: uuid.Nil, Blockupperexternalidbound: v7Reference, Indexfilekey: "blocks/shared.idx"},
		{Payloadinsertedatdate: partitionDateOf(now), Blocklowerexternalidbound: uuid.Nil, Blockupperexternalidbound: ownExternalId, Indexfilekey: "blocks/own.idx"},
	} {
		require.NoError(t, queries.CreateOffloadedPayloadIndexBlock(ctx, pool, block))
	}

	sharedKey := RetrieveFromExternalOpts{IndexFileKey: "blocks/shared.idx", ExternalId: v7Reference}
	ownKey := RetrieveFromExternalOpts{IndexFileKey: "blocks/own.idx", ExternalId: ownExternalId}
	external.payloads[sharedKey] = []byte(`{"offloaded":"shared"}`)
	external.payloads[ownKey] = []byte(`{"offloaded":"own"}`)

	referrer := PayloadRef[TaskInputShape]{TenantId: internalTenantId, Id: 20, InsertedAt: timestamptz(now), ExternalId: uuid.New(), PayloadReferenceId: &v7Reference}
	ownRow := PayloadRef[TaskInputShape]{TenantId: internalTenantId, Id: 21, InsertedAt: timestamptz(now), ExternalId: ownExternalId}

	payloads, err := store.TaskInputs().Retrieve(ctx, pool, referrer, ownRow)
	require.NoError(t, err)

	assert.ElementsMatch(t, []RetrieveFromExternalOpts{sharedKey, ownKey}, external.requested)
	assert.JSONEq(t, `{"offloaded":"shared"}`, string(payloads[referrer].Bytes()))
	assert.JSONEq(t, `{"offloaded":"own"}`, string(payloads[ownRow].Bytes()))
}

func TestPayloadReferenceInsertedAtMatchesGoPartitionTime(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	fallback := timestamptz(time.Date(2026, 10, 22, 12, 34, 56, 789_000, time.UTC))

	cases := []struct {
		name string
		id   uuid.UUID
	}{
		{"unix epoch", v7ReferenceIdAt(time.UnixMilli(0))},
		{"one millisecond", v7ReferenceIdAt(time.UnixMilli(1))},
		{"sub-millisecond part dropped", v7ReferenceIdAt(time.Unix(1_700_000_000, 999_999))},
		{"start of a day", v7ReferenceIdAt(time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC))},
		{"last millisecond of a day", v7ReferenceIdAt(time.Date(2026, 10, 23, 23, 59, 59, 999_000_000, time.UTC))},
		{"2038 rollover", v7ReferenceIdAt(time.UnixMilli(1 << 31 * 1000))},
		// extract_uuid7_timestamp multiplies the millisecond count by an interval in float8, which is exact
		// up to 2^53 microseconds (year 2255). Ids minted from the wall clock never get near that.
		{"largest exactly representable timestamp", v7ReferenceIdAt(time.UnixMilli((1 << 53) / 1000))},
		{"fresh reference id", NewPayloadReferenceId()},
		{"version 4 falls back", uuid.New()},
		{"zero uuid falls back", uuid.Nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := tc.id
			want := RetrievePayloadOpts{InsertedAt: fallback, PayloadReferenceId: &id}.partitionTime()

			var got pgtype.Timestamptz
			require.NoError(t, pool.QueryRow(ctx, "SELECT v1_payload_reference_inserted_at($1, $2)", id, fallback).Scan(&got))
			require.True(t, got.Valid)

			assert.Equal(t, want.Time.UnixMicro(), got.Time.UnixMicro(), "want %s, got %s", want.Time, got.Time)
		})
	}
}
