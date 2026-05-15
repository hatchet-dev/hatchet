import { SettingsPageHeader } from '../components/settings-page-header';
import {
  limitDurationMap,
  limitedResources,
  LimitIndicator,
} from './components/resource-limit-columns';
import { Subscription } from '@/components/v1/cloud/billing';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries, TenantMemberRole, TenantResourceLimit } from '@/lib/api';
import { useAppContext } from '@/providers/app-context';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

const BILLING_SYNC_REFETCH_INTERVAL_MS = 5000;

export default function ResourceLimits() {
  const { tenantId } = useCurrentTenantId();
  const { membership } = useAppContext();
  const isOwner = membership === TenantMemberRole.OWNER;

  const { cloud, isCloudEnabled } = useCloud();
  const { isControlPlaneEnabled } = useControlPlane();

  const billingEnabled =
    isControlPlaneEnabled && isCloudEnabled && !!cloud?.canBill;
  const billingSyncRefetchInterval = billingEnabled
    ? BILLING_SYNC_REFETCH_INTERVAL_MS
    : false;

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(tenantId),
    refetchInterval: billingSyncRefetchInterval,
  });

  const billingState = useQuery({
    ...queries.controlPlane.billing(tenantId),
    enabled: billingEnabled,
    refetchInterval: billingSyncRefetchInterval,
  });

  const isDedicatedCloud = !isControlPlaneEnabled && !!cloud?.canBill;

  const resourceLimits = resourcePolicyQuery.data?.limits || [];

  const resourceLimitColumns = useMemo(
    () => [
      {
        columnLabel: 'Resource',
        cellRenderer: (limit: TenantResourceLimit) => (
          <div className="flex flex-row items-center gap-3">
            <LimitIndicator
              value={limit.value}
              alarmValue={limit.alarmValue}
              limitValue={limit.limitValue}
            />
            <span className="font-medium text-foreground">
              {limitedResources[limit.resource]}
            </span>
          </div>
        ),
      },
      {
        columnLabel: 'Current Value',
        cellRenderer: (limit: TenantResourceLimit) => (
          <span className="tabular-nums">{limit.value}</span>
        ),
      },
      {
        columnLabel: 'Limit Value',
        cellRenderer: (limit: TenantResourceLimit) => (
          <span className="tabular-nums">{limit.limitValue}</span>
        ),
      },
      {
        columnLabel: 'Alarm Value',
        cellRenderer: (limit: TenantResourceLimit) => (
          <span className="tabular-nums">{limit.alarmValue || 'N/A'}</span>
        ),
      },
      {
        columnLabel: 'Meter Window',
        cellRenderer: (limit: TenantResourceLimit) =>
          (limit.window || '-') in limitDurationMap
            ? limitDurationMap[limit.window || '-']
            : limit.window,
      },
      {
        columnLabel: 'Last Refill',
        cellRenderer: (limit: TenantResourceLimit) =>
          !limit.window
            ? 'N/A'
            : limit.lastRefill && <RelativeDate date={limit.lastRefill} />,
      },
    ],
    [],
  );

  if (resourcePolicyQuery.isLoading || billingState.isLoading) {
    return (
      <div className="h-full w-full flex-grow px-4 sm:px-6 lg:px-8">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="Resource limit settings"
          description="Review billing details and the resource limits currently applied to this tenant."
        />

        {billingEnabled && !isDedicatedCloud && (
          <>
            {isOwner ? (
              <Subscription
                active={billingState.data?.currentSubscription}
                upcoming={billingState.data?.upcomingSubscription}
                plans={billingState.data?.plans}
                coupons={billingState.data?.coupons}
              />
            ) : (
              <Alert variant="destructive">
                <ExclamationTriangleIcon className="size-4" />
                <AlertTitle>Unauthorized</AlertTitle>
                <AlertDescription>
                  You do not have permission to view billing information. Only
                  tenant owners can access billing details.
                </AlertDescription>
              </Alert>
            )}
            <Separator className="my-8" />
          </>
        )}

        {isDedicatedCloud && (
          <Alert variant="destructive">
            <ExclamationTriangleIcon className="size-4" />
            <AlertTitle>Dedicated Cloud</AlertTitle>
            <AlertDescription>
              Please contact us to discuss your plan.
            </AlertDescription>
          </Alert>
        )}

        {resourceLimits.length > 0 ? (
          <SimpleTable
            columns={resourceLimitColumns}
            data={resourceLimits}
            rowKey={(row) => row.metadata.id}
          />
        ) : (
          <div className="py-8 text-center text-sm text-muted-foreground">
            No resource limits configured. Upgrade your plan or{' '}
            <a
              href="https://hatchet.run/office-hours"
              className="text-primary/70 hover:text-primary hover:underline"
            >
              contact us
            </a>{' '}
            to adjust your limits.
          </div>
        )}
      </div>
    </div>
  );
}
