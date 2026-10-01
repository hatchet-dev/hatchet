-- +goose Up
-- +goose StatementBegin
ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'STREAM_TOPIC';
ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'STREAM_MESSAGE';
ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'STREAM_RETENTION';

-- ensure_v1_stream_message_partition makes sure tenantId has a partition of
-- v1_stream_message (itself partitioned by hour) and an hourly partition for
-- the UTC hour containing targetTime. Tables are created detached and then
-- attached, which only takes SHARE UPDATE EXCLUSIVE on the parent so
-- concurrent inserts are never blocked. Returns the number of tables created.
CREATE OR REPLACE FUNCTION ensure_v1_stream_message_partition(
    tenantId uuid,
    targetTime timestamptz
) RETURNS integer
    LANGUAGE plpgsql AS
$$
DECLARE
    tenantTable text := 'v1_stream_message_' || replace(tenantId::text, '-', '');
    hourStart timestamptz := date_trunc('hour', targetTime AT TIME ZONE 'UTC') AT TIME ZONE 'UTC';
    hourTable text;
    created integer := 0;
BEGIN
    hourTable := tenantTable || '_' || to_char(hourStart AT TIME ZONE 'UTC', 'YYYYMMDDHH24');

    PERFORM set_config('lock_timeout', '5s', true);

    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = tenantTable) THEN
        BEGIN
            EXECUTE format('CREATE TABLE %I (LIKE v1_stream_message INCLUDING DEFAULTS INCLUDING CONSTRAINTS INCLUDING INDEXES) PARTITION BY RANGE (inserted_at)', tenantTable);
            EXECUTE format('ALTER TABLE v1_stream_message ATTACH PARTITION %I FOR VALUES IN (%L)', tenantTable, tenantId);
            created := created + 1;
        EXCEPTION WHEN duplicate_table OR unique_violation THEN
            -- a concurrent caller created it first
        END;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = hourTable) THEN
        BEGIN
            EXECUTE format('CREATE TABLE %I (LIKE v1_stream_message INCLUDING DEFAULTS INCLUDING CONSTRAINTS INCLUDING INDEXES)', hourTable);
            EXECUTE format('ALTER TABLE %I SET (
                autovacuum_vacuum_scale_factor = ''0.1'',
                autovacuum_analyze_scale_factor = ''0.05'',
                autovacuum_vacuum_threshold = ''25'',
                autovacuum_analyze_threshold = ''25''
            )', hourTable);
            EXECUTE format('ALTER TABLE %I ATTACH PARTITION %I FOR VALUES FROM (%L) TO (%L)', tenantTable, hourTable, hourStart, hourStart + INTERVAL '1 hour');
            created := created + 1;
        EXCEPTION WHEN duplicate_table OR unique_violation THEN
            -- a concurrent caller created it first
        END;
    END IF;

    RETURN created;
END;
$$;

-- get_v1_stream_message_retention_start returns the start of tenantId's oldest
-- hourly partition, reading only that tenant's partitions. NULL if it has none.
CREATE OR REPLACE FUNCTION get_v1_stream_message_retention_start(
    tenantId uuid
) RETURNS timestamptz
    LANGUAGE plpgsql STABLE AS
$$
DECLARE
    tenantTable regclass := to_regclass('v1_stream_message_' || replace(tenantId::text, '-', ''));
BEGIN
    IF tenantTable IS NULL THEN
        RETURN NULL;
    END IF;

    RETURN (
        SELECT MIN(to_timestamp(right(c.relname, 10), 'YYYYMMDDHH24')::timestamp AT TIME ZONE 'UTC')
        FROM pg_inherits i
        JOIN pg_class c ON c.oid = i.inhrelid
        WHERE i.inhparent = tenantTable
            AND NOT i.inhdetachpending
    );
END;
$$;

-- list_v1_stream_message_hour_partitions lists every hourly partition of
-- v1_stream_message with its tenant and hour.
CREATE OR REPLACE FUNCTION list_v1_stream_message_hour_partitions()
RETURNS TABLE(parent_table text, partition_name text, tenant_id uuid, hour_start timestamptz)
    LANGUAGE plpgsql STABLE AS
