-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_payload_cutover_job_offset
ADD COLUMN lease_process_id UUID NOT NULL DEFAULT gen_random_uuid(),
ADD COLUMN lease_expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_payload_cutover_job_offset
DROP COLUMN lease_expires_at,
DROP COLUMN lease_process_id;
-- +goose StatementEnd
