-- atlas:txmode none

-- 'Creating an updatedAt index that will be useful later';
create index CONCURRENTLY IF NOT EXISTS "StepRun_updatedAt_idx" on "StepRun" ("updatedAt");
-- 'Created index';
DO $$
DECLARE
    retry_count INT := 0;
    max_retries INT := 10;
    sleep_interval INT := 5000;
    rec RECORD;
    sql_statement TEXT;
    newest_record RECORD;
BEGIN
    WHILE retry_count < max_retries LOOP
        BEGIN

            SET LOCAL lock_timeout = '30s';


            CREATE TABLE "StepRun_new" (
                "id" uuid NOT NULL,
                "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
                "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
                "deletedAt" timestamp(3),
                "tenantId" uuid NOT NULL,
                "jobRunId" uuid NOT NULL,
                "stepId" uuid NOT NULL,
                "order" BIGSERIAL,
                "workerId" uuid,
                "tickerId" uuid,
                "status" "StepRunStatus" NOT NULL DEFAULT 'PENDING'::"StepRunStatus",
                "input" jsonb,
                "output" jsonb,
                "requeueAfter" timestamp(3) without time zone,
                "scheduleTimeoutAt" timestamp(3) without time zone,
                "error" text,
                "startedAt" timestamp(3)  without time zone,
                "finishedAt" timestamp(3)  without time zone,
                "timeoutAt" timestamp(3)  without time zone,
                "cancelledAt" timestamp(3)  without time zone,
                "cancelledReason" text,
                "cancelledError" text,
                "inputSchema" jsonb,
                "callerFiles" jsonb,
                "gitRepoBranch" text,
                "retryCount" integer NOT NULL DEFAULT 0,
                "semaphoreReleased" boolean NOT NULL DEFAULT false,
                "queue" text NOT NULL DEFAULT 'default'::text,
                "priority" integer,
                PRIMARY KEY ( "status", "id")
            ) PARTITION BY LIST ("status");

            RAISE NOTICE 'Created table "StepRun_new"';


            CREATE TABLE "StepRun_volatile" PARTITION OF "StepRun_new"
            FOR VALUES IN ('PENDING','PENDING_ASSIGNMENT','ASSIGNED','RUNNING','CANCELLING')
            WITH (fillfactor = 50);

            CREATE TABLE "StepRun_stable" PARTITION OF "StepRun_new"
            FOR VALUES IN ('FAILED','CANCELLED','SUCCEEDED') WITH (fillfactor = 100);

            RAISE NOTICE 'Created partitions for "StepRun_new"';

            RAISE NOTICE 'Inserting data into "StepRun_new"';

            INSERT INTO "StepRun_new" SELECT * FROM "StepRun"  ;

            RAISE NOTICE 'Inserted data into "StepRun_new"';



            RAISE NOTICE 'Inserted data into "StepRun_new"';

            RAISE NOTICE 'Creating indexes for "StepRun_new"';
            ALTER INDEX IF EXISTS "StepRun_createdAt_idx" RENAME TO "StepRun_createdAt_idx_old";
            ALTER INDEX IF EXISTS "StepRun_deletedAt_idx" RENAME TO "StepRun_deletedAt_idx_old";
            ALTER INDEX IF EXISTS "StepRun_updatedAt_idx" RENAME TO "StepRun_updatedAt_idx_old";

            ALTER INDEX IF EXISTS "StepRun_id_tenantId_idx" RENAME TO "StepRun_id_tenantId_idx_old";
            ALTER INDEX IF EXISTS "StepRun_jobRunId_status_idx" RENAME TO "StepRun_jobRunId_status_idx_old";
            ALTER INDEX IF EXISTS "StepRun_jobRunId_tenantId_order_idx" RENAME TO "StepRun_jobRunId_tenantId_order_idx_old";
            ALTER INDEX IF EXISTS "StepRun_stepId_idx" RENAME TO "StepRun_stepId_idx_old";
            ALTER INDEX IF EXISTS "StepRun_tenantId_idx" RENAME TO "StepRun_tenantId_idx_old";
            ALTER INDEX IF EXISTS "StepRun_workerId_idx" RENAME TO "StepRun_workerId_idx_old";


            CREATE INDEX "StepRun_createdAt_idx" ON "StepRun_new" ("createdAt");
            CREATE INDEX "StepRun_deletedAt_idx" ON "StepRun_new" ("deletedAt");
            CREATE INDEX "StepRun_updatedAt_idx" ON "StepRun_new" ("updatedAt");
            CREATE INDEX "StepRun_id_tenantId_idx" ON "StepRun_new" ("id", "tenantId");
            CREATE INDEX "StepRun_jobRunId_status_idx" ON "StepRun_new" ("jobRunId", "status");
            CREATE INDEX "StepRun_jobRunId_tenantId_order_idx" ON "StepRun_new" ("jobRunId", "tenantId", "order");
            CREATE INDEX "StepRun_stepId_idx" ON "StepRun_new" ("stepId");
            CREATE INDEX "StepRun_tenantId_idx" ON "StepRun_new" ("tenantId");
            CREATE INDEX "StepRun_workerId_idx" ON "StepRun_new" ("workerId");
            CREATE INDEX "StepRun_status_tenantId_idx" ON "StepRun_new" ("status", "tenantId");


            RAISE NOTICE 'Created indexes for "StepRun_new"';

            RAISE NOTICE 'Checking for data since the last select';


            INSERT INTO "StepRun_new" SELECT * FROM "StepRun" where "updatedAt" >= (SELECT max("updatedAt") FROM "StepRun_new") AND NOT EXISTS (
              SELECT 1 FROM "StepRun_new" WHERE "StepRun_new"."id" = "StepRun"."id"
            );
            ALTER TABLE "StepRun_volatile"
            SET (autovacuum_vacuum_threshold = '1000',
                autovacuum_vacuum_scale_factor = '0.01',
                autovacuum_analyze_threshold = '500',
                autovacuum_analyze_scale_factor = '0.01');


            RAISE NOTICE 'Renaming tables and copying any new data';
            BEGIN
                LOCK TABLE "StepRun" IN SHARE MODE;
                LOCK TABLE "StepRun_new" IN SHARE MODE;

                INSERT INTO "StepRun_new"
                SELECT *
                FROM "StepRun"
                WHERE "updatedAt" >= (SELECT max("updatedAt") FROM "StepRun_new")
                AND NOT EXISTS (
              SELECT 1 FROM "StepRun_new" WHERE "StepRun_new"."id" = "StepRun"."id"
            );


                ALTER TABLE "StepRun" RENAME TO "StepRun_old";
                ALTER TABLE "StepRun_new" RENAME TO "StepRun";
            END;


            FOR rec IN
                SELECT
                    conname AS constraint_name,
                    conrelid::regclass AS referencing_table,
                    a.attname AS referencing_column,
                    confrelid::regclass AS referenced_table,
                    af.attname AS referenced_column
                FROM
                    pg_constraint c
                JOIN
                    pg_attribute a ON a.attnum = ANY(c.conkey) AND a.attrelid = c.conrelid
                JOIN
                    pg_attribute af ON af.attnum = ANY(c.confkey) AND af.attrelid = c.confrelid
                WHERE
                    confrelid = '"StepRun_old"'::regclass
                    AND contype = 'f'
            LOOP

                RAISE NOTICE 'Referencing column: %, Referenced table: %, Referenced column: %', rec.referencing_column, rec.referenced_table, rec.referenced_column;

                sql_statement = 'CREATE OR REPLACE FUNCTION ' || 'StepRun' || rec.referencing_column || '_fk_trigger_function() RETURNS TRIGGER AS $function_body$
                BEGIN
                    IF NEW."'|| rec.referencing_column || '" IS NOT NULL THEN
                    IF NOT EXISTS (
                        SELECT 1
                        FROM "StepRun"
                        WHERE "' || rec.referenced_column || '" = NEW."' || rec.referencing_column || '"
                    ) THEN
                        RAISE EXCEPTION ''Foreign key violation: ' || 'StepRun' || ' with ' || rec.referenced_column || ' = % does not exist.'', NEW."' || rec.referencing_column || '";
                    END IF;
                    END IF;
                    RETURN NEW;
                END;
                $function_body$ LANGUAGE plpgsql;';

                RAISE NOTICE 'Executing: %', sql_statement;

                EXECUTE sql_statement;

                RAISE NOTICE 'Created trigger function for %', rec.constraint_name;

                sql_statement = 'CREATE TRIGGER "' || rec.constraint_name || '_fk_insert_trigger"
                BEFORE INSERT ON ' || rec.referencing_table || '
                FOR EACH ROW
                EXECUTE FUNCTION ' || 'StepRun'||rec.referencing_column|| '_fk_trigger_function();';
                RAISE NOTICE 'Executing: %', sql_statement;
                EXECUTE sql_statement;

                sql_statement = 'CREATE TRIGGER "' || rec.constraint_name || '_fk_update_trigger"
                BEFORE UPDATE ON ' || rec.referencing_table || '
                FOR EACH ROW
                EXECUTE FUNCTION ' || 'StepRun' || rec.referencing_column|| '_fk_trigger_function();';
                RAISE NOTICE 'Executing: %', sql_statement;



                EXECUTE sql_statement;

                sql_statement = 'ALTER TABLE ' || rec.referencing_table || ' DROP CONSTRAINT "' || rec.constraint_name || '"';
                RAISE NOTICE 'Executing: %', sql_statement;
                EXECUTE sql_statement;


            END LOOP;



            RAISE NOTICE 'Migration successful EXIT';
            EXIT;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Migration failed, retrying...';
                RAISE NOTICE 'SQLSTATE: %, Message: %', SQLSTATE, SQLERRM;
                ROLLBACK;
                retry_count := retry_count + 1;
                RAISE NOTICE 'Attempt %', retry_count;
                PERFORM pg_sleep(sleep_interval / 1000.0);
        END;
    END LOOP;


    IF retry_count = max_retries THEN
        RAISE EXCEPTION 'Migration failed after % attempts.', max_retries;
    END IF;
    RAISE NOTICE 'Migration successful COMMIT';
    DROP TABLE "StepRun_old";
    COMMIT;
