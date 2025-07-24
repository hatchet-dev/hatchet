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
