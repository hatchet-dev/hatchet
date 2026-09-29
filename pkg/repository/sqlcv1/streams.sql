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
WHERE id = @id::bigint;

-- name: CountStreamTopics :one
SELECT COUNT(*) FROM v1_stream_topic WHERE tenant_id = @tenantId::uuid;

-- name: InsertOrderedStreamMessage :one
-- Atomically advances v1_stream_producer_cursor and inserts the message, but
-- only if producer_seq is exactly one past the producer's last durably
-- applied sequence -- the ON CONFLICT ... WHERE clause is a compare-and-swap
-- that also handles the very first message (no conflict, plain insert, but
-- only for producer_seq 0 so a reordered seq>0 can't claim the first slot).
-- inserted=false means this message was NOT applied; current_last_seq (the
-- watermark as of this call) tells the caller whether that's a gap worth
-- retrying (current_last_seq < producer_seq - 1) or a stale redelivery of an
-- already-applied message (current_last_seq >= producer_seq).
WITH advanced AS (
    INSERT INTO v1_stream_producer_cursor (tenant_id, namespace, topic, producer_id, last_seq)
    SELECT @tenantId::uuid, @namespace::text, @topic::text, @producerId::text, @producerSeq::bigint
    WHERE @producerSeq::bigint = 0 OR EXISTS (
        SELECT 1 FROM v1_stream_producer_cursor
        WHERE tenant_id = @tenantId::uuid AND namespace = @namespace::text AND topic = @topic::text AND producer_id = @producerId::text
    )
    ON CONFLICT (tenant_id, namespace, topic, producer_id) DO UPDATE
    SET last_seq = @producerSeq::bigint
    -- computed by the caller as producerSeq - 1, rather than written as an
    -- expression here, so sqlc binds it as one plain parameter
    WHERE v1_stream_producer_cursor.last_seq = @expectedPrevSeq::bigint
    RETURNING 1
), inserted_row AS (
    INSERT INTO v1_stream_message (tenant_id, namespace, topic, payload, producer_id, producer_seq)
    SELECT @tenantId::uuid, @namespace::text, @topic::text, @payload::bytea, @producerId::text, @producerSeq::bigint
    WHERE EXISTS (SELECT 1 FROM advanced)
    RETURNING 1
)
-- current_last_seq is read via a LEFT JOIN, not a scalar subquery against
-- v1_stream_producer_cursor directly: per Postgres's WITH-clause semantics, a
-- plain subquery in the same statement as advanced cannot see the row
-- advanced just inserted (only ever-so-slightly stale reads are visible to
-- "other parts of the query"), which made this column spuriously NULL --
-- and therefore unscannable into a NOT NULL Go field -- on every producer's
-- very first message. The LEFT JOIN sees the same pre-statement snapshot but
-- sqlc correctly infers it as nullable, and NULL here is meaningful anyway:
-- it means this producer has never had a row at all.
SELECT
    EXISTS (SELECT 1 FROM inserted_row) AS inserted,
    cur.last_seq AS current_last_seq
FROM (SELECT 1) AS one
LEFT JOIN v1_stream_producer_cursor cur
    ON cur.tenant_id = @tenantId::uuid AND cur.namespace = @namespace::text AND cur.topic = @topic::text AND cur.producer_id = @producerId::text;

-- name: ForceInsertOrderedStreamMessage :exec
-- Used only once InsertOrderedStreamMessage has been unable to close a gap
-- for too long (see maxProducerGapWait in the streams controller): inserts
-- unconditionally and bumps the watermark forward with GREATEST so it never
-- regresses, accepting an out-of-order delivery rather than blocking this
-- producer's topic forever on a message that never arrived.
WITH bumped AS (
    INSERT INTO v1_stream_producer_cursor (tenant_id, namespace, topic, producer_id, last_seq)
    VALUES (@tenantId::uuid, @namespace::text, @topic::text, @producerId::text, @producerSeq::bigint)
    ON CONFLICT (tenant_id, namespace, topic, producer_id) DO UPDATE
    SET last_seq = GREATEST(v1_stream_producer_cursor.last_seq, @producerSeq::bigint)
)
INSERT INTO v1_stream_message (tenant_id, namespace, topic, payload, producer_id, producer_seq)
VALUES (@tenantId::uuid, @namespace::text, @topic::text, @payload::bytea, @producerId::text, @producerSeq::bigint);

-- name: ListStreamMessagesAfterCursor :many
-- Keyset pagination on id. xact_id < pg_snapshot_xmin(...) excludes rows
-- whose inserting transaction may still be in flight, so a concurrent batch
-- insert can't let a higher id become visible before a lower one commits.
SELECT *
FROM v1_stream_message
WHERE tenant_id = @tenantId::uuid
    AND namespace = @namespace::text
    AND topic = @topic::text
    AND id > @afterId::bigint
    AND xact_id < pg_snapshot_xmin(pg_current_snapshot())
ORDER BY id ASC
LIMIT sqlc.arg('limit')::integer;
