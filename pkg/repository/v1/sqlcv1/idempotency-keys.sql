-- name: CreateIdempotencyKey :one
WITH upserted AS (
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
    ON CONFLICT (tenant_id, key, expires_at) DO NOTHING
    RETURNING 1
)

SELECT NOT EXISTS (
    SELECT 1
    FROM upserted
) AS already_existed;