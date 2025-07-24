-- +goose Up
-- +goose StatementBegin
CREATE TYPE v1_payload_type AS ENUM ('WORKFLOW_INPUT', 'TASK_OUTPUT');

CREATE TABLE v1_payload (
    tenant_id UUID NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    type v1_payload_type NOT NULL,
    value JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, task_inserted_at, task_id, type)
) PARTITION BY RANGE(task_inserted_at);

SELECT create_v1_range_partition('v1_payload'::TEXT, NOW()::DATE);
SELECT create_v1_range_partition('v1_payload'::TEXT, (NOW() + INTERVAL '1 day')::DATE);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_payload;
DROP TYPE v1_payload_type;
-- +goose StatementEnd
