-- name: ListIntervalsByOperationId :many
SELECT
    *
FROM
    v1_operation_interval_settings
WHERE
    operation_id = @operationId::text;

-- name: UpsertInterval :one
INSERT INTO v1_operation_interval_settings (
    tenant_id,
    operation_id,
    interval_nanoseconds
) VALUES (
    @tenantId::uuid,
    @operationId::text,
    @intervalNanoseconds::bigint
) ON CONFLICT (tenant_id, operation_id) DO UPDATE
SET
    interval_nanoseconds = EXCLUDED.interval_nanoseconds
RETURNING
    *;

-- name: ReadInterval :one
SELECT
    *
FROM
    v1_operation_interval_settings
WHERE
    tenant_id = @tenantId::uuid
    AND operation_id = @operationId::text
LIMIT 1;
