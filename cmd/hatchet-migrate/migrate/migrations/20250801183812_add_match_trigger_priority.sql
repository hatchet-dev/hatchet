-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_match
ADD COLUMN trigger_priority INTEGER;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_match
DROP COLUMN trigger_priority;
-- +goose StatementEnd
