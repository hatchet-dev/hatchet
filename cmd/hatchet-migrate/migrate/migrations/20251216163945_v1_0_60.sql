-- +goose Up
-- +goose StatementBegin
DROP FUNCTION list_paginated_payloads_for_offload(date, uuid, timestamptz, bigint, v1_payload_type, uuid, timestamptz, bigint, v1_payload_type);
CREATE OR REPLACE FUNCTION list_paginated_payloads_for_offload(
    partition_date date,
    last_tenant_id uuid,
    last_inserted_at timestamptz,
    last_id bigint,
    last_type v1_payload_type,
    next_tenant_id uuid,
    next_inserted_at timestamptz,
    next_id bigint,
    next_type v1_payload_type,
    batch_size integer
) RETURNS TABLE (
    tenant_id UUID,
    id BIGINT,
    inserted_at TIMESTAMPTZ,
    external_id UUID,
    type v1_payload_type,
    location v1_payload_location,
    external_location_key TEXT,
    inline_content JSONB,
    updated_at TIMESTAMPTZ
)
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    query text;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH candidates AS (
            SELECT tenant_id, id, inserted_at, external_id, type, location,
                external_location_key, inline_content, updated_at
            FROM %I
            WHERE
                (tenant_id, inserted_at, id, type) >= ($1, $2, $3, $4)
            ORDER BY tenant_id, inserted_at, id, type
            LIMIT $9 * 2
        )

        SELECT tenant_id, id, inserted_at, external_id, type, location,
               external_location_key, inline_content, updated_at
        FROM candidates
        WHERE
            (tenant_id, inserted_at, id, type) >= ($1, $2, $3, $4)
            AND (tenant_id, inserted_at, id, type) <= ($5, $6, $7, $8)
        ORDER BY tenant_id, inserted_at, id, type
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_inserted_at, last_id, last_type, next_tenant_id, next_inserted_at, next_id, next_type, batch_size;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION list_paginated_payloads_for_offload(date, uuid, timestamptz, bigint, v1_payload_type, uuid, timestamptz, bigint, v1_payload_type, integer);
CREATE OR REPLACE FUNCTION list_paginated_payloads_for_offload(
    partition_date date,
    last_tenant_id uuid,
    last_inserted_at timestamptz,
    last_id bigint,
    last_type v1_payload_type,
    next_tenant_id uuid,
    next_inserted_at timestamptz,
    next_id bigint,
    next_type v1_payload_type
) RETURNS TABLE (
    tenant_id UUID,
    id BIGINT,
    inserted_at TIMESTAMPTZ,
    external_id UUID,
    type v1_payload_type,
    location v1_payload_location,
    external_location_key TEXT,
    inline_content JSONB,
    updated_at TIMESTAMPTZ
)
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    query text;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        SELECT tenant_id, id, inserted_at, external_id, type, location,
               external_location_key, inline_content, updated_at
        FROM %I
        WHERE
            (tenant_id, inserted_at, id, type) >= ($1, $2, $3, $4)
            AND (tenant_id, inserted_at, id, type) <= ($5, $6, $7, $8)
        ORDER BY tenant_id, inserted_at, id, type
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_inserted_at, last_id, last_type, next_tenant_id, next_inserted_at, next_id, next_type;
END;
$$;
-- +goose StatementEnd
