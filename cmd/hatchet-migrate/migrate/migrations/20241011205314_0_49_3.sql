-- +goose Up
-- Modify "EventKey" table
ALTER TABLE "EventKey" DROP CONSTRAINT "EventKey_pkey", ADD COLUMN "id" bigserial NOT NULL, ADD PRIMARY KEY ("id");
