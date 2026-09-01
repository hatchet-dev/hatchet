-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS "UserOAuth_userId_key";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE UNIQUE INDEX "UserOAuth_userId_key" ON "UserOAuth" ("userId");
-- +goose StatementEnd