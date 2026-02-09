-- +goose Up
-- +goose StatementBegin
DROP TABLE v1_payload_wal;
DROP TYPE v1_payload_wal_operation;
DROP TABLE v1_payload_cutover_queue_item;
DROP FUNCTION find_matching_tenants_in_payload_wal_partition(INT);
DROP FUNCTION find_matching_tenants_in_payload_cutover_queue_item_partition(INT);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TYPE v1_payload_wal_operation AS ENUM ('CREATE', 'UPDATE', 'DELETE');

CREATE TABLE v1_payload_wal (
    tenant_id UUID NOT NULL,
    offload_at TIMESTAMPTZ NOT NULL,
    payload_id BIGINT NOT NULL,
    payload_inserted_at TIMESTAMPTZ NOT NULL,
    payload_type v1_payload_type NOT NULL,
    operation v1_payload_wal_operation NOT NULL DEFAULT 'CREATE',

    -- todo: we probably should shuffle this index around - it'd make more sense now for the order to be
    -- (tenant_id, offload_at, payload_id, payload_inserted_at, payload_type)
    -- so we can filter by the tenant id first, then order by offload_at
    PRIMARY KEY (offload_at, tenant_id, payload_id, payload_inserted_at, payload_type)
) PARTITION BY HASH (tenant_id);

CREATE INDEX v1_payload_wal_payload_lookup_idx ON v1_payload_wal (payload_id, payload_inserted_at, payload_type, tenant_id);
CREATE INDEX v1_payload_wal_poll_idx ON v1_payload_wal (tenant_id, offload_at);

SELECT create_v1_hash_partitions('v1_payload_wal'::TEXT, 4);

CREATE TABLE v1_payload_cutover_queue_item (
    tenant_id UUID NOT NULL,
    cut_over_at TIMESTAMPTZ NOT NULL,
    payload_id BIGINT NOT NULL,
    payload_inserted_at TIMESTAMPTZ NOT NULL,
    payload_type v1_payload_type NOT NULL,

    PRIMARY KEY (cut_over_at, tenant_id, payload_id, payload_inserted_at, payload_type)
) PARTITION BY HASH (tenant_id);

SELECT create_v1_hash_partitions('v1_payload_cutover_queue_item'::TEXT, 4);

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
