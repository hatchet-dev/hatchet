-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_tenant_control (
    tenant_id UUID NOT NULL REFERENCES "Tenant" ("id") ON DELETE CASCADE,
    ingest_per_second INTEGER NOT NULL CHECK (ingest_per_second >= 1),
    reason TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT v1_tenant_control_pkey PRIMARY KEY (tenant_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_tenant_control;
-- +goose StatementEnd
