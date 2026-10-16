-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task ADD COLUMN payload_reference_id UUID;
ALTER TABLE v1_dag ADD COLUMN payload_reference_id UUID;
ALTER TABLE v1_event ADD COLUMN payload_reference_id UUID;
ALTER TABLE v1_task_event ADD COLUMN payload_reference_id UUID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task DROP COLUMN payload_reference_id;
ALTER TABLE v1_dag DROP COLUMN payload_reference_id;
ALTER TABLE v1_event DROP COLUMN payload_reference_id;
ALTER TABLE v1_task_event DROP COLUMN payload_reference_id;
-- +goose StatementEnd
