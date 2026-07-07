import { RunsTable } from './components/runs-table';
import { RunsProvider } from './hooks/runs-provider';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { useOnboardingActions } from '@/components/v1/molecules/empty-state/workflows-guard';
import { Loading } from '@/components/v1/ui/loading';
import { queries } from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { useMemo } from 'react';

export default function RunsPage() {
  const { tenant: tenantId } = useParams({ from: appRoutes.tenantRoute.to });
  const actions = useOnboardingActions({
    href: docsPages.v1.quickstart.href,
    description: 'Learn about running tasks',
  });

  const since24h = useMemo(
    () => new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
    [],
  );

  const workflowCountQuery = useQuery(
    queries.workflows.list(tenantId, { limit: 1, offset: 0 }),
  );
  const recentRunsQuery = useQuery(
    queries.v1WorkflowRuns.list(tenantId, {
      limit: 1,
      offset: 0,
      since: since24h,
      only_tasks: false,
    }),
  );

  if (workflowCountQuery.isLoading || recentRunsQuery.isLoading) {
    return <Loading />;
  }

  const hasWorkflows = (workflowCountQuery.data?.rows?.length ?? 0) > 0;
  const hasRecentRuns = (recentRunsQuery.data?.rows?.length ?? 0) > 0;

  // Fail open on probe errors: the table's own error handling is more useful
  // than trapping the user on the onboarding placeholder.
  const probesErrored = workflowCountQuery.isError || recentRunsQuery.isError;

  if (!probesErrored && !hasWorkflows && !hasRecentRuns) {
    return (
      <div className="flex h-full items-center justify-center">
        <EmptyState
          title="No runs found"
          description="Runs are individual executions of your tasks and workflows. Dispatch a task to see runs appear here."
          actions={actions}
        />
      </div>
    );
  }

  return (
    <div className="size-full flex-grow">
      <RunsProvider
        tableKey="workflow-runs-main"
        persistColumnVisibilityKey="workflow-runs-main"
      >
        <RunsTable />
      </RunsProvider>
    </div>
  );
}
