import { billingUsageDismissKey } from './dismissed';
import { Notification, NotificationColor } from './types';
import { isPeriodUsageFeature } from '@/components/v1/cloud/billing/usage-features';
import useControlPlane from '@/hooks/use-control-plane';
import { useTenantDetails } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { OrganizationUsageFeature } from '@/lib/api/generated/control-plane/data-contracts';
import {
  getUsageLimitStatus,
  ResourceLimitStatus,
} from '@/lib/resource-limit-status';
import { useAppContext } from '@/providers/app-context';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

const TWO_MINUTES_MS = 2 * 60_000;

const statusToColor: Record<
  Exclude<ResourceLimitStatus, 'ok'>,
  NotificationColor
> = {
  warn: 'yellow',
  exhausted: 'red',
};

function formatUsageCount(value: number) {
  return new Intl.NumberFormat('en-US').format(value);
}

const featureToNotification = (
  feature: OrganizationUsageFeature,
  organizationId: string,
  organizationName: string,
  timestamp: string,
): Notification | null => {
  // Period included amounts for task runs / events are billing thresholds,
  // not hard caps. Daily-limit features still notify.
  if (isPeriodUsageFeature(feature.featureId)) {
    return null;
  }

  const status = getUsageLimitStatus(feature);

  if (status === 'ok') {
    return null;
  }

  return {
    color: statusToColor[status],
    shortTitle: status === 'exhausted' ? 'Limit reached' : 'Approaching limit',
    title:
      status === 'exhausted'
        ? `${feature.name} limit reached`
        : `Approaching ${feature.name} limit`,
    message: `${organizationName}: ${formatUsageCount(feature.usage)} / ${formatUsageCount(feature.includedUsage)} used`,
    timestamp,
    dismissKey: billingUsageDismissKey(
      organizationId,
      feature.featureId,
      status,
    ),
    url: appRoutes.organizationBillingRoute.to.replace(
      '$organization',
      organizationId,
    ),
  };
};

export const useBillingUsageNotifications = () => {
  const { organizationId } = useTenantDetails();
  const appContext = useAppContext();
  const { isControlPlaneEnabled, canBill } = useControlPlane();

  const organization = useMemo(() => {
    if (!organizationId || !appContext.isUserUniverseLoaded) {
      return undefined;
    }

    return appContext.organizations?.find(
      (org) => org.metadata.id === organizationId,
    );
  }, [
    appContext.isUserUniverseLoaded,
    appContext.organizations,
    organizationId,
  ]);
  const usageQuery = useQuery({
    ...queries.controlPlane.usage(organizationId ?? ''),
    refetchInterval: TWO_MINUTES_MS,
    enabled: isControlPlaneEnabled && canBill && !!organizationId,
  });

  const notifications = useMemo(() => {
    if (!organizationId) {
      return [];
    }

    const timestamp =
      usageQuery.data?.periodEnd ??
      usageQuery.data?.periodStart ??
      new Date(0).toISOString();
    const organizationName = organization?.name ?? organizationId;

    return (usageQuery.data?.features ?? [])
      .map((feature) =>
        featureToNotification(
          feature,
          organizationId,
          organizationName,
          timestamp,
        ),
      )
      .filter((n): n is Notification => n !== null);
  }, [
    organization?.name,
    organizationId,
    usageQuery.data?.features,
    usageQuery.data?.periodEnd,
    usageQuery.data?.periodStart,
  ]);

  return {
    notifications,
    isLoading: usageQuery.isLoading,
  };
};
