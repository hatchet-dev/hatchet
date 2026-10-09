-- +goose Up
ALTER TABLE "APIToken" ADD COLUMN "readOnly" BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE "APIToken" DROP COLUMN "readOnly";
