import { EmptyState, EmptyStateAction } from './empty-state';
import { usePylon } from '@/components/support-chat';
import { Loading } from '@/components/v1/ui/loading';
import { queries } from '@/lib/api';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { BookOpen, Calendar, MessageCircle, Rocket } from 'lucide-react';

type OnboardingDocs = {
  href: string;
  description: string;
};

export function useOnboardingActions(docs: OnboardingDocs) {
  const { tenant: tenantId } = useParams({ from: appRoutes.tenantRoute.to });
  const pylon = usePylon();

  const actions: EmptyStateAction[] = [
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
    pylon.enabled
      ? {
          icon: <MessageCircle className="size-4" />,
          label: 'Talk to us',
          description: 'Chat with our support team',
          onClick: pylon.show,
        }
      : {
          icon: <MessageCircle className="size-4" />,
          label: 'Join Discord',
          description: 'Chat with the Hatchet community',
          href: 'https://discord.com/invite/ZMeUafwH89',
          external: true,
        },
    {
      icon: <Calendar className="size-4" />,
      label: 'Book office hours',
      description: 'Schedule time with the Hatchet team',
      href: 'https://cal.com/team/hatchet/talk-to-us',
      external: true,
    },
  ];

  return actions;
}

type WorkflowsGuardProps = {
  title: string;
  description: string;
  docs: OnboardingDocs;
  children: React.ReactNode;
};

// Shows an onboarding placeholder instead of the page content until the tenant
// has at least one registered workflow.
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

  if (!workflowCountQuery.isSuccess) {
    return <Loading />;
  }

  const hasWorkflows = (workflowCountQuery.data?.rows?.length ?? 0) > 0;

  if (hasWorkflows) {
    return <>{children}</>;
  }

  return (
    <div className="flex h-full items-center justify-center">
      <EmptyState title={title} description={description} actions={actions} />
    </div>
  );
}
