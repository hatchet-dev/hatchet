-- +goose Up
-- Add value to enum type: "InternalQueue"
ALTER TYPE "InternalQueue" ADD VALUE 'STEP_RUN_UPDATE_V2';
