-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_log_line ADD COLUMN workflow_id UUID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_log_line DROP COLUMN workflow_id;
-- +goose StatementEnd
