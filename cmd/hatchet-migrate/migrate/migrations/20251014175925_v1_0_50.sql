-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_operation_interval_settings (
    tenant_id UUID NOT NULL,
    operation_id TEXT NOT NULL,
    -- The interval represents a Go time.Duration, hence the nanoseconds
    interval_nanoseconds BIGINT NOT NULL,
    PRIMARY KEY (tenant_id, operation_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS v1_operation_interval_settings;
-- +goose StatementEnd
