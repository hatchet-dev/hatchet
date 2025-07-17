-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION find_matching_tenants_in_task_status_updates_tmp_partition(
    partition_number INT,
    tenant_ids UUID[]
) RETURNS UUID[]
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
    result UUID[];
BEGIN
    partition_table := 'v1_task_status_updates_tmp_' || partition_number::text;

    EXECUTE format(
        'SELECT ARRAY(
            SELECT DISTINCT e.tenant_id
            FROM %I e
            WHERE e.tenant_id = ANY($1)
              AND e.requeue_after <= CURRENT_TIMESTAMP
        )',
        partition_table)
    USING tenant_ids
    INTO result;

    RETURN result;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION find_matching_tenants_in_task_status_updates_tmp_partition(
    partition_number INT,
    tenant_ids UUID[]
);
-- +goose StatementEnd
