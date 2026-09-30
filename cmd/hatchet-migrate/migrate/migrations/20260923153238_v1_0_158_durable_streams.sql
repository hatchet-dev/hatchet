-- +goose Up
-- +goose StatementBegin
ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'STREAM_TOPIC';
ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'STREAM_MESSAGE';
ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'STREAM_RETENTION';

-- create_v1_hourly_range_partition attaches a partition covering the UTC hour
-- containing targetTime, named <table>_YYYYMMDDHH.
CREATE OR REPLACE FUNCTION create_v1_hourly_range_partition(
    targetTableName text,
    targetTime timestamptz
) RETURNS integer
    LANGUAGE plpgsql AS
$$
DECLARE
    hourStart timestamptz;
    newTableName varchar;
BEGIN
    hourStart := date_trunc('hour', targetTime AT TIME ZONE 'UTC') AT TIME ZONE 'UTC';
    newTableName := lower(format('%s_%s', targetTableName, to_char(hourStart AT TIME ZONE 'UTC', 'YYYYMMDDHH24')));

    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = newTableName) THEN
        RETURN 0;
    END IF;

    EXECUTE
        format('CREATE TABLE %s (LIKE %s INCLUDING INDEXES INCLUDING CONSTRAINTS)', newTableName, targetTableName);
    EXECUTE
        format('ALTER TABLE %I SET (
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor=''0.05'',
            autovacuum_vacuum_threshold=''25'',
            autovacuum_analyze_threshold=''25'',
            autovacuum_vacuum_cost_delay=''10'',
            autovacuum_vacuum_cost_limit=''1000''
        )', newTableName);
    EXECUTE
        format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (%L) TO (%L)', targetTableName, newTableName, hourStart, hourStart + INTERVAL '1 hour');
    RETURN 1;
END;
$$;

-- get_v1_hourly_partitions_before lists hourly partitions whose whole hour
-- ends at or before targetTime.
CREATE OR REPLACE FUNCTION get_v1_hourly_partitions_before(
    targetTableName text,
    targetTime timestamptz
) RETURNS TABLE(partition_name text)
    LANGUAGE plpgsql AS
$$
BEGIN
    RETURN QUERY
    SELECT
        inhrelid::regclass::text AS partition_name
    FROM
        pg_inherits
    WHERE
        inhparent = targetTableName::regclass
        AND substring(inhrelid::regclass::text, format('%s_(\d{10})$', targetTableName)) ~ '^\d{10}$'
        AND (to_timestamp(substring(inhrelid::regclass::text, format('%s_(\d{10})$', targetTableName)), 'YYYYMMDDHH24')::timestamp AT TIME ZONE 'UTC') + INTERVAL '1 hour' <= targetTime;
END;
$$;

-- metadata for streams
CREATE TABLE v1_stream_topic (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id UUID NOT NULL,
    namespace TEXT NOT NULL DEFAULT '',
    topic TEXT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_published_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT v1_stream_topic_tenant_ns_topic_key UNIQUE (tenant_id, namespace, topic)
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

    CONSTRAINT v1_stream_message_pkey PRIMARY KEY (tenant_id, namespace, topic, inserted_at, id)
) PARTITION BY RANGE(inserted_at);

-- the partition job keeps a day of hourly partitions ahead; seed the first day here
SELECT create_v1_hourly_range_partition('v1_stream_message', NOW() + make_interval(hours => h))
FROM generate_series(0, 24) AS h;

-- v1_stream_producer_cursor tracks, per (tenant, namespace, topic,
-- producer_id), the last producer_seq durably applied to v1_stream_message.
-- Partitioned by the UTC day a row was written so idle producers age out,
-- retained several times longer than v1_stream_message; a producer's
-- watermark is its row in its latest bucket, copied forward on its first
-- write of each day.
CREATE TABLE v1_stream_producer_cursor (
    tenant_id UUID NOT NULL,
    namespace TEXT NOT NULL DEFAULT '',
    topic TEXT NOT NULL,
    producer_id TEXT NOT NULL,
    bucket DATE NOT NULL,
    last_seq BIGINT NOT NULL,
    -- when last_seq last advanced; a gap only counts as permanent once this stalls
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT v1_stream_producer_cursor_pkey PRIMARY KEY (tenant_id, namespace, topic, producer_id, bucket)
) PARTITION BY RANGE(bucket);

SELECT create_v1_range_partition('v1_stream_producer_cursor', (NOW() AT TIME ZONE 'UTC')::DATE);
SELECT create_v1_range_partition('v1_stream_producer_cursor', (NOW() AT TIME ZONE 'UTC')::DATE + 1);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_stream_producer_cursor;
DROP TABLE v1_stream_message;
DROP TABLE v1_stream_topic;
DROP FUNCTION IF EXISTS get_v1_hourly_partitions_before(text, timestamptz);
DROP FUNCTION IF EXISTS create_v1_hourly_range_partition(text, timestamptz);
-- +goose StatementEnd
