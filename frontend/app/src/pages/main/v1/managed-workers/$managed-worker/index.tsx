import { Separator } from '@/components/v1/ui/separator';
import { queries } from '@/lib/api';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';
import { relativeDate } from '@/lib/utils';
import { CpuChipIcon } from '@heroicons/react/24/outline';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { useState } from 'react';
import { Button } from '@/components/v1/ui/button';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { ManagedWorkerLogs } from './components/managed-worker-logs';
import { ManagedWorkerMetrics } from './components/managed-worker-metrics';
import { ManagedWorkerActivity } from './components/managed-worker-activity';
import { UpdateManagedWorkerRequest } from '@/lib/api/generated/cloud/data-contracts';
import { ManagedWorkerInstancesTable } from './components/managed-worker-instances-table';
import UpdateWorkerForm from './components/update-form';
import { cloudApi } from '@/lib/api/api';
import { useApiError } from '@/lib/hooks';
import GithubButton from './components/github-button';

export default function ExpandedWorkflow() {
  const navigate = useNavigate();
  const [deleteWorker, setDeleteWorker] = useState(false);

  const params = useParams();
  invariant(params['managed-worker']);

  const managedWorkerQuery = useQuery({
    ...queries.cloud.getManagedWorker(params['managed-worker']),
    refetchInterval: 5000,
  });

  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const updateManagedWorkerMutation = useMutation({
    mutationKey: ['managed-worker:update', params['managed-worker']],
    mutationFn: async (data: UpdateManagedWorkerRequest) => {
      invariant(managedWorker);

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
    mutationKey: ['managed-worker:delete', params['managed-worker']],
    mutationFn: async () => {
      invariant(managedWorker);
      const res = await cloudApi.managedWorkerDelete(managedWorker.metadata.id);
      return res.data;
    },
    onSuccess: () => {
      setDeleteWorker(false);
      navigate('/managed-workers');
    },
    onError: handleApiError,
  });

  if (managedWorkerQuery.isLoading || !managedWorkerQuery.data) {
    return <Loading />;
  }

  const managedWorker = managedWorkerQuery.data;

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center">
            <CpuChipIcon className="h-6 w-6 text-foreground mt-1" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              {managedWorker.name}
            </h2>
          </div>
        </div>
        <div className="flex flex-row justify-start items-center mt-4 gap-6">
          <div className="text-sm text-gray-700 dark:text-gray-300">
            Created {relativeDate(managedWorker.metadata.createdAt)}
          </div>
          <GithubButton
            buildConfig={managedWorker.buildConfig}
            prefix="Deploys from"
          />
        </div>
        <div className="flex flex-row justify-start items-center mt-4"></div>
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
            <h3 className="text-xl font-bold leading-tight text-foreground mt-4">
              Instances
            </h3>
            <Separator className="my-4" />
            <ManagedWorkerInstancesTable
              managedWorkerId={managedWorker.metadata.id}
            />
          </TabsContent>
          <TabsContent value="logs">
            <h3 className="text-xl font-bold leading-tight text-foreground mt-4">
              Logs
            </h3>
            <Separator className="my-4" />
            <ManagedWorkerLogs managedWorker={managedWorker} />
          </TabsContent>
          <TabsContent value="metrics">
            <ManagedWorkerMetrics managedWorker={managedWorker} />
          </TabsContent>
          <TabsContent value="configuration">
            <h3 className="text-xl font-bold leading-tight text-foreground mt-4">
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
            <h4 className="text-lg font-bold leading-tight text-foreground mt-8">
              Danger Zone
            </h4>
            <Separator className="my-4" />
            <Button
              variant="destructive"
              className="mt-2"
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
