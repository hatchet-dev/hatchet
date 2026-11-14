-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_tenant_control (
    tenant_id UUID NOT NULL REFERENCES "Tenant" ("id") ON DELETE CASCADE,
    paused BOOLEAN NOT NULL DEFAULT FALSE,
    paused_at TIMESTAMPTZ,
    db_conn_percent INTEGER CHECK (db_conn_percent BETWEEN 1 AND 100),
    db_time_percent INTEGER CHECK (db_time_percent BETWEEN 1 AND 100),
    ingest_per_second INTEGER CHECK (ingest_per_second >= 1),
    reason TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT v1_tenant_control_pkey PRIMARY KEY (tenant_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_tenant_control;
-- +goose StatementEnd
