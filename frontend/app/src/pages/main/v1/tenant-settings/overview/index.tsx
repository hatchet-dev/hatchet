import { Button } from '@/components/v1/ui/button';
import { Separator } from '@/components/v1/ui/separator';
import { TenantContextType } from '@/lib/outlet';
import { useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import api, { Tenant, UpdateTenantRequest } from '@/lib/api';
import { Switch } from '@/components/v1/ui/switch';
import { Label } from '@radix-ui/react-label';
import { Spinner } from '@/components/v1/ui/loading';
import { capitalize } from '@/lib/utils';
import { UpdateTenantForm } from './components/update-tenant-form';

export default function TenantSettings() {
  const { tenant } = useOutletContext<TenantContextType>();

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          {capitalize(tenant.name)} Overview
        </h2>
        <Separator className="my-4" />
        <UpdateTenant tenant={tenant} />
        <Separator className="my-4" />
        <AnalyticsOptOut tenant={tenant} />
      </div>
    </div>
  );
}

const UpdateTenant: React.FC<{ tenant: Tenant }> = ({ tenant }) => {
  const [isLoading, setIsLoading] = useState(false);

  const { handleApiError } = useApiError({});

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenant.metadata.id, data);
    },
    onMutate: () => {
      setIsLoading(true);
    },
    onSuccess: () => {
      window.location.reload();
    },
    onError: handleApiError,
  });

  return (
    <div className="w-fit">
      <UpdateTenantForm
        isLoading={isLoading}
        onSubmit={(data) => {
          updateMutation.mutate(data);
        }}
        tenant={tenant}
      />
    </div>
  );
};

const AnalyticsOptOut: React.FC<{ tenant: Tenant }> = ({ tenant }) => {
  const checked = !!tenant.analyticsOptOut;

  const [changed, setChanged] = useState(false);
  const [checkedState, setChecked] = useState(checked);
  const [isLoading, setIsLoading] = useState(false);

  const { handleApiError } = useApiError({});

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenant.metadata.id, data);
    },
    onMutate: () => {
      setIsLoading(true);
    },
    onSuccess: () => {
      window.location.reload();
    },
    onSettled: () => {
      setTimeout(() => {
        setIsLoading(false);
      }, 1000);
    },
    onError: handleApiError,
  });

  const save = () => {
    updateMutation.mutate({
      analyticsOptOut: checkedState,
    });
  };

  return (
    <>
      <h2 className="text-xl font-semibold leading-tight text-foreground">
        Analytics Opt-Out
      </h2>
      <Separator className="my-4" />
      <p className="text-gray-700 dark:text-gray-300 my-4">
        Choose whether to opt out of all analytics tracking.
      </p>
      <div className="flex items-center space-x-2">
        <Switch
          id="aoo"
          checked={checkedState}
          onClick={() => {
            setChecked((checkedState) => !checkedState);
            setChanged(true);
          }}
        />
        <Label htmlFor="aoo" className="text-sm">
          Analytics Opt-Out
        </Label>
      </div>
      {changed &&
        (isLoading ? (
          <Spinner />
        ) : (
          <Button onClick={save} className="mt-4">
            Save and Reload
          </Button>
        ))}
    </>
  );
};
