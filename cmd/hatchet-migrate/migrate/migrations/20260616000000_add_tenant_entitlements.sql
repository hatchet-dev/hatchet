-- +goose Up
-- +goose StatementBegin
CREATE TABLE tenant_entitlement (
    tenant_id UUID NOT NULL,

    audit_logs BOOLEAN NOT NULL DEFAULT FALSE,

    prometheus_metrics BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT tenant_entitlement_pkey PRIMARY KEY (tenant_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE tenant_entitlement;
-- +goose StatementEnd
