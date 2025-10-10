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
    type,
    location,
    external_location_key,
    inline_content
)
SELECT
    i.tenant_id,
    i.id,
    i.inserted_at,
    i.type,
    i.location,
    CASE WHEN i.external_location_key = '' OR i.location != 'EXTERNAL' THEN NULL ELSE i.external_location_key END,
    i.inline_content
FROM
    inputs i
ON CONFLICT (tenant_id, id, inserted_at, type)
DO UPDATE SET
    location = EXCLUDED.location,
    external_location_key = CASE WHEN EXCLUDED.external_location_key = '' OR EXCLUDED.location != 'EXTERNAL' THEN NULL ELSE EXCLUDED.external_location_key END,
    inline_content = EXCLUDED.inline_content,
    updated_at = NOW()
;


-- name: WritePayloadWAL :exec
WITH inputs AS (
    SELECT
        UNNEST(@payloadIds::BIGINT[]) AS payload_id,
        UNNEST(@payloadInsertedAts::TIMESTAMPTZ[]) AS payload_inserted_at,
        UNNEST(CAST(@payloadTypes::TEXT[] AS v1_payload_type[])) AS payload_type,
        UNNEST(@offloadAts::TIMESTAMPTZ[]) AS offload_at,
        UNNEST(@tenantIds::UUID[]) AS tenant_id
)

INSERT INTO v1_payload_wal (
    tenant_id,
    offload_at,
    payload_id,
    payload_inserted_at,
    payload_type
)
SELECT
    i.tenant_id,
    i.offload_at,
    i.payload_id,
    i.payload_inserted_at,
    i.payload_type
FROM
    inputs i
ON CONFLICT DO NOTHING
;

-- name: PollPayloadWALForRecordsToReplicate :many
WITH tenants AS (
    SELECT UNNEST(
        find_matching_tenants_in_payload_wal_partition(
            @partitionNumber::INT
        )
    ) AS tenant_id
)

SELECT *
FROM v1_payload_wal
WHERE tenant_id = ANY(SELECT tenant_id FROM tenants)
ORDER BY offload_at
LIMIT @pollLimit::INT
FOR UPDATE SKIP LOCKED
;

-- name: SetPayloadExternalKeys :exec
WITH inputs AS (
    SELECT
        UNNEST(@ids::BIGINT[]) AS id,
        UNNEST(@insertedAts::TIMESTAMPTZ[]) AS inserted_at,
        UNNEST(CAST(@payloadTypes::TEXT[] AS v1_payload_type[])) AS type,
        UNNEST(@offloadAts::TIMESTAMPTZ[]) AS offload_at,
        UNNEST(@externalLocationKeys::TEXT[]) AS external_location_key,
        UNNEST(@tenantIds::UUID[]) AS tenant_id
), payload_updates AS (
    UPDATE v1_payload
    SET
        external_location_key = i.external_location_key,
        updated_at = NOW()
    FROM inputs i
    WHERE
        v1_payload.id = i.id
        AND v1_payload.inserted_at = i.inserted_at
        AND v1_payload.tenant_id = i.tenant_id
), cutover_queue_items AS (
    INSERT INTO v1_payload_cutover_queue_item (
        tenant_id,
        cut_over_at,
        payload_id,
        payload_inserted_at,
        payload_type
    )
    SELECT
        i.tenant_id,
        i.offload_at,
        i.id,
        i.inserted_at,
        i.type
    FROM
        inputs i
    ON CONFLICT DO NOTHING
)

DELETE FROM v1_payload_wal
WHERE
    (offload_at, payload_id, payload_inserted_at, payload_type, tenant_id) IN (
        SELECT offload_at, id, inserted_at, type, tenant_id
        FROM inputs
    )
;


-- name: CutOverPayloadsToExternal :one
WITH tenants AS (
    SELECT UNNEST(
        find_matching_tenants_in_payload_cutover_queue_item_partition(
            @partitionNumber::INT
        )
    ) AS tenant_id
), queue_items AS (
    SELECT *
    FROM v1_payload_cutover_queue_item
    WHERE
        tenant_id = ANY(SELECT tenant_id FROM tenants)
        AND cut_over_at <= NOW()
    ORDER BY cut_over_at
    LIMIT @pollLimit::INT
    FOR UPDATE SKIP LOCKED
), payload_updates AS (
    UPDATE v1_payload
    SET
        location = 'EXTERNAL',
        inline_content = NULL,
        updated_at = NOW()
    FROM queue_items qi
    WHERE
        v1_payload.id = qi.payload_id
        AND v1_payload.inserted_at = qi.payload_inserted_at
        AND v1_payload.tenant_id = qi.tenant_id
        AND v1_payload.type = qi.payload_type
        AND v1_payload.external_location_key IS NOT NULL
), deletions AS (
    DELETE FROM v1_payload_cutover_queue_item
    WHERE
        (cut_over_at, payload_id, payload_inserted_at, payload_type, tenant_id) IN (
            SELECT cut_over_at, payload_id, payload_inserted_at, payload_type, tenant_id
            FROM queue_items
        )
)

SELECT COUNT(*)
FROM queue_items
;

-- name: AnalyzeV1Payload :exec
ANALYZE v1_payload;
