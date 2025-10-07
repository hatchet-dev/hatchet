-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_payload DROP CONSTRAINT v1_payload_check;
ALTER TABLE v1_payload ADD CONSTRAINT v1_payload_check CHECK (
    location = 'INLINE'
    OR
    (location = 'EXTERNAL' AND inline_content IS NULL AND external_location_key IS NOT NULL)
) NOT VALID;

CREATE TABLE v1_payload_cutover_queue_item (
    tenant_id UUID NOT NULL,
    cut_over_at TIMESTAMPTZ NOT NULL,
    payload_id BIGINT NOT NULL,
    payload_inserted_at TIMESTAMPTZ NOT NULL,
    payload_type v1_payload_type NOT NULL,

    PRIMARY KEY (cut_over_at, tenant_id, payload_id, payload_inserted_at, payload_type),
    CONSTRAINT "v1_payload_cutover_queue_item_payload" FOREIGN KEY (payload_id, payload_inserted_at, payload_type, tenant_id) REFERENCES v1_payload (id, inserted_at, type, tenant_id) ON DELETE CASCADE
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

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_payload DROP CONSTRAINT v1_payload_check;
ALTER TABLE v1_payload ADD CONSTRAINT v1_payload_check CHECK (
    (location = 'INLINE' AND external_location_key IS NULL)
    OR
    (location = 'EXTERNAL' AND inline_content IS NULL AND external_location_key IS NOT NULL)
) NOT VALID;

DROP TABLE v1_payload_cutover_queue_item;

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
            WHERE e.offload_at < NOW()
        )',
        partition_table)
    INTO result;

    RETURN result;
END;
$$;

DROP FUNCTION find_matching_tenants_in_payload_cutover_queue_item_partition(INT);
-- +goose StatementEnd
