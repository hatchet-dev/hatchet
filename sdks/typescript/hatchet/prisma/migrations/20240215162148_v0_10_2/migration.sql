-- Create sequence and alter the table for StepRun
CREATE SEQUENCE step_run_order_seq;
ALTER TABLE "StepRun" ALTER COLUMN "order" TYPE BIGINT;
ALTER SEQUENCE step_run_order_seq OWNED BY "StepRun"."order";
ALTER TABLE "StepRun" ALTER COLUMN "order" SET DEFAULT nextval('step_run_order_seq'::regclass);

-- Create sequence and alter the table for WorkflowVersion
CREATE SEQUENCE workflow_version_order_seq;
ALTER TABLE "WorkflowVersion" ALTER COLUMN "order" TYPE BIGINT;
ALTER SEQUENCE workflow_version_order_seq OWNED BY "WorkflowVersion"."order";
ALTER TABLE "WorkflowVersion" ALTER COLUMN "order" SET DEFAULT nextval('workflow_version_order_seq'::regclass);

