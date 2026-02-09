-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS v1_task_evicted (
    task_id bigint NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    retry_count INTEGER NOT NULL,
    worker_id UUID,
    tenant_id UUID NOT NULL,
    timeout_at TIMESTAMP(3) NOT NULL,
    evicted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT v1_task_evicted_pkey PRIMARY KEY (task_id, task_inserted_at, retry_count)
);

-- TODO concurrently
CREATE INDEX IF NOT EXISTS v1_task_evicted_tenantId_timeoutAt_idx
    ON v1_task_evicted (tenant_id ASC, timeout_at ASC);

CREATE INDEX IF NOT EXISTS v1_task_evicted_tenantId_workerId_idx
    ON v1_task_evicted (tenant_id ASC, worker_id ASC)
    WHERE worker_id IS NOT NULL;

alter table v1_task_evicted set (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor='0.05',
    autovacuum_vacuum_threshold='25',
    autovacuum_analyze_threshold='25',
    autovacuum_vacuum_cost_delay='10',
    autovacuum_vacuum_cost_limit='1000'
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS v1_task_evicted_tenantId_workerId_idx;
DROP INDEX IF EXISTS v1_task_evicted_tenantId_timeoutAt_idx;
DROP TABLE IF EXISTS v1_task_evicted;

-- +goose StatementEnd
