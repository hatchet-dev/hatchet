-- +goose Up
-- +goose StatementBegin
CREATE TYPE v1_audit_log_actor_type AS ENUM ('user', 'api_key');

CREATE TABLE v1_audit_log (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    actor_type v1_audit_log_actor_type NOT NULL,
    actor_id UUID NOT NULL,
    action TEXT NOT NULL,
    correlation_id TEXT,
    resource_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,

    PRIMARY KEY (tenant_id, inserted_at, id)
) PARTITION BY RANGE(inserted_at);

CREATE INDEX v1_audit_log_tenant_id_correlation_id_idx ON v1_audit_log (tenant_id, correlation_id DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS v1_audit_log;
DROP TYPE IF EXISTS v1_audit_log_actor_type;
-- +goose StatementEnd
