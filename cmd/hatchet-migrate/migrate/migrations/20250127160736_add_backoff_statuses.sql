-- +goose Up
-- Add value to enum type: "WorkflowRunStatus"
ALTER TYPE "WorkflowRunStatus" ADD VALUE 'BACKOFF';
-- Add value to enum type: "StepRunStatus"
ALTER TYPE "StepRunStatus" ADD VALUE 'BACKOFF';
-- Add value to enum type: "JobRunStatus"
ALTER TYPE "JobRunStatus" ADD VALUE 'BACKOFF';
