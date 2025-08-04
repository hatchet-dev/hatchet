-- name: CreateIdempotencyKey :exec
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

-- name: ClaimIdempotencyKeys :many
WITH inputs AS (
    SELECT
        UNNEST(@keys::TEXT[]) AS key,
        UNNEST(@claimedByExternalIds::UUID[]) AS claimed_by_external_id
), claims AS (
    UPDATE v1_idempotency_key k
    SET
        claimed_by_external_id = i.claimed_by_external_id,
        updated_at = NOW()
    FROM inputs i
    WHERE
        k.tenant_id = @tenantId::UUID
        AND k.key = i.key
        AND k.claimed_by_external_id IS NULL
    RETURNING k.key, k.claimed_by_external_id
)

SELECT
    key::TEXT AS key,
    key NOT IN (
        SELECT key
        FROM claims
    ) AS was_already_claimed
FROM inputs
;
