-- Modify "WorkflowRun" table
ALTER TABLE "WorkflowRun" DROP COLUMN "gitRepoBranch";
-- Create "WebhookWorker" table
CREATE TABLE "WebhookWorker" ("id" uuid NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "name" text NOT NULL, "secret" text NOT NULL, "url" text NOT NULL, "tokenValue" text NULL, "deleted" boolean NOT NULL DEFAULT false, "tokenId" uuid NULL, "tenantId" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "WebhookWorker_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON UPDATE CASCADE ON DELETE CASCADE, CONSTRAINT "WebhookWorker_tokenId_fkey" FOREIGN KEY ("tokenId") REFERENCES "APIToken" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "WebhookWorker_id_key" to table: "WebhookWorker"
CREATE UNIQUE INDEX "WebhookWorker_id_key" ON "WebhookWorker" ("id");
-- Create index "WebhookWorker_url_key" to table: "WebhookWorker"
CREATE UNIQUE INDEX "WebhookWorker_url_key" ON "WebhookWorker" ("url");
-- Create "WebhookWorkerWorkflow" table
CREATE TABLE "WebhookWorkerWorkflow" ("id" uuid NOT NULL, "webhookWorkerId" uuid NOT NULL, "workflowId" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "WebhookWorkerWorkflow_webhookWorkerId_fkey" FOREIGN KEY ("webhookWorkerId") REFERENCES "WebhookWorker" ("id") ON UPDATE CASCADE ON DELETE CASCADE, CONSTRAINT "WebhookWorkerWorkflow_workflowId_fkey" FOREIGN KEY ("workflowId") REFERENCES "Workflow" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "WebhookWorkerWorkflow_id_key" to table: "WebhookWorkerWorkflow"
CREATE UNIQUE INDEX "WebhookWorkerWorkflow_id_key" ON "WebhookWorkerWorkflow" ("id");
-- Create index "WebhookWorkerWorkflow_webhookWorkerId_workflowId_key" to table: "WebhookWorkerWorkflow"
CREATE UNIQUE INDEX "WebhookWorkerWorkflow_webhookWorkerId_workflowId_key" ON "WebhookWorkerWorkflow" ("webhookWorkerId", "workflowId");
-- Drop "WorkflowDeploymentConfig" table
DROP TABLE "WorkflowDeploymentConfig";
-- Drop "_GithubAppInstallationToGithubWebhook" table
DROP TABLE "_GithubAppInstallationToGithubWebhook";
-- Drop "GithubAppInstallation" table
DROP TABLE "GithubAppInstallation";
-- Drop "_GithubAppOAuthToUser" table
DROP TABLE "_GithubAppOAuthToUser";
-- Drop "GithubAppOAuth" table
DROP TABLE "GithubAppOAuth";
-- Drop "GithubPullRequestComment" table
DROP TABLE "GithubPullRequestComment";
-- Drop "_GithubPullRequestToWorkflowRun" table
DROP TABLE "_GithubPullRequestToWorkflowRun";
-- Drop "GithubPullRequest" table
DROP TABLE "GithubPullRequest";
-- Drop "GithubWebhook" table
DROP TABLE "GithubWebhook";
