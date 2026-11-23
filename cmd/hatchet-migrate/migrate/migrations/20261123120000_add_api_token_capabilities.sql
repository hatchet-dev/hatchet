-- +goose Up
-- NULL means the token predates capabilities and keeps access to every gRPC procedure.
ALTER TABLE "APIToken" ADD COLUMN "capabilities" TEXT[];

-- +goose Down
ALTER TABLE "APIToken" DROP COLUMN "capabilities";
