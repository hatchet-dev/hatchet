-- +goose Up
-- +goose StatementBegin
-- kept apart from "RateLimit" so a key can be paused before its first run and upserts can't clear a pause
CREATE TABLE v1_rate_limit_pause (
    tenant_id UUID NOT NULL,
    key TEXT NOT NULL,
    -- the key's window is frozen from paused_at until paused_until, and shifted by that span once the pause ends
    paused_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paused_until TIMESTAMPTZ NOT NULL,
    CONSTRAINT v1_rate_limit_pause_pkey PRIMARY KEY (tenant_id, key)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_rate_limit_pause;
-- +goose StatementEnd
