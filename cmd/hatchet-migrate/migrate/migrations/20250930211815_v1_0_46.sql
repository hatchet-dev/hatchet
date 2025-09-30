-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task_event ADD COLUMN external_id UUID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task_event DROP COLUMN external_id;
-- +goose StatementEnd
