import { RunsTable } from './components/runs-table';
import { RunsProvider } from './hooks/runs-provider';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { usePylon } from '@/components/support-chat';
import { Loading } from '@/components/v1/ui/loading';
import { queries } from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { BookOpen, Calendar, MessageCircle, Rocket } from 'lucide-react';
import { useMemo } from 'react';

export default function Tasks() {
  const { tenant: tenantId } = useParams({ from: appRoutes.tenantRoute.to });
  const pylon = usePylon();

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

  if (!workflowCountQuery.isSuccess || !recentRunsQuery.isSuccess) {
    return <Loading />;
  }

  const hasWorkflows = (workflowCountQuery.data?.rows?.length ?? 0) > 0;
  const hasRecentRuns = (recentRunsQuery.data?.rows?.length ?? 0) > 0;

  if (!hasWorkflows && !hasRecentRuns) {
    return (
      <div className="flex h-full items-center justify-center">
        <EmptyState
          title="No runs found"
          description="Runs are individual executions of your tasks and workflows. Dispatch a task to see runs appear here."
          actions={[
            {
              icon: <Rocket className="size-4" />,
              label: 'Get started',
              description: 'Follow our onboarding guide',
              href: `/tenants/${tenantId}/overview`,
            },
            {
              icon: <BookOpen className="size-4" />,
              label: 'Read the docs',
              description: 'Learn about running tasks',
              href: docsPages.v1.quickstart.href,
              external: true,
            },
            ...(pylon.enabled
              ? [
                  {
                    icon: <MessageCircle className="size-4" />,
                    label: 'Talk to us',
                    description: 'Chat with our support team',
                    onClick: pylon.show,
                  } as const,
                ]
              : [
                  {
                    icon: <MessageCircle className="size-4" />,
                    label: 'Join Discord',
                    description: 'Chat with the Hatchet community',
                    href: 'https://discord.com/invite/ZMeUafwH89',
                    external: true,
                  } as const,
                ]),
            {
              icon: <Calendar className="size-4" />,
              label: 'Book office hours',
              description: 'Schedule time with the Hatchet team',
              href: 'https://hatchet.run/office-hours',
              external: true,
            },
          ]}
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
