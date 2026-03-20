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
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          Billing & Limits
        </h2>
        <Separator className="my-4" />

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

        <h3 className="text-xl font-semibold leading-tight text-foreground">
          Resource Limits
        </h3>
        <Separator className="my-4" />
        <p className="text-sm text-muted-foreground mb-4">
          Resource limits control usage within your tenant. When a limit is
          reached, the system will take action based on the limit type. Upgrade
          your plan or{' '}
          <a
            href="https://hatchet.run/office-hours"
            className="text-primary/70 hover:text-primary hover:underline"
          >
            contact us
          </a>{' '}
          to adjust your limits.
        </p>

        {resourceLimits.length > 0 ? (
          <SimpleTable columns={resourceLimitColumns} data={resourceLimits} />
        ) : (
          <div className="py-8 text-center text-sm text-muted-foreground">
            No resource limits configured.
          </div>
        )}
      </div>
    </div>
  );
}
