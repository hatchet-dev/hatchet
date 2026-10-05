import { EmptyState, EmptyStateAction } from './empty-state';
import { usePylon } from '@/components/support-chat';
import { queries } from '@/lib/api';
import { DISCORD_INVITE_URL, OFFICE_HOURS_URL } from '@/lib/external-links';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { BookOpen, Calendar, MessageCircle, Rocket } from 'lucide-react';
import { useMemo } from 'react';

type OnboardingDocs = {
  href: string;
  description: string;
};

export function useOnboardingActions(docs: OnboardingDocs) {
  const { tenant: tenantId } = useParams({ from: appRoutes.tenantRoute.to });
  const { enabled: pylonEnabled, show: showPylon } = usePylon();

  return useMemo<EmptyStateAction[]>(
    () => [
      {
        icon: <Rocket className="size-4" />,
        label: 'Get started',
        description: 'Follow our onboarding guide',
        href: `/tenants/${tenantId}/overview`,
      },
      {
        icon: <BookOpen className="size-4" />,
        label: 'Read the docs',
        description: docs.description,
        href: docs.href,
        external: true,
      },
      pylonEnabled
        ? {
            icon: <MessageCircle className="size-4" />,
            label: 'Talk to us',
            description: 'Chat with our support team',
            onClick: showPylon,
          }
        : {
            icon: <MessageCircle className="size-4" />,
            label: 'Join Discord',
            description: 'Chat with the Hatchet community',
            href: DISCORD_INVITE_URL,
            external: true,
          },
      {
        icon: <Calendar className="size-4" />,
        label: 'Book office hours',
        description: 'Schedule time with the Hatchet team',
        href: OFFICE_HOURS_URL,
        external: true,
      },
    ],
    [tenantId, docs.href, docs.description, pylonEnabled, showPylon],
  );
}

type WorkflowsGuardProps = {
  title: string;
  description: string;
  docs: OnboardingDocs;
  children: React.ReactNode;
};

// Shows an onboarding placeholder instead of the page content once the probe
// confirms the tenant has no registered workflows. The page renders while the
// probe is in flight so it never blocks on the request.
export function WorkflowsGuard({
  title,
  description,
  docs,
  children,
}: WorkflowsGuardProps) {
  const { tenant: tenantId } = useParams({ from: appRoutes.tenantRoute.to });
  const actions = useOnboardingActions(docs);

  const workflowCountQuery = useQuery(
    queries.workflows.list(tenantId, { limit: 1, offset: 0 }),
  );

  const hasWorkflows = (workflowCountQuery.data?.rows?.length ?? 0) > 0;
  const confirmedNoWorkflows = workflowCountQuery.isSuccess && !hasWorkflows;

  if (!confirmedNoWorkflows) {
    return <>{children}</>;
  }

  return (
    <div className="flex h-full items-center justify-center">
      <EmptyState title={title} description={description} actions={actions} />
    </div>
  );
}
