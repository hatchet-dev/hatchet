-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_failed_run_alerts_olap (
    tenant_id UUID NOT NULL,
    run_id BIGINT NOT NULL,
    run_inserted_at TIMESTAMPTZ NOT NULL,
    kind v1_run_kind NOT NULL,
    external_id UUID NOT NULL,
    failed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, run_inserted_at, run_id)
);

CREATE INDEX v1_failed_run_alerts_olap_tenant_id_failed_at_idx ON v1_failed_run_alerts_olap (tenant_id, failed_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS v1_failed_run_alerts_olap;
-- +goose StatementEnd
