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
