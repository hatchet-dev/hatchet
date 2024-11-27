-- atlas:txmode none

create index CONCURRENTLY IF NOT EXISTS "StepRun_updatedAt_idx" on "StepRun" ("updatedAt");

-- 'Created index';
-- print the time
DO $$
BEGIN
    RAISE NOTICE 'Starting at time=%', clock_timestamp();
END $$;

BEGIN TRANSACTION;

DO $$
DECLARE
    retry_count INT := 0;
    max_retries INT := 20;
    sleep_interval INT := 5000;
    rec RECORD;
    sql_statement TEXT;
    newest_record RECORD;
    start_time TIMESTAMP;
    end_time TIMESTAMP;
BEGIN


            SET LOCAL lock_timeout = '25s';
            SET LOCAL statement_timeout = '0';



            CREATE TABLE "StepRun_new" ( LIKE "StepRun" INCLUDING DEFAULTS, "identityId" BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY,
            PRIMARY KEY ( "status", "id"),   
            UNIQUE ("status", "identityId") )
            PARTITION BY LIST ("status");
          

               
        

            

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


-- Create index "StepRun_id_key" to table: "StepRun"
CREATE UNIQUE INDEX IF NOT EXISTS  "StepRun_new_id_key" ON "StepRun_new" ("id", "status");
-- Create index "StepRun_jobRunId_status_tenantId_idx" to table: "StepRun"
CREATE INDEX IF NOT EXISTS "StepRun_new_jobRunId_status_tenantId_idx" ON "StepRun_new" ("jobRunId", "status", "tenantId") WHERE (status = 'PENDING'::"StepRunStatus");
   RAISE NOTICE 'Created indexes for "StepRun_new"';
   RAISE NOTICE 'Creating identity constraint for "StepRun_new"';
   ALTER TABLE "StepRun_new"
    ADD CONSTRAINT "StepRun_new_identityId_status_unique" UNIQUE ("identityId","status");
   RAISE NOTICE 'Created identity constraint for "StepRun_new"';
   RAISE NOTICE 'StepRun_new table created successfully now we try and swap the tables';

    WHILE retry_count < max_retries
     LOOP
        BEGIN
            SET LOCAL lock_timeout = '25s';
            SET LOCAL statement_timeout = '0';

         

        RAISE NOTICE 'Checking for data since the last select';


            INSERT INTO "StepRun_new" SELECT * FROM "StepRun" where "updatedAt" = (SELECT max("updatedAt") FROM "StepRun_new") AND NOT EXISTS (
              SELECT 1 FROM "StepRun_new" WHERE "StepRun_new"."id" = "StepRun"."id"
            );

                  INSERT INTO "StepRun_new" SELECT * FROM "StepRun" where "updatedAt" > (SELECT max("updatedAt") FROM "StepRun_new") ;
            ALTER TABLE "StepRun_volatile"
            SET (autovacuum_vacuum_threshold = '1000',
                autovacuum_vacuum_scale_factor = '0.01',
                autovacuum_analyze_threshold = '500',
                autovacuum_analyze_scale_factor = '0.01');


            RAISE NOTICE 'Renaming tables and copying any new data';

            BEGIN
            SET LOCAL lock_timeout = '10s';
            SET LOCAL statement_timeout = '0';
            RAISE NOTICE 'Acquiring locks on dependent tables';

            start_time := clock_timestamp();
            LOCK TABLE "StepRun" IN ACCESS EXCLUSIVE MODE;
            end_time := clock_timestamp();
            RAISE NOTICE 'Acquired StepRun lock in % ms', EXTRACT(MILLISECOND FROM end_time - start_time);



            RAISE NOTICE 'Acquired locks on dependent tables';



                INSERT INTO "StepRun_new"
                SELECT *
                FROM "StepRun"
                WHERE "updatedAt" = (SELECT max("updatedAt") FROM "StepRun_new")
                AND NOT EXISTS (
              SELECT 1 FROM "StepRun_new" WHERE "StepRun_new"."id" = "StepRun"."id"
            );

                RAISE NOTICE 'Inserted data with the same updatedAt';

    -- only need to check for equality on the updatedAt column greater than should be fine
     INSERT INTO "StepRun_new"
                SELECT *
                FROM "StepRun"
                WHERE "updatedAt" > (SELECT max("updatedAt") FROM "StepRun_new");
                
               

              RAISE NOTICE 'Inserted data with a greater updatedAt';


                ALTER TABLE "StepRun" RENAME TO "StepRun_old";
                ALTER TABLE "StepRun_new" RENAME TO "StepRun";

                RAISE NOTICE 'Renamed tables now we try and create the triggers';

          
            END;

            ALTER SEQUENCE "step_run_order_seq" OWNED BY "StepRun".order;

            RAISE NOTICE 'Migration successful EXIT';
            EXIT;
    
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Migration failed, retrying...';
                RAISE NOTICE 'SQLSTATE: %, Message: %', SQLSTATE, SQLERRM;
               
                retry_count := retry_count + 1;
                RAISE NOTICE 'Attempt %', retry_count;
                PERFORM pg_sleep(sleep_interval / 1000.0);
                IF retry_count = max_retries THEN
                    RAISE NOTICE 'Migration failed after % attempts', retry_count;
                    RAISE NOTICE 'SQLSTATE: %, Message: %', SQLSTATE, SQLERRM;
                    RAISE EXCEPTION 'Migration failed after % attempts', retry_count;
                END IF;
        END;
    END LOOP;
END $$;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE 'Manually creating the FK triggers';
END $$;

BEGIN TRANSACTION;

