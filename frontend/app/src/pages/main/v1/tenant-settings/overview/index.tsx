import { Button } from '@/components/v1/ui/button';
import { Separator } from '@/components/v1/ui/separator';
import { TenantContextType } from '@/lib/outlet';
import { useState } from 'react';
import {
  createSearchParams,
  useNavigate,
  useOutletContext,
} from 'react-router-dom';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import api, {
  queries,
  Tenant,
  TenantUIVersion,
  TenantVersion,
  UpdateTenantRequest,
} from '@/lib/api';
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
import { AxiosError } from 'axios';
import useCloudFeatureFlags from '@/pages/auth/hooks/use-cloud-feature-flags';

export default function TenantSettings() {
  const { tenant } = useOutletContext<TenantContextType>();
  const featureFlags = useCloudFeatureFlags(tenant?.metadata.id || '');

  const hasUIVersionFlag =
    featureFlags?.data['has-ui-version-upgrade-available'] === 'true';

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          {capitalize(tenant.name)} Overview
        </h2>
        <Separator className="my-4" />
        <UpdateTenant tenant={tenant} />
        <Separator className="my-4" />
        <AnalyticsOptOut tenant={tenant} />
        <Separator className="my-4" />
        <TenantVersionSwitcher />
        {hasUIVersionFlag && (
          <>
            {' '}
            <Separator className="my-4" />
            <UIVersionSwitcher />
          </>
        )}{' '}
      </div>
    </div>
  );
}

const TenantVersionSwitcher = () => {
  const { tenant } = useOutletContext<TenantContextType>();
  const queryClient = useQueryClient();
  const [showDowngradeModal, setShowDowngradeModal] = useState(false);

  const { handleApiError } = useApiError({});

  const { mutate: updateTenant, isPending } = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenant.metadata.id, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queries.user.listTenantMemberships.queryKey,
      });

      window.location.reload();
    },
    onError: handleApiError,
  });

  // Only show for V1 tenants
  if (tenant.version === TenantVersion.V0) {
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

function UIVersionSwitcher() {
  const { tenant } = useOutletContext<TenantContextType>();
  const navigate = useNavigate();
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);
  const { handleApiError } = useApiError({});

  const { mutateAsync: updateTenant, isPending } = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      return api.tenantUpdate(tenant.metadata.id, data);
    },
    onError: (error: AxiosError) => {
      setShowUpgradeModal(false);
      handleApiError(error);
    },
  });

  // Only show for V0 tenants
  if (!tenant.uiVersion || tenant?.uiVersion === TenantUIVersion.V1) {
    return null;
  }

  return (
    <div className="flex flex-col gap-y-2">
      <h2 className="text-xl font-semibold leading-tight text-foreground">
        UI Version
      </h2>
      <p className="text-sm text-muted-foreground">
        You can downgrade your UI to V0 if needed.
      </p>
      <Button
        onClick={() => setShowUpgradeModal(true)}
        disabled={isPending}
        variant="default"
        className="w-fit"
      >
        Upgrade to the V1 UI
      </Button>

      <Dialog open={showUpgradeModal} onOpenChange={setShowUpgradeModal}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Upgrade to the V1 UI</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            Please confirm your upgrade to the V1 UI version. Note that this
            will have no effect on any of your workflows, and is a UI-only
            change.
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowUpgradeModal(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button
              variant="default"
              onClick={async () => {
                const tenant = await updateTenant({
                  uiVersion: TenantUIVersion.V1,
                });

                if (tenant.data.uiVersion !== TenantUIVersion.V1) {
                  return;
                }

                setShowUpgradeModal(false);
                navigate(
                  {
                    pathname: '/next',
                    search: createSearchParams({
                      tenant: tenant.data.metadata.id,
                    }).toString(),
                  },
                  {
                    replace: false,
                  },
                );
              }}
              disabled={isPending}
            >
              Confirm Upgrade 🎉🎉
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
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
