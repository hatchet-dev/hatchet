-- name: UpsertStreamTopic :one
-- Implicitly creates a topic on first publish. The `inserted` column lets the
-- caller distinguish a brand-new topic (subject to a per-tenant topic-count
-- limit check) from a publish to an already-existing topic.
INSERT INTO v1_stream_topic (tenant_id, namespace, topic)
VALUES (@tenantId::uuid, @namespace::text, @topic::text)
ON CONFLICT (tenant_id, namespace, topic) DO UPDATE
SET last_published_at = NOW()
RETURNING sqlc.embed(v1_stream_topic), (xmax = 0) AS inserted;

-- name: DeleteStreamTopic :exec
-- Used to roll back UpsertStreamTopic when a newly-created topic turns out to
-- violate the tenant's topic-count limit.
DELETE FROM v1_stream_topic
WHERE tenant_id = @tenantId::uuid AND namespace = @namespace::text AND topic = @topic::text;

-- name: DeleteIdleStreamTopics :execrows
-- Removes topics nobody has published to within their tenant's retention, in
-- batches so one sweep never holds a long lock. last_published_at is
-- refreshed at least every streamTopicSeenCache TTL while a topic is in use.
DELETE FROM v1_stream_topic
WHERE (tenant_id, namespace, topic) IN (
    SELECT t.tenant_id, t.namespace, t.topic
    FROM v1_stream_topic t
    LEFT JOIN "TenantResourceLimit" l ON l."tenantId" = t.tenant_id AND l."resource" = 'STREAM_RETENTION'
    WHERE t.last_published_at < NOW() - make_interval(hours => LEAST(COALESCE(l."limitValue", @defaultRetentionHours::integer), @maxRetentionHours::integer))
    LIMIT @batchSize::integer
);

-- name: ListStreamProducerCursorPartitionsBeforeDate :many
-- Kept separate from ListPartitionsBeforeDate because producer cursors have
-- their own retention.
SELECT
    'v1_stream_producer_cursor' AS parent_table,
    p::text AS partition_name
FROM get_v1_partitions_before_date('v1_stream_producer_cursor', @date::date) AS p;

-- name: CountStreamTopics :one
SELECT COUNT(*) FROM v1_stream_topic WHERE tenant_id = @tenantId::uuid;

-- name: ReserveStreamTopicOffsets :batchone
-- Reserves @count consecutive offsets for a topic and returns the last. The
-- row lock it takes is held until the publishing transaction commits, so a
-- concurrent publisher to the topic waits, then takes strictly later offsets
-- and commits after it. Re-creates a topic the idle sweep removed; its
-- offsets restart, which is safe because every message it held had already
-- passed retention.
INSERT INTO v1_stream_topic (tenant_id, namespace, topic, last_offset)
VALUES (@tenantId::uuid, @namespace::text, @topic::text, @count::bigint)
ON CONFLICT (tenant_id, namespace, topic) DO UPDATE
SET last_offset = v1_stream_topic.last_offset + EXCLUDED.last_offset, last_published_at = NOW()
RETURNING last_offset;

-- name: InsertOrderedStreamMessage :batchone
-- Atomically advances the producer's watermark and inserts the message, but
-- only if producer_seq is exactly one past the producer's last durably
-- applied sequence. The watermark is the producer's row in its latest
-- bucket; the compare-and-swap runs against that one row, so concurrent
-- attempts serialize on its row lock even across a UTC day boundary. The
-- first write of a new day also copies the watermark into today's bucket so
-- that an active producer's cursor outlives partition retention. Only buckets
-- from @minBucket on are read, so planning and locking touch just the
-- partitions within cursor retention; an older watermark counts as none.
-- inserted=false means this message was NOT applied; current_last_seq (the
-- watermark as of this call, NULL if the producer has no row) tells the
-- caller whether that's a gap worth retrying (current_last_seq <
-- producer_seq - 1) or a stale redelivery (current_last_seq >= producer_seq).
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
    INSERT INTO v1_stream_message (id, tenant_id, namespace, topic, payload, producer_id, producer_seq)
    -- the offset reserved for this message by ReserveStreamTopicOffsets
    SELECT @messageOffset::bigint, @tenantId::uuid, @namespace::text, @topic::text, @payload::bytea, @producerId::text, @producerSeq::bigint
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
-- The start of the tenant's oldest hourly partition: its rows inserted before
-- it have been dropped. NULL if the tenant has no partitions.
SELECT get_v1_stream_message_retention_start(@tenantId::uuid)::timestamptz AS retention_start;

-- name: EnsureStreamMessagePartition :exec
-- Creates the tenant's partitions for this hour and the next, for an insert
-- that found none.
SELECT
    ensure_v1_stream_message_partition(@tenantId::uuid, NOW()),
    ensure_v1_stream_message_partition(@tenantId::uuid, NOW() + INTERVAL '1 hour');

-- name: EnsureActiveStreamMessagePartitions :exec
-- Pre-creates this hour's and next hour's partitions for tenants that
-- published recently, so their inserts rarely create one inline.
SELECT ensure_v1_stream_message_partition(t.tenant_id, NOW() + hours.offset_hours * INTERVAL '1 hour')
FROM (
    SELECT DISTINCT tenant_id
    FROM v1_stream_topic
    WHERE last_published_at > NOW() - INTERVAL '2 hours'
) AS t
CROSS JOIN generate_series(0, 1) AS hours(offset_hours);

-- name: ListStreamMessageHourPartitions :many
SELECT
    p.parent_table::text AS parent_table,
    p.partition_name::text AS partition_name,
    p.tenant_id::uuid AS tenant_id,
    p.hour_start::timestamptz AS hour_start
FROM list_v1_stream_message_hour_partitions() AS p;

-- name: ListDroppableStreamMessageTenantPartitions :many
-- Tenant partitions with no hourly partitions left and no topics, so nothing
-- will write to them again before a new topic re-creates them.
SELECT p.partition_name::text AS partition_name
FROM list_v1_stream_message_empty_tenant_partitions() AS p
WHERE NOT EXISTS (SELECT 1 FROM v1_stream_topic t WHERE t.tenant_id = p.tenant_id);

-- name: ListStreamMessagesAfterCursor :many
-- Keyset pagination on id, a topic's offset: offsets become visible strictly
-- in order (see ReserveStreamTopicOffsets), so nothing after the cursor can
-- appear later behind it. A page also stops once its payloads reach @maxBytes (always keeping its
-- first row). Payloads over ~2KB are stored out of line, and octet_length
-- reads their size without fetching them, so only the rows returned have
-- their payloads loaded.
SELECT id, inserted_at, tenant_id, namespace, topic, payload, producer_id, producer_seq
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