CREATE OR REPLACE FUNCTION StepRunA_fk_trigger_function() RETURNS TRIGGER AS $function_body$
                BEGIN
                    IF NEW."A" IS NOT NULL THEN
                    IF NOT EXISTS (
                        SELECT 1
                        FROM "StepRun"
                        WHERE "id" = NEW."A"
                    ) THEN
                        RAISE EXCEPTION 'Foreign key violation: StepRun with id = % does not exist.', NEW."A";
                    END IF;
                    END IF;
                    RETURN NEW;
                END;
                $function_body$ LANGUAGE plpgsql;

CREATE TRIGGER "_StepRunOrder_A_fkey_fk_insert_trigger" BEFORE INSERT ON "_StepRunOrder" FOR EACH ROW
EXECUTE FUNCTION StepRunA_fk_trigger_function();

CREATE TRIGGER "_StepRunOrder_A_fkey_fk_update_trigger" BEFORE
UPDATE ON "_StepRunOrder" FOR EACH ROW
EXECUTE FUNCTION StepRunA_fk_trigger_function();

COMMIT;




ALTER TABLE "_StepRunOrder"
DROP CONSTRAINT "_StepRunOrder_A_fkey";

DO $$
BEGIN
    RAISE NOTICE 'Dropped _StepRunOrder_A_fkey';
END $$;

BEGIN TRANSACTION;

CREATE OR REPLACE FUNCTION StepRunB_fk_trigger_function() RETURNS TRIGGER AS $function_body$
                BEGIN
                    IF NEW."B" IS NOT NULL THEN
                    IF NOT EXISTS (
                        SELECT 1
                        FROM "StepRun"
                        WHERE "id" = NEW."B"
                    ) THEN
                        RAISE EXCEPTION 'Foreign key violation: StepRun with id = % does not exist.', NEW."B";
                    END IF;
                    END IF;
                    RETURN NEW;
                END;
                $function_body$ LANGUAGE plpgsql;

CREATE TRIGGER "_StepRunOrder_B_fkey_fk_insert_trigger" BEFORE INSERT ON "_StepRunOrder" FOR EACH ROW
EXECUTE FUNCTION StepRunB_fk_trigger_function();

CREATE TRIGGER "_StepRunOrder_B_fkey_fk_update_trigger" BEFORE
UPDATE ON "_StepRunOrder" FOR EACH ROW
EXECUTE FUNCTION StepRunB_fk_trigger_function();

COMMIT;

ALTER TABLE "_StepRunOrder"
DROP CONSTRAINT "_StepRunOrder_B_fkey";
BEGIN TRANSACTION ;
CREATE OR REPLACE FUNCTION StepRunstepRunId_fk_trigger_function() RETURNS TRIGGER AS $function_body$
                BEGIN
                    IF NEW."stepRunId" IS NOT NULL THEN
                    IF NOT EXISTS (
                        SELECT 1
                        FROM "StepRun"
                        WHERE "id" = NEW."stepRunId"
                    ) THEN
                        RAISE EXCEPTION 'Foreign key violation: StepRun with id = % does not exist.', NEW."stepRunId";
                    END IF;
                    END IF;
                    RETURN NEW;
                END;
                $function_body$ LANGUAGE plpgsql;

CREATE TRIGGER "StepRunResultArchive_stepRunId_fkey_fk_insert_trigger" BEFORE INSERT ON "StepRunResultArchive" FOR EACH ROW
EXECUTE FUNCTION StepRunstepRunId_fk_trigger_function();

CREATE TRIGGER "StepRunResultArchive_stepRunId_fkey_fk_update_trigger" BEFORE
UPDATE ON "StepRunResultArchive" FOR EACH ROW
EXECUTE FUNCTION StepRunstepRunId_fk_trigger_function();

COMMIT;

DO $$
BEGIN
    RAISE NOTICE 'created StepRunResultArchive triggers';
END $$;

ALTER TABLE "StepRunResultArchive"
DROP CONSTRAINT "StepRunResultArchive_stepRunId_fkey";

BEGIN TRANSACTION;

CREATE
OR REPLACE FUNCTION StepRunstepRunId_fk_trigger_function() RETURNS TRIGGER AS $function_body$
                BEGIN
                    IF NEW."stepRunId" IS NOT NULL THEN
                    IF NOT EXISTS (
                        SELECT 1
                        FROM "StepRun"
                        WHERE "id" = NEW."stepRunId"
                    ) THEN
                        RAISE EXCEPTION 'Foreign key violation: StepRun with id = % does not exist.', NEW."stepRunId";
                    END IF;
                    END IF;
                    RETURN NEW;
                END;
                $function_body$ LANGUAGE plpgsql;

CREATE TRIGGER "LogLine_stepRunId_fkey_fk_insert_trigger" BEFORE INSERT ON "LogLine" FOR EACH ROW
EXECUTE FUNCTION StepRunstepRunId_fk_trigger_function();

CREATE TRIGGER "LogLine_stepRunId_fkey_fk_update_trigger" BEFORE
UPDATE ON "LogLine" FOR EACH ROW
EXECUTE FUNCTION StepRunstepRunId_fk_trigger_function();

COMMIT;

ALTER TABLE "LogLine"
DROP CONSTRAINT "LogLine_stepRunId_fkey";

BEGIN TRANSACTION;

CREATE
OR REPLACE FUNCTION StepRunparentStepRunId_fk_trigger_function() RETURNS TRIGGER AS $function_body$
                BEGIN
                    IF NEW."parentStepRunId" IS NOT NULL THEN
                    IF NOT EXISTS (
                        SELECT 1
                        FROM "StepRun"
                        WHERE "id" = NEW."parentStepRunId"
                    ) THEN
                        RAISE EXCEPTION 'Foreign key violation: StepRun with id = % does not exist.', NEW."parentStepRunId";
                    END IF;
                    END IF;
                    RETURN NEW;
                END;
                $function_body$ LANGUAGE plpgsql;

