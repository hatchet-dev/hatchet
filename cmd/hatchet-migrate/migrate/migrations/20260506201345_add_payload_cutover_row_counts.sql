-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_payloads_olap_cutover_job_offset
    ADD COLUMN final_source_table_row_count BIGINT,
    ADD COLUMN final_target_table_row_count BIGINT,
    ADD COLUMN final_row_count_diff BIGINT
;

ALTER TABLE v1_payload_cutover_job_offset
    ADD COLUMN final_source_table_row_count BIGINT,
    ADD COLUMN final_target_table_row_count BIGINT,
    ADD COLUMN final_row_count_diff BIGINT
;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_payloads_olap_cutover_job_offset
    DROP COLUMN final_source_table_row_count,
    DROP COLUMN final_target_table_row_count,
    DROP COLUMN final_row_count_diff
;

ALTER TABLE v1_payload_cutover_job_offset
    DROP COLUMN final_source_table_row_count,
    DROP COLUMN final_target_table_row_count,
    DROP COLUMN final_row_count_diff
;
-- +goose StatementEnd
