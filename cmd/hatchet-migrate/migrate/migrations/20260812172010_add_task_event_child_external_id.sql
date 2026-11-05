-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task_event ADD COLUMN child_external_id UUID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task_event DROP COLUMN child_external_id;
-- +goose StatementEnd
