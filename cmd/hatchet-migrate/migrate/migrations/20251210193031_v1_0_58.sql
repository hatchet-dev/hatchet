-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION list_paginated_payloads_for_offload(
    partition_date date,
    limit_param int,
    last_tenant_id uuid,
    last_inserted_at timestamptz,
    last_id bigint,
    last_type v1_payload_type
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
        WHERE (tenant_id, inserted_at, id, type) >= ($1, $2, $3, $4)
        ORDER BY tenant_id, inserted_at, id, type
        LIMIT $5
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_inserted_at, last_id, last_type, limit_param;
END;
$$;

CREATE OR REPLACE FUNCTION list_paginated_olap_payloads_for_offload(
    partition_date date,
    limit_param int,
    last_tenant_id uuid,
    last_external_id uuid,
    last_inserted_at timestamptz
) RETURNS TABLE (
    tenant_id UUID,
    external_id UUID,
    location v1_payload_location_olap,
    external_location_key TEXT,
    inline_content JSONB,
    inserted_at TIMESTAMPTZ,
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
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        SELECT tenant_id, external_id, location, external_location_key, inline_content, inserted_at, updated_at
        FROM %I
        WHERE (tenant_id, external_id, inserted_at) >= ($1, $2, $3)
        ORDER BY tenant_id, external_id, inserted_at
        LIMIT $4
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_external_id, last_inserted_at, limit_param;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION list_paginated_payloads_for_offload(
    partition_date date,
    limit_param int,
    last_tenant_id uuid,
    last_inserted_at timestamptz,
    last_id bigint,
    last_type v1_payload_type
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
        WHERE (tenant_id, inserted_at, id, type) > ($1, $2, $3, $4)
        ORDER BY tenant_id, inserted_at, id, type
        LIMIT $5
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_inserted_at, last_id, last_type, limit_param;
END;
$$;

CREATE OR REPLACE FUNCTION list_paginated_olap_payloads_for_offload(
    partition_date date,
    limit_param int,
    last_tenant_id uuid,
    last_external_id uuid,
    last_inserted_at timestamptz
) RETURNS TABLE (
    tenant_id UUID,
    external_id UUID,
    location v1_payload_location_olap,
    external_location_key TEXT,
    inline_content JSONB,
    inserted_at TIMESTAMPTZ,
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
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        SELECT tenant_id, external_id, location, external_location_key, inline_content, inserted_at, updated_at
        FROM %I
        WHERE (tenant_id, external_id, inserted_at) > ($1, $2, $3)
        ORDER BY tenant_id, external_id, inserted_at
        LIMIT $4
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_external_id, last_inserted_at, limit_param;
END;
$$;
-- +goose StatementEnd
