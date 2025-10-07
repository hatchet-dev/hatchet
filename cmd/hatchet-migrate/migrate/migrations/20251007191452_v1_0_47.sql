-- +goose Up
-- +goose StatementBegin
ALTER TYPE v1_payload_type ADD VALUE IF NOT EXISTS 'TASK_EVENT_DATA';
ALTER TABLE v1_task_event
    ADD COLUMN external_id UUID,
    ADD COLUMN inserted_at TIMESTAMPTZ DEFAULT NOW();
;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task_event
DROP COLUMN external_id,
DROP COLUMN inserted_at;
-- +goose StatementEnd
