-- +goose NO TRANSACTION

-- +goose Up
DROP INDEX v1_filter_unique_idx;
CREATE UNIQUE INDEX CONCURRENTLY v1_filter_unique_tenant_workflow_id_scope_expression_payload ON v1_filter (
    tenant_id ASC,
    workflow_id ASC,
    scope ASC,
    expression ASC,
    payload
);

-- +goose Down
DROP INDEX v1_filter_unique_tenant_workflow_id_scope_expression_payload;
CREATE UNIQUE INDEX CONCURRENTLY v1_filter_unique_idx ON v1_filter (
    tenant_id ASC,
    workflow_id ASC,
    scope ASC,
    expression ASC
);
