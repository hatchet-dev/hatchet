--  these are stub views to satisfy the sqlc generator for pg features

CREATE VIEW pg_stat_user_tables AS
SELECT
    NULL::oid AS relid,
    NULL::name AS schemaname,
    NULL::name AS relname,
    NULL::bigint AS seq_scan,
    NULL::bigint AS seq_tup_read,
    NULL::bigint AS idx_scan,
    NULL::bigint AS idx_tup_fetch,
    NULL::bigint AS n_tup_ins,
    NULL::bigint AS n_tup_upd,
    NULL::bigint AS n_tup_del,
    NULL::bigint AS n_tup_hot_upd,
    NULL::bigint AS n_live_tup,
    NULL::bigint AS n_dead_tup,
    NULL::bigint AS n_mod_since_analyze,
    NULL::bigint AS n_ins_since_vacuum,
    NULL::bigint AS vacuum_count,
    NULL::bigint AS autovacuum_count,
    NULL::bigint AS analyze_count,
    NULL::bigint AS autoanalyze_count,
    NULL::timestamp with time zone AS last_vacuum,
    NULL::timestamp with time zone AS last_autovacuum,
    NULL::timestamp with time zone AS last_analyze,
    NULL::timestamp with time zone AS last_autoanalyze
WHERE
    false;

CREATE VIEW pg_statio_user_tables AS
SELECT
    NULL::oid AS relid,
    NULL::name AS schemaname,
    NULL::name AS relname,
    NULL::bigint AS heap_blks_read,
    NULL::bigint AS heap_blks_hit,
    NULL::bigint AS idx_blks_read,
    NULL::bigint AS idx_blks_hit,
    NULL::bigint AS toast_blks_read,
    NULL::bigint AS toast_blks_hit,
    NULL::bigint AS tidx_blks_read,
    NULL::bigint AS tidx_blks_hit
WHERE
    false;

CREATE VIEW pg_available_extensions AS
SELECT
    NULL::name AS name,
    NULL::text AS default_version,
    NULL::text AS installed_version,
    NULL::text AS comment
WHERE
    false;

CREATE VIEW pg_stat_statements AS
SELECT
    NULL::oid AS userid,
    NULL::oid AS dbid,
    NULL::bigint AS queryid,
    NULL::text AS query,
    NULL::bigint AS calls,
    NULL::double precision AS total_exec_time,
    NULL::bigint AS rows,
    NULL::bigint AS shared_blks_hit,
    NULL::bigint AS shared_blks_read,
    NULL::bigint AS shared_blks_dirtied,
    NULL::bigint AS shared_blks_written,
    NULL::bigint AS local_blks_hit,
    NULL::bigint AS local_blks_read,
    NULL::bigint AS local_blks_dirtied,
    NULL::bigint AS local_blks_written,
    NULL::bigint AS temp_blks_read,
    NULL::bigint AS temp_blks_written,
    NULL::double precision AS blk_read_time,
    NULL::double precision AS blk_write_time
WHERE
    false;

CREATE VIEW pg_stat_progress_vacuum AS
SELECT
    NULL::integer AS pid,
    NULL::oid AS datid,
    NULL::name AS datname,
    NULL::oid AS relid,
    NULL::text AS phase,
    NULL::bigint AS heap_blks_total,
    NULL::bigint AS heap_blks_scanned,
    NULL::bigint AS heap_blks_vacuumed,
    NULL::bigint AS heap_blks_frozen,
    NULL::bigint AS index_vacuum_count,
    NULL::bigint AS max_dead_tuples,
    NULL::bigint AS num_dead_tuples
WHERE
    false;

CREATE VIEW pg_stat_activity AS
SELECT
    NULL::integer AS pid,
    NULL::name AS usename,
    NULL::text AS application_name,
    NULL::inet AS client_addr,
    NULL::text AS state,
    NULL::timestamp with time zone AS query_start,
    NULL::text AS wait_event_type,
    NULL::text AS wait_event,
    NULL::text AS query
WHERE
    false;
