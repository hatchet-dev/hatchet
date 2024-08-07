-- CreateEnum
CREATE TYPE "WorkflowKind" AS ENUM ('FUNCTION', 'DURABLE', 'DAG');

-- AlterTable
ALTER TABLE "WorkflowVersion" ADD COLUMN     "kind" "WorkflowKind" NOT NULL DEFAULT 'DAG';
