import { RunsTable } from './components/runs-table';
import { RunsProvider } from './hooks/runs-provider';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { useOnboardingActions } from '@/components/v1/molecules/empty-state/workflows-guard';
import useControlPlane from '@/hooks/use-control-plane';
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
  const { isSelfHosted } = useControlPlane();

  const since24h = useMemo(
    () => new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
    [],
  );

  const workflowCountQuery = useQuery(
    queries.workflows.list(tenantId, { limit: 1, offset: 0 }),
  );
  const hasWorkflows = (workflowCountQuery.data?.rows?.length ?? 0) > 0;
  const confirmedNoWorkflows = workflowCountQuery.isSuccess && !hasWorkflows;
  const recentRunsQuery = useQuery({
    ...queries.v1WorkflowRuns.list(
      tenantId,
      {
        limit: 1,
        offset: 0,
        since: since24h,
        only_tasks: false,
      },
      isSelfHosted,
    ),
    enabled: confirmedNoWorkflows,
  });

  const confirmedNoRecentRuns =
    recentRunsQuery.isSuccess &&
    recentRunsQuery.data !== 'timeout' &&
    (recentRunsQuery.data?.rows?.length ?? 0) === 0;

  // The table renders while the probes are in flight and is only replaced by
  // onboarding once both probes have confirmed there is nothing to show.
  if (confirmedNoWorkflows && confirmedNoRecentRuns) {
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
