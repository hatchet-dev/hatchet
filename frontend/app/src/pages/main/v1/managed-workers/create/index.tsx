import { Separator } from '@/components/v1/ui/separator';
import invariant from 'tiny-invariant';
import { useNavigate, useOutletContext } from 'react-router-dom';
import { TenantContextType } from '@/lib/outlet';
import { ServerStackIcon } from '@heroicons/react/24/outline';
import CreateWorkerForm from './components/create-worker-form';
import { useMutation } from '@tanstack/react-query';
import { CreateManagedWorkerRequest } from '@/lib/api/generated/cloud/data-contracts';
import { cloudApi } from '@/lib/api/api';
import { useState } from 'react';
import { useApiError } from '@/lib/hooks';
import { useTenant } from '@/lib/atoms';
import { managedCompute } from '@/lib/can/features/managed-compute';
import { RejectReason } from '@/lib/can/shared/permission.base';
import { BillingRequired } from '../components/billing-required';

export default function CreateWorker() {
  const navigate = useNavigate();
  const { tenant: contextTenant } = useOutletContext<TenantContextType>();
  const { tenant, billing, can } = useTenant();
  invariant(contextTenant);

  const [portalLoading, setPortalLoading] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });
  // Check if billing is required
  const [canCreateManagedWorker, rejectReason] = can(managedCompute.create());
  const isBillingRequired = rejectReason === RejectReason.BILLING_REQUIRED;

  const manageClicked = async () => {
    try {
      if (portalLoading) {
        return;
      }
      setPortalLoading(true);
      billing?.setPollBilling(true);
      const link = await cloudApi.billingPortalLinkGet(tenant!.metadata.id);
      window.open(link.data.url, '_blank');
    } catch (e) {
      handleApiError(e as any);
    } finally {
      setPortalLoading(false);
    }
  };

  const createManagedWorkerMutation = useMutation({
    mutationKey: ['managed-worker:create', contextTenant],
    mutationFn: async (data: CreateManagedWorkerRequest) => {
      const dataCopy = { ...data };

      if (dataCopy.isIac) {
        delete dataCopy.runtimeConfig;
      }

      const res = await cloudApi.managedWorkerCreate(
        contextTenant.metadata.id,
        dataCopy,
      );
      return res.data;
    },
    onSuccess: (data) => {
      navigate(`/v1/managed-workers/${data.metadata.id}`);
    },
    onError: handleApiError,
  });

  // Show billing required page if billing is required
  if (isBillingRequired) {
    return (
      <BillingRequired
        tenant={tenant}
        billing={billing}
        manageClicked={manageClicked}
        portalLoading={portalLoading}
      />
    );
  }

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center justify-between">
            <ServerStackIcon className="h-6 w-6 text-foreground mt-1" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              New Service
            </h2>
          </div>
        </div>
        <Separator className="my-4" />
        <CreateWorkerForm
          onSubmit={createManagedWorkerMutation.mutate}
          isLoading={createManagedWorkerMutation.isPending}
          fieldErrors={fieldErrors}
        />
      </div>
    </div>
  );
}
