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

export default function CreateWorker() {
  const navigate = useNavigate();

  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const createManagedWorkerMutation = useMutation({
    mutationKey: ['managed-worker:create', tenant],
    mutationFn: async (data: CreateManagedWorkerRequest) => {
      const dataCopy = { ...data };

      if (dataCopy.isIac) {
        delete dataCopy.runtimeConfig;
      }

      const res = await cloudApi.managedWorkerCreate(
        tenant.metadata.id,
        dataCopy,
      );
      return res.data;
    },
    onSuccess: (data) => {
      navigate(`/managed-workers/${data.metadata.id}`);
    },
    onError: handleApiError,
  });

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center justify-between">
            <ServerStackIcon className="h-6 w-6 text-foreground mt-1" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              New Managed Worker
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
