-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION convert_duration_to_interval(duration text) RETURNS interval AS $$
DECLARE
    num_value INT;
BEGIN
    num_value := substring(duration from '^\d+');

    RETURN CASE
        WHEN duration LIKE '%ms' THEN make_interval(secs => num_value::float / 1000)
        WHEN duration LIKE '%s' THEN make_interval(secs => num_value)
        WHEN duration LIKE '%m' THEN make_interval(mins => num_value)
        WHEN duration LIKE '%h' THEN make_interval(hours => num_value)
        WHEN duration LIKE '%d' THEN make_interval(days => num_value)
        WHEN duration LIKE '%w' THEN make_interval(days => num_value * 7)
        WHEN duration LIKE '%y' THEN make_interval(months => num_value * 12)
        ELSE '5 minutes'::interval
    END;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
