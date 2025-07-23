-- +goose Up
-- +goose StatementBegin
CREATE TYPE v1_payload_type AS ENUM ('WORKFLOW_INPUT', 'TASK_OUTPUT');

CREATE TABLE v1_payload (
    tenant_id UUID NOT NULL,
    key TEXT NOT NULL,
    type v1_payload_type NOT NULL,
    value JSONB NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, key, type)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_payload;
DROP TYPE v1_payload_type;
-- +goose StatementEnd
