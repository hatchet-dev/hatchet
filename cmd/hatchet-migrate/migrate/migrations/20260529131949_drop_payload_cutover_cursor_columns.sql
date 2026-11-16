-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_payload_cutover_job_offset
    DROP COLUMN last_tenant_id,
    DROP COLUMN last_id,
    DROP COLUMN last_inserted_at,
    DROP COLUMN last_type,
    DROP COLUMN final_source_table_row_count,
    DROP COLUMN final_target_table_row_count,
    DROP COLUMN final_row_count_diff
;

ALTER TABLE v1_payloads_olap_cutover_job_offset
    DROP COLUMN last_tenant_id,
    DROP COLUMN last_inserted_at,
    DROP COLUMN final_source_table_row_count,
    DROP COLUMN final_target_table_row_count,
    DROP COLUMN final_row_count_diff
;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_payload_cutover_job_offset
    ADD COLUMN last_tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID,
    ADD COLUMN last_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN last_inserted_at TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00',
    ADD COLUMN last_type v1_payload_type NOT NULL DEFAULT 'TASK_INPUT',
    ADD COLUMN final_source_table_row_count BIGINT,
    ADD COLUMN final_target_table_row_count BIGINT,
    ADD COLUMN final_row_count_diff BIGINT
;

ALTER TABLE v1_payloads_olap_cutover_job_offset
    ADD COLUMN last_tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID,
    ADD COLUMN last_inserted_at TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00',
    ADD COLUMN final_source_table_row_count BIGINT,
    ADD COLUMN final_target_table_row_count BIGINT,
    ADD COLUMN final_row_count_diff BIGINT
;
-- +goose StatementEnd
