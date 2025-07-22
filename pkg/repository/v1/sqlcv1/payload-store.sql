-- name: ReadPayload :one
SELECT *
FROM v1_payload
WHERE
    tenant_id = @tenantId::UUID
    AND type = @type::v1_payload_type
    AND key = @key::TEXT
;

-- name: WritePayload :exec
INSERT INTO v1_payload (
    tenant_id,
    key,
    type,
    value
)
VALUES (
    @tenantId::UUID,
    @key::TEXT,
    @type::v1_payload_type,
    @payload::JSONB
)
;