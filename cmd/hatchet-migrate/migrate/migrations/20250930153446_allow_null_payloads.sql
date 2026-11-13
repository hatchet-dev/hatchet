-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_payload DROP CONSTRAINT v1_payload_check;
ALTER TABLE v1_payload ADD CONSTRAINT v1_payload_check CHECK (
    (location = 'INLINE' AND external_location_key IS NULL)
    OR
    (location = 'EXTERNAL' AND inline_content IS NULL AND external_location_key IS NOT NULL)
) NOT VALID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_payload DROP CONSTRAINT v1_payload_check;
ALTER TABLE v1_payload ADD CONSTRAINT v1_payload_check CHECK (
    (location = 'INLINE' AND inline_content IS NOT NULL AND external_location_key IS NULL)
    OR
    (location = 'EXTERNAL' AND inline_content IS NULL AND external_location_key IS NOT NULL)
) NOT VALID;
-- +goose StatementEnd
