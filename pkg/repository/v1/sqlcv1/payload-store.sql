-- name: ReadPayload :one
SELECT *
FROM v1_payload
WHERE
    tenant_id = @tenantId::UUID
    AND type = @type::v1_payload_type
    AND key = @key::TEXT
;

-- name: WritePayloads :exec
WITH inputs AS (
    SELECT
        UNNEST(@keys::TEXT[]) AS key,
        UNNEST(@types::v1_payload_type[]) AS type,
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