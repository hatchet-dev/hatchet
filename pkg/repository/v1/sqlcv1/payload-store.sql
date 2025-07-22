-- name: ReadPayload :one
SELECT *
FROM v1_payload
WHERE
    tenant_id = @tenantId::UUID
    AND type = @type::v1_payload_type
    AND key = @key::TEXT
;

-- name: ReadPayloads :many
WITH inputs AS (
    SELECT
        UNNEST(@keys::TEXT[]) AS key,
        UNNEST(CAST(@types::TEXT[] AS v1_payload_type[])) AS type
)

SELECT *
FROM v1_payload
WHERE
    tenant_id = @tenantId::UUID
    AND (key, type) IN (
        SELECT key, type
        FROM inputs
    )
;

-- name: WritePayloads :exec
WITH inputs AS (
    SELECT
        UNNEST(@keys::TEXT[]) AS key,
        UNNEST(CAST(@types::TEXT[] AS v1_payload_type[])) AS type,
        UNNEST(@payloads::JSONB[]) AS payload
)
INSERT INTO v1_payload (
    tenant_id,
    key,
    type,
    value
)
SELECT
    @tenantId::UUID,
    i.key,
    i.type,
    i.payload
FROM
    inputs i
;