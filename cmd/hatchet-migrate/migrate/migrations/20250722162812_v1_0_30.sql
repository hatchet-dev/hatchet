-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_payload (
    tenant_id UUID NOT NULL,
    key TEXT NOT NULL,
    value JSONB NOT NULL,

    PRIMARY KEY (tenant_id, key)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_payload;
-- +goose StatementEnd
