-- +goose Up
-- +goose StatementBegin
CREATE TYPE v1_payload_type_olap AS ENUM ('TASK_INPUT', 'DAG_INPUT', 'TASK_OUTPUT', 'TASK_EVENT_DATA');
CREATE TYPE v1_payload_location_olap AS ENUM ('INLINE', 'EXTERNAL');

CREATE TABLE v1_payloads_olap (
    tenant_id UUID NOT NULL,
    id BIGINT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL,
    type v1_payload_type_olap NOT NULL,
    location v1_payload_location_olap NOT NULL,
    external_location_key TEXT,
    inline_content JSONB,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, inserted_at, id, type),
    CHECK (
        location = 'INLINE'
        OR
        (location = 'EXTERNAL' AND inline_content IS NULL AND external_location_key IS NOT NULL)
    )
) PARTITION BY RANGE(inserted_at);

SELECT create_v1_range_partition('v1_payloads_olap'::TEXT, NOW()::DATE);
SELECT create_v1_range_partition('v1_payloads_olap'::TEXT, (NOW() + INTERVAL '1 day')::DATE);
SELECT create_v1_range_partition('v1_payloads_olap'::TEXT, (NOW() + INTERVAL '2 day')::DATE);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_payloads_olap;

DROP TYPE v1_payload_location_olap;
DROP TYPE v1_payload_type_olap;
-- +goose StatementEnd
