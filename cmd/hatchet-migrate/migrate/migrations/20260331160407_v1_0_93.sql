-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_log_line
    ADD COLUMN workflow_id UUID,
    ADD COLUMN step_id UUID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_log_line
    DROP COLUMN workflow_id,
    DROP COLUMN step_id;
-- +goose StatementEnd
