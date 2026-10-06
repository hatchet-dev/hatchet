-- name: UpsertStreamTopic :one
-- `inserted` marks a new topic, which counts against the topic limit.
INSERT INTO v1_stream_topic (tenant_id, namespace, topic)
VALUES (@tenantId::uuid, @namespace::text, @topic::text)
ON CONFLICT (tenant_id, namespace, topic) DO UPDATE
SET last_published_at = NOW()
RETURNING sqlc.embed(v1_stream_topic), (xmax = 0) AS inserted;

-- name: DeleteStreamTopic :exec
-- Undoes UpsertStreamTopic for a topic over the limit.
DELETE FROM v1_stream_topic
WHERE tenant_id = @tenantId::uuid AND namespace = @namespace::text AND topic = @topic::text;

-- name: DeleteIdleStreamTopics :execrows
-- Topics with no publish within their tenant's retention, in batches to keep
-- each statement short.
DELETE FROM v1_stream_topic
WHERE (tenant_id, namespace, topic) IN (
    SELECT t.tenant_id, t.namespace, t.topic
    FROM v1_stream_topic t
    LEFT JOIN "TenantResourceLimit" l ON l."tenantId" = t.tenant_id AND l."resource" = 'STREAM_RETENTION'
    WHERE t.last_published_at < NOW() - make_interval(hours => LEAST(COALESCE(l."limitValue", @defaultRetentionHours::integer), @maxRetentionHours::integer))
    LIMIT @batchSize::integer
);

-- name: ListStreamProducerCursorPartitionsBeforeDate :many
-- Separate from ListPartitionsBeforeDate: producer cursors have their own retention.
SELECT
    'v1_stream_producer_cursor' AS parent_table,
    p::text AS partition_name
FROM get_v1_partitions_before_date('v1_stream_producer_cursor', @date::date) AS p;

-- name: CountStreamTopics :one
SELECT COUNT(*) FROM v1_stream_topic WHERE tenant_id = @tenantId::uuid;

-- name: ReserveStreamTopicOffsets :batchone
-- Reserves @count offsets and returns the last. The row lock is held until
-- commit, so offsets become visible in order. A topic the idle sweep removed
-- restarts at 1, safe since all its messages had passed retention.
INSERT INTO v1_stream_topic (tenant_id, namespace, topic, last_offset)
VALUES (@tenantId::uuid, @namespace::text, @topic::text, @count::bigint)
ON CONFLICT (tenant_id, namespace, topic) DO UPDATE
SET last_offset = v1_stream_topic.last_offset + EXCLUDED.last_offset, last_published_at = NOW()
RETURNING last_offset;

-- name: InsertOrderedStreamMessage :batchone
-- Inserts the message only if producer_seq is one past the producer's
-- watermark: its row in the latest bucket, compare-and-swapped so concurrent
-- attempts serialize on one row lock. A day's first write copies the
-- watermark into today's bucket, so an active producer outlives cursor
-- retention. Reading only from @minBucket keeps planning to retained
-- partitions; an older watermark counts as none. current_last_seq is the
-- watermark (NULL if none), telling a gap from a duplicate.
WITH latest AS (
    SELECT bucket, last_seq
    FROM v1_stream_producer_cursor
    WHERE tenant_id = @tenantId::uuid AND namespace = @namespace::text AND topic = @topic::text AND producer_id = @producerId::text
        AND bucket >= @minBucket::date
    ORDER BY bucket DESC
    LIMIT 1
), cas AS (
    UPDATE v1_stream_producer_cursor c
    SET last_seq = @producerSeq::bigint
    FROM latest
    WHERE c.tenant_id = @tenantId::uuid AND c.namespace = @namespace::text AND c.topic = @topic::text AND c.producer_id = @producerId::text
        AND c.bucket = latest.bucket
        AND c.bucket >= @minBucket::date
        -- computed by the caller as producerSeq - 1 so sqlc binds one plain parameter
        AND c.last_seq = @expectedPrevSeq::bigint
    RETURNING c.bucket
), first_message AS (
    -- only seq 0 may create a producer's first row, so a reordered seq>0 can't claim it
    INSERT INTO v1_stream_producer_cursor (tenant_id, namespace, topic, producer_id, bucket, last_seq)
    SELECT @tenantId::uuid, @namespace::text, @topic::text, @producerId::text, (NOW() AT TIME ZONE 'UTC')::date, @producerSeq::bigint
    WHERE @producerSeq::bigint = 0 AND NOT EXISTS (SELECT 1 FROM latest)
    ON CONFLICT DO NOTHING
    RETURNING 1
), carried_forward AS (
    INSERT INTO v1_stream_producer_cursor (tenant_id, namespace, topic, producer_id, bucket, last_seq)
    SELECT @tenantId::uuid, @namespace::text, @topic::text, @producerId::text, (NOW() AT TIME ZONE 'UTC')::date, @producerSeq::bigint
    FROM cas
    WHERE cas.bucket < (NOW() AT TIME ZONE 'UTC')::date
    ON CONFLICT (tenant_id, namespace, topic, producer_id, bucket) DO UPDATE
    SET last_seq = GREATEST(v1_stream_producer_cursor.last_seq, EXCLUDED.last_seq)
), applied AS (
    SELECT 1 FROM cas
    UNION ALL
    SELECT 1 FROM first_message
), inserted_row AS (
    INSERT INTO v1_stream_message (id, tenant_id, namespace, topic, payload, producer_id, producer_seq, payload_id, payload_inserted_at)
    -- the offset reserved for this message by ReserveStreamTopicOffsets
    SELECT @messageOffset::bigint, @tenantId::uuid, @namespace::text, @topic::text, @payload::bytea, @producerId::text, @producerSeq::bigint,
        sqlc.narg('payloadId')::uuid, sqlc.narg('payloadInsertedAt')::timestamptz
    WHERE EXISTS (SELECT 1 FROM applied)
    RETURNING 1
)
-- a LEFT JOIN rather than a scalar subquery so sqlc infers current_last_seq as nullable
SELECT
    EXISTS (SELECT 1 FROM inserted_row) AS inserted,
    latest.last_seq AS current_last_seq
