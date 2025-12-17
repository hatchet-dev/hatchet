import { BillingRequired } from '../components/billing-required';
import CreateWorkerForm from './components/create-worker-form';
import { Separator } from '@/components/v1/ui/separator';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import { cloudApi } from '@/lib/api/api';
import { CreateManagedWorkerRequest } from '@/lib/api/generated/cloud/data-contracts';
import { managedCompute } from '@/lib/can/features/managed-compute';
import { RejectReason } from '@/lib/can/shared/permission.base';
import { useApiError } from '@/lib/hooks';
import { appRoutes } from '@/router';
import { ServerStackIcon } from '@heroicons/react/24/outline';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { useState } from 'react';

export default function CreateWorker() {
  const navigate = useNavigate();
  const { billing, can } = useTenantDetails();
  const { tenantId } = useCurrentTenantId();

  const [portalLoading, setPortalLoading] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  // Check if billing is required
  const [, rejectReason] = can(managedCompute.create());
  const isBillingRequired = rejectReason === RejectReason.BILLING_REQUIRED;

  const manageClicked = async () => {
    try {
      if (portalLoading) {
        return;
      }
      setPortalLoading(true);
      billing?.setPollBilling(true);
      const link = await cloudApi.billingPortalLinkGet(tenantId);
      window.open(link.data.url, '_blank');
    } catch (e) {
      handleApiError(e as any);
    } finally {
      setPortalLoading(false);
    }
  };

  const createManagedWorkerMutation = useMutation({
    mutationKey: ['managed-worker:create', tenantId],
    mutationFn: async (data: CreateManagedWorkerRequest) => {
      const dataCopy = { ...data };

      if (dataCopy.isIac) {
        delete dataCopy.runtimeConfig;
      }

      const res = await cloudApi.managedWorkerCreate(tenantId, dataCopy);
      return res.data;
    },
    onSuccess: (data) => {
      navigate({
        to: appRoutes.tenantManagedWorkerRoute.to,
        params: { tenant: tenantId, managedWorker: data.metadata.id },
      });
    },
    onError: handleApiError,
  });

  // Show billing required page if billing is required
  if (isBillingRequired) {
    return (
      <BillingRequired
        tenant={tenantId}
        billing={billing}
        manageClicked={manageClicked}
        portalLoading={portalLoading}
      />
    );
  }

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-row items-center justify-between">
          <div className="flex flex-row items-center justify-between gap-4">
            <ServerStackIcon className="mt-1 h-6 w-6 text-foreground" />
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
