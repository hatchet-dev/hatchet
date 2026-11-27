-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_tenant_backlog_hold (
    tenant_id UUID PRIMARY KEY,
    backlog BIGINT NOT NULL,
    backlog_limit BIGINT NOT NULL,
    largest_state TEXT NOT NULL,
    held_until TIMESTAMPTZ NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_tenant_backlog_hold;
-- +goose StatementEnd
