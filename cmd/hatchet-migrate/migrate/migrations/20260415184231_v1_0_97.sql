-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_incoming_webhook ADD COLUMN return_event_as_response_payload BOOLEAN NOT NULL DEFAULT TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_incoming_webhook DROP COLUMN return_event_as_response_payload;
-- +goose StatementEnd
