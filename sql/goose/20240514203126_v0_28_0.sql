-- +goose Up
-- AlterEnum
ALTER TYPE "StepRunEventReason" ADD VALUE 'TIMEOUT_REFRESHED';
