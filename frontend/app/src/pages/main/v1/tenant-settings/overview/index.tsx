import { UpdateTenantForm } from './components/update-tenant-form';
import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import { Switch } from '@/components/v1/ui/switch';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import api, { UpdateTenantRequest } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { useApiError } from '@/lib/hooks';
import { capitalize } from '@/lib/utils';
import { Label } from '@radix-ui/react-label';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useState } from 'react';

export default function TenantSettings() {
  const { tenant } = useTenantDetails();

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          {capitalize(tenant?.name || '')} Overview
        </h2>
        <Separator className="my-4" />
        <UpdateTenant />
        <Separator className="my-4" />
        <TenantColor />
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
      <p className="my-4 text-gray-700 dark:text-gray-300">
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
          <Button onClick={save}>Save and Reload</Button>
        ))}
    </>
  );
};

const TENANT_COLOR_PALETTE = [
  '#3B82F6', // blue
  '#22C55E', // green
  '#F97316', // orange
  '#A855F7', // purple
  '#EF4444', // red
  '#14B8A6', // teal
  '#EAB308', // yellow
  '#64748B', // slate
] as const;

const DEFAULT_TENANT_COLOR = '#3B82F6';

const TenantColor: React.FC = () => {
  const { tenant } = useTenantDetails();
  const { tenantId } = useCurrentTenantId();

  const initial =
    tenant?.color && TENANT_COLOR_PALETTE.includes(tenant.color as any)
      ? tenant.color
      : DEFAULT_TENANT_COLOR;

  const [selected, setSelected] = useState<string>(initial);
  const [isLoading, setIsLoading] = useState(false);
  const [changed, setChanged] = useState(false);

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
      }, 300);
    },
    onError: handleApiError,
  });

  const save = () => {
    updateMutation.mutate({ color: selected });
  };

  return (
    <>
      <h2 className="text-xl font-semibold leading-tight text-foreground">
        Tenant Color
      </h2>
      <Separator className="my-4" />
      <p className="my-4 text-gray-700 dark:text-gray-300">
        Choose a color to help identify this tenant in the tenant switcher.
      </p>

      <div className="flex flex-wrap items-center gap-2">
        {TENANT_COLOR_PALETTE.map((color) => (
          <button
            key={color}
            type="button"
            aria-label={`Select tenant color ${color}`}
            onClick={() => {
              setSelected(color);
              setChanged(color !== (tenant?.color ?? DEFAULT_TENANT_COLOR));
            }}
            className={[
              'h-8 w-8 rounded-full border',
              selected === color
                ? 'ring-2 ring-foreground ring-offset-2 ring-offset-background'
                : 'opacity-90 hover:opacity-100',
            ].join(' ')}
            style={{ backgroundColor: color }}
          />
        ))}
      </div>

      <div className="mt-4">
        {changed &&
          (isLoading ? <Spinner /> : <Button onClick={save}>Save</Button>)}
      </div>
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
          <p className="my-4 text-gray-700 dark:text-gray-300">
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
          <p className="my-4 text-gray-700 dark:text-gray-300">
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
