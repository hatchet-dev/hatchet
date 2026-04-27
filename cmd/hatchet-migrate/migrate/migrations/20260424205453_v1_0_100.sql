-- +goose Up
-- +goose StatementBegin
DROP FUNCTION IF EXISTS create_v1_olap_partition_with_date_and_status(text, date);

DO $$
DECLARE
    base_name  text;
    old_parent text;
    new_parent text;
    child      RECORD;
    idx        RECORD;
    con        RECORD;
    new_name   text;
BEGIN
    FOREACH base_name IN ARRAY ARRAY['v1_runs_olap', 'v1_tasks_olap', 'v1_dags_olap'] LOOP
        old_parent := base_name;
        new_parent := base_name || '_new';

        EXECUTE format('LOCK TABLE %I IN ACCESS EXCLUSIVE MODE', old_parent);
        EXECUTE format('DROP TABLE IF EXISTS %I CASCADE', old_parent);
        EXECUTE format('ALTER TABLE %I RENAME TO %I', new_parent, old_parent);

        FOR idx IN (SELECT indexname FROM pg_indexes WHERE tablename = old_parent) LOOP
            new_name := replace(idx.indexname, base_name || '_new', base_name);
            IF new_name != idx.indexname THEN
                EXECUTE format('ALTER INDEX %I RENAME TO %I', idx.indexname, new_name);
            END IF;
        END LOOP;

        FOR con IN (
            SELECT conname FROM pg_constraint
            WHERE conrelid = old_parent::regclass AND contype = 'p'
        ) LOOP
            new_name := replace(con.conname, base_name || '_new', base_name);
            IF new_name != con.conname THEN
                EXECUTE format('ALTER TABLE %I RENAME CONSTRAINT %I TO %I', old_parent, con.conname, new_name);
            END IF;
        END LOOP;

        FOR child IN (
            SELECT c.relname
            FROM   pg_class c
            JOIN   pg_inherits i ON c.oid = i.inhrelid
            JOIN   pg_class    p ON p.oid = i.inhparent
            WHERE  p.relname = old_parent
              AND  c.relkind IN ('r', 'p')
        ) LOOP
            FOR idx IN (SELECT indexname FROM pg_indexes WHERE tablename = child.relname) LOOP
                new_name := replace(idx.indexname, base_name || '_new', base_name);
                IF new_name != idx.indexname THEN
                    EXECUTE format('ALTER INDEX %I RENAME TO %I', idx.indexname, new_name);
                END IF;
            END LOOP;

            FOR con IN (
                SELECT conname FROM pg_constraint
                WHERE conrelid = child.relname::regclass AND contype = 'p'
            ) LOOP
                new_name := replace(con.conname, base_name || '_new', base_name);
                IF new_name != con.conname THEN
                    EXECUTE format('ALTER TABLE %I RENAME CONSTRAINT %I TO %I', child.relname, con.conname, new_name);
                END IF;
            END LOOP;

            new_name := replace(child.relname, base_name || '_new_', base_name || '_');
            EXECUTE format('ALTER TABLE %I RENAME TO %I', child.relname, new_name);
        END LOOP;
    END LOOP;
END;
$$;

CREATE OR REPLACE FUNCTION v1_tasks_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_runs_olap (
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        kind,
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id
    )
    SELECT
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        'TASK',
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id
    FROM new_rows
    WHERE dag_id IS NULL
    ON CONFLICT (inserted_at, id) DO NOTHING;

    INSERT INTO v1_lookup_table_olap (
        tenant_id,
        external_id,
        task_id,
        inserted_at
    )
    SELECT
        tenant_id,
        external_id,
        id,
        inserted_at
    FROM new_rows
    ON CONFLICT (external_id) DO NOTHING;

    -- If the task has a dag_id and dag_inserted_at, insert into the lookup table
    INSERT INTO v1_dag_to_task_olap (
        dag_id,
        dag_inserted_at,
        task_id,
        task_inserted_at
    )
    SELECT
        dag_id,
        dag_inserted_at,
        id,
        inserted_at
    FROM new_rows
    WHERE dag_id IS NOT NULL
    ON CONFLICT (dag_id, dag_inserted_at, task_id, task_inserted_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION v1_dags_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_runs_olap (
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        kind,
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id
    )
    SELECT
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        'DAG',
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id
    FROM new_rows
    ON CONFLICT (inserted_at, id) DO NOTHING;

    INSERT INTO v1_lookup_table_olap (
        tenant_id,
        external_id,
        dag_id,
        inserted_at
    )
    SELECT
        tenant_id,
        external_id,
        id,
        inserted_at
    FROM new_rows
    ON CONFLICT (external_id) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER v1_tasks_olap_status_insert_trigger
AFTER INSERT ON v1_tasks_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_tasks_olap_insert_function();

CREATE OR REPLACE TRIGGER v1_tasks_olap_status_delete_trigger
AFTER DELETE ON v1_tasks_olap
REFERENCING OLD TABLE AS old_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_tasks_olap_delete_function();

CREATE OR REPLACE TRIGGER v1_tasks_olap_status_update_trigger
AFTER UPDATE ON v1_tasks_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_tasks_olap_status_update_function();

CREATE OR REPLACE TRIGGER v1_dags_olap_status_insert_trigger
AFTER INSERT ON v1_dags_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_dags_olap_insert_function();

CREATE OR REPLACE TRIGGER v1_dags_olap_status_delete_trigger
AFTER DELETE ON v1_dags_olap
REFERENCING OLD TABLE AS old_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_dags_olap_delete_function();

CREATE OR REPLACE TRIGGER v1_dags_olap_status_update_trigger
AFTER UPDATE ON v1_dags_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_dags_olap_status_update_function();

CREATE OR REPLACE TRIGGER v1_runs_olap_status_insert_trigger
AFTER INSERT ON v1_runs_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_runs_olap_insert_function();

CREATE OR REPLACE TRIGGER v1_runs_olap_status_update_trigger
AFTER UPDATE ON v1_runs_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_runs_olap_status_update_function();

DROP FUNCTION IF EXISTS v1_runs_olap_mirror_fn();
DROP FUNCTION IF EXISTS v1_tasks_olap_mirror_fn();
DROP FUNCTION IF EXISTS v1_dags_olap_mirror_fn();
-- +goose StatementEnd

-- +goose Down
-- intentionally blank
