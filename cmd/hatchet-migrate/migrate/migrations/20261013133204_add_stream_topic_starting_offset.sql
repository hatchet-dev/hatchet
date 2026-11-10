-- +goose Up
-- +goose StatementBegin
-- subscribers without a cursor start after this offset (see ResetStreamTopic)
ALTER TABLE v1_stream_topic ADD COLUMN starting_offset BIGINT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_stream_topic DROP COLUMN starting_offset;
-- +goose StatementEnd
