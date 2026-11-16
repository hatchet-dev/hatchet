-- +goose Up
-- +goose NO TRANSACTION
CREATE INDEX CONCURRENTLY IF NOT EXISTS ix_user_session_cre_at_exp_at ON "UserSession" ("createdAt") INCLUDE ("expiresAt");

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX CONCURRENTLY IF EXISTS ix_user_session_cre_at_exp_at;