CREATE TRIGGER "WorkflowTriggerScheduledRef_parentStepRunId_fkey_fk_insert_trigger" BEFORE INSERT ON "WorkflowTriggerScheduledRef" FOR EACH ROW
EXECUTE FUNCTION StepRunparentStepRunId_fk_trigger_function();

CREATE TRIGGER "WorkflowTriggerScheduledRef_parentStepRunId_fkey_fk_update_trigger" BEFORE
UPDATE ON "WorkflowTriggerScheduledRef" FOR EACH ROW
EXECUTE FUNCTION StepRunparentStepRunId_fk_trigger_function();

COMMIT;

ALTER TABLE "WorkflowTriggerScheduledRef"
DROP CONSTRAINT "WorkflowTriggerScheduledRef_parentStepRunId_fkey";

BEGIN TRANSACTION;

CREATE OR REPLACE FUNCTION StepRunparentStepRunId_fk_trigger_function() RETURNS TRIGGER AS $function_body$
                BEGIN
                    IF NEW."parentStepRunId" IS NOT NULL THEN
                    IF NOT EXISTS (
                        SELECT 1
                        FROM "StepRun"
                        WHERE "id" = NEW."parentStepRunId"
                    ) THEN
                        RAISE EXCEPTION 'Foreign key violation: StepRun with id = % does not exist.', NEW."parentStepRunId";
                    END IF;
                    END IF;
                    RETURN NEW;
                END;
                $function_body$ LANGUAGE plpgsql;

CREATE TRIGGER "WorkflowRun_parentStepRunId_fkey_fk_insert_trigger" BEFORE INSERT ON "WorkflowRun" FOR EACH ROW
EXECUTE FUNCTION StepRunparentStepRunId_fk_trigger_function();

CREATE TRIGGER "WorkflowRun_parentStepRunId_fkey_fk_update_trigger" BEFORE
UPDATE ON "WorkflowRun" FOR EACH ROW
EXECUTE FUNCTION StepRunparentStepRunId_fk_trigger_function();

COMMIT;

ALTER TABLE "WorkflowRun"
DROP CONSTRAINT "WorkflowRun_parentStepRunId_fkey"; 

DO $$
BEGIN
    RAISE NOTICE 'dropped WorkflowRun_parentStepRunId_fkey';
END $$;


BEGIN TRANSACTION;

CREATE
OR REPLACE FUNCTION StepRunstepRunId_fk_trigger_function() RETURNS TRIGGER AS $function_body$
                BEGIN
                    IF NEW."stepRunId" IS NOT NULL THEN
                    IF NOT EXISTS (
                        SELECT 1
                        FROM "StepRun"
                        WHERE "id" = NEW."stepRunId"
                    ) THEN
                        RAISE EXCEPTION 'Foreign key violation: StepRun with id = % does not exist.', NEW."stepRunId";
                    END IF;
                    END IF;
                    RETURN NEW;
                END;
                $function_body$ LANGUAGE plpgsql;

CREATE TRIGGER "StreamEvent_stepRunId_fkey_fk_insert_trigger" BEFORE INSERT ON "StreamEvent" FOR EACH ROW
EXECUTE FUNCTION StepRunstepRunId_fk_trigger_function();

CREATE TRIGGER "StreamEvent_stepRunId_fkey_fk_update_trigger" BEFORE
UPDATE ON "StreamEvent" FOR EACH ROW
EXECUTE FUNCTION StepRunstepRunId_fk_trigger_function();

COMMIT;

ALTER TABLE "StreamEvent"
DROP CONSTRAINT "StreamEvent_stepRunId_fkey";

DO $$

BEGIN

    RAISE NOTICE 'Finished manually converting the FK to triggers';

END $$;

-- we are done with the transaction and did not exception out
DO $$
BEGIN

    RAISE NOTICE 'Let everything else have a chance';
    PERFORM pg_sleep(10);
    RAISE NOTICE 'OK done sleeping';
END $$;


DO $$
DECLARE
    attempt INT;
BEGIN
    FOR attempt IN 1..20 LOOP
        BEGIN
            RAISE NOTICE 'Attempting to drop table "StepRun_old" (attempt %)...', attempt;
         

            DROP TABLE "StepRun_old" CASCADE;

       

            EXIT;
            EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Attempt % failed, retrying...', attempt;
            IF attempt = 20 THEN
                RAISE NOTICE 'Error:  %', SQLERRM;
                RAISE EXCEPTION 'All attempts failed after 20 retries.';
            END IF;
        END;
    END LOOP;     
END;
$$ LANGUAGE plpgsql;




DO $$
BEGIN

    RAISE NOTICE 'Let everything else have a chance';
    PERFORM pg_sleep(10);
    RAISE NOTICE 'OK done sleeping now Event table';
END $$;




CREATE TABLE
    "Event_new" (
        LIKE "Event" INCLUDING ALL,
        "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY
    );

INSERT INTO "Event_new" SELECT * FROM "Event";


BEGIN TRANSACTION;

DO $$
BEGIN
    RAISE NOTICE 'GRABBING LOCK on Event at time=%', clock_timestamp();
    LOCK TABLE "Event" IN ACCESS EXCLUSIVE MODE;
END $$;

ALTER TABLE "Event"
RENAME TO "Event_backup";

ALTER TABLE "Event_new"
RENAME TO "Event";

INSERT INTO
    "Event"
SELECT
    *
FROM
    "Event_backup"
