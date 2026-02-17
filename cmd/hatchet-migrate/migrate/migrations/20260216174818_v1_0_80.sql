-- +goose Up
-- +goose StatementBegin
DROP FUNCTION IF EXISTS create_v1_range_partition(text, date);
CREATE OR REPLACE FUNCTION create_v1_range_partition(
    targetTableName text,
    targetDate date,
    fillfactor integer DEFAULT 100
) RETURNS integer
    LANGUAGE plpgsql AS
$$
DECLARE
    targetDateStr varchar;
    targetDatePlusOneDayStr varchar;
    newTableName varchar;
BEGIN
    SELECT to_char(targetDate, 'YYYYMMDD') INTO targetDateStr;
    SELECT to_char(targetDate + INTERVAL '1 day', 'YYYYMMDD') INTO targetDatePlusOneDayStr;
    SELECT lower(format('%s_%s', targetTableName, targetDateStr)) INTO newTableName;
    -- exit if the table exists
    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = newTableName) THEN
        RETURN 0;
    END IF;

    EXECUTE
        format('CREATE TABLE %s (LIKE %s INCLUDING INDEXES INCLUDING CONSTRAINTS)', newTableName, targetTableName);
    EXECUTE format('ALTER TABLE %I SET (
            fillfactor = %s,
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor=''0.05'',
            autovacuum_vacuum_threshold=''25'',
            autovacuum_analyze_threshold=''25'',
            autovacuum_vacuum_cost_delay=''10'',
            autovacuum_vacuum_cost_limit=''1000''
        )', newTableName, fillfactor);
    EXECUTE
        format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneDayStr);
    RETURN 1;
END;
$$;

-- v1_durable_event_log represents the log file for the durable event history
-- of a durable task. This table stores metadata like sequence values for entries.
--
-- Important: writers to v1_durable_event_log_entry should lock this row to increment the sequence value.
CREATE TABLE v1_durable_event_log_file (
    tenant_id UUID NOT NULL,
    -- The id and inserted_at of the durable task which created this entry
    durable_task_id BIGINT NOT NULL,
    durable_task_inserted_at TIMESTAMPTZ NOT NULL,

    latest_inserted_at TIMESTAMPTZ NOT NULL,

    latest_invocation_count BIGINT NOT NULL,

    -- A monotonically increasing node id for this durable event log scoped to the durable task.
    -- Starts at 0 and increments by 1 for each new entry.
    latest_node_id BIGINT NOT NULL,
    -- The latest branch id. Branches represent different execution paths on a replay.
    latest_branch_id BIGINT NOT NULL,
    -- The parent node id which should be linked to the first node in a new branch to its parent node.
    latest_branch_first_parent_node_id BIGINT NOT NULL,

    CONSTRAINT v1_durable_event_log_file_pkey PRIMARY KEY (durable_task_id, durable_task_inserted_at)
) PARTITION BY RANGE(durable_task_inserted_at);

SELECT create_v1_range_partition('v1_durable_event_log_file', NOW()::DATE);
SELECT create_v1_range_partition('v1_durable_event_log_file', (NOW() + INTERVAL '1 day')::DATE);

CREATE TYPE v1_durable_event_log_kind AS ENUM (
    'RUN',
    'WAIT_FOR',
    'MEMO'
);

