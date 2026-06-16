-- +goose no transaction
-- +goose Up
-- Backs the retention cron's `DELETE FROM "UserSession" WHERE "expiresAt" < NOW()`.
-- Without it each cleanup batch sequentially scans the table (see #3913).
CREATE INDEX CONCURRENTLY IF NOT EXISTS "UserSession_expiresAt_idx" ON "UserSession" ("expiresAt");

-- +goose Down
DROP INDEX IF EXISTS "UserSession_expiresAt_idx";
