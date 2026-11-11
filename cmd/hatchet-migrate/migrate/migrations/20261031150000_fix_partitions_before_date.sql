-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION get_v1_partitions_before_date(
    targetTableName text,
    targetDate date
) RETURNS TABLE(partition_name text)
    LANGUAGE plpgsql AS
$$
BEGIN
    RETURN QUERY
    SELECT
        partition_bounds.child_name
    FROM (
        SELECT
            i.inhrelid::regclass::text AS child_name,
            (regexp_match(pg_get_expr(c.relpartbound, c.oid), 'TO \(''([^'']+)''\)'))[1] AS upper_bound
        FROM
            pg_inherits i
        JOIN
            pg_class c ON c.oid = i.inhrelid
        WHERE
            i.inhparent = targetTableName::regclass
            AND substring(i.inhrelid::regclass::text, format('%s_(\d{8})', targetTableName)) ~ '^\d{8}'
    ) partition_bounds
    WHERE
        partition_bounds.upper_bound::timestamptz <= targetDate::timestamptz;
END;
$$;

CREATE OR REPLACE FUNCTION get_v1_weekly_partitions_before_date(
    targetTableName text,
    targetDate date
) RETURNS TABLE(partition_name text)
    LANGUAGE plpgsql AS
$$
BEGIN
    RETURN QUERY
    SELECT * FROM get_v1_partitions_before_date(targetTableName, targetDate);
END;
$$;

CREATE OR REPLACE FUNCTION get_v1_monthly_partitions_before_date(
    targetTableName text,
    targetDate date
) RETURNS TABLE(partition_name text)
    LANGUAGE plpgsql AS
$$
BEGIN
    RETURN QUERY
    SELECT * FROM get_v1_partitions_before_date(targetTableName, targetDate);
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION get_v1_partitions_before_date(
    targetTableName text,
    targetDate date
) RETURNS TABLE(partition_name text)
    LANGUAGE plpgsql AS
$$
BEGIN
    RETURN QUERY
    SELECT
        inhrelid::regclass::text AS partition_name
    FROM
        pg_inherits
    WHERE
        inhparent = targetTableName::regclass
        AND substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName)) ~ '^\d{8}'
        AND (substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName))::date) < targetDate;
END;
$$;

CREATE OR REPLACE FUNCTION get_v1_weekly_partitions_before_date(
    targetTableName text,
    targetDate date
) RETURNS TABLE(partition_name text)
    LANGUAGE plpgsql AS
$$
BEGIN
    RETURN QUERY
    SELECT
        inhrelid::regclass::text AS partition_name
    FROM
        pg_inherits
    WHERE
        inhparent = targetTableName::regclass
        AND substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName)) ~ '^\d{8}'
        AND (substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName))::date) < targetDate
        AND (substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName))::date) < NOW() - INTERVAL '1 week'
    ;
END;
$$;

CREATE OR REPLACE FUNCTION get_v1_monthly_partitions_before_date(
    targetTableName text,
    targetDate date
) RETURNS TABLE(partition_name text)
    LANGUAGE plpgsql AS
$$
BEGIN
    RETURN QUERY
    SELECT
        inhrelid::regclass::text AS partition_name
    FROM
        pg_inherits
    WHERE
        inhparent = targetTableName::regclass
        AND substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName)) ~ '^\d{8}'
        AND (substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName))::date + INTERVAL '1 month') <= targetDate
    ;
END;
$$;
-- +goose StatementEnd
