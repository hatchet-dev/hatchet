-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION btree_gist;

CREATE TYPE uuidrange AS RANGE (
    SUBTYPE = UUID
);

CREATE TABLE v1_payload_offloaded_block_index (
    payload_inserted_at_date DATE NOT NULL,
    block_external_id_range uuidrange NOT NULL,
    index_file_key TEXT NOT NULL,
    CONSTRAINT v1_payload_offloaded_block_index_date_range_excl
        EXCLUDE USING GIST (payload_inserted_at_date WITH =, block_external_id_range WITH &&)
);

CREATE TABLE v1_payloads_olap_offloaded_block_index (
    payload_inserted_at_date DATE NOT NULL,
    block_external_id_range uuidrange NOT NULL,
    index_file_key TEXT NOT NULL,
    CONSTRAINT v1_payloads_olap_offloaded_block_index_date_range_excl
        EXCLUDE USING GIST (payload_inserted_at_date WITH =, block_external_id_range WITH &&)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_payload_offloaded_block_index;
DROP TABLE v1_payloads_olap_offloaded_block_index;
DROP TYPE uuidrange;
DROP EXTENSION btree_gist;
-- +goose StatementEnd
