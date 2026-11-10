-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_audit_log_archive_cursor (
    partition_date DATE PRIMARY KEY,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    lease_process_id UUID NOT NULL,
    lease_expires_at TIMESTAMPTZ NOT NULL,

    next_file_index INTEGER NOT NULL,
    last_tenant_id UUID NOT NULL,
    last_inserted_at TIMESTAMPTZ NOT NULL,
    last_id UUID NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_audit_log_archive_cursor;
-- +goose StatementEnd
