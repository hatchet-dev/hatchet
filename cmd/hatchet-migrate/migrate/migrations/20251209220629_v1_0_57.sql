-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_payload_cutover_job_offset
ADD COLUMN last_tenant_id UUID NOT NULL DEFAULT gen_random_uuid(),
ADD COLUMN last_inserted_at TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00',
ADD COLUMN last_id BIGINT NOT NULL DEFAULT 0,
ADD COLUMN last_type v1_payload_type NOT NULL DEFAULT 'TASK_INPUT',
DROP COLUMN last_offset;

ALTER TABLE v1_payloads_olap_cutover_job_offset
ADD COLUMN last_tenant_id UUID NOT NULL DEFAULT gen_random_uuid(),
ADD COLUMN last_external_id UUID NOT NULL DEFAULT gen_random_uuid(),
ADD COLUMN last_inserted_at TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00',
DROP COLUMN last_offset;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_payload_cutover_job_offset
ADD COLUMN last_offset BIGINT NOT NULL DEFAULT 0,
DROP COLUMN last_tenant_id,
DROP COLUMN last_inserted_at,
DROP COLUMN last_id,
DROP COLUMN last_type;

ALTER TABLE v1_payloads_olap_cutover_job_offset
ADD COLUMN last_offset BIGINT NOT NULL DEFAULT 0,
DROP COLUMN last_tenant_id,
DROP COLUMN last_external_id,
DROP COLUMN last_inserted_at
;
-- +goose StatementEnd