$$
BEGIN
    RETURN QUERY
    SELECT
        tp.relname::text,
        hp.relname::text,
        substring(tp.relname FROM 19)::uuid,
        to_timestamp(right(hp.relname, 10), 'YYYYMMDDHH24')::timestamp AT TIME ZONE 'UTC'
    FROM pg_inherits ti
    JOIN pg_class tp ON tp.oid = ti.inhrelid
    JOIN pg_inherits hi ON hi.inhparent = tp.oid AND NOT hi.inhdetachpending
    JOIN pg_class hp ON hp.oid = hi.inhrelid
    WHERE ti.inhparent = 'v1_stream_message'::regclass;
END;
$$;

-- list_v1_stream_message_empty_tenant_partitions lists tenant partitions of
-- v1_stream_message with no hourly partitions left.
CREATE OR REPLACE FUNCTION list_v1_stream_message_empty_tenant_partitions()
RETURNS TABLE(partition_name text, tenant_id uuid)
    LANGUAGE plpgsql STABLE AS
$$
BEGIN
    RETURN QUERY
    SELECT tp.relname::text, substring(tp.relname FROM 19)::uuid
    FROM pg_inherits ti
    JOIN pg_class tp ON tp.oid = ti.inhrelid
    WHERE ti.inhparent = 'v1_stream_message'::regclass
        AND NOT ti.inhdetachpending
        AND NOT EXISTS (SELECT 1 FROM pg_inherits hi WHERE hi.inhparent = tp.oid);
END;
$$;

-- metadata for streams
CREATE TABLE v1_stream_topic (
    tenant_id UUID NOT NULL,
    namespace TEXT NOT NULL DEFAULT '',
    topic TEXT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_published_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT v1_stream_topic_pkey PRIMARY KEY (tenant_id, namespace, topic)
);

-- xact_id stores the transaction id that this row was inserted with
-- when querying, we filter based on the minimum in-flight transaction number,
-- thus all transactions started after the oldest in flight transaction are ignored
-- strictly ordering messages by transaction *start* rather than commit.
CREATE TABLE v1_stream_message (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    namespace TEXT NOT NULL DEFAULT '',
    topic TEXT NOT NULL,
    payload BYTEA NOT NULL,
    xact_id XID8 NOT NULL DEFAULT pg_current_xact_id(),
    producer_id TEXT NOT NULL,
    producer_seq BIGINT NOT NULL,

    -- id before inserted_at, so a topic's index is in id order for keyset
    -- reads (inserted_at is only here because every partition key must be)
    CONSTRAINT v1_stream_message_pkey PRIMARY KEY (tenant_id, namespace, topic, id, inserted_at)
) PARTITION BY LIST(tenant_id);

-- each tenant gets its own hourly-partitioned table on its first message (see
-- ensure_v1_stream_message_partition)

-- v1_stream_producer_cursor tracks, per (tenant, namespace, topic,
-- producer_id), the last producer_seq durably applied to v1_stream_message.
-- Partitioned by the UTC day a row was written so idle producers age out
-- after a few days (streamProducerCursorRetention); a producer's watermark is
-- its row in its latest bucket, copied forward on its first write of each day.
CREATE TABLE v1_stream_producer_cursor (
    tenant_id UUID NOT NULL,
    namespace TEXT NOT NULL DEFAULT '',
    topic TEXT NOT NULL,
    producer_id TEXT NOT NULL,
    bucket DATE NOT NULL,
    last_seq BIGINT NOT NULL,

    CONSTRAINT v1_stream_producer_cursor_pkey PRIMARY KEY (tenant_id, namespace, topic, producer_id, bucket)
) PARTITION BY RANGE(bucket);

-- headroom so each publish's last_seq update can stay on its page (HOT)
SELECT create_v1_range_partition('v1_stream_producer_cursor', (NOW() AT TIME ZONE 'UTC')::DATE, 80);
SELECT create_v1_range_partition('v1_stream_producer_cursor', (NOW() AT TIME ZONE 'UTC')::DATE + 1, 80);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_stream_producer_cursor;
DROP TABLE v1_stream_message;
DROP TABLE v1_stream_topic;
DROP FUNCTION IF EXISTS list_v1_stream_message_empty_tenant_partitions();
DROP FUNCTION IF EXISTS list_v1_stream_message_hour_partitions();
DROP FUNCTION IF EXISTS get_v1_stream_message_retention_start(uuid);
DROP FUNCTION IF EXISTS ensure_v1_stream_message_partition(uuid, timestamptz);
-- +goose StatementEnd
