-- name: ReadPayloads :many
WITH inputs AS (
    SELECT
        UNNEST(@ids::BIGINT[]) AS id,
        UNNEST(@insertedAts::TIMESTAMPTZ[]) AS inserted_at,
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(CAST(@types::TEXT[] AS v1_payload_type[])) AS type
)

SELECT *
FROM v1_payload
WHERE (tenant_id, id, inserted_at, type) IN (
        SELECT tenant_id, id, inserted_at, type
        FROM inputs
    )
;

-- name: WritePayloads :exec
WITH inputs AS (
    SELECT DISTINCT
        UNNEST(@ids::BIGINT[]) AS id,
        UNNEST(@insertedAts::TIMESTAMPTZ[]) AS inserted_at,
        UNNEST(@externalIds::UUID[]) AS external_id,
        UNNEST(CAST(@types::TEXT[] AS v1_payload_type[])) AS type,
        UNNEST(CAST(@locations::TEXT[] AS v1_payload_location[])) AS location,
        UNNEST(@externalLocationKeys::TEXT[]) AS external_location_key,
        UNNEST(@inlineContents::JSONB[]) AS inline_content,
        UNNEST(@tenantIds::UUID[]) AS tenant_id
)

INSERT INTO v1_payload (
    tenant_id,
    id,
    inserted_at,
    external_id,
    type,
    location,
    external_location_key,
    inline_content
)
SELECT
    i.tenant_id,
    i.id,
    i.inserted_at,
    i.external_id,
    i.type,
    i.location,
    CASE WHEN i.external_location_key = '' OR i.location != 'EXTERNAL' THEN NULL ELSE i.external_location_key END,
    i.inline_content
FROM
    inputs i
ORDER BY i.tenant_id, i.inserted_at, i.id, i.type
ON CONFLICT (tenant_id, id, inserted_at, type)
DO UPDATE SET
    location = EXCLUDED.location,
    external_location_key = CASE WHEN EXCLUDED.external_location_key = '' OR EXCLUDED.location != 'EXTERNAL' THEN NULL ELSE EXCLUDED.external_location_key END,
    inline_content = EXCLUDED.inline_content,
    updated_at = NOW()
;

-- name: AnalyzeV1Payload :exec
ANALYZE v1_payload;

-- name: ListPaginatedPayloadsForOffload :many
WITH payloads AS (
    SELECT
        (p).*
    FROM list_paginated_payloads_for_offload(
        @partitionDate::DATE,
        @lastTenantId::UUID,
        @lastInsertedAt::TIMESTAMPTZ,
        @lastId::BIGINT,
        @lastType::v1_payload_type,
        @nextTenantId::UUID,
        @nextInsertedAt::TIMESTAMPTZ,
        @nextId::BIGINT,
        @nextType::v1_payload_type,
        @batchSize::INTEGER
    ) p
)
SELECT
    tenant_id::UUID,
    id::BIGINT,
    inserted_at::TIMESTAMPTZ,
    external_id::UUID,
    type::v1_payload_type,
    location::v1_payload_location,
    COALESCE(external_location_key, '')::TEXT AS external_location_key,
    inline_content::JSONB AS inline_content,
    updated_at::TIMESTAMPTZ
FROM payloads;

-- name: CreatePayloadRangeChunks :many
WITH chunks AS (
    SELECT
        (p).*
    FROM create_payload_offload_range_chunks(
        @partitionDate::DATE,
        @windowSize::INTEGER,
        @chunkSize::INTEGER,
        @lastTenantId::UUID,
        @lastInsertedAt::TIMESTAMPTZ,
        @lastId::BIGINT,
        @lastType::v1_payload_type
    ) p
)

SELECT
    lower_tenant_id::UUID,
    lower_id::BIGINT,
    lower_inserted_at::TIMESTAMPTZ,
    lower_type::v1_payload_type,
    upper_tenant_id::UUID,
    upper_id::BIGINT,
    upper_inserted_at::TIMESTAMPTZ,
    upper_type::v1_payload_type
FROM chunks
;

-- name: CreateV1PayloadCutoverTemporaryTable :exec
SELECT copy_v1_payload_partition_structure(@date::DATE);

-- name: SwapV1PayloadPartitionWithTemp :exec
SELECT swap_v1_payload_partition_with_temp(@date::DATE);

