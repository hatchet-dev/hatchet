-- name: ReadPayload :one
SELECT *
FROM v1_payload
WHERE
    tenant_id = @tenantId::UUID
    AND key = @key::TEXT
;

-- name: WritePayload :exec
INSERT INTO v1_payload (
    tenant_id,
    key,
    value
)
VALUES (
    @tenantId::UUID,
    @key::TEXT,
    @payload::JSONB
)
;