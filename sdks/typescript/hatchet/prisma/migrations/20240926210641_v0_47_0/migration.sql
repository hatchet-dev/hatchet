-- CreateEnum
CREATE TYPE "StepRateLimitKind" AS ENUM ('STATIC', 'DYNAMIC');

-- CreateEnum
CREATE TYPE "StepExpressionKind" AS ENUM ('DYNAMIC_RATE_LIMIT_KEY', 'DYNAMIC_RATE_LIMIT_VALUE', 'DYNAMIC_RATE_LIMIT_UNITS', 'DYNAMIC_RATE_LIMIT_WINDOW');

-- AlterEnum
ALTER TYPE "StepRunEventReason" ADD VALUE 'RATE_LIMIT_ERROR';

-- DropForeignKey
ALTER TABLE "StepRateLimit" DROP CONSTRAINT "StepRateLimit_stepId_fkey";

-- AlterTable
ALTER TABLE "StepRateLimit" ADD COLUMN     "kind" "StepRateLimitKind" NOT NULL DEFAULT 'STATIC';

-- CreateTable
CREATE TABLE "StepExpression" (
    "key" TEXT NOT NULL,
    "stepId" UUID NOT NULL,
    "expression" TEXT NOT NULL,
    "kind" "StepExpressionKind" NOT NULL,

    CONSTRAINT "StepExpression_pkey" PRIMARY KEY ("key","stepId","kind")
);

-- CreateTable
CREATE TABLE "StepRunExpressionEval" (
    "key" TEXT NOT NULL,
    "stepRunId" UUID NOT NULL,
    "valueStr" TEXT,
    "valueInt" INTEGER,
    "kind" "StepExpressionKind" NOT NULL,

    CONSTRAINT "StepRunExpressionEval_pkey" PRIMARY KEY ("key","stepRunId","kind")
);