WHERE
    "updatedAt" >= (
        SELECT
            max("updatedAt")
        FROM
            "Event"
    )
    AND NOT EXISTS (
        SELECT
            1
        FROM
            "Event"
        WHERE
            "Event"."id" = "Event_backup"."id"
    );

;

ALTER TABLE "Event"
ADD CONSTRAINT "Event_replayedFromId_fkey" FOREIGN KEY ("replayedFromId") REFERENCES "Event" ("id") ON DELETE SET NULL ON UPDATE CASCADE NOT VALID;

DO $$
BEGIN
    RAISE NOTICE 'RELEASED LOCK on Event at time=%', clock_timestamp();
END $$;

COMMIT;



DO $$
DECLARE
    attempt INT;
BEGIN
    FOR attempt IN 1..20 LOOP
        BEGIN
            RAISE NOTICE 'Attempting to drop table "Event_backup" (attempt %)...', attempt;
      

            DROP TABLE "Event_backup" CASCADE;

           

            EXIT;
        EXCEPTION WHEN OTHERS THEN
     
      
            RAISE NOTICE 'Attempt % failed, retrying...', attempt;
                if attempt = 20 THEN
        RAISE EXCEPTION 'All attempts failed after 20 retries.';
    END IF;
        END;
    END LOOP;

END;
$$ LANGUAGE plpgsql;



-- Indexes:
--     "Event_pkey" PRIMARY KEY, btree (id)
--     "Event_createdAt_idx" btree ("createdAt")
--     "Event_id_key" UNIQUE, btree (id)
--     "Event_tenantId_createdAt_idx" btree ("tenantId", "createdAt")
--     "Event_tenantId_idx" btree ("tenantId")
-- Foreign-key constraints:
--     "Event_replayedFromId_fkey" FOREIGN KEY ("replayedFromId") REFERENCES "Event"(id) ON UPDATE CASCADE ON DELETE SET NULL
-- Referenced by:
--     TABLE ""Event"" CONSTRAINT "Event_replayedFromId_fkey" FOREIGN KEY ("replayedFromId") REFERENCES "Event"(id) ON UPDATE CASCADE ON DELETE SET NULL

DO $$
BEGIN
    RAISE NOTICE 'Migrating JobRun time=%', clock_timestamp();
END $$;

CREATE TABLE
    "JobRun_new" (
        LIKE "JobRun" INCLUDING ALL,
        "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY
    );

INSERT INTO
    "JobRun_new"
SELECT
    *
FROM
    "JobRun";


DO $$
BEGIN
    RAISE NOTICE 'About to modify Jobrun=%', clock_timestamp();
END $$;

DO $$
DECLARE
    retry_count INT := 0;
    max_retries INT := 20;
BEGIN
    WHILE retry_count < max_retries
    LOOP
    BEGIN

    RAISE NOTICE 'GRABBING LOCK on JobRun at time=%', clock_timestamp();
    LOCK TABLE "JobRun" IN ACCESS EXCLUSIVE MODE;

RAISE NOTICE 'Altering JobRun table';
ALTER TABLE "JobRun" RENAME TO "JobRun_backup";
ALTER TABLE "JobRun_new" RENAME TO "JobRun";

INSERT INTO "JobRun" SELECT * FROM "JobRun_backup" WHERE "updatedAt" >= (SELECT max("updatedAt") FROM "JobRun") AND NOT EXISTS (
              SELECT 1 FROM "JobRun" WHERE "JobRun"."id" = "JobRun_backup"."id"
            );



    RAISE NOTICE 'RELEASED LOCK on JobRun at time=%', clock_timestamp();
    EXIT;
      EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Migration failed, retrying...';
                RAISE NOTICE 'SQLSTATE: %, Message: %', SQLSTATE, SQLERRM;
               
                retry_count := retry_count + 1;
                RAISE NOTICE 'Attempt %', retry_count;
                PERFORM pg_sleep(5);
                IF retry_count = max_retries THEN
                    RAISE NOTICE 'JobRun Migration failed after % attempts', retry_count;
                    RAISE NOTICE 'SQLSTATE: %, Message: %', SQLSTATE, SQLERRM;
                    RAISE EXCEPTION 'Migration failed after % attempts', retry_count;
                END IF;
        END;
    END LOOP;
END $$;








DO $$
BEGIN
    RAISE NOTICE 'Finished Jobrun bulk changes';
END $$;

DO $$
DECLARE
    attempt INT := 0;
    max_retries INT := 20;
BEGIN
    FOR attempt IN 1..max_retries LOOP
        BEGIN
            RAISE NOTICE 'Attempting to add constraint (attempt %)...', attempt;
            
            ALTER TABLE "JobRun"
            ADD CONSTRAINT "JobRun_workflowRunId_fkey" 
            FOREIGN KEY ("workflowRunId") 
            REFERENCES "WorkflowRun" ("id") 
            ON DELETE CASCADE 
            ON UPDATE CASCADE 
            NOT VALID;

            
            RETURN;
        EXCEPTION 
            WHEN SQLSTATE '40001' THEN  
                RAISE NOTICE 'Deadlock detected on attempt %; retrying...', attempt;
                PERFORM pg_sleep(5);  -- Sleep for 5 seconds before retrying
            WHEN OTHERS THEN
                RAISE NOTICE 'Error on attempt %: %', attempt, SQLERRM;
                PERFORM pg_sleep(5);  -- Sleep before retrying on other errors
        END;
    END LOOP;
    
    RAISE EXCEPTION 'All attempts failed after % retries.', max_retries;
END;


$$ LANGUAGE plpgsql;
DO $$
BEGIN
    RAISE NOTICE 'added jobrun constraint';
END $$;










