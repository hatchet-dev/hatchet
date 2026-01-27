import {
  limitDurationMap,
  limitedResources,
  LimitIndicator,
} from './components/resource-limit-columns';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries, TenantResourceLimit } from '@/lib/api';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { useAppContext } from '@/providers/app-context';
import { appRoutes } from '@/router';
import {
  ArrowRightIcon,
  InformationCircleIcon,
} from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { useMemo } from 'react';

export default function ResourceLimits() {
  const { tenantId } = useCurrentTenantId();
  const { getCurrentOrganization } = useAppContext();

  const { cloud, isCloudEnabled } = useCloud();

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(tenantId),
  });

  const billingEnabled = isCloudEnabled && cloud?.canBill;
  const currentOrg = getCurrentOrganization();

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

  if (resourcePolicyQuery.isLoading) {
    return (
      <div className="h-full w-full flex-grow px-4 sm:px-6 lg:px-8">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="h-full w-full flex-grow">
      {billingEnabled && currentOrg && (
        <>
          <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
            <Alert>
              <InformationCircleIcon className="size-4" />
              <AlertTitle>Subscription Management</AlertTitle>
              <AlertDescription className="flex items-center justify-between">
                <span>
                  Subscription and billing is now managed at the organization
                  level.
                </span>
                <Link
                  to={appRoutes.organizationsRoute.to}
                  params={{ organization: currentOrg.metadata.id }}
                >
                  <Button variant="outline" size="sm">
                    Go to Organization Billing
                    <ArrowRightIcon className="ml-2 size-4" />
                  </Button>
                </Link>
              </AlertDescription>
            </Alert>
          </div>
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
