-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_events_olap (
    tenant_id UUID NOT NULL,
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    generated_at TIMESTAMPTZ NOT NULL,
    external_id UUID NOT NULL,
    key TEXT NOT NULL,
    payload JSONB NOT NULL,
    additional_metadata JSONB,

    PRIMARY KEY (tenant_id, id, inserted_at)
) PARTITION BY RANGE(inserted_at);

CREATE INDEX v1_events_olap_external_id_idx ON v1_events_olap (tenant_id, external_id);

CREATE TABLE v1_event_to_run_olap (
    run_id BIGINT NOT NULL,
    run_inserted_at TIMESTAMPTZ NOT NULL,
    event_id BIGINT NOT NULL,
    event_inserted_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (event_id, event_inserted_at, run_id, run_inserted_at)
) PARTITION BY RANGE(event_inserted_at);

SELECT create_v1_range_partition('v1_events_olap', DATE 'today');
SELECT create_v1_range_partition('v1_event_to_run_olap', DATE 'today');


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_event_to_run_olap;
DROP TABLE v1_events_olap;
-- +goose StatementEnd
