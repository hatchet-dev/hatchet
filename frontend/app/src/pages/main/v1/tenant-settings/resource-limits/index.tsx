import { SettingsPageHeader } from '../components/settings-page-header';
import { resourceLimitColumns } from './components/resource-limit-columns';
import { Subscription } from '@/components/v1/cloud/billing';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import useCloud from '@/hooks/use-cloud';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries, TenantMemberRole } from '@/lib/api';
import { useAppContext } from '@/providers/app-context';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';

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
