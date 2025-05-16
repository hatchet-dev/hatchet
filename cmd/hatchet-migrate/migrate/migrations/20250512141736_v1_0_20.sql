-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_filter (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    scope TEXT NOT NULL,
    expression TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::JSONB,

    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, id)
);

CREATE UNIQUE INDEX v1_filter_unique_idx ON v1_filter (
    tenant_id ASC,
    workflow_id ASC,
    scope ASC,
    expression ASC
);


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_filter;
-- +goose StatementEnd
