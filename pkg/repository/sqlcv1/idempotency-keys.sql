-- name: CleanUpExpiredIdempotencyKeys :exec
DELETE FROM v1_idempotency_key
WHERE
    tenant_id = @tenantId::UUID
    AND expires_at < NOW()
;

-- name: ClaimIdempotencyKeys :many
WITH inputs AS (
    SELECT
        UNNEST(@keys::TEXT[]) AS key,
        UNNEST(@expiresAts::TIMESTAMPTZ[]) AS expires_at,
        UNNEST(@claimedByExternalIds::UUID[]) AS claimed_by_external_id
), locked_existing_keys AS (
    SELECT *
    FROM v1_idempotency_key
    WHERE
        tenant_id = @tenantId::UUID
        AND key IN (
            SELECT key
            FROM inputs
        )
    FOR UPDATE SKIP LOCKED
), already_claimed_keys AS (
    SELECT *
    FROM locked_existing_keys
    WHERE expires_at > NOW()
), claimable_keys AS (
    SELECT *
    FROM locked_existing_keys
    WHERE expires_at <= NOW()
), claims AS (
    INSERT INTO v1_idempotency_key (tenant_id, key, expires_at, claimed_by_external_id)
    SELECT key, expires_at, @tenantId::UUID, claimed_by_external_id
    FROM inputs
    ON CONFLICT (tenant_id, key) DO UPDATE
    SET
        expires_at = CASE
            WHEN (v1_idempotency_key.tenant_id, v1_idempotency_key.key) IN (SELECT tenant_id, key FROM claimable_keys) THEN EXCLUDED.expires_at
            ELSE v1_idempotency_key.expires_at
        END,
        claimed_by_external_id = CASE
            WHEN (v1_idempotency_key.tenant_id, v1_idempotency_key.key) IN (SELECT tenant_id, key FROM claimable_keys) THEN EXCLUDED.claimed_by_external_id
            ELSE v1_idempotency_key.claimed_by_external_id
        END
    RETURNING *
)

SELECT
    i.key::TEXT AS key,
    c.expires_at::TIMESTAMPTZ AS expires_at,
    (c.claimed_by_external_id = i.claimed_by_external_id)::BOOLEAN AS was_successfully_claimed,
    c.claimed_by_external_id
FROM inputs i
LEFT JOIN claims c USING (key)
;
