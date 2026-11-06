-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_support_grant_revocation (
    grant_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (grant_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS v1_support_grant_revocation;
-- +goose StatementEnd
