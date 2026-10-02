-- +goose Up
-- +goose StatementBegin
ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'STREAM_TOPIC';
ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'STREAM_MESSAGE';
ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'STREAM_RETENTION';

-- durable streams are off for a tenant until it's entitled to them
ALTER TABLE tenant_entitlement ADD COLUMN durable_streams BOOLEAN NOT NULL DEFAULT FALSE;

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

-- get_v1_hourly_partitions_before lists hourly partitions ending at or before targetTime.
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

-- fillfactor leaves room for every publish batch's last_offset update to stay HOT
CREATE TABLE v1_stream_topic (
    tenant_id UUID NOT NULL,
    namespace TEXT NOT NULL DEFAULT '',
    topic TEXT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_published_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- the last offset reserved (see v1_stream_message)
    last_offset BIGINT NOT NULL DEFAULT 0,

    CONSTRAINT v1_stream_topic_pkey PRIMARY KEY (tenant_id, namespace, topic)
) WITH (fillfactor = 80);

-- id is the message's offset in its topic, reserved from last_offset under a
-- row lock held until commit, so offsets become visible in order and readers
-- page on id alone.
CREATE TABLE v1_stream_message (
    id BIGINT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    namespace TEXT NOT NULL DEFAULT '',
    topic TEXT NOT NULL,
    payload BYTEA NOT NULL,
    producer_id TEXT NOT NULL,
    producer_seq BIGINT NOT NULL,

    -- id first so reads seek by offset; inserted_at is required as the partition key
    CONSTRAINT v1_stream_message_pkey PRIMARY KEY (tenant_id, namespace, topic, id, inserted_at)
) PARTITION BY RANGE(inserted_at);

-- the partition job keeps a day of hourly partitions ahead; seed the first day here
SELECT create_v1_hourly_range_partition('v1_stream_message', NOW() + make_interval(hours => h))
FROM generate_series(0, 24) AS h;

-- v1_stream_producer_cursor holds each producer's last stored producer_seq.
-- Day buckets let idle producers age out; an active producer's watermark is
-- copied into each new day's bucket.
CREATE TABLE v1_stream_producer_cursor (
    tenant_id UUID NOT NULL,
    namespace TEXT NOT NULL DEFAULT '',
    topic TEXT NOT NULL,
    producer_id TEXT NOT NULL,
    bucket DATE NOT NULL,
    last_seq BIGINT NOT NULL,

    CONSTRAINT v1_stream_producer_cursor_pkey PRIMARY KEY (tenant_id, namespace, topic, producer_id, bucket)
) PARTITION BY RANGE(bucket);

-- room for each publish's last_seq update to stay HOT
SELECT create_v1_range_partition('v1_stream_producer_cursor', (NOW() AT TIME ZONE 'UTC')::DATE, 80);
SELECT create_v1_range_partition('v1_stream_producer_cursor', (NOW() AT TIME ZONE 'UTC')::DATE + 1, 80);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenant_entitlement DROP COLUMN durable_streams;
DROP TABLE v1_stream_producer_cursor;
DROP TABLE v1_stream_message;
DROP TABLE v1_stream_topic;
DROP FUNCTION IF EXISTS get_v1_hourly_partitions_before(text, timestamptz);
DROP FUNCTION IF EXISTS create_v1_hourly_range_partition(text, timestamptz);
-- +goose StatementEnd