DO $$
DECLARE
    attempt INT;
BEGIN
    FOR attempt IN 1..20 LOOP
        BEGIN

         

            DROP TABLE "JobRun_backup" CASCADE;

     

            RETURN;
            EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Error is %', SQLERRM;
            RAISE NOTICE 'Attempt % failed, retrying...', attempt;
            IF attempt = 20 THEN
                RAISE EXCEPTION 'All attempts failed after 20 retries.';
            END IF;
        END;
    END LOOP;

END;
$$ LANGUAGE plpgsql;



-- can skip this cause we create it in a second
-- ALTER TABLE "JobRunLookupData" 
--     ADD CONSTRAINT "JobRunLookupData_jobRunId_fkey" 
--     FOREIGN KEY ("jobRunId") 
--     REFERENCES "JobRun" ("id") 
--     ON DELETE CASCADE 
--     ON UPDATE CASCADE 
--     NOT VALID;
-- Indexes:
--     "JobRun_pkey" PRIMARY KEY, btree (id)
--     "JobRun_deletedAt_idx" btree ("deletedAt")
--     "JobRun_id_key" UNIQUE, btree (id)
--     "JobRun_workflowRunId_tenantId_idx" btree ("workflowRunId", "tenantId")
-- Foreign-key constraints:
--     "JobRun_workflowRunId_fkey" FOREIGN KEY ("workflowRunId") REFERENCES "WorkflowRun"(id) ON UPDATE CASCADE ON DELETE CASCADE
-- Referenced by:
--     TABLE ""JobRunLookupData"" CONSTRAINT "JobRunLookupData_jobRunId_fkey" FOREIGN KEY ("jobRunId") REFERENCES "JobRun"(id) ON UPDATE CASCADE ON DELETE CASCADE
--     TABLE ""StepRun"" CONSTRAINT "StepRun_jobRunId_fkey" FOREIGN KEY ("jobRunId") REFERENCES "JobRun"(id) ON UPDATE CASCADE ON DELETE CASCADE
CREATE TABLE
    "JobRunLookupData_new" (
        LIKE "JobRunLookupData" INCLUDING ALL,
        "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY
    );

INSERT INTO
    "JobRunLookupData_new"
SELECT
    *
FROM
    "JobRunLookupData";

BEGIN TRANSACTION;

SET
    LOCAL lock_timeout = '25s';

SET
    LOCAL statement_timeout = '0';

DO $$
DECLARE
    retry_count INT := 0;
    max_retries INT := 20;
BEGIN
    WHILE retry_count < max_retries 

LOOP
        
BEGIN


    RAISE NOTICE 'GRABBING LOCK on JobRunLookupData at time=%', clock_timestamp();
    LOCK TABLE "JobRunLookupData" IN ACCESS EXCLUSIVE MODE;

    ALTER TABLE "JobRunLookupData" RENAME TO "JobRunLookupData_backup";
    ALTER TABLE "JobRunLookupData_new" RENAME TO "JobRunLookupData";

    INSERT INTO "JobRunLookupData" 
    SELECT * FROM "JobRunLookupData_backup" 
    WHERE "updatedAt" = (SELECT max("updatedAt") FROM "JobRunLookupData") AND NOT EXISTS (
              SELECT 1 FROM "JobRunLookupData" WHERE "JobRunLookupData"."id" = "JobRunLookupData_backup"."id"
            );

    INSERT INTO "JobRunLookupData" 
    SELECT * FROM "JobRunLookupData_backup" 
    WHERE "updatedAt" > (SELECT max("updatedAt") FROM "JobRunLookupData") AND NOT EXISTS (
              SELECT 1 FROM "JobRunLookupData" WHERE "JobRunLookupData"."id" = "JobRunLookupData_backup"."id"
            );



    RAISE NOTICE 'RELEASED LOCK on JobRunLookupData at time=%', clock_timestamp();
    EXIT;
      EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Migration failed, retrying...';
                RAISE NOTICE 'SQLSTATE: %, Message: %', SQLSTATE, SQLERRM;
               
                retry_count := retry_count + 1;
                RAISE NOTICE 'Attempt %', retry_count;
                PERFORM pg_sleep(5);
                IF retry_count = max_retries THEN
                    
                    RAISE NOTICE 'JobRun Migration failed after % attempts', retry_count;
                    RAISE NOTICE 'SQLSTATE: %, Message: %', SQLSTATE, SQLERRM;
                    RAISE EXCEPTION 'Migration on JobRunLookupData failed after % attempts', retry_count;
                END IF;
        END;
    END LOOP;    
END $$;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE 'modified JobRunLookupData';
END $$;

ALTER TABLE "JobRunLookupData"
ADD CONSTRAINT "JobRunLookupData_jobRunId_fkey" FOREIGN KEY ("jobRunId") REFERENCES "JobRun" ("id") ON DELETE CASCADE ON UPDATE CASCADE NOT VALID;

DO $$
BEGIN
    RAISE NOTICE 'added JobRunLookupData constraint';
END $$;

BEGIN TRANSACTION;

DO $$
DECLARE
    attempt INT;
BEGIN
    FOR attempt IN 1..20 LOOP
        BEGIN
            RAISE NOTICE 'Attempting to drop table "JobRunLookupData_backup" (attempt %)...', attempt;
    

            DROP TABLE "JobRunLookupData_backup" CASCADE;

       

            EXIT;
        EXCEPTION WHEN OTHERS THEN
            -- Rollback if there's an error
            RAISE NOTICE 'Attempt % failed, retrying...', attempt;
             if attempt = 20 THEN
        RAISE EXCEPTION 'All attempts failed after 20 retries.';
    END IF;
        END;
    END LOOP;

   
