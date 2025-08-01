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

-- name: CleanUpExpiredIdempotencyKeys :exec
DELETE FROM v1_idempotency_key
WHERE
    tenant_id = ANY(@tenantIds::UUID[])
    AND expires_at < (NOW() - INTERVAL '1 minute')
;

-- name: ClaimIdempotencyKey :one
WITH claim AS (
    UPDATE v1_idempotency_key
    SET
        claimed_by_external_id = @claimedByExternalId::UUID,
        updated_at = NOW()
    WHERE
        tenant_id = @tenantId::UUID
        AND key = @key::TEXT
        AND claimed_by_external_id IS NULL
    RETURNING 1
)

SELECT NOT EXISTS (
    SELECT 1
    FROM claim
) AS already_claimed
;
