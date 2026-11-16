import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { ResourceLimitStatus } from '@/lib/resource-limit-status';
import { useCallback } from 'react';

export const DISMISSED_LIMIT_NOTIFICATIONS_KEY =
  'hatchet:dismissed-limit-notifications';

export type DismissedLimitNotifications = Record<string, true>;

export function billingUsageDismissKey(
  organizationId: string,
  featureId: string,
  status: Exclude<ResourceLimitStatus, 'ok'>,
) {
  return `billing:${organizationId}:${featureId}:${status}`;
}

export function tenantResourceDismissKey(
  tenantId: string,
  resource: string,
  status: Exclude<ResourceLimitStatus, 'ok'>,
) {
  return `tenant:${tenantId}:${resource}:${status}`;
}

export function isLimitNotificationDismissed(
  dismissed: DismissedLimitNotifications,
  key?: string,
) {
  return !!key && dismissed[key] === true;
}

export function useDismissedLimitNotifications() {
  const [dismissed, setDismissed] =
    useLocalStorageState<DismissedLimitNotifications>(
      DISMISSED_LIMIT_NOTIFICATIONS_KEY,
      {},
    );

  const dismiss = useCallback(
    (key: string) => {
      setDismissed((prev) => ({ ...prev, [key]: true }));
    },
    [setDismissed],
  );

  return { dismissed, dismiss };
}