FROM (SELECT 1) AS one
LEFT JOIN latest ON true;

-- name: GetStreamMessageRetentionStart :one
-- The oldest partition's start: anything earlier was dropped. NULL if none.
SELECT MIN(to_timestamp(substring(p::text, 'v1_stream_message_(\d{10})$'), 'YYYYMMDDHH24')::timestamp AT TIME ZONE 'UTC')::timestamptz AS retention_start
FROM get_v1_hourly_partitions_before('v1_stream_message', 'infinity'::timestamptz) AS p;

-- name: CreateStreamMessagePartitions :exec
SELECT create_v1_hourly_range_partition('v1_stream_message', hours.hour_start)
FROM generate_series(@fromTime::timestamptz, @toTime::timestamptz, INTERVAL '1 hour') AS hours(hour_start);

-- name: CreateStreamPayloadPartitions :exec
SELECT create_v1_hourly_range_partition('v1_stream_payload', hours.hour_start)
FROM generate_series(@fromTime::timestamptz, @toTime::timestamptz, INTERVAL '1 hour') AS hours(hour_start);

-- name: ListStreamPayloadPartitionsBefore :many
SELECT
    'v1_stream_payload' AS parent_table,
    p::text AS partition_name
FROM get_v1_hourly_partitions_before('v1_stream_payload', @before::timestamptz) AS p;

-- name: ListStreamMessagePartitionsBefore :many
SELECT
    'v1_stream_message' AS parent_table,
    p::text AS partition_name
FROM get_v1_hourly_partitions_before('v1_stream_message', @before::timestamptz) AS p;

-- name: GetMaxStreamRetentionHours :one
-- Shared partitions can only be dropped once every tenant is done with them.
-- Tenants without a row use the default.
SELECT GREATEST(COALESCE(MAX("limitValue"), 0), @defaultRetentionHours::integer)::integer AS max_hours
FROM "TenantResourceLimit"
WHERE "resource" = 'STREAM_RETENTION';

-- name: ListStreamRetentionDeleteCandidates :many
-- Tenants with retention under @maxRetentionHours, whose expired messages sit
-- in partitions not yet droppable. Tenants without a row use the default.
SELECT "tenantId"::uuid AS tenant_id, "limitValue"::integer AS retention_hours
FROM "TenantResourceLimit"
WHERE "resource" = 'STREAM_RETENTION'
    AND "limitValue" > 0
    AND "limitValue" < @maxRetentionHours::integer
UNION
SELECT DISTINCT t.tenant_id, @defaultRetentionHours::integer
FROM v1_stream_topic t
WHERE @defaultRetentionHours::integer < @maxRetentionHours::integer
    AND NOT EXISTS (
        SELECT 1 FROM "TenantResourceLimit" l
        WHERE l."tenantId" = t.tenant_id AND l."resource" = 'STREAM_RETENTION'
    );

