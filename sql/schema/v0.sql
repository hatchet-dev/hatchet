-- CreateEnum
CREATE TYPE "ConcurrencyLimitStrategy" AS ENUM (
    'CANCEL_IN_PROGRESS',
    'DROP_NEWEST', -- DEPRECATED
    'QUEUE_NEWEST', -- DEPRECATED
    'GROUP_ROUND_ROBIN',
    'CANCEL_NEWEST'
);


-- CreateEnum
CREATE TYPE "InternalQueue" AS ENUM (
    'WORKER_SEMAPHORE_COUNT',
    'STEP_RUN_UPDATE',
    'WORKFLOW_RUN_UPDATE',
    'WORKFLOW_RUN_PAUSED',
    'STEP_RUN_UPDATE_V2'
);

-- CreateEnum
CREATE TYPE "InviteLinkStatus" AS ENUM ('PENDING', 'ACCEPTED', 'REJECTED');

-- CreateEnum
CREATE TYPE "JobKind" AS ENUM ('DEFAULT', 'ON_FAILURE');

-- CreateEnum
CREATE TYPE "JobRunStatus" AS ENUM (
    'PENDING',
    'RUNNING',
    'SUCCEEDED',
    'FAILED',
    'CANCELLED',
    'BACKOFF'
);

-- CreateEnum
CREATE TYPE "LeaseKind" AS ENUM ('WORKER', 'QUEUE', 'CONCURRENCY_STRATEGY');

-- CreateEnum
CREATE TYPE "LimitResource" AS ENUM (
    'TASK_RUN',
    'EVENT',
    'WORKER',
    'WORKER_SLOT',
    'CRON',
    'SCHEDULE',
    'INCOMING_WEBHOOK'
);

-- CreateEnum
CREATE TYPE "LogLineLevel" AS ENUM ('DEBUG', 'INFO', 'WARN', 'ERROR');

-- CreateEnum
CREATE TYPE "StepExpressionKind" AS ENUM (
    'DYNAMIC_RATE_LIMIT_KEY',
    'DYNAMIC_RATE_LIMIT_VALUE',
    'DYNAMIC_RATE_LIMIT_UNITS',
    'DYNAMIC_RATE_LIMIT_WINDOW'
);

-- CreateEnum
CREATE TYPE "StepRateLimitKind" AS ENUM ('STATIC', 'DYNAMIC');

-- CreateEnum
CREATE TYPE "StepRunEventReason" AS ENUM (
    'REQUEUED_NO_WORKER',
    'REQUEUED_RATE_LIMIT',
    'SCHEDULING_TIMED_OUT',
    'ASSIGNED',
    'STARTED',
    'FINISHED',
    'FAILED',
    'RETRYING',
    'CANCELLED',
    'TIMED_OUT',
    'REASSIGNED',
    'SLOT_RELEASED',
    'TIMEOUT_REFRESHED',
    'RETRIED_BY_USER',
    'SENT_TO_WORKER',
    'WORKFLOW_RUN_GROUP_KEY_SUCCEEDED',
    'WORKFLOW_RUN_GROUP_KEY_FAILED',
    'RATE_LIMIT_ERROR',
    'ACKNOWLEDGED'
);

-- CreateEnum
CREATE TYPE "StepRunEventSeverity" AS ENUM ('INFO', 'WARNING', 'CRITICAL');

-- CreateEnum
CREATE TYPE "StepRunStatus" AS ENUM (
    'PENDING',
    'PENDING_ASSIGNMENT',
    'ASSIGNED',
    'RUNNING',
    'SUCCEEDED',
    'FAILED',
    'CANCELLED',
    'CANCELLING',
    'BACKOFF'
);

-- CreateEnum
CREATE TYPE "StickyStrategy" AS ENUM ('SOFT', 'HARD');

-- CreateEnum
CREATE TYPE "TenantMemberRole" AS ENUM ('OWNER', 'ADMIN', 'MEMBER');

-- CreateEnum
-- IMPORTANT: keep values in sync with api-contracts/openapi/components/schemas/tenant.yaml#TenantEnvironment
CREATE TYPE "TenantEnvironment" AS ENUM ('local', 'development', 'production');

-- CreateEnum
CREATE TYPE "TenantResourceLimitAlertType" AS ENUM ('Alarm', 'Exhausted');

-- CreateEnum
CREATE TYPE "VcsProvider" AS ENUM ('GITHUB');

-- CreateEnum
CREATE TYPE "WebhookWorkerRequestMethod" AS ENUM ('GET', 'POST', 'PUT');

-- CreateEnum
CREATE TYPE "WorkerLabelComparator" AS ENUM (
    'EQUAL',
    'NOT_EQUAL',
    'GREATER_THAN',
    'GREATER_THAN_OR_EQUAL',
    'LESS_THAN',
    'LESS_THAN_OR_EQUAL'
);

-- CreateEnum
CREATE TYPE "WorkerSDKS" AS ENUM ('UNKNOWN', 'GO', 'PYTHON', 'TYPESCRIPT', 'RUBY');

-- CreateEnum
CREATE TYPE "WorkerType" AS ENUM ('WEBHOOK', 'MANAGED', 'SELFHOSTED');

-- CreateEnum
CREATE TYPE "WorkflowKind" AS ENUM ('FUNCTION', 'DURABLE', 'DAG');

-- CreateEnum
CREATE TYPE "WorkflowRunStatus" AS ENUM ('PENDING', 'RUNNING', 'SUCCEEDED', 'FAILED', 'QUEUED', 'CANCELLING', 'CANCELLED', 'BACKOFF');

