-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_match SET (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor = '0.05',
    autovacuum_vacuum_threshold = '25',
    autovacuum_analyze_threshold = '25',
    autovacuum_vacuum_cost_delay = '10',
    autovacuum_vacuum_cost_limit = '1000'
);

ALTER TABLE v1_match_condition SET (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor = '0.05',
    autovacuum_vacuum_threshold = '25',
    autovacuum_analyze_threshold = '25',
    autovacuum_vacuum_cost_delay = '10',
    autovacuum_vacuum_cost_limit = '1000'
);

ALTER TABLE v1_workflow_concurrency_slot SET (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor = '0.05',
    autovacuum_vacuum_threshold = '25',
    autovacuum_analyze_threshold = '25',
    autovacuum_vacuum_cost_delay = '10',
    autovacuum_vacuum_cost_limit = '1000'
);

ALTER TABLE v1_concurrency_slot SET (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor = '0.05',
    autovacuum_vacuum_threshold = '25',
    autovacuum_analyze_threshold = '25',
    autovacuum_vacuum_cost_delay = '10',
    autovacuum_vacuum_cost_limit = '1000'
);

ALTER TABLE v1_retry_queue_item SET (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor = '0.05',
    autovacuum_vacuum_threshold = '25',
    autovacuum_analyze_threshold = '25',
    autovacuum_vacuum_cost_delay = '10',
    autovacuum_vacuum_cost_limit = '1000'
);

ALTER TABLE v1_durable_sleep SET (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor = '0.05',
    autovacuum_vacuum_threshold = '25',
    autovacuum_analyze_threshold = '25',
    autovacuum_vacuum_cost_delay = '10',
    autovacuum_vacuum_cost_limit = '1000'
);

ALTER TABLE v1_task_runtime_slot SET (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor = '0.05',
    autovacuum_vacuum_threshold = '25',
    autovacuum_analyze_threshold = '25',
    autovacuum_vacuum_cost_delay = '10',
    autovacuum_vacuum_cost_limit = '1000'
);

ALTER TABLE v1_batch_runtime SET (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor = '0.05',
    autovacuum_vacuum_threshold = '25',
    autovacuum_analyze_threshold = '25',
    autovacuum_vacuum_cost_delay = '10',
    autovacuum_vacuum_cost_limit = '1000'
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_match RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_analyze_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_analyze_threshold,
    autovacuum_vacuum_cost_delay,
    autovacuum_vacuum_cost_limit
);

ALTER TABLE v1_match_condition RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_analyze_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_analyze_threshold,
    autovacuum_vacuum_cost_delay,
    autovacuum_vacuum_cost_limit
);

ALTER TABLE v1_workflow_concurrency_slot RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_analyze_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_analyze_threshold,
    autovacuum_vacuum_cost_delay,
    autovacuum_vacuum_cost_limit
);

ALTER TABLE v1_concurrency_slot RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_analyze_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_analyze_threshold,
    autovacuum_vacuum_cost_delay,
    autovacuum_vacuum_cost_limit
);

ALTER TABLE v1_retry_queue_item RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_analyze_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_analyze_threshold,
    autovacuum_vacuum_cost_delay,
    autovacuum_vacuum_cost_limit
);

ALTER TABLE v1_durable_sleep RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_analyze_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_analyze_threshold,
    autovacuum_vacuum_cost_delay,
    autovacuum_vacuum_cost_limit
);

ALTER TABLE v1_task_runtime_slot RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_analyze_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_analyze_threshold,
    autovacuum_vacuum_cost_delay,
    autovacuum_vacuum_cost_limit
);

ALTER TABLE v1_batch_runtime RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_analyze_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_analyze_threshold,
    autovacuum_vacuum_cost_delay,
    autovacuum_vacuum_cost_limit
);
-- +goose StatementEnd
