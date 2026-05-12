-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION convert_duration_to_interval(duration text) RETURNS interval AS $$
DECLARE
    rest text;
    sign_factor double precision := 1;
    total_seconds double precision := 0;
    m text[];
    val double precision;
    unit text;
    factor double precision;
    consumed integer;
BEGIN
    IF duration IS NULL OR length(duration) = 0 THEN
        RETURN '5 minutes'::interval;
    END IF;

    -- Legacy single-unit suffixes (d, w, y) keep calendar semantics for
    -- backward compatibility. Only accepted as the entire input.
    m := regexp_match(duration, '^([0-9]+)(d|w|y)$');
    IF m IS NOT NULL THEN
        val := m[1]::double precision;
        unit := m[2];
        CASE unit
            WHEN 'd' THEN RETURN make_interval(days => val::int);
            WHEN 'w' THEN RETURN make_interval(days => (val * 7)::int);
            WHEN 'y' THEN RETURN make_interval(months => (val * 12)::int);
        END CASE;
    END IF;

    rest := duration;

    IF left(rest, 1) = '-' THEN
        sign_factor := -1;
        rest := substring(rest from 2);
    ELSIF left(rest, 1) = '+' THEN
        rest := substring(rest from 2);
    END IF;

    IF length(rest) = 0 THEN
        RETURN '5 minutes'::interval;
    END IF;

    LOOP
        EXIT WHEN length(rest) = 0;

        m := regexp_match(rest, '^([0-9]+(?:\.[0-9]*)?|\.[0-9]+)(ms|s|m|h)');

        IF m IS NULL THEN
            RETURN '5 minutes'::interval;
        END IF;

        val := m[1]::double precision;
        unit := m[2];

        CASE unit
            WHEN 'ms' THEN factor := 1e-3;
            WHEN 's' THEN factor := 1;
            WHEN 'm' THEN factor := 60;
            WHEN 'h' THEN factor := 3600;
        END CASE;

        total_seconds := total_seconds + val * factor;

        consumed := length(m[1]) + length(m[2]);
        rest := substring(rest from consumed + 1);
    END LOOP;

    RETURN make_interval(secs => sign_factor * total_seconds);
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
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
