-- +goose Up
-- NULL means the token predates capabilities and resolves to "all", or "read-only" for read-only tokens.
ALTER TABLE "APIToken" ADD COLUMN "capabilities" TEXT[];

-- +goose Down
ALTER TABLE "APIToken" DROP COLUMN "capabilities";
