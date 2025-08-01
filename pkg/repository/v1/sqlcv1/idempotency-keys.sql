-- name: CreateIdempotencyKey :one
INSERT INTO v1_idempotency_key (
    tenant_id,
    key,
    expires_at
)
VALUES (
    @tenantId::UUID,
    @key::TEXT,
    @expiresAt::TIMESTAMPTZ
)
RETURNING *;

-- name: MarkIdempotencyKeyFilled :exec
UPDATE v1_idempotency_key
SET is_filled = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE tenant_id = @tenantId::UUID
  AND key = @key::TEXT
;

-- name: CheckIfIdempotencyKeyFilled :one
SELECT EXISTS (
    SELECT 1
    FROM v1_idempotency_key
    WHERE tenant_id = @tenantId::UUID
      AND key = @key::TEXT
      AND is_filled = TRUE
)::BOOLEAN AS is_filled;
