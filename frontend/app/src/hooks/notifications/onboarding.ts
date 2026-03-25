import { Notification } from './types';
import { queries } from '@/lib/api';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

function exactlyOneElement<T>(array: T[]): array is [T] {
  return array.length === 1;
}

export const useOnboardingNotifications = () => {
  const { tenantMemberships } = useUserUniverse();

  const tenants = useMemo(
    () =>
      (tenantMemberships ?? [])
        .map((m) => m.tenant)
        .filter((t): t is NonNullable<typeof t> => t != null),
    [tenantMemberships],
  );

  const hasOneTenant = exactlyOneElement(tenants);
  const tenantId = hasOneTenant ? tenants[0].metadata.id : '';

  const workflowQuery = useQuery({
    ...queries.workflows.list(tenantId, { limit: 1 }),
    enabled: hasOneTenant,
  });

  const tokenQuery = useQuery({
    ...queries.tokens.list(tenantId),
    enabled: hasOneTenant,
  });

  const isLoading = workflowQuery.isLoading || tokenQuery.isLoading;

  const notifications = useMemo((): Notification[] => {
    // Why not when they have zero tenants?  Because in that case they're getting redirected to the tenant creation screen
    if (!hasOneTenant || isLoading) {
      return [];
    }

    const hasWorkflows = (workflowQuery.data?.rows?.length ?? 0) > 0;
    const hasTokens = (tokenQuery.data?.rows?.length ?? 0) > 0;

    if (hasWorkflows || hasTokens) {
      return [];
    }

    return [
      {
        color: 'blue',
        title: 'Get started with Hatchet',
        message: 'Create an API token and run your first workflow',
        timestamp: new Date().toISOString(),
        url: appRoutes.tenantOverviewRoute.to.replace('$tenant', tenantId),
      },
    ];
  }, [hasOneTenant, isLoading, tenantId, workflowQuery.data, tokenQuery.data]);

  return {
    notifications,
    isLoading,
  };
};
