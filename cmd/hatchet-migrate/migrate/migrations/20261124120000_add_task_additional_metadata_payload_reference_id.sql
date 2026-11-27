-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task ADD COLUMN additional_metadata_payload_reference_id UUID;
ALTER TYPE v1_payload_type ADD VALUE IF NOT EXISTS 'TASK_ADDITIONAL_METADATA';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task DROP COLUMN additional_metadata_payload_reference_id;
-- +goose StatementEnd
