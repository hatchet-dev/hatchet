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

-- name: AcquireLeases :many
-- Attempts to acquire leases for a set of resources. Returns the acquired leases.
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

-- name: RenewLeases :many
-- Renews a set of leases by their IDs. Returns the renewed leases.
UPDATE "Lease" l
SET
    "expiresAt" = now() + COALESCE(sqlc.narg('leaseDuration')::interval, '30 seconds'::interval)
FROM (
    SELECT
        unnest(@leaseIds::bigint[]) AS "id"
    ) AS input
WHERE
    l."id" = input."id"
RETURNING l.*;

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
