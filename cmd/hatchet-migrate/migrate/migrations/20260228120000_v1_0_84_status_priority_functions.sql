-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_status_priority(s v1_readable_status_olap)
RETURNS int IMMUTABLE LANGUAGE sql AS $$
    SELECT CASE s
        WHEN 'QUEUED'    THEN 1
        WHEN 'RUNNING'   THEN 2
        WHEN 'EVICTED'   THEN 3
        WHEN 'CANCELLED' THEN 4
        WHEN 'FAILED'    THEN 5
        WHEN 'COMPLETED' THEN 6
    END;
$$;

CREATE OR REPLACE FUNCTION v1_status_from_priority(p int)
RETURNS v1_readable_status_olap IMMUTABLE LANGUAGE sql AS $$
    SELECT CASE p
        WHEN 1 THEN 'QUEUED'
        WHEN 2 THEN 'RUNNING'
        WHEN 3 THEN 'EVICTED'
        WHEN 4 THEN 'CANCELLED'
        WHEN 5 THEN 'FAILED'
        WHEN 6 THEN 'COMPLETED'
    END::v1_readable_status_olap;
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS v1_status_from_priority(int);
DROP FUNCTION IF EXISTS v1_status_priority(v1_readable_status_olap);