END;
$$ LANGUAGE plpgsql;

COMMIT;

BEGIN TRANSACTION;

-- Indexes:
--     "JobRunLookupData_pkey" PRIMARY KEY, btree (id)
--     "JobRunLookupData_id_key" UNIQUE, btree (id)
--     "JobRunLookupData_jobRunId_key" UNIQUE, btree ("jobRunId")
--     "JobRunLookupData_jobRunId_tenantId_key" UNIQUE, btree ("jobRunId", "tenantId")
-- Foreign-key constraints:
--     "JobRunLookupData_jobRunId_fkey" FOREIGN KEY ("jobRunId") REFERENCES "JobRun"(id) ON UPDATE CASCADE ON DELETE CASCADE
DO $$
DECLARE
    retry_count INT := 0;
    max_retries INT := 20;


BEGIN 
    RAISE NOTICE 'Starting with WorkflowRunTriggeredBy at time=%', clock_timestamp();
    CREATE TABLE "WorkflowRunTriggeredBy_new" (
        LIKE "WorkflowRunTriggeredBy" INCLUDING ALL, 
        "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY
    );

    INSERT INTO "WorkflowRunTriggeredBy_new" SELECT * FROM "WorkflowRunTriggeredBy";
    while retry_count < max_retries
    LOOP
    BEGIN
    RAISE NOTICE 'GRABBING LOCK on WorkflowRunTriggeredBy at time=%', clock_timestamp();
    LOCK TABLE "WorkflowRunTriggeredBy" IN ACCESS EXCLUSIVE MODE;

    ALTER TABLE "WorkflowRunTriggeredBy" RENAME TO "WorkflowRunTriggeredBy_backup";
    ALTER TABLE "WorkflowRunTriggeredBy_new" RENAME TO "WorkflowRunTriggeredBy";

    INSERT INTO "WorkflowRunTriggeredBy" 
    SELECT * FROM "WorkflowRunTriggeredBy_backup" 
    WHERE "updatedAt" = (SELECT max("updatedAt") FROM "WorkflowRunTriggeredBy") AND NOT EXISTS (
              SELECT 1 FROM "WorkflowRunTriggeredBy" WHERE "WorkflowRunTriggeredBy"."id" = "WorkflowRunTriggeredBy_backup"."id"
            );


    INSERT INTO "WorkflowRunTriggeredBy" 
    SELECT * FROM "WorkflowRunTriggeredBy_backup" 
    WHERE "updatedAt" > (SELECT max("updatedAt") FROM "WorkflowRunTriggeredBy") ;


    RAISE NOTICE 'RELEASED LOCK on WorkflowRunTriggeredBy at time=%', clock_timestamp();
    EXIT;
      EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Migration failed, retrying...';
                RAISE NOTICE 'SQLSTATE: %, Message: %', SQLSTATE, SQLERRM;
               
                retry_count := retry_count + 1;
                RAISE NOTICE 'Attempt %', retry_count;
                PERFORM pg_sleep(5);
                IF retry_count = max_retries THEN
                    RAISE NOTICE 'JobRun Migration failed after % attempts', retry_count;
                    RAISE NOTICE 'SQLSTATE: %, Message: %', SQLSTATE, SQLERRM;
                    RAISE EXCEPTION 'Migration on WorkflowRunTriggeredBy failed after % attempts', retry_count;
                END IF;
        END;
    END LOOP;        
END $$;

COMMIT;



DO $$
DECLARE
    attempt INT := 0;
    max_retries INT := 20;
BEGIN
    FOR attempt IN 1..max_retries LOOP
        BEGIN
            RAISE NOTICE 'Attempting to add constraint (attempt %)...', attempt;

            -- Attempt to add the constraint
            ALTER TABLE "WorkflowRunTriggeredBy"
            ADD CONSTRAINT "WorkflowRunTriggeredBy_cronParentId_cronSchedule_fkey" 
            FOREIGN KEY ("cronParentId", "cronSchedule") 
            REFERENCES "WorkflowTriggerCronRef" ("parentId", "cron") 
            ON DELETE SET NULL 
            ON UPDATE CASCADE 
            NOT VALID;

            -- If successful, exit the loop and end the DO block
            RAISE NOTICE 'Constraint added successfully on attempt %', attempt;
            RETURN;
        EXCEPTION 
            WHEN SQLSTATE '40001' THEN  -- Deadlock error code
                RAISE NOTICE 'Deadlock detected on attempt %; retrying...', attempt;
                PERFORM pg_sleep(5);  -- Sleep for 5 seconds before retrying
            WHEN OTHERS THEN
                RAISE NOTICE 'Error on attempt %: %', attempt, SQLERRM;
                PERFORM pg_sleep(5);  -- Sleep before retrying on other errors
        END;
    END LOOP;

    -- If we exhaust all attempts, raise an exception
    RAISE EXCEPTION 'All attempts failed after % retries.', max_retries;
END;
$$ LANGUAGE plpgsql;




DO $$
DECLARE
    attempt INT := 0;
    max_retries INT := 20;
