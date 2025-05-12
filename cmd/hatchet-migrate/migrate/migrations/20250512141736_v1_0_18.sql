-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_filter (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    workflow_version_id UUID NOT NULL,
    resource_hint TEXT NOT NULL,
    expression TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::JSONB,

    PRIMARY KEY (tenant_id, workflow_id, workflow_version_id, resource_hint, id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_filter;
-- +goose StatementEnd
