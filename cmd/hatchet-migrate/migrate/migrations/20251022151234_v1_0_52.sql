-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION find_matching_tenants_in_payload_wal_partition(
    partition_number INT
) RETURNS UUID[]
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
    result UUID[];
BEGIN
    partition_table := 'v1_payload_wal_' || partition_number::text;

    EXECUTE format(
        'SELECT ARRAY(
            SELECT t.id
            FROM "Tenant" t
            WHERE EXISTS (
                SELECT 1
                FROM %I e
                WHERE e.tenant_id = t.id
                LIMIT 1
            )
        )',
        partition_table)
    INTO result;

    RETURN result;
END;
$$;

CREATE OR REPLACE FUNCTION find_matching_tenants_in_payload_cutover_queue_item_partition(
    partition_number INT
) RETURNS UUID[]
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
    result UUID[];
BEGIN
    partition_table := 'v1_payload_cutover_queue_item_' || partition_number::text;

    EXECUTE format(
        'SELECT ARRAY(
            SELECT t.id
            FROM "Tenant" t
            WHERE EXISTS (
                SELECT 1
                FROM %I e
                WHERE e.tenant_id = t.id
                  AND e.cut_over_at <= NOW()
                LIMIT 1
            )
        )',
        partition_table)
    INTO result;

    RETURN result;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION find_matching_tenants_in_payload_wal_partition(
    partition_number INT
) RETURNS UUID[]
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
    result UUID[];
BEGIN
    partition_table := 'v1_payload_wal_' || partition_number::text;

    EXECUTE format(
        'SELECT ARRAY(
            SELECT DISTINCT e.tenant_id
            FROM %I e
        )',
        partition_table)
    INTO result;

    RETURN result;
END;
$$;

CREATE OR REPLACE FUNCTION find_matching_tenants_in_payload_cutover_queue_item_partition(
    partition_number INT
) RETURNS UUID[]
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
    result UUID[];
BEGIN
    partition_table := 'v1_payload_cutover_queue_item_' || partition_number::text;

    EXECUTE format(
        'SELECT ARRAY(
            SELECT DISTINCT e.tenant_id
            FROM %I e
            WHERE e.cut_over_at <= NOW()
        )',
        partition_table)
    INTO result;

    RETURN result;
END;
$$;
-- +goose StatementEnd