BEGIN
    FOR attempt IN 1..max_retries LOOP
        BEGIN
            RAISE NOTICE 'Attempting to add constraint (attempt %)...', attempt;

            -- Attempt to add the constraint
            ALTER TABLE "WorkflowRunTriggeredBy"
            ADD CONSTRAINT "WorkflowRunTriggeredBy_scheduledId_fkey" 
            FOREIGN KEY ("scheduledId") 
            REFERENCES "WorkflowTriggerScheduledRef" ("id") 
            ON DELETE SET NULL 
            ON UPDATE CASCADE 
            NOT VALID;

            -- If successful, exit the loop and end the DO block
            RAISE NOTICE 'Constraint added successfully on attempt %', attempt;
            RETURN;
        EXCEPTION 
            WHEN SQLSTATE '40001' THEN  -- Deadlock error code
                RAISE NOTICE 'Deadlock detected on attempt %; retrying...', attempt;
                PERFORM pg_sleep(5);  -- Sleep for 5 seconds before retrying
            WHEN OTHERS THEN
                RAISE NOTICE 'Error on attempt %: %', attempt, SQLERRM;
                PERFORM pg_sleep(5);  -- Sleep before retrying on other errors
        END;
    END LOOP;

    -- If we exhaust all attempts, raise an exception
    RAISE EXCEPTION 'All attempts failed after % retries.', max_retries;
END;
$$ LANGUAGE plpgsql;




DO $$
DECLARE
    attempt INT;
BEGIN
    FOR attempt IN 1..20 LOOP
        BEGIN
            RAISE NOTICE 'Attempting to drop table "WorkflowRunTriggeredBy_backup" (attempt %)...', attempt;
         

            DROP TABLE "WorkflowRunTriggeredBy_backup" CASCADE;

    

            RETURN;
        EXCEPTION WHEN OTHERS THEN
          
            RAISE NOTICE 'Attempt % failed, retrying...', attempt;
                if attempt = 20 THEN
        RAISE EXCEPTION 'All attempts failed after 20 retries.';
    END IF;
        END;
    END LOOP;

END;
$$ LANGUAGE plpgsql;



-- Indexes:
--     "WorkflowRunTriggeredBy_pkey" PRIMARY KEY, btree (id)
--     "WorkflowRunTriggeredBy_eventId_idx" btree ("eventId")
--     "WorkflowRunTriggeredBy_id_key" UNIQUE, btree (id)
--     "WorkflowRunTriggeredBy_parentId_idx" btree ("parentId")
--     "WorkflowRunTriggeredBy_parentId_key" UNIQUE, btree ("parentId")
--     "WorkflowRunTriggeredBy_scheduledId_key" UNIQUE, btree ("scheduledId")
--     "WorkflowRunTriggeredBy_tenantId_idx" btree ("tenantId")
-- Foreign-key constraints:
--     "WorkflowRunTriggeredBy_cronParentId_cronSchedule_fkey" FOREIGN KEY ("cronParentId", "cronSchedule") REFERENCES "WorkflowTriggerCronRef"("parentId", cron) ON UPDATE CASCADE ON DELETE SET NULL
--     "WorkflowRunTriggeredBy_scheduledId_fkey" FOREIGN KEY ("scheduledId") REFERENCES "WorkflowTriggerScheduledRef"(id) ON UPDATE CASCADE ON DELETE SET NULL
DO $$
BEGIN
    RAISE NOTICE 'Ending at time=%', clock_timestamp();
END $$;

-- Step run should have
-- Indexes:
--     "StepRun_pkey" PRIMARY KEY, btree (id)
--     "StepRun_createdAt_idx" btree ("createdAt")
--     "StepRun_deletedAt_idx" btree ("deletedAt")
--     "StepRun_id_key" UNIQUE, btree (id)
--     "StepRun_id_tenantId_idx" btree (id, "tenantId")
--     "StepRun_jobRunId_status_idx" btree ("jobRunId", status)
--     "StepRun_jobRunId_status_tenantId_idx" btree ("jobRunId", status, "tenantId") WHERE status = 'PENDING'::"StepRunStatus"
--     "StepRun_jobRunId_status_tenantId_requeueAfter_idx" btree ("jobRunId", status, "tenantId", "requeueAfter")
--     "StepRun_jobRunId_tenantId_order_idx" btree ("jobRunId", "tenantId", "order")
--     "StepRun_queue_createdAt_idx" btree (queue, "createdAt")
--     "StepRun_requeueAfter_idx" btree ("requeueAfter")
--     "StepRun_status_idx" btree (status)
--     "StepRun_status_timeoutAt_tickerId_idx" btree (status, "timeoutAt", "tickerId")
--     "StepRun_stepId_idx" btree ("stepId")
--     "StepRun_tenantId_idx" btree ("tenantId")
--     "StepRun_timeoutAt_idx" btree ("timeoutAt")
--     "StepRun_workerId_idx" btree ("workerId")
-- Foreign-key constraints:
--     "StepRun_jobRunId_fkey" FOREIGN KEY ("jobRunId") REFERENCES "JobRun"(id) ON UPDATE CASCADE ON DELETE CASCADE
--     "StepRun_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"(id) ON UPDATE CASCADE ON DELETE SET NULL
-- Referenced by:
--     TABLE ""LogLine"" CONSTRAINT "LogLine_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"(id) ON UPDATE CASCADE ON DELETE SET NULL
--     TABLE ""StepRunResultArchive"" CONSTRAINT "StepRunResultArchive_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"(id) ON UPDATE CASCADE ON DELETE CASCADE
--     TABLE ""StreamEvent"" CONSTRAINT "StreamEvent_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"(id) ON UPDATE CASCADE ON DELETE SET NULL
--     TABLE ""WorkflowRun"" CONSTRAINT "WorkflowRun_parentStepRunId_fkey" FOREIGN KEY ("parentStepRunId") REFERENCES "StepRun"(id) ON UPDATE CASCADE ON DELETE SET NULL
--     TABLE ""WorkflowTriggerScheduledRef"" CONSTRAINT "WorkflowTriggerScheduledRef_parentStepRunId_fkey" FOREIGN KEY ("parentStepRunId") REFERENCES "StepRun"(id) ON UPDATE CASCADE ON DELETE SET NULL
--     TABLE ""_StepRunOrder"" CONSTRAINT "_StepRunOrder_A_fkey" FOREIGN KEY ("A") REFERENCES "StepRun"(id) ON UPDATE CASCADE ON DELETE CASCADE
--     TABLE ""_StepRunOrder"" CONSTRAINT "_StepRunOrder_B_fkey" FOREIGN KEY ("B") REFERENCES "StepRun"(id) ON UPDATE CASCADE ON DELETE CASCADE
DO $$
BEGIN
    RAISE NOTICE 'Renaming indexes starting at time=%', clock_timestamp();
