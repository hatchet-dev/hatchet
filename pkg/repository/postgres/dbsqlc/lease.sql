-- name: GetLeasesToAcquire :exec
SELECT
    *
FROM
    "Lease"
WHERE
    "tenantId" = @tenantId::uuid
    AND "kind" = @kind::"LeaseKind"
    AND "expiresAt" < now()
    AND "resourceId" = ANY(@resourceIds::text[])
FOR UPDATE;

-- name: AcquireOrExtendLeases :many
-- Attempts to acquire leases for a set of resources, and extends the leases if we already have them.
-- Returns the acquired leases.
INSERT INTO "Lease" (
    "expiresAt",
    "tenantId",
    "resourceId",
    "kind"
)
SELECT
    now() + COALESCE(sqlc.narg('leaseDuration')::interval, '30 seconds'::interval),
    @tenantId::uuid,
    input."resourceId",
    @kind::"LeaseKind"
FROM (
    SELECT
        unnest(@resourceIds::text[]) AS "resourceId"
    ) AS input
-- On conflict, acquire the lease if the existing lease has expired.
ON CONFLICT ("tenantId", "kind", "resourceId") DO UPDATE
SET
    "expiresAt" = EXCLUDED."expiresAt"
WHERE
    "Lease"."expiresAt" < now() OR
    "Lease"."id" = ANY(@existingLeaseIds::bigint[])
RETURNING *;

-- name: ReleaseLeases :many
-- Releases a set of leases by their IDs. Returns the released leases.
DELETE FROM "Lease" l
USING (
    SELECT
        unnest(@leaseIds::bigint[]) AS "id"
    ) AS input
WHERE
    l."id" = input."id"
RETURNING l.*;
