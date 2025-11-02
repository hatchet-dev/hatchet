-- name: CheckPGStatStatementsEnabled :one
SELECT COUNT(*) FROM pg_available_extensions WHERE name = 'pg_stat_statements';

-- name: CheckBloat :many
SELECT
    schemaname,
    relname AS tablename,
    pg_size_pretty(pg_total_relation_size(quote_ident(schemaname)||'.'||quote_ident(relname))) AS total_size,
    pg_size_pretty(pg_relation_size(quote_ident(schemaname)||'.'||quote_ident(relname))) AS table_size,
    n_live_tup AS live_tuples,
    n_dead_tup AS dead_tuples,
    ROUND(100 * n_dead_tup::numeric / NULLIF(n_live_tup, 0), 2) AS dead_pct,
    last_vacuum,
    last_autovacuum,
    last_analyze
FROM
    pg_stat_user_tables
WHERE
    n_dead_tup > 1000 
    AND n_live_tup > 1000 
    AND ROUND(100 * n_dead_tup::numeric / NULLIF(n_live_tup, 0), 2) > 50 
    AND relname NOT IN (
        'Lease'
        )
ORDER BY
    dead_pct DESC NULLS LAST;

-- name: CheckLongRunningQueries :many
SELECT 
    pid,
    usename,
    application_name,
    client_addr,
    state,
    now() - query_start AS duration,
    query
FROM 
    pg_stat_activity
WHERE 
    state = 'active'
    AND now() - query_start > interval '5 minutes'
    AND query NOT LIKE 'autovacuum:%'
    AND query NOT LIKE '%pg_stat_activity%'
ORDER BY 
    query_start;

-- name: CheckQueryCaches :many
SELECT 
    schemaname,
    relname AS tablename,
    heap_blks_read,
    heap_blks_hit,
    heap_blks_hit + heap_blks_read AS total_reads,
    ROUND(
        100.0 * heap_blks_hit / NULLIF(heap_blks_hit + heap_blks_read, 0),
        2
    )::float AS cache_hit_ratio_pct,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||relname)) AS total_size
FROM 
    pg_statio_user_tables
WHERE 
    heap_blks_read + heap_blks_hit > 0
ORDER BY 
    heap_blks_read DESC
LIMIT 20;

-- name: LongRunningVacuum :many
SELECT 
    p.pid,
    p.relid::regclass AS table_name,
    pg_size_pretty(pg_total_relation_size(p.relid)) AS table_size,
    p.phase,
    p.heap_blks_total AS total_blocks,
    p.heap_blks_scanned AS scanned_blocks,
    ROUND(100.0 * p.heap_blks_scanned / NULLIF(p.heap_blks_total, 0), 2) AS pct_scanned,
    p.heap_blks_vacuumed AS vacuumed_blocks,
    p.index_vacuum_count,
    now() - a.query_start AS elapsed_time,
    CASE 
        WHEN p.heap_blks_scanned > 0 THEN
            ((now() - a.query_start) * (p.heap_blks_total - p.heap_blks_scanned) / p.heap_blks_scanned)
        ELSE interval '0'
    END AS estimated_time_remaining,
    a.wait_event_type,
    a.wait_event,
    a.query
FROM 
    pg_stat_progress_vacuum p
    JOIN pg_stat_activity a ON p.pid = a.pid
WHERE 
    now() - a.query_start > interval '3 hours'
ORDER BY 
    a.query_start;