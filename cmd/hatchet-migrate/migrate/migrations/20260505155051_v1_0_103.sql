-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry ADD COLUMN result_payload_external_id UUID NOT NULL DEFAULT gen_random_uuid();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry DROP COLUMN result_payload_external_id;
-- +goose StatementEnd
