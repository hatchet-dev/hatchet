import { SettingsPageHeader } from '../components/settings-page-header';
import { resourceLimitColumns } from './components/resource-limit-columns';
import { Subscription } from '@/components/v1/cloud/billing';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries, TenantMemberRole } from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { useAppContext } from '@/providers/app-context';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';

export default function ResourceLimits() {
  const { tenantId } = useCurrentTenantId();
  const appContext = useAppContext();
  const isOwner = appContext.membership === TenantMemberRole.OWNER;

  const { cloud, isCloudEnabled } = useCloud(tenantId);
  const { isControlPlaneEnabled } = useControlPlane();
  const billingEnabled = isCloudEnabled && !!cloud?.canBill;
  const organizationId =
    appContext.isUserUniverseLoaded && appContext.isCloudEnabled
      ? appContext.organizations.find((org) =>
          org.tenants.some((tenant) => tenant.id === tenantId),
        )?.metadata.id
      : undefined;

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(tenantId),
  });

  const resourceLimits = resourcePolicyQuery.data?.limits || [];

  if (resourcePolicyQuery.isLoading || !appContext.isUserUniverseLoaded) {
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
          title={billingEnabled ? 'Billing & usage' : 'Resource limits'}
          description={
            billingEnabled
              ? 'Review billing details and resource limits for this tenant.'
              : 'Review currently configured resource limits for this tenant. Once limits are exceeded, requests will be rejected.'
          }
        />

        {billingEnabled && (
          <>
            {isControlPlaneEnabled ? (
              isOwner ? (
                <Subscription
                  tenantId={tenantId}
                  organizationId={organizationId}
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
              )
            ) : (
              <div className="py-8 text-center text-sm text-muted-foreground">
                Contact us to discuss plan options.
              </div>
            )}
            <Separator className="my-8" />
          </>
        )}

        {resourceLimits.length > 0 ? (
          <>
            {billingEnabled && (
              <>
                <h3 className="flex flex-row items-center gap-2 text-xl font-semibold leading-tight text-foreground">
                  Resource limits
                </h3>
                <Separator className="my-4" />
              </>
            )}
            <SimpleTable
              columns={resourceLimitColumns}
              data={resourceLimits}
              rowKey={(row) => row.metadata.id}
            />
          </>
        ) : (
          <>
            {billingEnabled ? (
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
            ) : (
              <div className="flex flex-col items-center gap-y-4 py-8 text-center text-sm text-muted-foreground">
                <p>No resource limits configured.</p>
                <DocsButton
                  doc={docsPages.v1['rate-limits']}
                  label="Learn about resource limits"
                />
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
