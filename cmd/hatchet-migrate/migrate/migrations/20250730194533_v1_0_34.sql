-- +goose Up
-- +goose StatementBegin
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
        AND targetDate < NOW() - INTERVAL '1 week'
    ;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION get_v1_weekly_partitions_before_date(
    targetTableName text,
    targetDate date
);
-- +goose StatementEnd
