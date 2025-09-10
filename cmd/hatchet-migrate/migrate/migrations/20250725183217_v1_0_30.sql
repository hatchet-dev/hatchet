-- +goose Up
-- +goose StatementBegin
CREATE TYPE v1_cel_evaluation_failure_source AS ENUM ('FILTER', 'WEBHOOK');

CREATE TABLE v1_cel_evaluation_failures_olap (
    id BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY,

    tenant_id UUID NOT NULL,

    source v1_cel_evaluation_failure_source NOT NULL,

    error TEXT NOT NULL,

    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (inserted_at, id)
) PARTITION BY RANGE(inserted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_cel_evaluation_failures_olap;
DROP TYPE v1_cel_evaluation_failure_source;
-- +goose StatementEnd
