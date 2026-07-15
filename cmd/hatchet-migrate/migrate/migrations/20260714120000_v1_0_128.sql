-- +goose Up
-- +goose StatementBegin
-- Rewrite v1_runs_olap_status_update_function from UPDATE ... FROM to an upsert.
--
-- We use INSERT INTO rather than UPDATE for ensuring this query is fast on TimescaleDB,
-- which does not have effective runtime partition pruning on UPDATE and would involve scanning
-- every partition in v1_statuses_olap.
CREATE OR REPLACE FUNCTION v1_runs_olap_status_update_function()
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
    -- DISTINCT ON: nothing constraint-enforces (external_id, inserted_at)
    -- uniqueness across new_rows, and ON CONFLICT DO UPDATE errors if a single
    -- statement affects the same row twice. On duplicates, keep the
    -- highest-priority status.
    SELECT DISTINCT ON (external_id, inserted_at)
        external_id,
        inserted_at,
        tenant_id,
        workflow_id,
        kind,
        readable_status
    FROM new_rows
    ORDER BY external_id, inserted_at, v1_status_to_priority(readable_status) DESC
    ON CONFLICT (external_id, inserted_at) DO UPDATE
    SET readable_status = EXCLUDED.readable_status
    WHERE v1_statuses_olap.readable_status IS DISTINCT FROM EXCLUDED.readable_status;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_runs_olap_status_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    UPDATE
        v1_statuses_olap s
    SET
        readable_status = n.readable_status
    FROM new_rows n
    WHERE
        s.external_id = n.external_id
        AND s.inserted_at = n.inserted_at;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd
