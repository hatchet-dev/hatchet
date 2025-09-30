-- +goose Up
-- +goose StatementBegin
ALTER TYPE v1_payload_type ADD VALUE IF NOT EXISTS 'TASK_EVENT_DATA';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- intentionally empty - can't remove value from enum
-- +goose StatementEnd
