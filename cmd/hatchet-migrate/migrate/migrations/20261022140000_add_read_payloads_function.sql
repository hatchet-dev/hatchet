-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_payload_inserted_at(
    payload_reference_id UUID,
    inserted_at TIMESTAMPTZ
) RETURNS TIMESTAMPTZ
    LANGUAGE sql
    IMMUTABLE
    PARALLEL SAFE
AS $$
    SELECT CASE
        WHEN payload_reference_id IS NOT NULL AND get_byte(uuid_send(payload_reference_id), 6) >> 4 = 7
        THEN extract_uuid7_timestamp(payload_reference_id)
        ELSE inserted_at
    END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION read_payloads(
    tenant_ids UUID[],
    ids BIGINT[],
    inserted_ats TIMESTAMPTZ[],
    types v1_payload_type[],
    payload_reference_ids UUID[]
) RETURNS SETOF v1_payload
    LANGUAGE sql
    STABLE
AS $$
    WITH inputs AS (
        SELECT
            UNNEST(tenant_ids) AS tenant_id,
            UNNEST(ids) AS id,
            UNNEST(inserted_ats) AS inserted_at,
            UNNEST(types) AS type,
            NULLIF(UNNEST(payload_reference_ids), '00000000-0000-0000-0000-000000000000'::UUID) AS payload_reference_id
    )

    SELECT *
    FROM v1_payload
    WHERE (tenant_id, id, inserted_at, type) IN (
        SELECT tenant_id, id, inserted_at, type
        FROM inputs
        WHERE payload_reference_id IS NULL
    )

    UNION ALL

    SELECT *
    FROM v1_payload
    WHERE (tenant_id, external_id, inserted_at) IN (
        SELECT tenant_id, payload_reference_id, v1_payload_inserted_at(payload_reference_id, inserted_at)
        FROM inputs
        WHERE payload_reference_id IS NOT NULL
    );
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION read_payloads(UUID[], BIGINT[], TIMESTAMPTZ[], v1_payload_type[], UUID[]);
-- +goose StatementEnd

-- +goose StatementBegin
DROP FUNCTION v1_payload_inserted_at(UUID, TIMESTAMPTZ);
-- +goose StatementEnd
