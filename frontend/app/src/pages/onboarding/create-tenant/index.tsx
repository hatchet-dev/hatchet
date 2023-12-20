import api, { CreateTenantRequest } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { TenantCreateForm } from './components/tenant-create-form';

export default function CreateTenant() {
  const navigate = useNavigate();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const createMutation = useMutation({
    mutationKey: ['user:update:login'],
    mutationFn: async (data: CreateTenantRequest) => {
      await api.tenantCreate(data);
    },
    onSuccess: () => {
      navigate('/');
    },
    onError: handleApiError,
  });

  return (
    <div className="flex flex-row flex-1 w-full h-full">
      <div className="container relative hidden flex-col items-center justify-center md:grid lg:max-w-none lg:grid-cols-2 lg:px-0">
        <div className="lg:p-8 mx-auto w-screen">
          <div className="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-[350px]">
            <div className="flex flex-col space-y-2 text-center">
              <h1 className="text-2xl font-semibold tracking-tight">
                Create a new tenant
              </h1>
            </div>
            <TenantCreateForm
              isLoading={createMutation.isPending}
              onSubmit={createMutation.mutate}
              fieldErrors={fieldErrors}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
