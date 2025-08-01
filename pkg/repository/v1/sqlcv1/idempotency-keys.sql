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

-- name: FillIdempotencyKey :one
WITH updated AS (
    UPDATE v1_idempotency_key
    SET
        is_filled = TRUE,
        updated_at = NOW()
    WHERE
        tenant_id = @tenantId::UUID
        AND key = @key::TEXT
        AND is_filled = FALSE
    RETURNING *
)

SELECT COUNT(*) > 0 AS successfully_filled
FROM updated;
