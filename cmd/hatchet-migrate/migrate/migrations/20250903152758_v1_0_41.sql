-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_dag_to_task_partitioned (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (dag_id, dag_inserted_at, task_id, task_inserted_at)
) PARTITION BY RANGE(dag_inserted_at);

DO $$
DECLARE
    new_table_name TEXT;
BEGIN
    -- We want to attach all of the existing data as today's partition, which
    -- includes everything up until the _end_ of today.
    -- The new partition will be named with today's date, but the end time will be tomorrow (midnight).
    new_table_name := 'v1_dag_to_task_' || TO_CHAR(NOW()::DATE, 'YYYYMMDD');

    RAISE NOTICE 'Renaming existing table to %', new_table_name;

    EXECUTE format('ALTER TABLE v1_dag_to_task RENAME TO %I', new_table_name);
    EXECUTE format('ALTER INDEX v1_dag_to_task_pkey RENAME TO %I', new_table_name || '_pkey');

    EXECUTE
        format('ALTER TABLE %s SET (
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor=''0.05'',
            autovacuum_vacuum_threshold=''25'',
            autovacuum_analyze_threshold=''25'',
            autovacuum_vacuum_cost_delay=''10'',
            autovacuum_vacuum_cost_limit=''1000''
        )', new_table_name);

    EXECUTE
        format(
            'ALTER TABLE v1_dag_to_task_partitioned ATTACH PARTITION %s FOR VALUES FROM (''19700101'') TO (''%s'')',
            new_table_name,
            TO_CHAR((NOW() + INTERVAL '1 day')::DATE, 'YYYYMMDD')
        );
END $$;

ALTER TABLE v1_dag_to_task_partitioned RENAME TO v1_dag_to_task;
ALTER INDEX v1_dag_to_task_partitioned_pkey RENAME TO v1_dag_to_task_pkey;

SELECT create_v1_range_partition('v1_dag_to_task', (NOW() + INTERVAL '1 day')::DATE);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE v1_dag_to_task_original (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (dag_id, dag_inserted_at, task_id, task_inserted_at)
);

INSERT INTO v1_dag_to_task_original
SELECT * FROM v1_dag_to_task;

DROP TABLE v1_dag_to_task;
ALTER TABLE v1_dag_to_task_original RENAME TO v1_dag_to_task;
ALTER INDEX v1_dag_to_task_original_pkey RENAME TO v1_dag_to_task_pkey;
-- +goose StatementEnd
