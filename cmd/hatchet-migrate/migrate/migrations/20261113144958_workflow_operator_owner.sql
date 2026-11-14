-- +goose Up
ALTER TABLE "APIToken" ADD COLUMN "operatorId" UUID;
ALTER TABLE "Workflow" ADD COLUMN "operatorId" UUID;

-- +goose Down
ALTER TABLE "Workflow" DROP COLUMN "operatorId";
ALTER TABLE "APIToken" DROP COLUMN "operatorId";
