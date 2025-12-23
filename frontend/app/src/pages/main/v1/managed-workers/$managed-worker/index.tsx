import GithubButton from './components/github-button';
import { ManagedWorkerActivity } from './components/managed-worker-activity';
import { ManagedWorkerInstancesTable } from './components/managed-worker-instances-table';
import { ManagedWorkerLogs } from './components/managed-worker-logs';
import { ManagedWorkerMetrics } from './components/managed-worker-metrics';
import UpdateWorkerForm from './components/update-form';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { Button } from '@/components/v1/ui/button';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { Separator } from '@/components/v1/ui/separator';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { UpdateManagedWorkerRequest } from '@/lib/api/generated/cloud/data-contracts';
import { shouldRetryQueryError } from '@/lib/error-utils';
import { useApiError } from '@/lib/hooks';
import { relativeDate } from '@/lib/utils';
import { ResourceNotFound } from '@/pages/error/components/resource-not-found';
import { appRoutes } from '@/router';
import { CpuChipIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from '@tanstack/react-router';
import { isAxiosError } from 'axios';
import { useState } from 'react';

export default function ExpandedWorkflow() {
  const navigate = useNavigate();
  const [deleteWorker, setDeleteWorker] = useState(false);
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();

  const params = useParams({ from: appRoutes.tenantManagedWorkerRoute.to });

  const managedWorkerQuery = useQuery({
    ...queries.cloud.getManagedWorker(params.managedWorker),
    refetchInterval,
    retry: (_failureCount, error) => shouldRetryQueryError(error),
  });

  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const updateManagedWorkerMutation = useMutation({
    mutationKey: ['managed-worker:update', params.managedWorker],
    mutationFn: async (data: UpdateManagedWorkerRequest) => {
      const dataCopy = { ...data };

      if (dataCopy.isIac) {
        delete dataCopy.runtimeConfig;
      }

      const res = await cloudApi.managedWorkerUpdate(
        managedWorker.metadata.id,
        dataCopy,
      );
      return res.data;
    },
    onSuccess: () => {
      managedWorkerQuery.refetch();
    },
    onError: handleApiError,
  });

  const deleteManagedWorkerMutation = useMutation({
    mutationKey: ['managed-worker:delete', params.managedWorker],
    mutationFn: async () => {
      const res = await cloudApi.managedWorkerDelete(managedWorker.metadata.id);
      return res.data;
    },
    onSuccess: () => {
      setDeleteWorker(false);
      navigate({
        to: appRoutes.tenantManagedWorkersRoute.to,
        params: { tenant: tenantId },
      });
    },
    onError: handleApiError,
  });

  if (managedWorkerQuery.isLoading) {
    return <Loading />;
  }

  if (managedWorkerQuery.isError) {
    if (
      isAxiosError(managedWorkerQuery.error) &&
      managedWorkerQuery.error.response?.status === 404
    ) {
      return (
        <ResourceNotFound
          resource="Managed worker"
          primaryAction={{
            label: 'Back to Managed Workers',
            navigate: {
              to: appRoutes.tenantManagedWorkersRoute.to,
              params: { tenant: tenantId },
            },
          }}
        />
      );
    }

    throw managedWorkerQuery.error;
  }

  if (!managedWorkerQuery.data) {
    return <Loading />;
  }

  const managedWorker = managedWorkerQuery.data;

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-row items-center justify-between">
          <div className="flex flex-row items-center gap-4">
            <CpuChipIcon className="mt-1 h-6 w-6 text-foreground" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              {managedWorker.name}
            </h2>
          </div>
        </div>
        <div className="mt-4 flex flex-row items-center justify-start gap-6">
          <div className="text-sm text-gray-700 dark:text-gray-300">
            Created {relativeDate(managedWorker.metadata.createdAt)}
          </div>
          {managedWorker.buildConfig && (
            <GithubButton
              buildConfig={managedWorker.buildConfig}
              prefix="Deploys from"
            />
          )}
        </div>
        <div className="mt-4 flex flex-row items-center justify-start"></div>
        <Tabs defaultValue="activity">
          <TabsList layout="underlined">
            <TabsTrigger variant="underlined" value="activity">
              Activity
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="instances">
              Instances
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="logs">
              Logs
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="metrics">
              Metrics
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="configuration">
              Configuration
            </TabsTrigger>
          </TabsList>
          <TabsContent value="activity">
            <ManagedWorkerActivity managedWorker={managedWorker} />
          </TabsContent>
          <TabsContent value="instances">
            <h3 className="mt-4 text-xl font-bold leading-tight text-foreground">
              Instances
            </h3>
            <Separator className="my-4" />
            <ManagedWorkerInstancesTable
              managedWorkerId={managedWorker.metadata.id}
            />
          </TabsContent>
          <TabsContent value="logs">
            <h3 className="mt-4 text-xl font-bold leading-tight text-foreground">
              Logs
            </h3>
            <Separator className="my-4" />
            <ManagedWorkerLogs managedWorker={managedWorker} />
          </TabsContent>
          <TabsContent value="metrics">
            <ManagedWorkerMetrics managedWorker={managedWorker} />
          </TabsContent>
          <TabsContent value="configuration">
            <h3 className="mt-4 text-xl font-bold leading-tight text-foreground">
              Configuration
            </h3>
            <Separator className="my-4" />
            <UpdateWorkerForm
              managedWorker={managedWorker}
              onSubmit={updateManagedWorkerMutation.mutate}
              isLoading={updateManagedWorkerMutation.isPending}
              fieldErrors={fieldErrors}
            />
            <Separator className="my-4" />
            <h4 className="mt-8 text-lg font-bold leading-tight text-foreground">
              Danger Zone
            </h4>
            <Separator className="my-4" />
            <Button
              variant="destructive"
              onClick={() => {
                setDeleteWorker(true);
              }}
            >
              Delete Managed Worker
            </Button>

            <ConfirmDialog
              title={`Delete managed worker`}
              description={`Are you sure you want to delete the managed worker ${managedWorker.name}? This action cannot be undone, and will immediately tear these workers down.`}
              submitLabel={'Delete'}
              onSubmit={deleteManagedWorkerMutation.mutate}
              onCancel={function (): void {
                setDeleteWorker(false);
              }}
              isLoading={deleteManagedWorkerMutation.isPending}
              isOpen={deleteWorker}
            />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
