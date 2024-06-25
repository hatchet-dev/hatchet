-- Modify "WorkflowRun" table
ALTER TABLE "WorkflowRun" DROP COLUMN "gitRepoBranch";
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
