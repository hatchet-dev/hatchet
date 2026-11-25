-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION write_payloads(
    tenant_ids UUID[],
    ids BIGINT[],
    inserted_ats TIMESTAMPTZ[],
    external_ids UUID[],
    types v1_payload_type[],
    inline_contents JSONB[],
    payload_reference_ids UUID[],
    overwrite_existing BOOLEAN
) RETURNS VOID
    LANGUAGE plpgsql
AS $$
BEGIN
    IF overwrite_existing THEN
        -- NOTE: this is an upsert rather than an UPDATE because the row being overwritten may have
        -- already been offloaded out of v1_payload (e.g. a task replayed after its partition was cut over
        -- to external storage). A plain UPDATE would silently no-op and readers would keep resolving the
        -- stale offloaded copy via the block index instead of the new content.
        WITH unnested AS (
            SELECT
                UNNEST(tenant_ids) AS tenant_id,
                UNNEST(ids) AS id,
                UNNEST(inserted_ats) AS inserted_at,
                UNNEST(external_ids) AS external_id,
                UNNEST(types) AS type,
                UNNEST(inline_contents) AS inline_content,
                NULLIF(UNNEST(payload_reference_ids), '00000000-0000-0000-0000-000000000000'::UUID) AS payload_reference_id
        ), inputs AS (
            SELECT DISTINCT
                u.tenant_id,
                u.id,
                v1_payload_inserted_at(u.payload_reference_id, u.inserted_at) AS inserted_at,
                COALESCE(u.payload_reference_id, u.external_id) AS external_id,
                u.type,
                u.inline_content
            FROM
                unnested u
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
            'INLINE',
            NULL,
            i.inline_content
        FROM
            inputs i
        ORDER BY i.tenant_id, i.inserted_at, i.id, i.type
        ON CONFLICT (tenant_id, inserted_at, id, type) DO UPDATE
        SET
            location = 'INLINE',
            external_location_key = NULL,
            inline_content = EXCLUDED.inline_content,
            updated_at = NOW();
    ELSE
        WITH unnested AS (
            SELECT
                UNNEST(tenant_ids) AS tenant_id,
                UNNEST(ids) AS id,
                UNNEST(inserted_ats) AS inserted_at,
                UNNEST(external_ids) AS external_id,
                UNNEST(types) AS type,
                UNNEST(inline_contents) AS inline_content,
                NULLIF(UNNEST(payload_reference_ids), '00000000-0000-0000-0000-000000000000'::UUID) AS payload_reference_id
        ), inputs AS (
            SELECT DISTINCT
                u.tenant_id,
                u.id,
                v1_payload_inserted_at(u.payload_reference_id, u.inserted_at) AS inserted_at,
                COALESCE(u.payload_reference_id, u.external_id) AS external_id,
                u.type,
                u.inline_content
            FROM
                unnested u
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
            'INLINE',
            NULL,
            i.inline_content
        FROM
            inputs i
        ORDER BY i.tenant_id, i.inserted_at, i.id, i.type
        ON CONFLICT DO NOTHING;
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS write_payloads(UUID[], BIGINT[], TIMESTAMPTZ[], UUID[], v1_payload_type[], JSONB[], UUID[], BOOLEAN);
-- +goose StatementEnd
