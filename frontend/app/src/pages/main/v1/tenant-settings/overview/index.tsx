import { Button } from '@/components/v1/ui/button';
import { Separator } from '@/components/v1/ui/separator';
import { useState } from 'react';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import api, { queries, TenantVersion, UpdateTenantRequest } from '@/lib/api';
import { Switch } from '@/components/v1/ui/switch';
import { Label } from '@radix-ui/react-label';
import { Spinner } from '@/components/v1/ui/loading';
import { capitalize } from '@/lib/utils';
import { UpdateTenantForm } from './components/update-tenant-form';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
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
        <Separator className="my-4" />
        <TenantVersionSwitcher />
      </div>
    </div>
  );
}

const TenantVersionSwitcher = () => {
  const { tenantId } = useCurrentTenantId();
  const { tenant } = useTenantDetails();
  const queryClient = useQueryClient();
  const [showDowngradeModal, setShowDowngradeModal] = useState(false);

  const { handleApiError } = useApiError({});

  const { mutate: updateTenant, isPending } = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenantId, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queries.user.listTenantMemberships.queryKey,
      });

      window.location.href = '/';
    },
    onError: handleApiError,
  });

  // Only show for V1 tenants
  if (tenant?.version === TenantVersion.V0) {
    return null;
  }

  return (
    <>
      <div className="flex flex-col gap-y-2">
        <h2 className="text-xl font-semibold leading-tight text-foreground">
          Tenant Version
        </h2>
        <p className="text-sm text-muted-foreground">
          You can downgrade your tenant to V0 if needed. Please help us improve
          V1 by reporting any bugs in our{' '}
          <a
            href="https://github.com/hatchet-dev/hatchet/issues"
            target="_blank"
            rel="noopener noreferrer"
            className="text-indigo-400 hover:underline"
          >
            Github issues.
          </a>
        </p>
        <Button
          onClick={() => setShowDowngradeModal(true)}
          disabled={isPending}
          variant="destructive"
          className="w-fit"
        >
          {isPending ? <Spinner /> : null}
          Downgrade to V0
        </Button>
      </div>

      <Dialog open={showDowngradeModal} onOpenChange={setShowDowngradeModal}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Downgrade to V0</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <Alert variant="warn">
              <AlertTitle>Warning</AlertTitle>
              <AlertDescription>
                Downgrading to V0 will remove access to V1 features and may
                affect your existing workflows. This action should only be taken
                if absolutely necessary.
              </AlertDescription>
            </Alert>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDowngradeModal(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                updateTenant({
                  version: TenantVersion.V0,
                });
              }}
              disabled={isPending}
            >
              {isPending ? <Spinner /> : null}
              Confirm Downgrade
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
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
