-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_log_line 
    ADD COLUMN retry_count INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_log_line 
    DROP COLUMN retry_count;
-- +goose StatementEnd
