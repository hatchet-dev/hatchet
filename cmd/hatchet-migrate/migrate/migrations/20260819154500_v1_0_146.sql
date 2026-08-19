-- +goose Up
-- +goose StatementBegin
-- NOTE: no-op so v1.0.145 (canViewPayloads) is penultimate. load-online-migrate
-- seeds against N-1, and HEAD already writes that column.
SELECT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
