-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task ADD COLUMN max_reassignments_allowed INTEGER;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task DROP COLUMN max_reassignments_allowed;
-- +goose StatementEnd
