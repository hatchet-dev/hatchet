-- +goose Up
-- +goose StatementBegin
-- subscribers without a cursor start after this offset (see ResetStreamTopic)
ALTER TABLE v1_stream_topic ADD COLUMN reset_offset BIGINT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_stream_topic DROP COLUMN reset_offset;
-- +goose StatementEnd
