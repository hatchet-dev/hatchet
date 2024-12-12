-- +goose Up
-- Modify "APIToken" table
ALTER TABLE "APIToken" ADD COLUMN "internal" boolean NOT NULL DEFAULT false;
