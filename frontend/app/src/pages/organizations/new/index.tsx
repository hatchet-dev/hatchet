import { NewOrganizationSaverForm } from '@/components/forms/new-organization-saver-form';
import { SetupCard, SetupScreen } from '@/components/layout/setup-card';
import { UpgradeGateContent } from '@/components/v1/cloud/billing/upgrade-gate-dialog';
import { Button } from '@/components/v1/ui/button';
import { useOrganizations } from '@/hooks/use-organizations';
import { useTenantDetails } from '@/hooks/use-tenant';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { XMarkIcon } from '@heroicons/react/24/outline';
import { useNavigate } from '@tanstack/react-router';
import { useState } from 'react';

export default function OrganizationsNew() {
  const navigate = useNavigate();
  const { canCreateOrganization, organizations, isLoaded } = useUserUniverse();
  const { tenant } = useTenantDetails();
  const { getOrganizationForTenant } = useOrganizations();
  const [limitReached, setLimitReached] = useState(false);
  const currentOrganization = tenant
    ? getOrganizationForTenant(tenant.metadata.id)
    : undefined;
  const ownedOrganization =
    currentOrganization ?? organizations?.find((org) => org.isOwner);
  const showUpgrade = Boolean(
    isLoaded && ownedOrganization && (!canCreateOrganization || limitReached),
  );

  return (
    <SetupScreen
      topRight={
        <Button
          variant="ghost"
          size="sm"
          onClick={() => window.history.back()}
          className="h-8 w-8 p-0"
        >
          <XMarkIcon className="size-4" />
        </Button>
      }
    >
      {showUpgrade ? (
        <UpgradeGateContent
          gate="organizations"
          organizationId={ownedOrganization!.metadata.id}
          organizationName={ownedOrganization!.name}
          onDismiss={() => window.history.back()}
        />
      ) : (
        <SetupCard title="Create a new organization">
          <NewOrganizationSaverForm
            onLimitReached={() => setLimitReached(true)}
            afterSave={({ tenant }) =>
              navigate({
                to: appRoutes.tenantOnboardingRoute.to,
                params: { tenant: tenant.id },
              })
            }
          />
        </SetupCard>
      )}
    </SetupScreen>
  );
}
