import { Notification, NotificationColor } from './types';
import { queries, TenantResource, TenantResourceLimit } from '@/lib/api';
import {
  getResourceLimitStatus,
  ResourceLimitStatus,
} from '@/lib/resource-limit-status';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { useQueries } from '@tanstack/react-query';
import { useMemo } from 'react';

const resourceLabels: Record<TenantResource, string> = {
  [TenantResource.WORKER]: 'Total Workers',
  [TenantResource.WORKER_SLOT]: 'Concurrency Slots',
  [TenantResource.EVENT]: 'Events',
  [TenantResource.TASK_RUN]: 'Task Runs',
  [TenantResource.CRON]: 'Cron Triggers',
  [TenantResource.SCHEDULE]: 'Schedule Triggers',
  [TenantResource.INCOMING_WEBHOOK]: 'Incoming Webhooks',
};

const statusToColor: Record<
  Exclude<ResourceLimitStatus, 'ok'>,
  NotificationColor
> = {
  warn: 'yellow',
  exhausted: 'red',
};

const limitToNotification = (
  limit: TenantResourceLimit,
  tenantId: string,
  tenantName: string,
): Notification | null => {
  const status = getResourceLimitStatus(limit);

  if (status === 'ok') {
    return null;
  }

  const label = resourceLabels[limit.resource] ?? limit.resource;

  return {
    color: statusToColor[status],
    title:
      status === 'exhausted'
        ? `${label} limit reached`
        : `Approaching ${label} limit`,
    message: `${tenantName}: ${limit.value} / ${limit.limitValue} used`,
    timestamp: limit.metadata.createdAt,
    url: appRoutes.tenantSettingsBillingRoute.to.replace('$tenant', tenantId),
  };
};

export const useResourceLimitNotifications = () => {
  const { tenantMemberships } = useUserUniverse();

  const tenants = useMemo(
    () =>
      (tenantMemberships ?? [])
        .map((m) => m.tenant)
        .filter((t): t is NonNullable<typeof t> => t != null),
    [tenantMemberships],
  );

  const resourcePolicyQueries = useQueries({
    queries: tenants.map((tenant) => ({
      ...queries.tenantResourcePolicy.get(tenant.metadata.id),
    })),
  });

  const notifications = useMemo(
    () =>
      tenants.flatMap((tenant, i) =>
        (resourcePolicyQueries[i]?.data?.limits ?? [])
          .map((limit) =>
            limitToNotification(limit, tenant.metadata.id, tenant.name),
          )
          .filter((n): n is Notification => n !== null),
      ),
    [tenants, resourcePolicyQueries],
  );

  return {
    notifications,
    isLoading: resourcePolicyQueries.some((q) => q.isLoading),
  };
};
