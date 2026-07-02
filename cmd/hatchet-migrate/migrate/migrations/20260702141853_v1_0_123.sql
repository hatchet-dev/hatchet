-- +goose Up
-- +goose NO TRANSACTION

-- Ensures UserSession.createdAt and UserSession.expiresAt are indexed correctly.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "UserSession_expiresAt_idx" ON "UserSession" ("expiresAt");
CREATE INDEX CONCURRENTLY IF NOT EXISTS "UserSession_createdAt_nullUser_idx" ON "UserSession" ("createdAt") WHERE "userId" IS NULL;

-- +goose Down
-- +goose NO TRANSACTION

DROP INDEX CONCURRENTLY IF EXISTS "UserSession_expiresAt_idx";
DROP INDEX CONCURRENTLY IF EXISTS "UserSession_createdAt_nullUser_idx";