END $$;

-- Drop index "StepRun_updatedAt_idx" from table: "StepRun"
DROP INDEX "StepRun_updatedAt_idx";
-- Create index "StepRun_id_key" to table: "StepRun"
CREATE UNIQUE INDEX "StepRun_id_key" ON "StepRun" ("id", "status");
-- Create index "StepRun_jobRunId_status_tenantId_idx" to table: "StepRun"
CREATE INDEX "StepRun_jobRunId_status_tenantId_idx" ON "StepRun" ("jobRunId", "status", "tenantId") WHERE (status = 'PENDING'::"StepRunStatus");


-- Modify "StepRun" table
ALTER TABLE "StepRun" ADD COLUMN "identityId" bigserial NOT NULL, ADD CONSTRAINT "step_run_identity_id_unique" UNIQUE ("identityId", "status");
-- Modify "Event" table
ALTER TABLE "Event" ADD COLUMN "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY;
-- Modify "JobRun" table
ALTER TABLE "JobRun" ADD COLUMN "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY;
-- Modify "JobRunLookupData" table
ALTER TABLE "JobRunLookupData" ADD COLUMN "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY;
-- Modify "WorkflowRun" table
ALTER TABLE "WorkflowRun" ADD COLUMN "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY;
-- Modify "WorkflowRunTriggeredBy" table
ALTER TABLE "WorkflowRunTriggeredBy" ADD COLUMN "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY;