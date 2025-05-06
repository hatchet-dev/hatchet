-- +goose Up
-- +goose StatementBegin
CREATE TABLE v1_events_olap (
    tenant_id UUID NOT NULL,
    id UUID NOT NULL,
    seen_at TIMESTAMPTZ NOT NULL,
    key TEXT NOT NULL,
    payload JSONB NOT NULL,
    additional_metadata JSONB,

    PRIMARY KEY (tenant_id, id, seen_at)
) PARTITION BY RANGE(seen_at);

CREATE TABLE v1_event_to_run_olap (
    run_id BIGINT NOT NULL,
    run_inserted_at TIMESTAMPTZ NOT NULL,
    event_id UUID NOT NULL,
    event_seen_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (event_id, event_seen_at, run_id, run_inserted_at)
) PARTITION BY RANGE(event_seen_at);


SELECT create_v1_range_partition('v1_events_olap', DATE 'today');
SELECT create_v1_range_partition('v1_event_to_run_olap', DATE 'today');


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_event_to_run_olap;
DROP TABLE v1_events_olap;
-- +goose StatementEnd