END $$;

-- Rename an index from "Event_new_createdAt_idx" to "Event_createdAt_idx"
ALTER INDEX "Event_new_createdAt_idx"
RENAME TO "Event_createdAt_idx";

-- Rename an index from "Event_new_id_idx" to "Event_id_key"
ALTER INDEX "Event_new_id_idx"
RENAME TO "Event_id_key";

-- Rename an index from "Event_new_tenantId_createdAt_idx" to "Event_tenantId_createdAt_idx"
ALTER INDEX "Event_new_tenantId_createdAt_idx"
RENAME TO "Event_tenantId_createdAt_idx";

-- Rename an index from "Event_new_tenantId_idx" to "Event_tenantId_idx"
ALTER INDEX "Event_new_tenantId_idx"
RENAME TO "Event_tenantId_idx";

-- Rename an index from "JobRun_new_deletedAt_idx" to "JobRun_deletedAt_idx"
ALTER INDEX "JobRun_new_deletedAt_idx"
RENAME TO "JobRun_deletedAt_idx";

-- Rename an index from "JobRun_new_id_idx" to "JobRun_id_key"
ALTER INDEX "JobRun_new_id_idx"
RENAME TO "JobRun_id_key";

-- Rename an index from "JobRun_new_workflowRunId_tenantId_idx" to "JobRun_workflowRunId_tenantId_idx"
ALTER INDEX "JobRun_new_workflowRunId_tenantId_idx"
RENAME TO "JobRun_workflowRunId_tenantId_idx";

-- Rename an index from "JobRunLookupData_new_id_idx" to "JobRunLookupData_id_key"
ALTER INDEX "JobRunLookupData_new_id_idx"
RENAME TO "JobRunLookupData_id_key";

-- Rename an index from "JobRunLookupData_new_jobRunId_idx" to "JobRunLookupData_jobRunId_key"
ALTER INDEX "JobRunLookupData_new_jobRunId_idx"
RENAME TO "JobRunLookupData_jobRunId_key";

-- Rename an index from "JobRunLookupData_new_jobRunId_tenantId_idx" to "JobRunLookupData_jobRunId_tenantId_key"
ALTER INDEX "JobRunLookupData_new_jobRunId_tenantId_idx"
RENAME TO "JobRunLookupData_jobRunId_tenantId_key";

-- Drop index "StepRun_updatedAt_idx" from table: "StepRun"
DROP INDEX "StepRun_updatedAt_idx";

-- Rename an index from "StepRun_new_id_key" to "StepRun_id_key"
ALTER INDEX "StepRun_new_id_key"
RENAME TO "StepRun_id_key";

-- Rename an index from "StepRun_new_jobRunId_status_tenantId_idx" to "StepRun_jobRunId_status_tenantId_idx"
ALTER INDEX "StepRun_new_jobRunId_status_tenantId_idx"
RENAME TO "StepRun_jobRunId_status_tenantId_idx";

-- Rename an index from "WorkflowRunTriggeredBy_new_eventId_idx" to "WorkflowRunTriggeredBy_eventId_idx"
ALTER INDEX "WorkflowRunTriggeredBy_new_eventId_idx"
RENAME TO "WorkflowRunTriggeredBy_eventId_idx";

-- Rename an index from "WorkflowRunTriggeredBy_new_id_idx" to "WorkflowRunTriggeredBy_id_key"
ALTER INDEX "WorkflowRunTriggeredBy_new_id_idx"
RENAME TO "WorkflowRunTriggeredBy_id_key";

-- Rename an index from "WorkflowRunTriggeredBy_new_parentId_idx" to "WorkflowRunTriggeredBy_parentId_key"
ALTER INDEX "WorkflowRunTriggeredBy_new_parentId_idx"
RENAME TO "WorkflowRunTriggeredBy_parentId_key";

-- Rename an index from "WorkflowRunTriggeredBy_new_parentId_idx1" to "WorkflowRunTriggeredBy_parentId_idx"
ALTER INDEX "WorkflowRunTriggeredBy_new_parentId_idx1"
RENAME TO "WorkflowRunTriggeredBy_parentId_idx";

-- Rename an index from "WorkflowRunTriggeredBy_new_scheduledId_idx" to "WorkflowRunTriggeredBy_scheduledId_key"
ALTER INDEX "WorkflowRunTriggeredBy_new_scheduledId_idx"
RENAME TO "WorkflowRunTriggeredBy_scheduledId_key";

-- Rename an index from "WorkflowRunTriggeredBy_new_tenantId_idx" to "WorkflowRunTriggeredBy_tenantId_idx"
ALTER INDEX "WorkflowRunTriggeredBy_new_tenantId_idx"
RENAME TO "WorkflowRunTriggeredBy_tenantId_idx";

-- Modify "StepRun" table
-- this constraint does not exist bu atlas things it does so we need to drop it 
ALTER TABLE "StepRun"
DROP CONSTRAINT IF EXISTS "StepRun_new_status_identityId_key";

DO $$
BEGIN
    RAISE NOTICE 'Renamed indexes ending at time=%', clock_timestamp();
END $$;