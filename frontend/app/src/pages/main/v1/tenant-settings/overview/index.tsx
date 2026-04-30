import { SettingsPageHeader } from '../components/settings-page-header';
import { UpdateTenantForm } from './components/update-tenant-form';
import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading';
import { Switch } from '@/components/v1/ui/switch';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import api, { UpdateTenantRequest } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';

export default function TenantSettings() {
  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="General settings"
          description="Update the tenant name and analytics preferences for this tenant."
        />

        <div className="divide-y divide-border">
          <SettingRow label="Tenant Name">
            <UpdateTenant />
          </SettingRow>
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

function SettingRow({
  label,
  description,
  children,
}: {
  label: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between py-4">
      <div>
        <p className="text-sm font-medium">{label}</p>
        {description && (
          <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
        )}
      </div>
      {children}
    </div>
  );
}

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
