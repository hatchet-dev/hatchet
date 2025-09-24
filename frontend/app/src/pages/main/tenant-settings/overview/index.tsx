import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { useState } from 'react';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import api, { UpdateTenantRequest } from '@/lib/api';
import { Switch } from '@/components/ui/switch';
import { Label } from '@radix-ui/react-label';
import { Spinner } from '@/components/ui/loading';
import { capitalize } from '@/lib/utils';
import { UpdateTenantForm } from './components/update-tenant-form';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import { cloudApi } from '@/lib/api/api';

export default function TenantSettings() {
  const { tenant } = useTenantDetails();

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          {capitalize(tenant?.name || '')} Overview
        </h2>
        <Separator className="my-4" />
        <UpdateTenant />
        <Separator className="my-4" />
        <AnalyticsOptOut />
        <Separator className="my-4" />
        <InactivityTimeout />
      </div>
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
      />
    </div>
  );
};

const AnalyticsOptOut: React.FC = () => {
  const { tenant } = useTenantDetails();
  const { tenantId } = useCurrentTenantId();
  const checked = !!tenant?.analyticsOptOut;

  const [changed, setChanged] = useState(false);
  const [checkedState, setChecked] = useState(checked);
  const [isLoading, setIsLoading] = useState(false);

  const { handleApiError } = useApiError({});

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenantId, data);
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

const InactivityTimeout: React.FC = () => {
  const { data: cloudMetadata } = useQuery({
    queryKey: ['metadata'],
    queryFn: async () => {
      const res = await cloudApi.metadataGet();
      return res.data;
    },
  });

  const formatTimeoutMs = (timeoutMs: number | undefined) => {
    if (!timeoutMs || timeoutMs <= 0) {
      return 'Disabled';
    }

    const minutes = Math.floor(timeoutMs / 60000);
    if (minutes < 60) {
      return `${minutes} minute${minutes !== 1 ? 's' : ''}`;
    }

    const hours = Math.floor(minutes / 60);
    const remainingMinutes = minutes % 60;

    if (remainingMinutes === 0) {
      return `${hours} hour${hours !== 1 ? 's' : ''}`;
    }

    return `${hours} hour${hours !== 1 ? 's' : ''} ${remainingMinutes} minute${remainingMinutes !== 1 ? 's' : ''}`;
  };

  const isDisabled =
    !cloudMetadata?.inactivityLogoutMs || cloudMetadata.inactivityLogoutMs <= 0;

  return (
    <>
      <h2 className="text-xl font-semibold leading-tight text-foreground">
        Inactivity Timeout
      </h2>
      <Separator className="my-4" />
      {isDisabled ? (
        <>
          <p className="text-gray-700 dark:text-gray-300 my-4">
            Inactivity timeout is currently <strong>disabled</strong>. This
            feature automatically logs out users after a period of inactivity to
            enhance security.
          </p>
          <Alert>
            <AlertDescription>
              To enable inactivity timeout for your tenant, please contact
              support.
            </AlertDescription>
          </Alert>
        </>
      ) : (
        <>
          <p className="text-gray-700 dark:text-gray-300 my-4">
            Current inactivity logout timeout:{' '}
            <strong>
              {formatTimeoutMs(cloudMetadata?.inactivityLogoutMs)}
            </strong>
          </p>
          <Alert>
            <AlertDescription>
              Please contact support to change this configuration.
            </AlertDescription>
          </Alert>
        </>
      )}
    </>
  );
};
