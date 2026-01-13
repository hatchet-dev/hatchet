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
);

-- name: CleanUpExpiredIdempotencyKeys :exec
DELETE FROM v1_idempotency_key
WHERE
    tenant_id = @tenantId::UUID
    AND expires_at < NOW()
;

-- name: ClaimIdempotencyKeys :many
WITH inputs AS (
    SELECT DISTINCT
        UNNEST(@keys::TEXT[]) AS key,
        UNNEST(@claimedByExternalIds::UUID[]) AS claimed_by_external_id
), incoming_claims AS (
    SELECT
        *,
        ROW_NUMBER() OVER(PARTITION BY key ORDER BY claimed_by_external_id) AS claim_index
    FROM inputs
), candidate_keys AS (
    -- Grab all of the keys that are attempting to be claimed
    SELECT
        tenant_id,
        expires_at,
        key,
        ROW_NUMBER() OVER(PARTITION BY tenant_id, key ORDER BY expires_at) AS key_index
    FROM v1_idempotency_key
    WHERE
        tenant_id = @tenantId::UUID
        AND key IN (
            SELECT key
            FROM incoming_claims
        )
        AND claimed_by_external_id IS NULL
        AND expires_at > NOW()
), to_update AS (
    SELECT
        ck.tenant_id,
        ck.expires_at,
        ck.key,
        ic.claimed_by_external_id
    FROM candidate_keys ck
    JOIN incoming_claims ic ON (ck.key, ck.key_index) = (ic.key, ic.claim_index)
    WHERE ck.tenant_id = @tenantId::UUID
    FOR UPDATE SKIP LOCKED
), claims AS (
    UPDATE v1_idempotency_key k
    SET
        claimed_by_external_id = u.claimed_by_external_id,
        updated_at = NOW()
    FROM to_update u
    WHERE (u.tenant_id, u.expires_at, u.key) = (k.tenant_id, k.expires_at, k.key)
    RETURNING k.*
)

SELECT
    i.key::TEXT AS key,
    c.expires_at::TIMESTAMPTZ AS expires_at,
    c.claimed_by_external_id IS NOT NULL::BOOLEAN AS was_successfully_claimed,
    c.claimed_by_external_id
FROM inputs i
LEFT JOIN claims c ON (i.key = c.key AND i.claimed_by_external_id = c.claimed_by_external_id)
;
