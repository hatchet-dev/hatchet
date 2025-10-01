-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_payload_wal DROP COLUMN operation;

DROP TYPE v1_payload_wal_operation;
CREATE TYPE v1_payload_wal_operation AS ENUM ('REPLICATE_TO_EXTERNAL', 'CUT_OVER_TO_EXTERNAL');

ALTER TABLE v1_payload_wal ADD COLUMN operation v1_payload_wal_operation NOT NULL DEFAULT 'REPLICATE_TO_EXTERNAL';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_payload_wal DROP COLUMN operation;

DROP TYPE v1_payload_wal_operation;
CREATE TYPE v1_payload_wal_operation AS ENUM ('CREATE', 'UPDATE', 'DELETE');

ALTER TABLE v1_payload_wal ADD COLUMN operation v1_payload_wal_operation NOT NULL DEFAULT 'CREATE';
-- +goose StatementEnd
