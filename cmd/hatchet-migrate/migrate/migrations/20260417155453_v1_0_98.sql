-- +goose Up
-- +goose StatementBegin
DROP FUNCTION IF EXISTS create_v1_olap_partition_with_date_and_status(text, date);
-- +goose StatementEnd

-- +goose StatementBegin
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
-- +goose StatementEnd

-- +goose StatementBegin
DROP TRIGGER IF EXISTS v1_tasks_olap_status_insert_trigger ON v1_tasks_olap;
CREATE TRIGGER v1_tasks_olap_status_insert_trigger
AFTER INSERT ON v1_tasks_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_tasks_olap_insert_function();

DROP TRIGGER IF EXISTS v1_tasks_olap_status_delete_trigger ON v1_tasks_olap;
CREATE TRIGGER v1_tasks_olap_status_delete_trigger
AFTER DELETE ON v1_tasks_olap
REFERENCING OLD TABLE AS old_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_tasks_olap_delete_function();

DROP TRIGGER IF EXISTS v1_tasks_olap_status_update_trigger ON v1_tasks_olap;
CREATE TRIGGER v1_tasks_olap_status_update_trigger
AFTER UPDATE ON v1_tasks_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_tasks_olap_status_update_function();

DROP TRIGGER IF EXISTS v1_dags_olap_status_insert_trigger ON v1_dags_olap;
CREATE TRIGGER v1_dags_olap_status_insert_trigger
AFTER INSERT ON v1_dags_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_dags_olap_insert_function();

DROP TRIGGER IF EXISTS v1_dags_olap_status_delete_trigger ON v1_dags_olap;
CREATE TRIGGER v1_dags_olap_status_delete_trigger
AFTER DELETE ON v1_dags_olap
REFERENCING OLD TABLE AS old_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_dags_olap_delete_function();

DROP TRIGGER IF EXISTS v1_dags_olap_status_update_trigger ON v1_dags_olap;
CREATE TRIGGER v1_dags_olap_status_update_trigger
AFTER UPDATE ON v1_dags_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_dags_olap_status_update_function();

DROP TRIGGER IF EXISTS v1_runs_olap_status_insert_trigger ON v1_runs_olap;
CREATE TRIGGER v1_runs_olap_status_insert_trigger
AFTER INSERT ON v1_runs_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_runs_olap_insert_function();

DROP TRIGGER IF EXISTS v1_runs_olap_status_update_trigger ON v1_runs_olap;
CREATE TRIGGER v1_runs_olap_status_update_trigger
AFTER UPDATE ON v1_runs_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_runs_olap_status_update_function();
-- +goose StatementEnd

-- +goose StatementBegin
DROP FUNCTION IF EXISTS v1_runs_olap_mirror_fn();
DROP FUNCTION IF EXISTS v1_tasks_olap_mirror_fn();
DROP FUNCTION IF EXISTS v1_dags_olap_mirror_fn();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION create_v1_olap_partition_with_date_and_status(
    targetTableName text,
    targetDate date
) RETURNS integer
    LANGUAGE plpgsql AS
$$
DECLARE
    targetDateStr varchar;
    targetDatePlusOneDayStr varchar;
    newTableName varchar;
BEGIN
    SELECT to_char(targetDate, 'YYYYMMDD') INTO targetDateStr;
    SELECT to_char(targetDate + INTERVAL '1 day', 'YYYYMMDD') INTO targetDatePlusOneDayStr;
    SELECT format('%s_%s', targetTableName, targetDateStr) INTO newTableName;
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = newTableName) THEN
        EXECUTE format('CREATE TABLE %s (LIKE %s INCLUDING INDEXES) PARTITION BY LIST (readable_status)', newTableName, targetTableName);
    END IF;

    PERFORM create_v1_partition_with_status(newTableName, 'QUEUED');
    PERFORM create_v1_partition_with_status(newTableName, 'RUNNING');
    PERFORM create_v1_partition_with_status(newTableName, 'COMPLETED');
    PERFORM create_v1_partition_with_status(newTableName, 'CANCELLED');
    PERFORM create_v1_partition_with_status(newTableName, 'FAILED');
    PERFORM create_v1_partition_with_status(newTableName, 'EVICTED');

    IF NOT EXISTS (SELECT 1 FROM pg_inherits WHERE inhrelid = newTableName::regclass) THEN
        EXECUTE format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneDayStr);
    END IF;

    RETURN 1;
END;
$$;
-- +goose StatementEnd
