import {
  Subscription,
  SubscriptionHistory,
} from '@/components/v1/cloud/billing';
import { Spinner } from '@/components/v1/ui/loading';
import useCloud from '@/hooks/use-cloud';
import { queries } from '@/lib/api';
import type { TenantResourceLimit } from '@/lib/api';
import type { TenantResourceLimit as ControlPlaneTenantResourceLimit } from '@/lib/api/generated/control-plane/data-contracts';
import { SettingsPageHeader } from '@/pages/main/v1/tenant-settings/components/settings-page-header';
import { TenantResourceLimitsTable } from '@/pages/main/v1/tenant-settings/resource-limits/components/tenant-resource-limits-table';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';

function toTenantResourceLimit(
  limit: ControlPlaneTenantResourceLimit,
): TenantResourceLimit {
  return {
    ...limit,
    resource: limit.resource as TenantResourceLimit['resource'],
  };
}

export default function OrganizationBillingPage() {
  const { organization } = useParams({
    from: appRoutes.organizationsRoute.to,
  });
  const { cloud, isCloudEnabled } = useCloud();

  const billingState = useQuery({
    ...queries.controlPlane.billing(organization),
    enabled: isCloudEnabled && !!cloud?.canBill,
  });

  const tenantResourceLimits = useQuery({
    ...queries.controlPlane.tenantResourceLimits(organization),
    enabled: isCloudEnabled,
  });

  const organizationTenants = tenantResourceLimits.data?.tenants ?? [];

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="Billing"
          description="Manage your organization subscription, payment methods, and plan changes."
        />
        <Subscription
          active={billingState.data?.currentSubscription}
          upcoming={billingState.data?.upcomingSubscription}
          plans={billingState.data?.plans}
          coupons={billingState.data?.coupons}
        />

        {tenantResourceLimits.isLoading || organizationTenants.length > 0 ? (
          <div className="mt-12 space-y-8">
            <div>
              <h2 className="text-lg font-semibold text-foreground">
                Resource limits
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                Usage limits applied to each tenant in this organization.
              </p>
            </div>

            {tenantResourceLimits.isLoading ? (
              <div className="py-6">
                <Spinner />
              </div>
            ) : (
              organizationTenants.map((tenant) => (
                <TenantResourceLimitsTable
                  key={tenant.tenantId}
                  tenantId={tenant.tenantId}
                  tenantName={
                    tenant.tenantName || tenant.tenantSlug || tenant.tenantId
                  }
                  limits={tenant.limits.map(toTenantResourceLimit)}
                />
              ))
            )}
          </div>
        ) : null}

        <div className="mt-12 space-y-4">
          <div>
            <h2 className="text-lg font-semibold text-foreground">
              Plan history
            </h2>
            <p className="mt-1 text-sm text-muted-foreground">
              A record of subscription changes for this organization.
            </p>
          </div>

          <SubscriptionHistory
            history={billingState.data?.subscriptionHistory}
            plans={billingState.data?.plans}
          />
        </div>
      </div>
    </div>
  );
}