-- name: DeleteExpiredStreamMessages :one
-- Small batches since payloads can be large. Uploaded payloads go with the
-- messages referencing them, one each; returns how many messages were deleted.
WITH deleted AS (
    DELETE FROM v1_stream_message
    WHERE inserted_at < @before::timestamptz
        AND (tenant_id, namespace, topic, id, inserted_at) IN (
            SELECT tenant_id, namespace, topic, id, inserted_at
            FROM v1_stream_message
            WHERE tenant_id = @tenantId::uuid
                AND inserted_at < @before::timestamptz
            LIMIT @batchSize::integer
        )
    RETURNING tenant_id, payload_id, payload_inserted_at
), deleted_payloads AS (
    DELETE FROM v1_stream_payload p
    USING deleted d
    WHERE p.tenant_id = d.tenant_id
        AND p.id = d.payload_id
        AND p.inserted_at = d.payload_inserted_at
)
SELECT COUNT(*)::bigint AS deleted_messages FROM deleted;

-- name: ListStreamMessagesAfterCursor :many
-- Pages by offset, which become visible in order (see ReserveStreamTopicOffsets),
-- so nothing can appear behind a cursor. A page ends once its payloads reach
-- @maxBytes, always keeping its first row; octet_length reads a TOASTed
-- payload's size without loading it.
SELECT id, inserted_at, tenant_id, namespace, topic, payload, producer_id, producer_seq, payload_id, payload_inserted_at
FROM (
    SELECT page.*, SUM(octet_length(page.payload)) OVER (ORDER BY page.id) - octet_length(page.payload) AS bytes_before
    FROM (
        SELECT *
        FROM v1_stream_message
        WHERE tenant_id = @tenantId::uuid
            AND namespace = @namespace::text
            AND topic = @topic::text
            AND id > @afterId::bigint
            -- the tenant's retention; also prunes partitions outside it
            AND inserted_at >= @retainedSince::timestamptz
        ORDER BY id ASC
        LIMIT sqlc.arg('limit')::integer
    ) AS page
) AS sized
WHERE bytes_before < @maxBytes::bigint
ORDER BY id ASC;

-- name: InsertStreamPayload :one
-- A payload uploaded ahead of the publish that references it. One no publish
-- references is left for its partition to be dropped.
INSERT INTO v1_stream_payload (id, tenant_id, payload)
VALUES (@id::uuid, @tenantId::uuid, @payload::bytea)
RETURNING inserted_at;

-- name: StreamPayloadExists :one
SELECT EXISTS (
    SELECT 1 FROM v1_stream_payload
    WHERE tenant_id = @tenantId::uuid AND id = @id::uuid AND inserted_at = @insertedAt::timestamptz
) AS exists;

-- name: CopyStreamPayload :one
-- Copies an upload into a new row, so it lands in the current partition and
-- lives as long as a message published now. No row when the original is gone.
INSERT INTO v1_stream_payload (id, tenant_id, payload)
SELECT @newId::uuid, tenant_id, payload
FROM v1_stream_payload
WHERE tenant_id = @tenantId::uuid
    AND id = @id::uuid
    AND inserted_at = @insertedAt::timestamptz
RETURNING inserted_at;

-- name: GetStreamPayload :one
-- inserted_at is the partition key, so this reads one partition.
SELECT payload
FROM v1_stream_payload
WHERE tenant_id = @tenantId::uuid
    AND id = @id::uuid
    AND inserted_at = @insertedAt::timestamptz
    AND inserted_at >= @retainedSince::timestamptz;

-- name: GetStreamTopicMetadata :one
-- No row when the topic doesn't exist. latest_id is 0 when no message is
-- retained, since offsets start at 1.
WITH topic AS (
    SELECT 1
    FROM v1_stream_topic
    WHERE tenant_id = @tenantId::uuid
        AND namespace = @namespace::text
        AND topic = @topic::text
), latest AS (
    SELECT id, inserted_at
    FROM v1_stream_message
    WHERE tenant_id = @tenantId::uuid
        AND namespace = @namespace::text
        AND topic = @topic::text
        AND inserted_at >= @retainedSince::timestamptz
    ORDER BY id DESC
    LIMIT 1
), retained AS (
    SELECT COUNT(*) AS message_count
    FROM v1_stream_message
    WHERE tenant_id = @tenantId::uuid
        AND namespace = @namespace::text
        AND topic = @topic::text
        AND inserted_at >= @retainedSince::timestamptz
)
SELECT
    COALESCE(latest.id, 0)::bigint AS latest_id,
    latest.inserted_at AS latest_inserted_at,
    retained.message_count::bigint AS message_count
FROM retained
LEFT JOIN latest ON true
WHERE EXISTS (SELECT 1 FROM topic);
