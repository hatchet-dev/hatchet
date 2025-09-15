-- name: ReadPayload :one
SELECT *
FROM v1_payload
WHERE
    tenant_id = @tenantId::UUID
    AND type = @type::v1_payload_type
    AND id = @id::BIGINT
    AND inserted_at = @insertedAt::TIMESTAMPTZ
;

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
    SELECT
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
    CASE WHEN i.external_location_key = '' OR i.location = 'EXTERNAL' THEN NULL ELSE i.external_location_key END,
    i.inline_content
FROM
    inputs i
;


-- name: WritePayloadWAL :exec
WITH inputs AS (
    SELECT
        UNNEST(@payloadIds::BIGINT[]) AS payload_id,
        UNNEST(@payloadInsertedAts::TIMESTAMPTZ[]) AS payload_inserted_at,
        UNNEST(CAST(@payloadTypes::TEXT[] AS v1_payload_type[])) AS payload_type,
        UNNEST(@offloadAts::TIMESTAMPTZ[]) AS offload_at,
        UNNEST(CAST(@operations::TEXT[] AS v1_payload_wal_operation[])) AS operation,
        UNNEST(@tenantIds::UUID[]) AS tenant_id
)

INSERT INTO v1_payload_wal (
    tenant_id,
    offload_at,
    payload_id,
    payload_inserted_at,
    payload_type,
    operation
)
SELECT
    i.tenant_id,
    i.offload_at,
    i.payload_id,
    i.payload_inserted_at,
    i.payload_type,
    i.operation
FROM
    inputs i
;

-- name: PollPayloadWALForRecordsToOffload :many
WITH tenants AS (
    SELECT UNNEST(
        find_matching_tenants_in_payload_wal_partition(
            @partitionNumber::BIGINT
        )
    ) AS tenant_id
)

SELECT *
FROM v1_payload_wal
WHERE
    offload_at < NOW()
    AND tenant_id = ANY(SELECT tenant_id FROM tenants)
ORDER BY offload_at, payload_id, payload_inserted_at, payload_type, tenant_id
FOR UPDATE
LIMIT @pollLimit::INT

;

-- name: FinalizePayloadOffloads :exec
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
        location = 'EXTERNAL',
        external_location_key = i.external_location_key,
        inline_content = NULL,
        updated_at = NOW()
    FROM inputs i
    WHERE
        v1_payload.id = i.id
        AND v1_payload.inserted_at = i.inserted_at
        AND v1_payload.tenant_id = i.tenant_id
)

DELETE FROM v1_payload_wal
WHERE
    (offload_at, payload_id, payload_inserted_at, payload_type, tenant_id) IN (
        SELECT offload_at, id, inserted_at, type, tenant_id
        FROM inputs
    )
;
