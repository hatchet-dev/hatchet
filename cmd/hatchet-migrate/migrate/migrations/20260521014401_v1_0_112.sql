-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_payload_offloaded_block_index (
    payload_inserted_at_date DATE NOT NULL,
    block_lower_external_id_bound UUID NOT NULL,
    block_upper_external_id_bound UUID NOT NULL,
    index_file_key TEXT NOT NULL,
    PRIMARY KEY (payload_inserted_at_date, block_lower_external_id_bound, block_upper_external_id_bound)
) PARTITION BY RANGE (payload_inserted_at_date);

SELECT create_v1_range_partition('v1_payload_offloaded_block_index', NOW()::DATE);
SELECT create_v1_range_partition('v1_payload_offloaded_block_index', (NOW() - INTERVAL '1 day')::DATE);
SELECT create_v1_range_partition('v1_payload_offloaded_block_index', (NOW() + INTERVAL '1 day')::DATE);
SELECT create_v1_range_partition('v1_payload_offloaded_block_index', (NOW() + INTERVAL '2 day')::DATE);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
