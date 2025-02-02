-- NOTE: this is just needed for sqlc to understand the schema, this shouldn't actually be run
CREATE SCHEMA IF NOT EXISTS timescaledb_information;

CREATE TABLE IF NOT EXISTS timescaledb_information.continuous_aggregates (
    materialization_hypertable_name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS timescaledb_information.jobs (
    job_id BIGINT NOT NULL,
    hypertable_name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS timescaledb_information.job_history (
    job_id BIGINT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    succeeded BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS timescaledb_information.job_stats (
    job_id BIGINT NOT NULL,
    last_successful_finish TIMESTAMP NOT NULL
);