-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_tasks_olap_delete_function()
RETURNS TRIGGER AS
$$
BEGIN
    DELETE FROM v1_runs_olap r
    USING old_rows o
    WHERE
        r.inserted_at = o.inserted_at
        AND r.id = o.id
        AND r.readable_status = o.readable_status
        AND r.kind = 'TASK'
        AND o.dag_id IS NULL;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_tasks_olap_status_delete_trigger
AFTER DELETE ON v1_tasks_olap
REFERENCING OLD TABLE AS old_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_tasks_olap_delete_function();

CREATE OR REPLACE FUNCTION v1_dags_olap_delete_function()
RETURNS TRIGGER AS
$$
BEGIN
    DELETE FROM v1_runs_olap r
    USING old_rows o
    WHERE
        r.inserted_at = o.inserted_at
        AND r.id = o.id
        AND r.readable_status = o.readable_status
        AND r.kind = 'DAG';

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_dags_olap_status_delete_trigger
AFTER DELETE ON v1_dags_olap
REFERENCING OLD TABLE AS old_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_dags_olap_delete_function();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS v1_tasks_olap_status_delete_trigger ON v1_tasks_olap;
DROP FUNCTION IF EXISTS v1_tasks_olap_delete_function();

DROP TRIGGER IF EXISTS v1_dags_olap_status_delete_trigger ON v1_dags_olap;
DROP FUNCTION IF EXISTS v1_dags_olap_delete_function();
-- +goose StatementEnd
