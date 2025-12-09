-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_payload_cutover_job_offset
ADD COLUMN last_tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID,
ADD COLUMN last_inserted_at TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00',
ADD COLUMN last_id BIGINT NOT NULL DEFAULT 0,
ADD COLUMN last_type v1_payload_type NOT NULL DEFAULT 'TASK_INPUT',
DROP COLUMN last_offset;

ALTER TABLE v1_payloads_olap_cutover_job_offset
ADD COLUMN last_tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID,
ADD COLUMN last_external_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID,
ADD COLUMN last_inserted_at TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00',
DROP COLUMN last_offset;

-- need to explicitly drop and replace because of the changes to the params
DROP FUNCTION IF EXISTS list_paginated_payloads_for_offload(date, int, bigint);
DROP FUNCTION IF EXISTS list_paginated_olap_payloads_for_offload(date, int, bigint);

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

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_payload_cutover_job_offset
ADD COLUMN last_offset BIGINT NOT NULL DEFAULT 0,
DROP COLUMN last_tenant_id,
DROP COLUMN last_inserted_at,
DROP COLUMN last_id,
DROP COLUMN last_type;

ALTER TABLE v1_payloads_olap_cutover_job_offset
ADD COLUMN last_offset BIGINT NOT NULL DEFAULT 0,
DROP COLUMN last_tenant_id,
DROP COLUMN last_external_id,
DROP COLUMN last_inserted_at
;

DROP FUNCTION IF EXISTS list_paginated_payloads_for_offload(date, int, uuid, timestamptz, bigint, v1_payload_type);
DROP FUNCTION IF EXISTS list_paginated_olap_payloads_for_offload(date, int, uuid, uuid, timestamptz);

CREATE OR REPLACE FUNCTION list_paginated_olap_payloads_for_offload(
    partition_date date,
    limit_param int,
    offset_param bigint
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
        ORDER BY tenant_id, external_id, inserted_at
        LIMIT $1
        OFFSET $2
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING limit_param, offset_param;
END;
$$;

CREATE OR REPLACE FUNCTION list_paginated_payloads_for_offload(
    partition_date date,
    limit_param int,
    offset_param bigint
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
        ORDER BY tenant_id, inserted_at, id, type
        LIMIT $1
        OFFSET $2
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING limit_param, offset_param;
END;
$$;
-- +goose StatementEnd
