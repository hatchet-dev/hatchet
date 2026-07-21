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
), inputs_with_rn AS (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY key ORDER BY expires_at DESC) AS rn
    FROM inputs
), deduplicated_potential_claims AS (
    SELECT *
    FROM inputs_with_rn
    WHERE rn = 1
), locked_existing_keys AS (
    SELECT *
    FROM v1_idempotency_key
    WHERE
        tenant_id = @tenantId::UUID
        AND key IN (
            SELECT key
            FROM deduplicated_potential_claims
        )
    ORDER BY tenant_id, expires_at, key
    FOR UPDATE
), claimable_keys AS (
    SELECT *
    FROM locked_existing_keys
    WHERE expires_at <= NOW()
), claims AS (
    INSERT INTO v1_idempotency_key (key, expires_at, tenant_id, claimed_by_external_id)
    SELECT key, expires_at, @tenantId::UUID, claimed_by_external_id
    FROM deduplicated_potential_claims
    ON CONFLICT (tenant_id, key) DO UPDATE
    SET
        expires_at = CASE
            WHEN (v1_idempotency_key.tenant_id, v1_idempotency_key.key) IN (SELECT tenant_id, key FROM claimable_keys) THEN EXCLUDED.expires_at
            ELSE v1_idempotency_key.expires_at
        END,
        claimed_by_external_id = CASE
            WHEN (v1_idempotency_key.tenant_id, v1_idempotency_key.key) IN (SELECT tenant_id, key FROM claimable_keys) THEN EXCLUDED.claimed_by_external_id
            ELSE v1_idempotency_key.claimed_by_external_id
        END,
        updated_at = CASE
            WHEN (v1_idempotency_key.tenant_id, v1_idempotency_key.key) IN (SELECT tenant_id, key FROM claimable_keys) THEN NOW()
            ELSE v1_idempotency_key.updated_at
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

-- name: ReleaseIdempotencyKeys :exec
-- !! IMPORTANT: this only gets called when a task reaches a terminal state (exhausted all retries, completed, etc.)
-- which means we want to evict any idempotency keys that still are live and tied to the task at this point
WITH input AS (
    SELECT
        UNNEST(@taskIds::BIGINT[]) AS task_id,
        UNNEST(@taskInsertedAts::TIMESTAMPTZ[]) AS task_inserted_at,
        UNNEST(@taskRetryCounts::INTEGER[]) AS retry_count
), relevant_tasks AS (
    SELECT t.tenant_id, t.idempotency_key
	FROM v1_task t
	JOIN "WorkflowVersion" wv ON t.workflow_version_id = wv.id
	JOIN "Step" s ON t.step_id = s.id
	WHERE
        (t.id, t.inserted_at, t.retry_count) IN (SELECT task_id, task_inserted_at, retry_count FROM input)
		AND t.idempotency_key IS NOT NULL
		AND wv."idempotencyMethod" = 'STATUS'
), keys_to_release AS (
    SELECT *
	FROM v1_idempotency_key
	WHERE (tenant_id, key) IN (
		SELECT tenant_id, idempotency_key
		FROM relevant_tasks
	)
	ORDER BY tenant_id, key
	FOR UPDATE
)

DELETE FROM v1_idempotency_key k
WHERE (tenant_id, key) IN (
	SELECT tenant_id, key
	FROM keys_to_release
);