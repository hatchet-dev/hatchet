-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_match ADD COLUMN existing_data JSONB;

ALTER TYPE v1_match_condition_action ADD VALUE 'CREATE_MATCH';

CREATE TYPE v1_step_match_condition_kind AS ENUM ('PARENT_OVERRIDE', 'USER_EVENT', 'SLEEP');

CREATE TABLE v1_step_match_condition (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    readable_data_key TEXT NOT NULL,
    action v1_match_condition_action NOT NULL DEFAULT 'CREATE',
    or_group_id UUID NOT NULL,
    expression TEXT,
    kind v1_step_match_condition_kind NOT NULL,
    -- If this is a SLEEP condition, this will be set to the sleep duration
    sleep_duration TEXT,
    -- If this is a USER_EVENT condition, this will be set to the user event key
    event_key TEXT,
    -- If this is a PARENT_OVERRIDE condition, this will be set to the parent readable_id
    parent_readable_id TEXT,
    PRIMARY KEY (step_id, id)
);

CREATE TABLE v1_durable_sleep (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    sleep_until TIMESTAMPTZ NOT NULL,
    sleep_duration TEXT NOT NULL,
    PRIMARY KEY (tenant_id, sleep_until, id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_match DROP COLUMN existing_data;

-- Note: Removing the enum value 'CREATE_MATCH' from v1_match_condition_action is not supported by PostgreSQL.

DROP TABLE v1_durable_sleep;
DROP TABLE v1_step_match_condition;
DROP TYPE v1_step_match_condition_kind;
-- +goose StatementEnd