-- name: AcquireOrExtendCutoverJobLease :one
WITH inputs AS (
    SELECT
        @key::DATE AS key,
        @leaseProcessId::UUID AS lease_process_id,
        @leaseExpiresAt::TIMESTAMPTZ AS lease_expires_at,
        @lastTenantId::UUID AS last_tenant_id,
        @lastInsertedAt::TIMESTAMPTZ AS last_inserted_at,
        @lastId::BIGINT AS last_id,
        @lastType::v1_payload_type AS last_type
), any_lease_held_by_other_process AS (
    -- need coalesce here in case there are no rows that don't belong to this process
    SELECT COALESCE(BOOL_OR(lease_expires_at > NOW()), FALSE) AS lease_exists
    FROM v1_payload_cutover_job_offset
    WHERE lease_process_id != @leaseProcessId::UUID
), to_insert AS (
    SELECT *
    FROM inputs
    -- if a lease is held by another process, we shouldn't try to insert a new row regardless
    -- of which key we're trying to acquire a lease on
    WHERE NOT (SELECT lease_exists FROM any_lease_held_by_other_process)
)

INSERT INTO v1_payload_cutover_job_offset (key, lease_process_id, lease_expires_at, last_tenant_id, last_inserted_at, last_id, last_type)
SELECT ti.key, ti.lease_process_id, ti.lease_expires_at, ti.last_tenant_id, ti.last_inserted_at, ti.last_id, ti.last_type
FROM to_insert ti
ON CONFLICT (key)
DO UPDATE SET
    -- if the lease is held by this process, then we extend the offset to the new tuple of (last_tenant_id, last_inserted_at, last_id, last_type)
    -- otherwise it's a new process acquiring the lease, so we should keep the offset where it was before
    last_tenant_id = CASE
        WHEN EXCLUDED.lease_process_id = v1_payload_cutover_job_offset.lease_process_id THEN EXCLUDED.last_tenant_id
        ELSE v1_payload_cutover_job_offset.last_tenant_id
    END,
    last_inserted_at = CASE
        WHEN EXCLUDED.lease_process_id = v1_payload_cutover_job_offset.lease_process_id THEN EXCLUDED.last_inserted_at
        ELSE v1_payload_cutover_job_offset.last_inserted_at
    END,
    last_id = CASE
        WHEN EXCLUDED.lease_process_id = v1_payload_cutover_job_offset.lease_process_id THEN EXCLUDED.last_id
        ELSE v1_payload_cutover_job_offset.last_id
    END,
    last_type = CASE
        WHEN EXCLUDED.lease_process_id = v1_payload_cutover_job_offset.lease_process_id THEN EXCLUDED.last_type
        ELSE v1_payload_cutover_job_offset.last_type
    END,

    lease_process_id = EXCLUDED.lease_process_id,
    lease_expires_at = EXCLUDED.lease_expires_at
WHERE v1_payload_cutover_job_offset.lease_expires_at < NOW() OR v1_payload_cutover_job_offset.lease_process_id = @leaseProcessId::UUID
RETURNING *
;

-- name: MarkCutoverJobAsCompleted :exec
UPDATE v1_payload_cutover_job_offset
SET is_completed = TRUE
WHERE key = @key::DATE
;

-- name: CleanUpCutoverJobOffsets :exec
DELETE FROM v1_payload_cutover_job_offset
WHERE NOT key = ANY(@keysToKeep::DATE[])
;

-- name: DiffPayloadSourceAndTargetPartitions :many
WITH payloads AS (
    SELECT
        (p).*
    FROM diff_payload_source_and_target_partitions(@partitionDate::DATE) p
)

SELECT
    tenant_id::UUID,
    id::BIGINT,
    inserted_at::TIMESTAMPTZ,
    external_id::UUID,
    type::v1_payload_type,
    location::v1_payload_location,
    COALESCE(external_location_key, '')::TEXT AS external_location_key,
    inline_content::JSONB AS inline_content,
    updated_at::TIMESTAMPTZ
FROM payloads
;

-- name: ComputePayloadBatchSize :one
SELECT compute_payload_batch_size(
    @partitionDate::DATE,
    @lastTenantId::UUID,
    @lastInsertedAt::TIMESTAMPTZ,
    @lastId::BIGINT,
    @lastType::v1_payload_type,
    @batchSize::INTEGER
) AS total_size_bytes;
