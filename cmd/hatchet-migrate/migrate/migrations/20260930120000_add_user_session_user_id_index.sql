-- +goose Up
-- +goose NO TRANSACTION
CREATE INDEX CONCURRENTLY IF NOT EXISTS ix_user_session_user_id ON "UserSession" ("userId") WHERE "userId" IS NOT NULL;

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX CONCURRENTLY IF EXISTS ix_user_session_user_id;
