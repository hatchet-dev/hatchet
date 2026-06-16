-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS v1_dag_to_task_partitioned (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (dag_id, dag_inserted_at, task_id, task_inserted_at)
) PARTITION BY RANGE(dag_inserted_at);

DO $$
DECLARE
    oldest_date DATE;
    d DATE;
    partition_count INTEGER;
BEGIN
    SELECT COUNT(*)
    INTO partition_count
    FROM pg_class c
    JOIN pg_inherits i ON c.oid = i.inhrelid
    JOIN pg_class parent ON i.inhparent = parent.oid
    WHERE parent.relname = 'v1_dag_to_task_partitioned'
    AND c.relkind = 'r';

    IF partition_count > 0 THEN
        RAISE NOTICE 'Partitions already exist for v1_dag_to_task_partitioned, skipping creation';
        RETURN;
    END IF;

    SELECT (WITH t AS (SELECT * FROM v1_task ORDER BY id LIMIT 1) SELECT inserted_at::DATE FROM t)
    INTO oldest_date;

    IF oldest_date IS NULL THEN
        oldest_date := NOW()::DATE;
    END IF;

    d := oldest_date;
    WHILE d <= (NOW() + INTERVAL '1 day')::DATE LOOP
        PERFORM create_v1_range_partition('v1_dag_to_task_partitioned', d);
        d := d + INTERVAL '1 day';
    END LOOP;
END $$;

CREATE OR REPLACE FUNCTION v1_dag_to_task_partitioned_insert_function()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO v1_dag_to_task_partitioned (
        dag_id,
        dag_inserted_at,
        task_id,
        task_inserted_at
    )
    SELECT
        dag_id,
        dag_inserted_at,
        task_id,
        task_inserted_at
    FROM new_rows
    ON CONFLICT (dag_id, dag_inserted_at, task_id, task_inserted_at) DO NOTHING;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER v1_dag_to_task_partitioned_insert_trigger
AFTER INSERT ON v1_dag_to_task
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_dag_to_task_partitioned_insert_function();
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO v1_dag_to_task_partitioned (
    dag_id,
    dag_inserted_at,
    task_id,
    task_inserted_at
)
SELECT
    dag_id,
    dag_inserted_at,
    task_id,
    task_inserted_at
FROM v1_dag_to_task
WHERE dag_id >= (
    SELECT MIN(id)
    FROM v1_dag
)
ON CONFLICT (dag_id, dag_inserted_at, task_id, task_inserted_at) DO NOTHING;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS v1_dag_to_task;
DROP FUNCTION IF EXISTS v1_dag_to_task_partitioned_insert_function;

SELECT rename_partitions('v1_dag_to_task_partitioned', 'v1_dag_to_task');

ALTER TABLE v1_dag_to_task_partitioned RENAME TO v1_dag_to_task;
ALTER INDEX v1_dag_to_task_partitioned_pkey RENAME TO v1_dag_to_task_pkey;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE v1_dag_to_task_original (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT v1_dag_to_task_pkey PRIMARY KEY (dag_id, dag_inserted_at, task_id, task_inserted_at)
);

INSERT INTO v1_dag_to_task_original
SELECT * FROM v1_dag_to_task;

DROP TABLE v1_dag_to_task;

ALTER TABLE v1_dag_to_task_original RENAME TO v1_dag_to_task;
-- +goose StatementEnd
