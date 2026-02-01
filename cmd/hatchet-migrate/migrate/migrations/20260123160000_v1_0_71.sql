-- +goose Up
-- Some deployments were created with a "TenantResourceLimitAlert" table missing the "resource" column.
-- The ticker poll query expects this column to exist (it inserts into it), so ensure it exists and backfill it.
DO $$
BEGIN
    -- If the table doesn't exist, there's nothing to do.
    IF to_regclass('"TenantResourceLimitAlert"') IS NULL THEN
        RETURN;
    END IF;

    -- Add the column if missing.
    IF NOT EXISTS (
        SELECT 1
        FROM pg_attribute
        WHERE attrelid = to_regclass('"TenantResourceLimitAlert"')
          AND attname = 'resource'
          AND NOT attisdropped
    ) THEN
        EXECUTE 'ALTER TABLE "TenantResourceLimitAlert" ADD COLUMN "resource" "LimitResource"';
    END IF;

    -- Backfill any missing values from the referenced resource limit.
    IF to_regclass('"TenantResourceLimit"') IS NOT NULL THEN
        EXECUTE '
            UPDATE "TenantResourceLimitAlert" trla
            SET "resource" = trl."resource"
            FROM "TenantResourceLimit" trl
            WHERE trla."resource" IS NULL
              AND trla."resourceLimitId" = trl."id"
        ';
    END IF;

    -- Only enforce NOT NULL if everything is populated.
    IF NOT EXISTS (SELECT 1 FROM "TenantResourceLimitAlert" WHERE "resource" IS NULL) THEN
        EXECUTE 'ALTER TABLE "TenantResourceLimitAlert" ALTER COLUMN "resource" SET NOT NULL';
    END IF;
END $$;

