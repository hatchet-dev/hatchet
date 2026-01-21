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
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries, TenantMemberRole, TenantResourceLimit } from '@/lib/api';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { useAppContext } from '@/providers/app-context';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

export default function ResourceLimits() {
  const { tenantId } = useCurrentTenantId();
  const { membership } = useAppContext();
  const isOwner = membership === TenantMemberRole.OWNER;

  const { cloud, isCloudEnabled } = useCloud();

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(tenantId),
  });

  const billingState = useQuery({
    ...queries.cloud.billing(tenantId),
    enabled: isCloudEnabled && !!cloud?.canBill,
  });

  const billingEnabled = isCloudEnabled && cloud?.canBill;

  const resourceLimits = resourcePolicyQuery.data?.limits || [];

  const resourceLimitColumns = useMemo(
    () => [
      {
        columnLabel: 'Resource',
        cellRenderer: (limit: TenantResourceLimit) => (
          <div className="flex flex-row items-center gap-4">
            <LimitIndicator
              value={limit.value}
              alarmValue={limit.alarmValue}
              limitValue={limit.limitValue}
            />
            {limitedResources[limit.resource]}
          </div>
        ),
      },
      {
        columnLabel: 'Current Value',
        cellRenderer: (limit: TenantResourceLimit) => limit.value,
      },
      {
        columnLabel: 'Limit Value',
        cellRenderer: (limit: TenantResourceLimit) => limit.limitValue,
      },
      {
        columnLabel: 'Alarm Value',
        cellRenderer: (limit: TenantResourceLimit) => limit.alarmValue || 'N/A',
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
      {billingEnabled && (
        <>
          {isOwner ? (
            <Subscription
              active={billingState.data?.currentSubscription}
              upcoming={billingState.data?.upcomingSubscription}
              plans={billingState.data?.plans}
              coupons={billingState.data?.coupons}
            />
          ) : (
            <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
              <Alert variant="destructive">
                <ExclamationTriangleIcon className="size-4" />
                <AlertTitle>Unauthorized</AlertTitle>
                <AlertDescription>
                  You do not have permission to view billing information. Only
                  tenant owners can access billing details.
                </AlertDescription>
              </Alert>
            </div>
          )}
          <Separator className="my-4" />
        </>
      )}

      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-row items-center justify-between">
          <h3 className="text-xl font-semibold leading-tight text-foreground">
            Resource Limits
          </h3>
        </div>
        <p className="my-4 text-gray-700 dark:text-gray-300">
          Resource limits are used to control the usage of resources within a
          tenant. When a limit is reached, the system will take action based on
          the limit type. Please upgrade your plan, or{' '}
          <a href="https://hatchet.run/office-hours" className="underline">
            contact us
          </a>{' '}
          if you need to adjust your limits.
        </p>

        <SimpleTable columns={resourceLimitColumns} data={resourceLimits} />
      </div>
    </div>
  );
}
