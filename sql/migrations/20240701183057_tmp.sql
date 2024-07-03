-- Create enum type "AffinityComparator"
CREATE TYPE "AffinityComparator" AS ENUM ('EQUAL', 'NOT_EQUAL', 'GREATER_THAN', 'GREATER_THAN_OR_EQUAL', 'LESS_THAN', 'LESS_THAN_OR_EQUAL');
-- Create "WorkerAffinity" table
CREATE TABLE "WorkerAffinity" ("id" bigserial NOT NULL, "workerId" uuid NOT NULL, "key" text NOT NULL, "value" jsonb NULL, "comparator" "AffinityComparator" NOT NULL DEFAULT 'EQUAL', "weight" integer NOT NULL DEFAULT 100, PRIMARY KEY ("id"), CONSTRAINT "WorkerAffinity_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
