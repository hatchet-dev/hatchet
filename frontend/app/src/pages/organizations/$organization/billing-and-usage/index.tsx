import { Subscription } from '@/components/v1/cloud/billing';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import { queries } from '@/lib/api';
import { SettingsPageHeader } from '@/pages/main/v1/tenant-settings/components/settings-page-header';
import { resourceLimitColumns } from '@/pages/main/v1/tenant-settings/resource-limits/components/resource-limit-columns';
import { useAppContext } from '@/providers/app-context';
import { appRoutes } from '@/router';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';

export default function OrganizationBillingAndUsage() {
  const { organization: organizationId } = useParams({
    from: appRoutes.organizationsRoute.to,
  });
  const appContext = useAppContext();
  const tenantId = appContext.tenantId;
  const { cloud, isCloudEnabled } = useCloud(tenantId);
  const { isControlPlaneEnabled } = useControlPlane();
  const billingEnabled = isCloudEnabled && cloud?.canBill;

  // TODO: cant this just be from the params?
  const organization =
    appContext.isUserUniverseLoaded && appContext.isCloudEnabled
      ? appContext.organizations.find(
          (org) => org.metadata.id === organizationId,
        )
      : undefined;
  const isOrganizationOwner = organization?.isOwner ?? false;

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(tenantId || ''),
    enabled: !!tenantId,
  });

  const resourceLimits = resourcePolicyQuery.data?.limits || [];

  if (!appContext.isUserUniverseLoaded || resourcePolicyQuery.isLoading) {
    return (
      <div className="h-full w-full flex-grow px-4 sm:px-6 lg:px-8">
        <Spinner />
      </div>
    );
  }

  if (!tenantId) {
    return (
      <div className="h-full w-full flex-grow">
        <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
          <SettingsPageHeader
            title="Billing & usage"
            description="Review billing details and resource limits for this organization."
          />
          <div className="py-8 text-center text-sm text-muted-foreground">
            Add a tenant to this organization to view billing and usage.
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="Billing & usage"
          description="Review billing details and resource limits for this organization."
        />

        {billingEnabled && (
          <>
            {isControlPlaneEnabled ? (
              isOrganizationOwner ? (
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
                    organization owners can access billing details.
                  </AlertDescription>
                </Alert>
              )
            ) : (
              <div className="py-8 text-center text-sm text-muted-foreground">
                Contact us to discuss plan options
              </div>
            )}
            <Separator className="my-8" />
          </>
        )}

        {resourceLimits.length > 0 ? (
          <>
            <h3 className="flex flex-row items-center gap-2 text-xl font-semibold leading-tight text-foreground">
              Resource limits
            </h3>
            <Separator className="my-4" />
            <SimpleTable
              columns={resourceLimitColumns}
              data={resourceLimits}
              rowKey={(row) => row.metadata.id}
            />
          </>
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
