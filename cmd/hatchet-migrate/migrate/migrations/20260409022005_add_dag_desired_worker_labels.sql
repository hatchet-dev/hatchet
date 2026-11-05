-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_dag ADD COLUMN desired_worker_labels JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_dag DROP COLUMN desired_worker_labels;
-- +goose StatementEnd