-- CreateTable
CREATE TABLE "APIToken" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "expiresAt" TIMESTAMP(3),
    "revoked" BOOLEAN NOT NULL DEFAULT false,
    "name" TEXT,
    "tenantId" UUID,
    "nextAlertAt" TIMESTAMP(3),
    "internal" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "APIToken_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Action" (
    "description" TEXT,
    "tenantId" UUID NOT NULL,
    "actionId" TEXT NOT NULL,
    "id" UUID NOT NULL,

    CONSTRAINT "Action_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "ControllerPartition" (
    "id" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastHeartbeat" TIMESTAMP(3),
    "name" TEXT,

    CONSTRAINT "ControllerPartition_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Dispatcher" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "lastHeartbeatAt" TIMESTAMP(3),
    "isActive" BOOLEAN NOT NULL DEFAULT true,

    CONSTRAINT "Dispatcher_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Event" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(6) NOT NULL DEFAULT clock_timestamp(),
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "key" TEXT NOT NULL,
    "tenantId" UUID NOT NULL,
    "replayedFromId" UUID,
    "data" JSONB,
    "additionalMetadata" JSONB,
    "insertOrder" INTEGER,

    CONSTRAINT "Event_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "EventKey" (
    "key" TEXT NOT NULL,
    "tenantId" UUID NOT NULL,
    "id" BIGSERIAL NOT NULL,

    CONSTRAINT "EventKey_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "GetGroupKeyRun" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "workerId" UUID,
    "tickerId" UUID,
    "status" "StepRunStatus" NOT NULL DEFAULT 'PENDING',
    "input" JSONB,
    "output" TEXT,
    "requeueAfter" TIMESTAMP(3),
    "error" TEXT,
    "startedAt" TIMESTAMP(3),
    "finishedAt" TIMESTAMP(3),
    "timeoutAt" TIMESTAMP(3),
    "cancelledAt" TIMESTAMP(3),
    "cancelledReason" TEXT,
    "cancelledError" TEXT,
    "workflowRunId" UUID NOT NULL,
    "scheduleTimeoutAt" TIMESTAMP(3),

    CONSTRAINT "GetGroupKeyRun_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "InternalQueueItem" (
    "id" BIGSERIAL NOT NULL,
    "queue" "InternalQueue" NOT NULL,
    "isQueued" BOOLEAN NOT NULL,
    "data" JSONB,
    "tenantId" UUID NOT NULL,
    "priority" INTEGER NOT NULL DEFAULT 1,
    "uniqueKey" TEXT,

    CONSTRAINT "InternalQueueItem_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Job" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "workflowVersionId" UUID NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "timeout" TEXT,
    "kind" "JobKind" NOT NULL DEFAULT 'DEFAULT',

    CONSTRAINT "Job_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "JobRun" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "jobId" UUID NOT NULL,
    "tickerId" UUID,
    "status" "JobRunStatus" NOT NULL DEFAULT 'PENDING',
    "result" JSONB,
    "startedAt" TIMESTAMP(3),
    "finishedAt" TIMESTAMP(3),
    "timeoutAt" TIMESTAMP(3),
    "cancelledAt" TIMESTAMP(3),
    "cancelledReason" TEXT,
    "cancelledError" TEXT,
    "workflowRunId" UUID NOT NULL,

    CONSTRAINT "JobRun_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "JobRunLookupData" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "jobRunId" UUID NOT NULL,
    "tenantId" UUID NOT NULL,
    "data" JSONB,

    CONSTRAINT "JobRunLookupData_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Lease" (
    "id" BIGSERIAL NOT NULL,
    "expiresAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "resourceId" TEXT NOT NULL,
    "kind" "LeaseKind" NOT NULL,

    CONSTRAINT "Lease_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "LogLine" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "stepRunId" UUID,
    "message" TEXT NOT NULL,
    "level" "LogLineLevel" NOT NULL DEFAULT 'INFO',
    "metadata" JSONB,

    CONSTRAINT "LogLine_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "LogLine_tenantId_stepRunId_idx" ON "LogLine" ("tenantId", "stepRunId" ASC);

-- CreateTable
CREATE TABLE "Queue" (
    "id" BIGSERIAL NOT NULL,
    "tenantId" UUID NOT NULL,
    "name" TEXT NOT NULL,
    "lastActive" TIMESTAMP(3),

    CONSTRAINT "Queue_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "QueueItem" (
    "id" BIGSERIAL NOT NULL,
    "stepRunId" UUID,
    "stepId" UUID,
    "actionId" TEXT,
    "scheduleTimeoutAt" TIMESTAMP(3),
    "stepTimeout" TEXT,
    "priority" INTEGER NOT NULL DEFAULT 1,
    "isQueued" BOOLEAN NOT NULL,
    "tenantId" UUID NOT NULL,
    "queue" TEXT NOT NULL,
    "sticky" "StickyStrategy",
    "desiredWorkerId" UUID,

    CONSTRAINT "QueueItem_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "RateLimit" (
    "tenantId" UUID NOT NULL,
    "key" TEXT NOT NULL,
    "limitValue" INTEGER NOT NULL,
    "value" INTEGER NOT NULL,
    "window" TEXT NOT NULL,
    "lastRefill" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- CreateTable
CREATE TABLE "SNSIntegration" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "topicArn" TEXT NOT NULL,

    CONSTRAINT "SNSIntegration_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SchedulerPartition" (
    "id" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastHeartbeat" TIMESTAMP(3),
    "name" TEXT,

    CONSTRAINT "SchedulerPartition_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SecurityCheckIdent" (
    "id" UUID NOT NULL,

    CONSTRAINT "SecurityCheckIdent_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SemaphoreQueueItem" (
    "stepRunId" UUID NOT NULL,
    "workerId" UUID NOT NULL,
    "tenantId" UUID NOT NULL,

    CONSTRAINT "SemaphoreQueueItem_pkey" PRIMARY KEY ("stepRunId")
);

-- CreateTable
CREATE TABLE "Service" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "name" TEXT NOT NULL,
    "description" TEXT,
    "tenantId" UUID NOT NULL,

    CONSTRAINT "Service_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SlackAppWebhook" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "teamId" TEXT NOT NULL,
    "teamName" TEXT NOT NULL,
    "channelId" TEXT NOT NULL,
    "channelName" TEXT NOT NULL,
    "webhookURL" BYTEA NOT NULL,

    CONSTRAINT "SlackAppWebhook_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Step" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "readableId" TEXT,
    "tenantId" UUID NOT NULL,
    "jobId" UUID NOT NULL,
    "actionId" TEXT NOT NULL,
    "timeout" TEXT,
    "customUserData" JSONB,
    "retries" INTEGER NOT NULL DEFAULT 0,
    -- a factor to use for exponential backoff: min(retryMaxBackoff, retryBackoffFactor^retryCount)
    "retryBackoffFactor" DOUBLE PRECISION,
    -- the maximum amount of time in seconds to wait between retries
    "retryMaxBackoff" INTEGER,
    "scheduleTimeout" TEXT NOT NULL DEFAULT '5m',
    "isDurable" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "Step_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "StepDesiredWorkerLabel" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "stepId" UUID NOT NULL,
    "key" TEXT NOT NULL,
    "strValue" TEXT,
    "intValue" INTEGER,
    "required" BOOLEAN NOT NULL,
    "comparator" "WorkerLabelComparator" NOT NULL,
    "weight" INTEGER NOT NULL,

    CONSTRAINT "StepDesiredWorkerLabel_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "StepExpression" (
    "key" TEXT NOT NULL,
    "stepId" UUID NOT NULL,
    "expression" TEXT NOT NULL,
    "kind" "StepExpressionKind" NOT NULL,

    CONSTRAINT "StepExpression_pkey" PRIMARY KEY ("key","stepId","kind")
);

-- CreateTable
CREATE TABLE "StepRateLimit" (
    "units" INTEGER NOT NULL,
    "stepId" UUID NOT NULL,
    "rateLimitKey" TEXT NOT NULL,
    "tenantId" UUID NOT NULL,
    "kind" "StepRateLimitKind" NOT NULL DEFAULT 'STATIC'
);

CREATE SEQUENCE steprun_identity_id_seq START 1;

-- CreateTable
CREATE TABLE "StepRun" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "jobRunId" UUID NOT NULL,
    "stepId" UUID NOT NULL,
    "order" BIGSERIAL NOT NULL,
    "workerId" UUID,
    "tickerId" UUID,
    "status" "StepRunStatus" NOT NULL DEFAULT 'PENDING',
    "input" JSONB,
    "output" JSONB,
    "requeueAfter" TIMESTAMP(3),
    "scheduleTimeoutAt" TIMESTAMP(3),
    "error" TEXT,
    "startedAt" TIMESTAMP(3),
    "finishedAt" TIMESTAMP(3),
    "timeoutAt" TIMESTAMP(3),
    "cancelledAt" TIMESTAMP(3),
    "cancelledReason" TEXT,
    "cancelledError" TEXT,
    "inputSchema" JSONB,
    "callerFiles" JSONB,
    "gitRepoBranch" TEXT,
    "retryCount" INTEGER NOT NULL DEFAULT 0,
    "semaphoreReleased" BOOLEAN NOT NULL DEFAULT false,
    "queue" TEXT NOT NULL DEFAULT 'default',
    "priority" INTEGER,
    "internalRetryCount" INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT "StepRun_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "StepRunEvent" (
    "id" BIGSERIAL NOT NULL,
    "timeFirstSeen" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "timeLastSeen" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "stepRunId" UUID,
    "reason" "StepRunEventReason" NOT NULL,
    "severity" "StepRunEventSeverity" NOT NULL,
    "message" TEXT NOT NULL,
    "count" INTEGER NOT NULL,
    "data" JSONB,
    "workflowRunId" UUID

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

-- CreateTable
CREATE TABLE "StepRunResultArchive" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "stepRunId" UUID NOT NULL,
    "order" BIGSERIAL NOT NULL,
    "input" JSONB,
    "output" JSONB,
    "error" TEXT,
    "startedAt" TIMESTAMP(3),
    "finishedAt" TIMESTAMP(3),
    "timeoutAt" TIMESTAMP(3),
    "cancelledAt" TIMESTAMP(3),
    "cancelledReason" TEXT,
    "cancelledError" TEXT,
    "retryCount" INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT "StepRunResultArchive_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "StreamEvent" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "stepRunId" UUID,
    "message" BYTEA NOT NULL,
    "metadata" JSONB,

    CONSTRAINT "StreamEvent_pkey" PRIMARY KEY ("id")
);

CREATE TYPE "TenantMajorEngineVersion" AS ENUM (
    'V0',
    'V1'
);

CREATE TYPE "TenantMajorUIVersion" AS ENUM (
    'V0',
    'V1'
);

-- CreateTable
CREATE TABLE "Tenant" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "version" "TenantMajorEngineVersion" NOT NULL DEFAULT 'V1',
    "uiVersion" "TenantMajorUIVersion" NOT NULL DEFAULT 'V0',
    "name" TEXT NOT NULL,
    "slug" TEXT NOT NULL,
    "analyticsOptOut" BOOLEAN NOT NULL DEFAULT false,
    "alertMemberEmails" BOOLEAN NOT NULL DEFAULT true,
    "controllerPartitionId" TEXT,
    "workerPartitionId" TEXT,
    "dataRetentionPeriod" TEXT NOT NULL DEFAULT '720h',
    "schedulerPartitionId" TEXT,
    "canUpgradeV1" BOOLEAN NOT NULL DEFAULT true,
    "onboardingData" JSONB,
    "environment" "TenantEnvironment",

    CONSTRAINT "Tenant_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TenantAlertEmailGroup" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "emails" TEXT NOT NULL,

    CONSTRAINT "TenantAlertEmailGroup_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TenantAlertingSettings" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "maxFrequency" TEXT NOT NULL DEFAULT '1h',
    "lastAlertedAt" TIMESTAMP(3),
    "tickerId" UUID,
    "enableExpiringTokenAlerts" BOOLEAN NOT NULL DEFAULT true,
    "enableWorkflowRunFailureAlerts" BOOLEAN NOT NULL DEFAULT false,
    "enableTenantResourceLimitAlerts" BOOLEAN NOT NULL DEFAULT true,

    CONSTRAINT "TenantAlertingSettings_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TenantInviteLink" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "inviterEmail" TEXT NOT NULL,
    "inviteeEmail" TEXT NOT NULL,
    "expires" TIMESTAMP(3) NOT NULL,
    "status" "InviteLinkStatus" NOT NULL DEFAULT 'PENDING',
    "role" "TenantMemberRole" NOT NULL DEFAULT 'OWNER',

    CONSTRAINT "TenantInviteLink_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TenantMember" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "userId" UUID NOT NULL,
    "role" "TenantMemberRole" NOT NULL,

    CONSTRAINT "TenantMember_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TenantResourceLimit" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "resource" "LimitResource" NOT NULL,
    "tenantId" UUID NOT NULL,
    "limitValue" INTEGER NOT NULL,
    "alarmValue" INTEGER,
    "value" INTEGER NOT NULL DEFAULT 0,
    "window" TEXT,
    "lastRefill" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "customValueMeter" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "TenantResourceLimit_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TenantResourceLimitAlert" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "resourceLimitId" UUID NOT NULL,
    "tenantId" UUID NOT NULL,
    "resource" "LimitResource" NOT NULL,
    "alertType" "TenantResourceLimitAlertType" NOT NULL,
    "value" INTEGER NOT NULL,
    "limit" INTEGER NOT NULL,

    CONSTRAINT "TenantResourceLimitAlert_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TenantVcsProvider" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "vcsProvider" "VcsProvider" NOT NULL,
    "config" JSONB,

    CONSTRAINT "TenantVcsProvider_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TenantWorkerPartition" (
    "id" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastHeartbeat" TIMESTAMP(3),
    "name" TEXT,

    CONSTRAINT "TenantWorkerPartition_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Ticker" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastHeartbeatAt" TIMESTAMP(3),
    "isActive" BOOLEAN NOT NULL DEFAULT true,

    CONSTRAINT "Ticker_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TimeoutQueueItem" (
    "id" BIGSERIAL NOT NULL,
    "stepRunId" UUID NOT NULL,
    "retryCount" INTEGER NOT NULL,
    "timeoutAt" TIMESTAMP(3) NOT NULL,
    "tenantId" UUID NOT NULL,
    "isQueued" BOOLEAN NOT NULL,

    CONSTRAINT "TimeoutQueueItem_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "User" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "email" TEXT NOT NULL,
    "emailVerified" BOOLEAN NOT NULL DEFAULT false,
    "name" TEXT,

    CONSTRAINT "User_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "UserOAuth" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "userId" UUID NOT NULL,
    "provider" TEXT NOT NULL,
    "providerUserId" TEXT NOT NULL,
    "expiresAt" TIMESTAMP(3),
    "accessToken" BYTEA NOT NULL,
    "refreshToken" BYTEA,

    CONSTRAINT "UserOAuth_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "UserPassword" (
    "hash" TEXT NOT NULL,
    "userId" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "UserSession" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "userId" UUID,
    "data" JSONB,
    "expiresAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "UserSession_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WebhookWorker" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" TEXT NOT NULL,
    "secret" TEXT NOT NULL,
    "url" TEXT NOT NULL,
    "tokenValue" TEXT,
    "deleted" BOOLEAN NOT NULL DEFAULT false,
    "tokenId" UUID,
    "tenantId" UUID NOT NULL,

    CONSTRAINT "WebhookWorker_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WebhookWorkerRequest" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "webhookWorkerId" UUID NOT NULL,
    "method" "WebhookWorkerRequestMethod" NOT NULL,
    "statusCode" INTEGER NOT NULL,

    CONSTRAINT "WebhookWorkerRequest_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WebhookWorkerWorkflow" (
    "id" UUID NOT NULL,
    "webhookWorkerId" UUID NOT NULL,
    "workflowId" UUID NOT NULL,

    CONSTRAINT "WebhookWorkerWorkflow_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Worker" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "lastHeartbeatAt" TIMESTAMP(3),
    "name" TEXT NOT NULL,
    "dispatcherId" UUID,
    -- FIXME: maxRuns is deprecated, remove this column in a future migration
    "maxRuns" INTEGER NOT NULL DEFAULT 100,
    "isActive" BOOLEAN NOT NULL DEFAULT false,
    "lastListenerEstablished" TIMESTAMP(3),
    "isPaused" BOOLEAN NOT NULL DEFAULT false,
    "type" "WorkerType" NOT NULL DEFAULT 'SELFHOSTED',
    "webhookId" UUID,
    "language" "WorkerSDKS",
    "languageVersion" TEXT,
    "os" TEXT,
    "runtimeExtra" TEXT,
    "sdkVersion" TEXT,

    CONSTRAINT "Worker_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkerAssignEvent" (
    "id" BIGSERIAL NOT NULL,
    "workerId" UUID,
    "assignedStepRuns" JSONB,

    CONSTRAINT "WorkerAssignEvent_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkerLabel" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "workerId" UUID NOT NULL,
    "key" TEXT NOT NULL,
    "strValue" TEXT,
    "intValue" INTEGER,

    CONSTRAINT "WorkerLabel_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Workflow" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "isPaused" BOOLEAN DEFAULT false,

    CONSTRAINT "Workflow_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkflowConcurrency" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "workflowVersionId" UUID NOT NULL,
    "getConcurrencyGroupId" UUID,
    "maxRuns" INTEGER NOT NULL DEFAULT 1,
    "limitStrategy" "ConcurrencyLimitStrategy" NOT NULL DEFAULT 'CANCEL_IN_PROGRESS',
    "concurrencyGroupExpression" TEXT,

    CONSTRAINT "WorkflowConcurrency_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkflowRun" (
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "workflowVersionId" UUID NOT NULL,
    "status" "WorkflowRunStatus" NOT NULL DEFAULT 'PENDING',
    "error" TEXT,
    "startedAt" TIMESTAMP(3),
    "finishedAt" TIMESTAMP(3),
    "concurrencyGroupId" TEXT,
    "displayName" TEXT,
    "id" UUID NOT NULL,
    "childIndex" INTEGER,
    "childKey" TEXT,
    "parentId" UUID,
    "parentStepRunId" UUID,
    "additionalMetadata" JSONB,
    "duration" BIGINT,
    "priority" INTEGER,
    "insertOrder" INTEGER,

    CONSTRAINT "WorkflowRun_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkflowRunDedupe" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "workflowId" UUID NOT NULL,
    "workflowRunId" UUID NOT NULL,
    "value" TEXT NOT NULL
);

-- CreateTable
CREATE TABLE "WorkflowRunStickyState" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "workflowRunId" UUID NOT NULL,
    "desiredWorkerId" UUID,
    "strategy" "StickyStrategy" NOT NULL,

    CONSTRAINT "WorkflowRunStickyState_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkflowRunTriggeredBy" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "eventId" UUID,
    "cronParentId" UUID,
    "cronSchedule" TEXT,
    "scheduledId" UUID,
    "input" JSONB,
    "parentId" UUID NOT NULL,
    "cronName" TEXT,

    CONSTRAINT "WorkflowRunTriggeredBy_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkflowTag" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "name" TEXT NOT NULL,
    "color" TEXT NOT NULL DEFAULT '#93C5FD',

    CONSTRAINT "WorkflowTag_pkey" PRIMARY KEY ("id")
);

-- CreateEnum
CREATE TYPE "WorkflowTriggerCronRefMethods" AS ENUM (
    'DEFAULT',
    'API'
);


-- CreateTable
CREATE TABLE "WorkflowTriggerCronRef" (
    "parentId" UUID NOT NULL,
    "cron" TEXT NOT NULL,
    "tickerId" UUID,
    "input" JSONB,
    "enabled" BOOLEAN NOT NULL DEFAULT true,
    "additionalMetadata" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" TEXT,
    "id" UUID NOT NULL,
    "method" "WorkflowTriggerCronRefMethods" NOT NULL DEFAULT 'DEFAULT',
    "priority" INTEGER NOT NULL DEFAULT 1,
    CONSTRAINT "WorkflowTriggerCronRef_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkflowTriggerEventRef" (
    "parentId" UUID NOT NULL,
    "eventKey" TEXT NOT NULL
);

-- CreateEnum
CREATE TYPE "WorkflowTriggerScheduledRefMethods" AS ENUM (
    'DEFAULT',
    'API'
);

-- CreateTable
CREATE TABLE "WorkflowTriggerScheduledRef" (
    "id" UUID NOT NULL,
    "parentId" UUID NOT NULL,
    "triggerAt" TIMESTAMP(3) NOT NULL,
    "tickerId" UUID,
    "input" JSONB,
    "childIndex" INTEGER,
    "childKey" TEXT,
    "parentStepRunId" UUID,
    "parentWorkflowRunId" UUID,
    "additionalMetadata" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "method" "WorkflowTriggerScheduledRefMethods" NOT NULL DEFAULT 'DEFAULT',
    "priority" INTEGER NOT NULL DEFAULT 1,
    CONSTRAINT "WorkflowTriggerScheduledRef_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE
    "WorkflowTriggers" (
        "id" UUID NOT NULL,
        "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "deletedAt" TIMESTAMP(3),
        "workflowVersionId" UUID NOT NULL,
        "tenantId" UUID NOT NULL,
        CONSTRAINT "WorkflowTriggers_pkey" PRIMARY KEY ("id")
    );

-- CreateTable
CREATE TABLE
    "WorkflowVersion" (
        "id" UUID NOT NULL,
        "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "deletedAt" TIMESTAMP(3),
        "version" TEXT,
        "order" BIGSERIAL NOT NULL,
        "workflowId" UUID NOT NULL,
        "checksum" TEXT NOT NULL,
        "scheduleTimeout" TEXT NOT NULL DEFAULT '5m',
        "onFailureJobId" UUID,
        "sticky" "StickyStrategy",
        "kind" "WorkflowKind" NOT NULL DEFAULT 'DAG',
        "defaultPriority" INTEGER,
        "createWorkflowVersionOpts" JSONB,
        "inputJsonSchema" JSONB,
        CONSTRAINT "WorkflowVersion_pkey" PRIMARY KEY ("id")
    );

-- CreateTable
CREATE TABLE
    "_ActionToWorker" ("B" UUID NOT NULL, "A" UUID NOT NULL);

-- CreateTable
CREATE TABLE
    "_ServiceToWorker" ("A" UUID NOT NULL, "B" UUID NOT NULL);

-- CreateTable
CREATE TABLE
    "_StepOrder" ("A" UUID NOT NULL, "B" UUID NOT NULL);

-- CreateTable
CREATE TABLE
    "_StepRunOrder" ("A" UUID NOT NULL, "B" UUID NOT NULL);

-- CreateTable
CREATE TABLE
    "_WorkflowToWorkflowTag" ("A" UUID NOT NULL, "B" UUID NOT NULL);

-- CreateTable
CREATE TABLE "MessageQueue" (
    "name" TEXT NOT NULL,
    "lastActive" TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP,
    "durable" BOOLEAN NOT NULL DEFAULT true,
    "autoDeleted" BOOLEAN NOT NULL DEFAULT false,
    "exclusive" BOOLEAN NOT NULL DEFAULT false,
    "exclusiveConsumerId" UUID,
    CONSTRAINT "MessageQueue_pkey" PRIMARY KEY ("name")
);

-- CreateEnum
CREATE TYPE "MessageQueueItemStatus" AS ENUM (
    'PENDING',
    'ASSIGNED'
);

-- CreateTable
CREATE TABLE "MessageQueueItem" (
    "id" bigint GENERATED ALWAYS AS IDENTITY,
    "payload" JSONB NOT NULL,
    "readAfter" TIMESTAMP(3),
    "expiresAt" TIMESTAMP(3),
    "queueId" TEXT,
    "status" "MessageQueueItemStatus" NOT NULL DEFAULT 'PENDING',
    CONSTRAINT "MessageQueueItem_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "MessageQueueItem_queueId_fkey" FOREIGN KEY ("queueId") REFERENCES "MessageQueue" ("name") ON DELETE SET NULL
);

-- Create an index for message queue item
CREATE INDEX "MessageQueueItem_queueId_expiresAt_readAfter_status_id_idx" ON "MessageQueueItem" ("expiresAt", "queueId", "readAfter", "status", "id");

-- CreateIndex
CREATE UNIQUE INDEX "APIToken_id_key" ON "APIToken" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Action_id_key" ON "Action" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Action_tenantId_actionId_key" ON "Action" ("tenantId" ASC, "actionId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "ControllerPartition_id_key" ON "ControllerPartition" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Dispatcher_id_key" ON "Dispatcher" ("id" ASC);

-- CreateIndex
CREATE INDEX "Event_createdAt_idx" ON "Event" ("createdAt" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Event_id_key" ON "Event" ("id" ASC);

-- CreateIndex
CREATE INDEX "Event_tenantId_createdAt_idx" ON "Event" ("tenantId" ASC, "createdAt" ASC);

-- CreateIndex
CREATE INDEX "Event_tenantId_idx" ON "Event" ("tenantId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "EventKey_key_tenantId_key" ON "EventKey" ("key" ASC, "tenantId" ASC);

-- CreateIndex
CREATE INDEX "GetGroupKeyRun_createdAt_idx" ON "GetGroupKeyRun" ("createdAt" ASC);

-- CreateIndex
CREATE INDEX "GetGroupKeyRun_deletedAt_idx" ON "GetGroupKeyRun" ("deletedAt" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "GetGroupKeyRun_id_key" ON "GetGroupKeyRun" ("id" ASC);

-- CreateIndex
CREATE INDEX "GetGroupKeyRun_status_deletedAt_timeoutAt_idx" ON "GetGroupKeyRun" ("status" ASC, "deletedAt" ASC, "timeoutAt" ASC);

-- CreateIndex
CREATE INDEX "GetGroupKeyRun_tenantId_deletedAt_status_idx" ON "GetGroupKeyRun" ("tenantId" ASC, "deletedAt" ASC, "status" ASC);

-- CreateIndex
CREATE INDEX "GetGroupKeyRun_tenantId_idx" ON "GetGroupKeyRun" ("tenantId" ASC);

-- CreateIndex
CREATE INDEX "GetGroupKeyRun_workerId_idx" ON "GetGroupKeyRun" ("workerId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "GetGroupKeyRun_workflowRunId_key" ON "GetGroupKeyRun" ("workflowRunId" ASC);

-- CreateIndex
CREATE INDEX "InternalQueueItem_isQueued_tenantId_queue_priority_id_idx" ON "InternalQueueItem" (
    "isQueued" ASC,
    "tenantId" ASC,
    "queue" ASC,
    "priority" DESC,
    "id" ASC
);

-- CreateIndex
CREATE UNIQUE INDEX "InternalQueueItem_tenantId_queue_uniqueKey_key" ON "InternalQueueItem" ("tenantId" ASC, "queue" ASC, "uniqueKey" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Job_id_key" ON "Job" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Job_workflowVersionId_name_key" ON "Job" ("workflowVersionId" ASC, "name" ASC);

-- CreateIndex
CREATE INDEX "JobRun_deletedAt_idx" ON "JobRun" ("deletedAt" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "JobRun_id_key" ON "JobRun" ("id" ASC);

-- CreateIndex
CREATE INDEX "JobRun_workflowRunId_tenantId_idx" ON "JobRun" ("workflowRunId" ASC, "tenantId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "JobRunLookupData_id_key" ON "JobRunLookupData" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "JobRunLookupData_jobRunId_key" ON "JobRunLookupData" ("jobRunId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "JobRunLookupData_jobRunId_tenantId_key" ON "JobRunLookupData" ("jobRunId" ASC, "tenantId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Lease_tenantId_kind_resourceId_key" ON "Lease" ("tenantId" ASC, "kind" ASC, "resourceId" ASC);

-- CreateIndex
CREATE INDEX "Queue_tenantId_lastActive_idx" ON "Queue" ("tenantId" ASC, "lastActive" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Queue_tenantId_name_key" ON "Queue" ("tenantId" ASC, "name" ASC);

-- CreateIndex
CREATE INDEX "QueueItem_isQueued_priority_tenantId_queue_id_idx_2" ON "QueueItem" (
    "isQueued" ASC,
    "tenantId" ASC,
    "queue" ASC,
    "priority" DESC,
    "id" ASC
);

-- CreateIndex
CREATE UNIQUE INDEX "RateLimit_tenantId_key_key" ON "RateLimit" ("tenantId" ASC, "key" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "SNSIntegration_id_key" ON "SNSIntegration" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "SNSIntegration_tenantId_topicArn_key" ON "SNSIntegration" ("tenantId" ASC, "topicArn" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "SchedulerPartition_id_key" ON "SchedulerPartition" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "SecurityCheckIdent_id_key" ON "SecurityCheckIdent" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "SemaphoreQueueItem_stepRunId_key" ON "SemaphoreQueueItem" ("stepRunId" ASC);

-- CreateIndex
CREATE INDEX "SemaphoreQueueItem_tenantId_workerId_idx" ON "SemaphoreQueueItem" ("tenantId" ASC, "workerId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Service_id_key" ON "Service" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Service_tenantId_name_key" ON "Service" ("tenantId" ASC, "name" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "SlackAppWebhook_id_key" ON "SlackAppWebhook" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "SlackAppWebhook_tenantId_teamId_channelId_key" ON "SlackAppWebhook" ("tenantId" ASC, "teamId" ASC, "channelId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Step_id_key" ON "Step" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Step_jobId_readableId_key" ON "Step" ("jobId" ASC, "readableId" ASC);

-- CreateIndex
CREATE INDEX "StepDesiredWorkerLabel_stepId_idx" ON "StepDesiredWorkerLabel" ("stepId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "StepDesiredWorkerLabel_stepId_key_key" ON "StepDesiredWorkerLabel" ("stepId" ASC, "key" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "StepRateLimit_stepId_rateLimitKey_key" ON "StepRateLimit" ("stepId" ASC, "rateLimitKey" ASC);

-- CreateIndex
CREATE INDEX "StepRun_createdAt_idx" ON "StepRun" ("createdAt" ASC);

-- CreateIndex
CREATE INDEX "StepRun_deletedAt_idx" ON "StepRun" ("deletedAt" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "StepRun_id_key" ON "StepRun"("id" ASC);

-- CreateIndex
CREATE INDEX "StepRun_id_tenantId_idx" ON "StepRun" ("id" ASC, "tenantId" ASC);

-- CreateIndex
CREATE INDEX "StepRun_jobRunId_status_idx" ON "StepRun" ("jobRunId" ASC, "status" ASC);

-- CreateIndex
CREATE INDEX "StepRun_jobRunId_tenantId_order_idx" ON "StepRun" ("jobRunId" ASC, "tenantId" ASC, "order" ASC);

-- CreateIndex
CREATE INDEX "StepRun_stepId_idx" ON "StepRun" ("stepId" ASC);

-- CreateIndex
CREATE INDEX "StepRun_tenantId_idx" ON "StepRun" ("tenantId" ASC);

-- CreateIndex
CREATE INDEX "StepRun_workerId_idx" ON "StepRun" ("workerId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "StepRunEvent_id_key" ON "StepRunEvent" ("id" ASC);

-- CreateIndex
CREATE INDEX "StepRunEvent_stepRunId_idx" ON "StepRunEvent" ("stepRunId" ASC);

-- CreateIndex
CREATE INDEX "StepRunEvent_workflowRunId_idx" ON "StepRunEvent" ("workflowRunId" ASC);

-- CreateIndex
CREATE INDEX "StepRunExpressionEval_stepRunId_idx" ON "StepRunExpressionEval" ("stepRunId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "StepRunResultArchive_id_key" ON "StepRunResultArchive" ("id" ASC);

-- CreateIndex
CREATE INDEX "Tenant_controllerPartitionId_idx" ON "Tenant" ("controllerPartitionId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Tenant_id_key" ON "Tenant" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Tenant_slug_key" ON "Tenant" ("slug" ASC);

-- CreateIndex
CREATE INDEX "Tenant_workerPartitionId_idx" ON "Tenant" ("workerPartitionId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "TenantAlertEmailGroup_id_key" ON "TenantAlertEmailGroup" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "TenantAlertingSettings_id_key" ON "TenantAlertingSettings" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "TenantAlertingSettings_tenantId_key" ON "TenantAlertingSettings" ("tenantId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "TenantInviteLink_id_key" ON "TenantInviteLink" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "TenantMember_id_key" ON "TenantMember" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "TenantMember_tenantId_userId_key" ON "TenantMember" ("tenantId" ASC, "userId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "TenantResourceLimit_id_key" ON "TenantResourceLimit" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "TenantResourceLimit_tenantId_resource_key" ON "TenantResourceLimit" ("tenantId" ASC, "resource" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "TenantResourceLimitAlert_id_key" ON "TenantResourceLimitAlert" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "TenantVcsProvider_id_key" ON "TenantVcsProvider" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "TenantVcsProvider_tenantId_vcsProvider_key" ON "TenantVcsProvider" ("tenantId" ASC, "vcsProvider" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "TenantWorkerPartition_id_key" ON "TenantWorkerPartition" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Ticker_id_key" ON "Ticker" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "TimeoutQueueItem_stepRunId_retryCount_key" ON "TimeoutQueueItem" ("stepRunId" ASC, "retryCount" ASC);

-- CreateIndex
CREATE INDEX "TimeoutQueueItem_tenantId_isQueued_timeoutAt_idx" ON "TimeoutQueueItem" ("tenantId" ASC, "isQueued" ASC, "timeoutAt" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "User_email_key" ON "User" ("email" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "User_id_key" ON "User" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "UserOAuth_id_key" ON "UserOAuth" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "UserOAuth_userId_key" ON "UserOAuth" ("userId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "UserOAuth_userId_provider_key" ON "UserOAuth" ("userId" ASC, "provider" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "UserPassword_userId_key" ON "UserPassword" ("userId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "UserSession_id_key" ON "UserSession" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorker_id_key" ON "WebhookWorker" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorker_url_key" ON "WebhookWorker" ("url" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorkerRequest_id_key" ON "WebhookWorkerRequest" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorkerWorkflow_id_key" ON "WebhookWorkerWorkflow" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorkerWorkflow_webhookWorkerId_workflowId_key" ON "WebhookWorkerWorkflow" ("webhookWorkerId" ASC, "workflowId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Worker_id_key" ON "Worker" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Worker_webhookId_key" ON "Worker" ("webhookId" ASC);

-- CreateIndex

CREATE INDEX "Worker_tenantId_lastHeartbeatAt_idx" ON "Worker" ("tenantId", "lastHeartbeatAt");

-- CreateIndex
CREATE INDEX "WorkerAssignEvent_workerId_id_idx" ON "WorkerAssignEvent" ("workerId" ASC, "id" ASC);

-- CreateIndex
CREATE INDEX "WorkerLabel_workerId_idx" ON "WorkerLabel" ("workerId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkerLabel_workerId_key_key" ON "WorkerLabel" ("workerId" ASC, "key" ASC);

-- CreateIndex
CREATE INDEX "Workflow_deletedAt_idx" ON "Workflow" ("deletedAt" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Workflow_id_key" ON "Workflow" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "Workflow_tenantId_name_key" ON "Workflow" ("tenantId" ASC, "name" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowConcurrency_id_key" ON "WorkflowConcurrency" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowConcurrency_workflowVersionId_key" ON "WorkflowConcurrency" ("workflowVersionId" ASC);

-- CreateIndex
CREATE INDEX "WorkflowRun_createdAt_idx" ON "WorkflowRun" ("createdAt" ASC);

-- CreateIndex
CREATE INDEX "WorkflowRun_deletedAt_idx" ON "WorkflowRun" ("deletedAt" ASC);

-- CreateIndex
CREATE INDEX "WorkflowRun_finishedAt_idx" ON "WorkflowRun" ("finishedAt" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRun_id_key" ON "WorkflowRun" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRun_parentId_parentStepRunId_childKey_key" ON "WorkflowRun" (
    "parentId" ASC,
    "parentStepRunId" ASC,
    "childKey" ASC
);

-- CreateIndex
CREATE INDEX "WorkflowRun_status_idx" ON "WorkflowRun" ("status" ASC);

-- CreateIndex
CREATE INDEX "WorkflowRun_tenantId_createdAt_idx" ON "WorkflowRun" ("tenantId" ASC, "createdAt" ASC);

-- CreateIndex
CREATE INDEX "WorkflowRun_tenantId_idx" ON "WorkflowRun" ("tenantId" ASC);

-- CreateIndex
CREATE INDEX "WorkflowRun_workflowVersionId_idx" ON "WorkflowRun" ("workflowVersionId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunDedupe_id_key" ON "WorkflowRunDedupe" ("id" ASC);

-- CreateIndex
CREATE INDEX "WorkflowRunDedupe_tenantId_value_idx" ON "WorkflowRunDedupe" ("tenantId" ASC, "value" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunDedupe_tenantId_workflowId_value_key" ON "WorkflowRunDedupe" ("tenantId" ASC, "workflowId" ASC, "value" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunStickyState_workflowRunId_key" ON "WorkflowRunStickyState" ("workflowRunId" ASC);

-- CreateIndex
CREATE INDEX "WorkflowRunTriggeredBy_eventId_idx" ON "WorkflowRunTriggeredBy" ("eventId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunTriggeredBy_id_key" ON "WorkflowRunTriggeredBy" ("id" ASC);

-- CreateIndex
CREATE INDEX "WorkflowRunTriggeredBy_parentId_idx" ON "WorkflowRunTriggeredBy" ("parentId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunTriggeredBy_parentId_key" ON "WorkflowRunTriggeredBy" ("parentId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunTriggeredBy_scheduledId_key" ON "WorkflowRunTriggeredBy" ("scheduledId" ASC);

-- CreateIndex
CREATE INDEX "WorkflowRunTriggeredBy_tenantId_idx" ON "WorkflowRunTriggeredBy" ("tenantId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTag_id_key" ON "WorkflowTag" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTag_tenantId_name_key" ON "WorkflowTag" ("tenantId" ASC, "name" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTriggerEventRef_parentId_eventKey_key" ON "WorkflowTriggerEventRef" ("parentId" ASC, "eventKey" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTriggerScheduledRef_id_key" ON "WorkflowTriggerScheduledRef" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTriggerScheduledRef_parentId_parentStepRunId_childK_key" ON "WorkflowTriggerScheduledRef" (
    "parentId" ASC,
    "parentStepRunId" ASC,
    "childKey" ASC
);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTriggers_id_key" ON "WorkflowTriggers" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTriggers_workflowVersionId_key" ON "WorkflowTriggers" ("workflowVersionId" ASC);

-- CreateIndex
CREATE INDEX "WorkflowVersion_deletedAt_idx" ON "WorkflowVersion" ("deletedAt" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowVersion_id_key" ON "WorkflowVersion" ("id" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowVersion_onFailureJobId_key" ON "WorkflowVersion" ("onFailureJobId" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "_ActionToWorker_AB_unique" ON "_ActionToWorker" ("A" ASC, "B" ASC);

-- CreateIndex
CREATE INDEX "_ActionToWorker_B_index" ON "_ActionToWorker" ("B" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "_ServiceToWorker_AB_unique" ON "_ServiceToWorker" ("A" ASC, "B" ASC);

-- CreateIndex
CREATE INDEX "_ServiceToWorker_B_index" ON "_ServiceToWorker" ("B" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "_StepOrder_AB_unique" ON "_StepOrder" ("A" ASC, "B" ASC);

-- CreateIndex
CREATE INDEX "_StepOrder_B_index" ON "_StepOrder" ("B" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "_StepRunOrder_AB_unique" ON "_StepRunOrder" ("A" ASC, "B" ASC);

-- CreateIndex
CREATE INDEX "_StepRunOrder_B_index" ON "_StepRunOrder" ("B" ASC);

-- CreateIndex
CREATE UNIQUE INDEX "_WorkflowToWorkflowTag_AB_unique" ON "_WorkflowToWorkflowTag" ("A" ASC, "B" ASC);

-- CreateIndex
CREATE INDEX "_WorkflowToWorkflowTag_B_index" ON "_WorkflowToWorkflowTag" ("B" ASC);

-- AddForeignKey
ALTER TABLE "APIToken" ADD CONSTRAINT "APIToken_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Action" ADD CONSTRAINT "Action_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Event" ADD CONSTRAINT "Event_replayedFromId_fkey" FOREIGN KEY ("replayedFromId") REFERENCES "Event" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GetGroupKeyRun" ADD CONSTRAINT "GetGroupKeyRun_tickerId_fkey" FOREIGN KEY ("tickerId") REFERENCES "Ticker" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GetGroupKeyRun" ADD CONSTRAINT "GetGroupKeyRun_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GetGroupKeyRun" ADD CONSTRAINT "GetGroupKeyRun_workflowRunId_fkey" FOREIGN KEY ("workflowRunId") REFERENCES "WorkflowRun" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Job" ADD CONSTRAINT "Job_workflowVersionId_fkey" FOREIGN KEY ("workflowVersionId") REFERENCES "WorkflowVersion" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "JobRun" ADD CONSTRAINT "JobRun_workflowRunId_fkey" FOREIGN KEY ("workflowRunId") REFERENCES "WorkflowRun" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "JobRunLookupData" ADD CONSTRAINT "JobRunLookupData_jobRunId_fkey" FOREIGN KEY ("jobRunId") REFERENCES "JobRun" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SNSIntegration" ADD CONSTRAINT "SNSIntegration_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Service" ADD CONSTRAINT "Service_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SlackAppWebhook" ADD CONSTRAINT "SlackAppWebhook_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Step" ADD CONSTRAINT "Step_actionId_tenantId_fkey" FOREIGN KEY ("actionId", "tenantId") REFERENCES "Action" ("actionId", "tenantId") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Step" ADD CONSTRAINT "Step_jobId_fkey" FOREIGN KEY ("jobId") REFERENCES "Job" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "StepDesiredWorkerLabel" ADD CONSTRAINT "StepDesiredWorkerLabel_stepId_fkey" FOREIGN KEY ("stepId") REFERENCES "Step" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "StepRateLimit" ADD CONSTRAINT "StepRateLimit_tenantId_rateLimitKey_fkey" FOREIGN KEY ("tenantId", "rateLimitKey") REFERENCES "RateLimit" ("tenantId", "key") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Tenant" ADD CONSTRAINT "Tenant_controllerPartitionId_fkey" FOREIGN KEY ("controllerPartitionId") REFERENCES "ControllerPartition" ("id") ON DELETE SET NULL ON UPDATE SET NULL;

-- AddForeignKey
ALTER TABLE "Tenant" ADD CONSTRAINT "Tenant_schedulerPartitionId_fkey" FOREIGN KEY ("schedulerPartitionId") REFERENCES "SchedulerPartition" ("id") ON DELETE SET NULL ON UPDATE SET NULL;

-- AddForeignKey
ALTER TABLE "Tenant" ADD CONSTRAINT "Tenant_workerPartitionId_fkey" FOREIGN KEY ("workerPartitionId") REFERENCES "TenantWorkerPartition" ("id") ON DELETE SET NULL ON UPDATE SET NULL;

-- AddForeignKey
ALTER TABLE "TenantAlertEmailGroup" ADD CONSTRAINT "TenantAlertEmailGroup_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantAlertingSettings" ADD CONSTRAINT "TenantAlertingSettings_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantAlertingSettings" ADD CONSTRAINT "TenantAlertingSettings_tickerId_fkey" FOREIGN KEY ("tickerId") REFERENCES "Ticker" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantInviteLink" ADD CONSTRAINT "TenantInviteLink_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantMember" ADD CONSTRAINT "TenantMember_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantMember" ADD CONSTRAINT "TenantMember_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantResourceLimit" ADD CONSTRAINT "TenantResourceLimit_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantResourceLimitAlert" ADD CONSTRAINT "TenantResourceLimitAlert_resourceLimitId_fkey" FOREIGN KEY ("resourceLimitId") REFERENCES "TenantResourceLimit" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantResourceLimitAlert" ADD CONSTRAINT "TenantResourceLimitAlert_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantVcsProvider" ADD CONSTRAINT "TenantVcsProvider_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "UserOAuth" ADD CONSTRAINT "UserOAuth_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "UserPassword" ADD CONSTRAINT "UserPassword_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "UserSession" ADD CONSTRAINT "UserSession_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WebhookWorker" ADD CONSTRAINT "WebhookWorker_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WebhookWorker" ADD CONSTRAINT "WebhookWorker_tokenId_fkey" FOREIGN KEY ("tokenId") REFERENCES "APIToken" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WebhookWorkerRequest" ADD CONSTRAINT "WebhookWorkerRequest_webhookWorkerId_fkey" FOREIGN KEY ("webhookWorkerId") REFERENCES "WebhookWorker" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WebhookWorkerWorkflow" ADD CONSTRAINT "WebhookWorkerWorkflow_webhookWorkerId_fkey" FOREIGN KEY ("webhookWorkerId") REFERENCES "WebhookWorker" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WebhookWorkerWorkflow" ADD CONSTRAINT "WebhookWorkerWorkflow_workflowId_fkey" FOREIGN KEY ("workflowId") REFERENCES "Workflow" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Worker" ADD CONSTRAINT "Worker_dispatcherId_fkey" FOREIGN KEY ("dispatcherId") REFERENCES "Dispatcher" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Worker" ADD CONSTRAINT "Worker_webhookId_fkey" FOREIGN KEY ("webhookId") REFERENCES "WebhookWorker" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkerAssignEvent" ADD CONSTRAINT "WorkerAssignEvent_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkerLabel" ADD CONSTRAINT "WorkerLabel_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowConcurrency" ADD CONSTRAINT "WorkflowConcurrency_getConcurrencyGroupId_fkey" FOREIGN KEY ("getConcurrencyGroupId") REFERENCES "Action" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowConcurrency" ADD CONSTRAINT "WorkflowConcurrency_workflowVersionId_fkey" FOREIGN KEY ("workflowVersionId") REFERENCES "WorkflowVersion" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRun" ADD CONSTRAINT "WorkflowRun_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "WorkflowRun" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRunStickyState" ADD CONSTRAINT "WorkflowRunStickyState_workflowRunId_fkey" FOREIGN KEY ("workflowRunId") REFERENCES "WorkflowRun" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRunTriggeredBy" ADD CONSTRAINT "WorkflowRunTriggeredBy_scheduledId_fkey" FOREIGN KEY ("scheduledId") REFERENCES "WorkflowTriggerScheduledRef" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTag" ADD CONSTRAINT "WorkflowTag_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggerCronRef" ADD CONSTRAINT "WorkflowTriggerCronRef_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "WorkflowTriggers" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggerCronRef" ADD CONSTRAINT "WorkflowTriggerCronRef_tickerId_fkey" FOREIGN KEY ("tickerId") REFERENCES "Ticker" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggerEventRef" ADD CONSTRAINT "WorkflowTriggerEventRef_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "WorkflowTriggers" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggerCronRef" ADD CONSTRAINT "WorkflowTriggerCronRef_parentId_cron_name_key" UNIQUE ("parentId", "cron", "name");
ALTER TABLE "WorkflowRunTriggeredBy" ADD CONSTRAINT "WorkflowRunTriggeredBy_cronParentId_cronSchedule_cronName_fkey" FOREIGN KEY ("cronParentId", "cronSchedule", "cronName") REFERENCES "WorkflowTriggerCronRef"("parentId", "cron", "name") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggerScheduledRef" ADD CONSTRAINT "WorkflowTriggerScheduledRef_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "WorkflowVersion" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggerScheduledRef" ADD CONSTRAINT "WorkflowTriggerScheduledRef_parentWorkflowRunId_fkey" FOREIGN KEY ("parentWorkflowRunId") REFERENCES "WorkflowRun" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggerScheduledRef" ADD CONSTRAINT "WorkflowTriggerScheduledRef_tickerId_fkey" FOREIGN KEY ("tickerId") REFERENCES "Ticker" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggers" ADD CONSTRAINT "WorkflowTriggers_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggers" ADD CONSTRAINT "WorkflowTriggers_workflowVersionId_fkey" FOREIGN KEY ("workflowVersionId") REFERENCES "WorkflowVersion" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowVersion" ADD CONSTRAINT "WorkflowVersion_onFailureJobId_fkey" FOREIGN KEY ("onFailureJobId") REFERENCES "Job" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowVersion" ADD CONSTRAINT "WorkflowVersion_workflowId_fkey" FOREIGN KEY ("workflowId") REFERENCES "Workflow" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_ActionToWorker" ADD CONSTRAINT "_ActionToWorker_A_fkey" FOREIGN KEY ("A") REFERENCES "Action" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_ActionToWorker" ADD CONSTRAINT "_ActionToWorker_B_fkey" FOREIGN KEY ("B") REFERENCES "Worker" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_ServiceToWorker" ADD CONSTRAINT "_ServiceToWorker_A_fkey" FOREIGN KEY ("A") REFERENCES "Service" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_ServiceToWorker" ADD CONSTRAINT "_ServiceToWorker_B_fkey" FOREIGN KEY ("B") REFERENCES "Worker" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_StepOrder" ADD CONSTRAINT "_StepOrder_A_fkey" FOREIGN KEY ("A") REFERENCES "Step" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_StepOrder" ADD CONSTRAINT "_StepOrder_B_fkey" FOREIGN KEY ("B") REFERENCES "Step" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "LogLine" ADD CONSTRAINT "LogLine_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "StepRun" ADD CONSTRAINT "StepRun_jobRunId_fkey" FOREIGN KEY ("jobRunId") REFERENCES "JobRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "StepRun" ADD CONSTRAINT "StepRun_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "StepRunResultArchive" ADD CONSTRAINT "StepRunResultArchive_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "StreamEvent" ADD CONSTRAINT "StreamEvent_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "WorkflowRun" ADD CONSTRAINT "WorkflowRun_parentStepRunId_fkey" FOREIGN KEY ("parentStepRunId") REFERENCES "StepRun"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "WorkflowTriggerScheduledRef" ADD CONSTRAINT "WorkflowTriggerScheduledRef_parentStepRunId_fkey" FOREIGN KEY ("parentStepRunId") REFERENCES "StepRun"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "_StepRunOrder" ADD CONSTRAINT "_StepRunOrder_A_fkey" FOREIGN KEY ("A") REFERENCES "StepRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "_StepRunOrder" ADD CONSTRAINT "_StepRunOrder_B_fkey" FOREIGN KEY ("B") REFERENCES "StepRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_WorkflowToWorkflowTag" ADD CONSTRAINT "_WorkflowToWorkflowTag_A_fkey" FOREIGN KEY ("A") REFERENCES "Workflow" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_WorkflowToWorkflowTag" ADD CONSTRAINT "_WorkflowToWorkflowTag_B_fkey" FOREIGN KEY ("B") REFERENCES "WorkflowTag" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- NOTE: this is a SQL script file that contains the constraints for the database
-- it is needed because prisma does not support constraints yet
-- Modify "QueueItem" table
ALTER TABLE "QueueItem" ADD CONSTRAINT "QueueItem_priority_check" CHECK (
    "priority" >= 1
    AND "priority" <= 4
);

-- Modify "WorkflowTriggerScheduledRef" table
ALTER TABLE "WorkflowTriggerScheduledRef" ADD CONSTRAINT "WorkflowTriggerScheduledRef_priority_check" CHECK (
    priority >= 1 AND priority <= 4
);

-- Modify "WorkflowTriggerCronRef" table
ALTER TABLE "WorkflowTriggerCronRef" ADD CONSTRAINT "WorkflowTriggerCronRef_priority_check" CHECK (
    priority >= 1 AND priority <= 4
);


-- Modify "InternalQueueItem" table
ALTER TABLE "InternalQueueItem" ADD CONSTRAINT "InternalQueueItem_priority_check" CHECK (
    "priority" >= 1
    AND "priority" <= 4
);

CREATE INDEX IF NOT EXISTS "StepRun_jobRunId_status_tenantId_idx" ON "StepRun" ("jobRunId", "status", "tenantId")
WHERE
    "status" = 'PENDING';

CREATE INDEX IF NOT EXISTS "WorkflowRun_parentStepRunId" ON "WorkflowRun" ("parentStepRunId" ASC);

-- Additional indexes on workflow run
CREATE INDEX IF NOT EXISTS idx_workflowrun_concurrency ON "WorkflowRun" ("concurrencyGroupId", "createdAt");

CREATE INDEX IF NOT EXISTS idx_workflowrun_main ON "WorkflowRun" (
    "tenantId",
    "deletedAt",
    "status",
    "workflowVersionId",
    "createdAt"
);

-- Additional indexes on workflow
CREATE INDEX IF NOT EXISTS idx_workflow_version_workflow_id_order ON "WorkflowVersion" ("workflowId", "order" DESC)
WHERE
    "deletedAt" IS NULL;

CREATE INDEX IF NOT EXISTS idx_workflow_tenant_id ON "Workflow" ("tenantId");

-- Additional indexes on WorkflowTriggers
CREATE INDEX IF NOT EXISTS idx_workflow_triggers_workflow_version_id ON "WorkflowTriggers" ("workflowVersionId");

-- Additional indexes on WorkflowTriggerEventRef
CREATE INDEX idx_workflow_trigger_event_ref_event_key_parent_id ON "WorkflowTriggerEventRef" ("eventKey", "parentId");

-- Additional indexes on WorkflowRun
CREATE INDEX IF NOT EXISTS "WorkflowRun_parentId_parentStepRunId_childIndex_key" ON "WorkflowRun" ("parentId", "parentStepRunId", "childIndex")
WHERE
    "deletedAt" IS NULL;

-- CreateTable
CREATE TABLE "RetryQueueItem" (
    "id" BIGSERIAL PRIMARY KEY,
    "retryAfter" TIMESTAMP(3) NOT NULL,
    "stepRunId" UUID NOT NULL,
    "tenantId" UUID NOT NULL,
    "isQueued" BOOLEAN NOT NULL
);

-- CreateIndex
CREATE INDEX "RetryQueueItem_isQueued_tenantId_retryAfter_idx" ON "RetryQueueItem" ("isQueued" ASC, "tenantId" ASC, "retryAfter" ASC);
