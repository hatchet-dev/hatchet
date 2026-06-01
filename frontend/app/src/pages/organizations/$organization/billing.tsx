import { Subscription } from '@/components/v1/cloud/billing';
import useCloud from '@/hooks/use-cloud';
import { queries } from '@/lib/api';
import { SettingsPageHeader } from '@/pages/main/v1/tenant-settings/components/settings-page-header';
import { TenantResourceLimitsTable } from '@/pages/main/v1/tenant-settings/resource-limits/components/tenant-resource-limits-table';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { useMemo } from 'react';

export default function OrganizationBillingPage() {
  const { organization } = useParams({
    from: appRoutes.organizationsRoute.to,
  });
  const { cloud, isCloudEnabled } = useCloud();
  const userUniverse = useUserUniverse();

  const billingState = useQuery({
    ...queries.controlPlane.billing(organization),
    enabled: isCloudEnabled && !!cloud?.canBill,
  });

  const organizationTenants = useMemo(() => {
    if (!userUniverse.isLoaded || !userUniverse.organizations) {
      return [];
    }

    return (
      userUniverse.organizations.find((org) => org.metadata.id === organization)
        ?.tenants ?? []
    );
  }, [organization, userUniverse.isLoaded, userUniverse.organizations]);

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

        {organizationTenants.length > 0 ? (
          <div className="mt-12 space-y-8">
            <div>
              <h2 className="text-lg font-semibold text-foreground">
                Resource limits
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                Usage limits applied to each tenant in this organization.
              </p>
            </div>

            {organizationTenants.map((tenant) => (
              <TenantResourceLimitsTable
                key={tenant.id}
                tenantId={tenant.id}
                tenantName={tenant.name ?? tenant.slug ?? tenant.id}
              />
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
}