CREATE TABLE v1_durable_event_log_entry (
    tenant_id UUID NOT NULL,
    -- need an external id for consistency with the payload store logic (unfortunately)
    external_id UUID NOT NULL,
    -- The id and inserted_at of the durable task which created this entry
    -- The inserted_at time of this event from a DB clock perspective.
    -- Important: for consistency, this should always be auto-generated via the CURRENT_TIMESTAMP!
    inserted_at TIMESTAMPTZ NOT NULL,
    id BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY,

    durable_task_id BIGINT NOT NULL,
    durable_task_inserted_at TIMESTAMPTZ NOT NULL,

    kind v1_durable_event_log_kind NOT NULL,
    -- The node number in the durable event log. This represents a monotonically increasing
    -- sequence value generated from v1_durable_event_log_file.latest_node_id
    node_id BIGINT NOT NULL,
    -- The parent node id for this event, if any. This can be null.
    parent_node_id BIGINT,
    -- The branch id when this event was first seen. A durable event log can be a part of many branches.
    branch_id BIGINT NOT NULL,
    -- Todo: Associated data for this event should be stored in the v1_payload table!
    -- data JSONB,
    -- The hash of the data stored in the v1_payload table to check non-determinism violations.
    -- This can be null for event types that don't have associated data.
    -- TODO: we can add CHECK CONSTRAINT for event types that require data_hash to be non-null.
    data_hash BYTEA,
    -- Can discuss: adds some flexibility for future hash algorithms
    data_hash_alg TEXT,
    -- Access patterns:
    -- Definite: we'll query directly for the node_id when a durable task is replaying its log
    -- Possible: we may want to query a range of node_ids for a durable task
    -- Possible: we may want to query a range of inserted_ats for a durable task

    -- Whether this event has been seen by the engine or not. Note that is_satisfied _may_ change multiple
    -- times through the lifecycle of a event, and readers should not assume that once it's true it will always be true.
    is_satisfied BOOLEAN NOT NULL DEFAULT FALSE,

    CONSTRAINT v1_durable_event_log_entry_pkey PRIMARY KEY (durable_task_id, durable_task_inserted_at, node_id)
) PARTITION BY RANGE(durable_task_inserted_at);

SELECT create_v1_range_partition('v1_durable_event_log_entry', NOW()::DATE, 80);
SELECT create_v1_range_partition('v1_durable_event_log_entry', (NOW() + INTERVAL '1 day')::DATE, 80);

ALTER TABLE v1_match
    ADD COLUMN durable_event_log_entry_durable_task_external_id UUID,
    ADD COLUMN durable_event_log_entry_node_id BIGINT,
    ADD COLUMN durable_event_log_entry_durable_task_id BIGINT,
    ADD COLUMN durable_event_log_entry_durable_task_inserted_at TIMESTAMPTZ;

ALTER TYPE v1_payload_type ADD VALUE IF NOT EXISTS 'DURABLE_EVENT_LOG_ENTRY_DATA';
ALTER TYPE v1_payload_type ADD VALUE IF NOT EXISTS 'DURABLE_EVENT_LOG_ENTRY_RESULT_DATA';

ALTER TABLE "Worker" ADD COLUMN "durableTaskDispatcherId" UUID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_durable_event_log_entry;
DROP TABLE v1_durable_event_log_file;
DROP TYPE v1_durable_event_log_kind;

ALTER TABLE v1_match
    DROP COLUMN durable_event_log_entry_durable_task_external_id,
    DROP COLUMN durable_event_log_entry_node_id,
    DROP COLUMN durable_event_log_entry_durable_task_id,
    DROP COLUMN durable_event_log_entry_durable_task_inserted_at;

ALTER TABLE "Worker" DROP COLUMN "durableTaskDispatcherId";

DROP FUNCTION IF EXISTS create_v1_range_partition(text, date, integer);
CREATE OR REPLACE FUNCTION create_v1_range_partition(
    targetTableName text,
    targetDate date
) RETURNS integer
    LANGUAGE plpgsql AS
$$
DECLARE
    targetDateStr varchar;
    targetDatePlusOneDayStr varchar;
    newTableName varchar;
BEGIN
    SELECT to_char(targetDate, 'YYYYMMDD') INTO targetDateStr;
    SELECT to_char(targetDate + INTERVAL '1 day', 'YYYYMMDD') INTO targetDatePlusOneDayStr;
    SELECT lower(format('%s_%s', targetTableName, targetDateStr)) INTO newTableName;
    -- exit if the table exists
    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = newTableName) THEN
        RETURN 0;
    END IF;

    EXECUTE
        format('CREATE TABLE %s (LIKE %s INCLUDING INDEXES INCLUDING CONSTRAINTS)', newTableName, targetTableName);
    EXECUTE
        format('ALTER TABLE %s SET (
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor=''0.05'',
            autovacuum_vacuum_threshold=''25'',
            autovacuum_analyze_threshold=''25'',
            autovacuum_vacuum_cost_delay=''10'',
            autovacuum_vacuum_cost_limit=''1000''
        )', newTableName);
    EXECUTE
        format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneDayStr);
    RETURN 1;
END;
$$;
-- +goose StatementEnd
