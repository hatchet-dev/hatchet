-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_tasks_olap_status_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    UPDATE
        v1_runs_olap r
    SET
        readable_status = n.readable_status
    FROM new_rows n
    WHERE
        r.id = n.id
        AND r.inserted_at = n.inserted_at
        AND r.kind = 'TASK';

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_tasks_olap_status_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    UPDATE
        v1_runs_olap r
    SET
        readable_status = n.readable_status
    FROM new_rows n
    WHERE
        r.id = n.id
        AND r.inserted_at = n.inserted_at
        AND r.kind = 'TASK';

    -- insert tmp events into task status updates table if we have a dag_id
    INSERT INTO v1_task_status_updates_tmp (
        tenant_id,
        dag_id,
        dag_inserted_at
    )
    SELECT
        tenant_id,
        dag_id,
        dag_inserted_at
    FROM new_rows
    WHERE dag_id IS NOT NULL;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd
