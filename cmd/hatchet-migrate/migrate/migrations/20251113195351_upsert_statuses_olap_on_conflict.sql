-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_runs_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_statuses_olap (
        external_id,
        inserted_at,
        tenant_id,
        workflow_id,
        kind,
        readable_status
    )
    SELECT
        external_id,
        inserted_at,
        tenant_id,
        workflow_id,
        kind,
        readable_status
    FROM new_rows
    ON CONFLICT (external_id, inserted_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
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
    ON CONFLICT (inserted_at, id, readable_status, kind) DO NOTHING;

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
-- +goose StatementEnd

-- +goose StatementBegin
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
    ON CONFLICT (inserted_at, id, readable_status, kind) DO NOTHING;

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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_runs_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_statuses_olap (
        external_id,
        inserted_at,
        tenant_id,
        workflow_id,
        kind,
        readable_status
    )
    SELECT
        external_id,
        inserted_at,
        tenant_id,
        workflow_id,
        kind,
        readable_status
    FROM new_rows;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
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
    ON CONFLICT (inserted_at, id, readable_status, kind) DO NOTHING;

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
    WHERE dag_id IS NOT NULL;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
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
    FROM new_rows;

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
-- +goose StatementEnd
