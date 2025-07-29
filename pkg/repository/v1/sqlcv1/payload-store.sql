-- name: ReadPayload :one
SELECT *
FROM v1_payload
WHERE
    tenant_id = @tenantId::UUID
    AND type = @type::v1_payload_type
    AND id = @id::BIGINT
    AND inserted_at = @insertedAt::TIMESTAMPTZ
;

-- name: ReadPayloads :many
WITH inputs AS (
    SELECT
        UNNEST(@ids::BIGINT[]) AS id,
        UNNEST(@insertedAts::TIMESTAMPTZ[]) AS inserted_at,
        UNNEST(CAST(@types::TEXT[] AS v1_payload_type[])) AS type
)

SELECT *
FROM v1_payload
WHERE
    tenant_id = @tenantId::UUID
    AND (id, inserted_at, type) IN (
        SELECT id, inserted_at, type
        FROM inputs
    )
;

-- name: WritePayloads :exec
WITH inputs AS (
    SELECT
        UNNEST(@ids::BIGINT[]) AS id,
        UNNEST(@insertedAts::TIMESTAMPTZ[]) AS inserted_at,
        UNNEST(CAST(@types::TEXT[] AS v1_payload_type[])) AS type,
        UNNEST(@payloads::JSONB[]) AS payload
)
INSERT INTO v1_payload (
    tenant_id,
    id,
    inserted_at,
    type,
    value
)
SELECT
    @tenantId::UUID,
    i.id,
    i.inserted_at,
    i.type,
    i.payload
FROM
    inputs i
;

-- name: OffloadPayloadsToExternalStore :exec
WITH inputs AS (
    SELECT
        UNNEST(@ids::BIGINT[]) AS id,
        UNNEST(@insertedAts::TIMESTAMPTZ[]) AS inserted_at,
        UNNEST(@values::JSONB[]) AS value
)

UPDATE v1_payload
SET
    value = i.value,
    updated_at = NOW()
FROM inputs i
WHERE
    v1_payload.tenant_id = @tenantId::UUID
    AND v1_payload.id = i.id
    AND v1_payload.inserted_at = i.inserted_at
;

-- name: WritePayloadWAL :exec
WITH inputs AS (
    SELECT
        UNNEST(@payloadIds::BIGINT[]) AS payload_id,
        UNNEST(@payloadInsertedAts::TIMESTAMPTZ[]) AS payload_inserted_at,
        UNNEST(CAST(@payloadTypes::TEXT[] AS v1_payload_type[])) AS payload_type,
        UNNEST(@offloadAts::TIMESTAMPTZ[]) AS offload_at,
        UNNEST(CAST(@operations::TEXT[] AS v1_payload_wal_operation[])) AS operation
)

INSERT INTO v1_payload_wal (
    tenant_id,
    offload_at,
    payload_id,
    payload_inserted_at,
    payload_type,
    operation
)
SELECT
    @tenantId::UUID,
    i.offload_at,
    i.payload_id,
    i.payload_inserted_at,
    i.payload_type,
    i.operation
FROM
    inputs i
;

-- name: PollPayloadWALForRecordsToOffload :many
SELECT *
FROM v1_payload_wal
WHERE
    tenant_id = @tenantId::UUID
    AND offload_at <= NOW()
ORDER BY offload_at, payload_id, payload_inserted_at, payload_type
LIMIT @pollLimit::INT
FOR UPDATE SKIP LOCKED
;

-- name: EvictPayloadWALRecords :exec
WITH inputs AS (
    SELECT
        UNNEST(@offloadAts::TIMESTAMPTZ[]) AS offload_at,
        UNNEST(@payloadIds::BIGINT[]) AS payload_id,
        UNNEST(@payloadInsertedAts::TIMESTAMPTZ[]) AS payload_inserted_at,
        UNNEST(CAST(@payloadTypes::TEXT[] AS v1_payload_type[])) AS payload_type
)
DELETE FROM v1_payload_wal
WHERE
    tenant_id = @tenantId::UUID
    AND (offload_at, payload_id, payload_inserted_at, payload_type) IN (
        SELECT offload_at, payload_id, payload_inserted_at, payload_type
        FROM inputs
    )
;