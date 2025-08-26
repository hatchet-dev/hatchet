import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { TenantContextType } from '@/lib/outlet';
import { useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import api, {
  queries,
  Tenant,
  TenantVersion,
  UpdateTenantRequest,
  TenantMemberRole,
} from '@/lib/api';
import { Switch } from '@/components/ui/switch';
import { Label } from '@radix-ui/react-label';
import { Spinner } from '@/components/ui/loading';
import { capitalize } from '@/lib/utils';
import { UpdateTenantForm } from './components/update-tenant-form';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { AxiosError } from 'axios';
import { useTenantDetails } from '@/hooks/use-tenant';

export default function TenantSettings() {
  const { tenant } = useOutletContext<TenantContextType>();

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
        <Separator className="my-4" />
        <DeleteTenant tenant={tenant} />
      </div>
    </div>
  );
}

const TenantVersionSwitcher = () => {
  const { tenant } = useOutletContext<TenantContextType>();
  const queryClient = useQueryClient();
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);
  const [upgradeRestrictedError, setUpgradeRestrictedError] =
    useState<boolean>(false);
  const { handleApiError } = useApiError({});

  const { mutate: updateTenant, isPending } = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      setUpgradeRestrictedError(false);
      await api.tenantUpdate(tenant.metadata.id, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queries.user.listTenantMemberships.queryKey,
      });

      window.location.reload();
    },
    onError: (error: AxiosError) => {
      if (error.response?.status === 403) {
        setUpgradeRestrictedError(true);
      } else {
        setShowUpgradeModal(false);
        handleApiError(error);
      }
    },
  });

  // Only show for V0 tenants
  if (tenant.version === TenantVersion.V1) {
    return null;
  }

  return (
    <>
      <div className="flex flex-col gap-y-4">
        <h2 className="text-xl font-semibold leading-tight text-foreground">
          Tenant Version
        </h2>
        <p className="text-sm text-muted-foreground">
          Upgrade your tenant to V1 to access new features and improvements.
        </p>
        <Button
          onClick={() => setShowUpgradeModal(true)}
          disabled={isPending}
          className="w-fit"
        >
          {isPending ? <Spinner /> : null}
          Upgrade to V1
        </Button>
      </div>

      <Dialog open={showUpgradeModal} onOpenChange={setShowUpgradeModal}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Upgrade to V1</DialogTitle>
          </DialogHeader>
          {!upgradeRestrictedError && (
            <div className="space-y-4 py-4">
              <p className="text-sm">Upgrading your tenant to V1 will:</p>
              <ul className="list-disc list-inside text-sm space-y-2">
                <li>Enable new V1 features and improvements</li>
                <li>Redirect you to the V1 interface</li>
              </ul>
              <Alert variant="warn">
                <AlertTitle>Warning</AlertTitle>
                <AlertDescription>
                  This upgrade will not automatically migrate your existing
                  workflows or in-progress runs. To ensure zero downtime during
                  the upgrade, please follow our migration guide which includes
                  steps for parallel operation of V0 and V1 environments.
                </AlertDescription>
              </Alert>

              <p className="text-sm">
                Please read our{' '}
                <a
                  href="https://github.com/hatchet-dev/hatchet/discussions/1348"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-indigo-400 hover:underline"
                >
                  V1 preview announcement
                </a>{' '}
                before proceeding.
              </p>
            </div>
          )}
          {upgradeRestrictedError && (
            <Alert variant="warn">
              <AlertDescription>
                Tenant version upgrade has been restricted for this tenant.
                Please contact us to request upgrade referencing tenant id:{' '}
                {tenant.metadata.id}
              </AlertDescription>
            </Alert>
          )}
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowUpgradeModal(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button
              onClick={() => {
                updateTenant({
                  version: TenantVersion.V1,
                });
              }}
              disabled={isPending}
            >
              {isPending ? <Spinner /> : null}
              Confirm Upgrade
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};

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

const DeleteTenant: React.FC<{ tenant: Tenant }> = ({ tenant }) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { membership } = useTenantDetails();
  const { handleApiError } = useApiError({});

  const isOwner = membership === TenantMemberRole.OWNER;

  const deleteTenantMutation = useMutation({
    mutationKey: ['tenant:delete'],
    mutationFn: async () => {
      await api.tenantDelete(tenant.metadata.id);
    },
    onSuccess: () => {
      window.location.href = '/';
    },
    onError: handleApiError,
  });

  const handleDelete = () => {
    deleteTenantMutation.mutate();
  };

  return (
    <>
      <div className="flex flex-col gap-y-4">
        <h2 className="text-xl font-semibold leading-tight text-foreground">
          Delete Tenant
        </h2>
        <p className="text-sm text-muted-foreground">
          Permanently delete this tenant and all associated data. This action
          cannot be undone.
        </p>

        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <div>
                <Button
                  variant="destructive"
                  onClick={() => setShowDeleteModal(true)}
                  disabled={!isOwner}
                  className="w-fit"
                >
                  Delete Tenant
                </Button>
              </div>
            </TooltipTrigger>
            {!isOwner && (
              <TooltipContent>
                Only tenant owners can delete the tenant
              </TooltipContent>
            )}
          </Tooltip>
        </TooltipProvider>
      </div>

      <Dialog open={showDeleteModal} onOpenChange={setShowDeleteModal}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Delete Tenant</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <Alert variant="destructive">
              <AlertTitle>Warning</AlertTitle>
              <AlertDescription>
                This action will permanently delete the tenant "{tenant.name}"
                and all associated data including:
              </AlertDescription>
            </Alert>
            <ul className="list-disc list-inside text-sm space-y-1 text-muted-foreground">
              <li>All workflows and workflow runs</li>
              <li>All step runs and logs</li>
              <li>All events and triggers</li>
              <li>All worker data</li>
              <li>All team members and invitations</li>
            </ul>
            <p className="text-sm text-muted-foreground">
              This action cannot be undone.
            </p>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDeleteModal(false)}
              disabled={deleteTenantMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteTenantMutation.isPending}
            >
              {deleteTenantMutation.isPending ? <Spinner /> : null}
              Delete Tenant
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};
