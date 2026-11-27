-- name: UpdateTenantBacklogHolds :many
-- Writes one backlog measurement pass. Each tenant in @heldTenantIds is held for two more minutes
-- with its measured backlog, overwriting an expired hold; each tenant in @releasedTenantIds loses
-- its hold. The two lists are disjoint. Holds are written in tenant_id order, so two schedulers
-- that measure the same tenants lock their rows in the same order. Expired holds of other tenants
-- are left alone: readers ignore them. Returns the tenants whose hold started (held is true) or
-- ended (held is false) in this statement; every CTE reads the table as it was before it.
WITH input AS (
    SELECT
        UNNEST(@heldTenantIds::UUID[]) AS tenant_id,
        UNNEST(@backlogs::BIGINT[]) AS backlog,
        UNNEST(@largestStates::TEXT[]) AS largest_state
), already_held AS (
    SELECT tenant_id
    FROM v1_tenant_backlog_hold
    WHERE tenant_id = ANY(@heldTenantIds::UUID[]) AND held_until > NOW()
), held AS (
    INSERT INTO v1_tenant_backlog_hold (tenant_id, backlog, backlog_limit, largest_state, held_until)
    SELECT tenant_id, backlog, @backlogLimit::BIGINT, largest_state, NOW() + INTERVAL '2 minutes'
    FROM input
    ORDER BY tenant_id
    ON CONFLICT (tenant_id) DO UPDATE SET
        backlog = EXCLUDED.backlog,
        backlog_limit = EXCLUDED.backlog_limit,
        largest_state = EXCLUDED.largest_state,
        held_until = EXCLUDED.held_until
    RETURNING tenant_id, backlog, backlog_limit, largest_state
), released AS (
    DELETE FROM v1_tenant_backlog_hold
    WHERE tenant_id = ANY(@releasedTenantIds::UUID[])
    RETURNING tenant_id, backlog, backlog_limit, largest_state
)
SELECT tenant_id, backlog, backlog_limit, largest_state, TRUE AS held
FROM held
WHERE tenant_id NOT IN (SELECT tenant_id FROM already_held)
UNION ALL
SELECT tenant_id, backlog, backlog_limit, largest_state, FALSE AS held
FROM released;

-- name: ListTenantBacklogHolds :many
SELECT *
FROM v1_tenant_backlog_hold
WHERE held_until > NOW();
