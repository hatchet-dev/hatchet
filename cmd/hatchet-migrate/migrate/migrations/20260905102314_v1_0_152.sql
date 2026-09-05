-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_payload ADD COLUMN IF NOT EXISTS inserted_at_date DATE;
ALTER TABLE v1_payload ALTER COLUMN inserted_at_date SET DEFAULT (NOW()::DATE);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'v1_payload'::regclass AND conname = 'v1_payload_inserted_at_date_not_null'
    ) THEN
        ALTER TABLE v1_payload
        ADD CONSTRAINT v1_payload_inserted_at_date_not_null CHECK (inserted_at_date IS NOT NULL) NOT VALID;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    batch_size INT := 10000;
    rows_updated INT;
BEGIN
    -- using a temp table so we only seq scan once
    CREATE TEMP TABLE v1_payload_rows_to_backfill (
        external_id UUID PRIMARY KEY
    );

    INSERT INTO v1_payload_rows_to_backfill (external_id)
    SELECT DISTINCT external_id
    FROM v1_payload
    WHERE inserted_at_date IS NULL;

    LOOP
        WITH batch AS (
            DELETE FROM v1_payload_rows_to_backfill
            WHERE external_id IN (
                SELECT external_id
                FROM v1_payload_rows_to_backfill
                ORDER BY external_id
                LIMIT batch_size
            )
            RETURNING external_id
        )

        UPDATE v1_payload p
        SET inserted_at_date = p.inserted_at::DATE
        FROM batch b
        WHERE
            p.external_id = b.external_id
            AND p.inserted_at_date IS NULL
        ;

        GET DIAGNOSTICS rows_updated = ROW_COUNT;

        EXIT WHEN rows_updated = 0;

        COMMIT;
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_payload VALIDATE CONSTRAINT v1_payload_inserted_at_date_not_null;
ALTER TABLE v1_payload ALTER COLUMN inserted_at_date SET NOT NULL;
ALTER TABLE v1_payload DROP CONSTRAINT v1_payload_inserted_at_date_not_null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_payload DROP COLUMN IF EXISTS inserted_at_date;
-- +goose StatementEnd
