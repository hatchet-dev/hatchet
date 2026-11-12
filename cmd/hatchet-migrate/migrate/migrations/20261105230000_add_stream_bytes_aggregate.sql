-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS v1_agg_stream_bytes_minute_olap (
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    published_bytes BIGINT NOT NULL DEFAULT 0,
    delivered_bytes BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, bucket)
) PARTITION BY RANGE (bucket);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_agg_stream_bytes_minute_olap;
-- +goose StatementEnd
