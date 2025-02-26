-- +goose Up
-- Modify "Event" table
ALTER TABLE "Event" ADD COLUMN "insertOrder" integer NULL;
