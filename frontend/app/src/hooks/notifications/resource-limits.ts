import { Notification, NotificationColor } from './types';
import { queries, TenantResource, TenantResourceLimit } from '@/lib/api';
import {
  getResourceLimitStatus,
  ResourceLimitStatus,
} from '@/lib/resource-limit-status';
import { useAppContext } from '@/providers/app-context';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

const TWO_MINUTES_MS = 2 * 60_000;

const resourceLabels = {
  [TenantResource.WORKER]: 'Total Workers',
  [TenantResource.WORKER_SLOT]: 'Concurrency Slots',
  [TenantResource.EVENT]: 'Events',
  [TenantResource.TASK_RUN]: 'Task Runs',
  [TenantResource.CRON]: 'Cron Triggers',
  [TenantResource.SCHEDULE]: 'Schedule Triggers',
  [TenantResource.INCOMING_WEBHOOK]: 'Incoming Webhooks',
} as const satisfies Record<TenantResource, string>;

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
    shortTitle: status === 'exhausted' ? 'Limit reached' : 'Approaching limit',
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
  const { tenantId, tenant } = useAppContext();

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(tenantId ?? ''),
    refetchInterval: TWO_MINUTES_MS,
    enabled: !!tenantId,
  });

  const notifications = useMemo(
    () =>
      (resourcePolicyQuery.data?.limits ?? [])
        .map((limit) =>
          limitToNotification(
            limit,
            tenantId ?? '',
            tenant?.name ?? tenantId ?? '',
          ),
        )
        .filter((n): n is Notification => n !== null),
    [tenantId, tenant?.name, resourcePolicyQuery.data],
  );

  return {
    notifications,
    isLoading: resourcePolicyQuery.isLoading,
  };
};
