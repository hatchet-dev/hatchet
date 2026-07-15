import { SettingsPageHeader } from '../components/settings-page-header';
import { ReadOnlyValue, SettingRow } from '../components/settings-row';
import { UpdateTenantForm } from './components/update-tenant-form';
import { TenantSwitcher } from '@/components/v1/molecules/nav-bar/tenant-switcher';
import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading';
import { Switch } from '@/components/v1/ui/switch';
import { useOrganizations } from '@/hooks/use-organizations';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import api, { UpdateTenantRequest } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { MembershipsContextType } from '@/lib/outlet';
import { useOutletContext } from '@/lib/router-helpers';
import { useMutation } from '@tanstack/react-query';
import { useMemo, useState } from 'react';

export default function TenantSettings() {
  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="General"
          description="Update the tenant name and analytics preferences for this tenant."
        />

        <div className="divide-y divide-border">
          <CurrentTenant />
          <SettingRow
            label="Tenant Name"
            description="The display name for this tenant, shown across the dashboard."
          >
            <UpdateTenant />
          </SettingRow>
          <TenantApiUrl />
          <TenantRegion />
          <SettingRow
            label="Analytics Opt-Out"
            description="Disable usage analytics collection for this tenant."
          >
            <AnalyticsOptOut />
          </SettingRow>
        </div>
      </div>
    </div>
  );
}

const CurrentTenant: React.FC = () => {
  const ctx = useOutletContext<MembershipsContextType>();
  const { organizationId } = useTenantDetails();
  const { getOrganizationIdForTenant } = useOrganizations();

  const organizationMemberships = useMemo(() => {
    const memberships = ctx?.memberships ?? [];

    if (!organizationId) {
      return memberships;
    }

    return memberships.filter(
      (m) =>
        m.tenant &&
        getOrganizationIdForTenant(m.tenant.metadata.id) === organizationId,
    );
  }, [ctx?.memberships, organizationId, getOrganizationIdForTenant]);

  return (
    <SettingRow
      label="Current Tenant"
      description={
        organizationId
          ? 'Switch between tenants in this organization.'
          : 'Switch between your tenants.'
      }
    >
      <TenantSwitcher
        memberships={organizationMemberships}
        className="w-[280px]"
      />
    </SettingRow>
  );
};

const TenantApiUrl: React.FC = () => {
  const { tenant } = useTenantDetails();
  const apiUrl = tenant?.serverUrl || window.location.origin;

  return (
    <SettingRow
      label="API URL"
      description="The base URL for this tenant's REST API."
    >
      <ReadOnlyValue value={apiUrl} />
    </SettingRow>
  );
};

const TenantRegion: React.FC = () => {
  const { tenant } = useTenantDetails();

  if (!tenant?.region) {
    return null;
  }

  return (
    <SettingRow
      label="Region"
      description="The control-plane region this tenant is deployed to."
    >
      <ReadOnlyValue value={tenant.region} />
    </SettingRow>
  );
};

const UpdateTenant: React.FC = () => {
  const [isLoading, setIsLoading] = useState(false);
  const { tenantId } = useCurrentTenantId();
  const { handleApiError } = useApiError({});

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenantId, data);
    },
    onMutate: () => setIsLoading(true),
    onSuccess: () => window.location.reload(),
    onError: handleApiError,
  });

  return (
    <UpdateTenantForm
      isLoading={isLoading}
      onSubmit={(data) => updateMutation.mutate(data)}
    />
  );
};

const AnalyticsOptOut: React.FC = () => {
  const { tenant } = useTenantDetails();
  const { tenantId } = useCurrentTenantId();
  const [changed, setChanged] = useState(false);
  const [checkedState, setChecked] = useState(!!tenant?.analyticsOptOut);
  const [isLoading, setIsLoading] = useState(false);
  const { handleApiError } = useApiError({});

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenantId, data);
    },
    onMutate: () => setIsLoading(true),
    onSuccess: () => window.location.reload(),
    onSettled: () => setTimeout(() => setIsLoading(false), 1000),
    onError: handleApiError,
  });

  return (
    <div className="flex items-center gap-3">
      <Switch
        id="aoo"
        checked={checkedState}
        onClick={() => {
          setChecked((s) => !s);
          setChanged(true);
        }}
      />
      {changed &&
        (isLoading ? (
          <Spinner />
        ) : (
          <Button
            size="sm"
            onClick={() =>
              updateMutation.mutate({ analyticsOptOut: checkedState })
            }
          >
            Save
          </Button>
        ))}
    </div>
  );
};
