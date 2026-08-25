-- +goose Up
-- +goose StatementBegin
ALTER TYPE v1_concurrency_strategy ADD VALUE IF NOT EXISTS 'CANCEL_EXCEPT_NEWEST';
ALTER TYPE v1_concurrency_strategy ADD VALUE IF NOT EXISTS 'CANCEL_EXCEPT_OLDEST';
-- +goose StatementEnd
