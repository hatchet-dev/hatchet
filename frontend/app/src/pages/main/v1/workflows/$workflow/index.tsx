import { RunsTable } from '../../workflow-runs-v1/components/runs-table';
import { workflowKey } from '../../workflow-runs-v1/components/v1/task-runs-columns';
import { RunsProvider } from '../../workflow-runs-v1/hooks/runs-provider';
import { WorkflowTags } from '../components/workflow-tags';
import { PauseWorkflowDialog } from './components/pause-workflow-dialog';
import { TriggerWorkflowForm } from './components/trigger-workflow-form';
import WorkflowGeneralSettings from './components/workflow-general-settings';
import { WorkflowStatusSettings } from './components/workflow-status-settings';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { Loading } from '@/components/v1/ui/loading.tsx';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import useCanWrite from '@/hooks/use-can-write';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, { PauseWorkflowRequest, queries, Workflow } from '@/lib/api';
import { shouldRetryQueryError } from '@/lib/error-utils';
import { relativeDate } from '@/lib/utils';
import { ResourceNotFound } from '@/pages/error/components/resource-not-found';
import { appRoutes } from '@/router';
import { Square3Stack3DIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from '@tanstack/react-router';
import { isAxiosError } from 'axios';
import { useState } from 'react';

export default function ExpandedWorkflow() {
  // TODO list previous versions and make selectable
  const [selectedVersion] = useState<string | undefined>();
  const { tenantId } = useCurrentTenantId();
  const canWrite = useCanWrite();

  const [triggerWorkflow, setTriggerWorkflow] = useState(false);
  const [deleteWorkflow, setDeleteWorkflow] = useState(false);
  const [pauseWorkflow, setPauseWorkflow] = useState(false);
  const [unpauseWorkflow, setUnpauseWorkflow] = useState(false);
  const { refetchInterval } = useRefetchInterval();
  const queryClient = useQueryClient();

  const params = useParams({ from: appRoutes.tenantWorkflowRoute.to });

  const workflowQueryKey = queries.workflows.get(params.workflow).queryKey;

  const workflowQuery = useQuery({
    ...queries.workflows.get(params.workflow),
    refetchInterval,
    retry: (_failureCount, error) => shouldRetryQueryError(error),
  });

  const workflowVersionQuery = useQuery({
    ...queries.workflows.getVersion(params.workflow, selectedVersion),
    refetchInterval,
  });

  const navigate = useNavigate();

  const deleteWorkflowMutation = useMutation({
    mutationKey: ['workflow:delete', workflowQuery?.data?.metadata.id],
    mutationFn: async () => {
      if (!workflowQuery?.data) {
        return;
      }

      const res = await api.workflowDelete(workflowQuery?.data.metadata.id);

      return res.data;
    },
    onSuccess: () => {
      navigate({
        to: appRoutes.tenantWorkflowsRoute.to,
        params: { tenant: tenantId },
      });
    },
  });

  const togglePauseMutation = useMutation({
    mutationKey: ['workflow:pause:toggle', params.workflow],
    mutationFn: async (opts: PauseWorkflowRequest) => {
      const res = await api.workflowUpdate(params.workflow, {
        pause: opts,
      });

      return res.data;
    },
    onMutate: async (opts) => {
      await queryClient.cancelQueries({ queryKey: workflowQueryKey });

      const previousWorkflow =
        queryClient.getQueryData<Workflow>(workflowQueryKey);

      queryClient.setQueryData<Workflow>(workflowQueryKey, (old) =>
        old ? { ...old, action: opts.action } : old,
      );

      return { previousWorkflow };
    },
    onError: (_err, _opts, context) => {
      if (context?.previousWorkflow) {
        queryClient.setQueryData(workflowQueryKey, context.previousWorkflow);
      }
    },
    onSettled: async () =>
      queryClient.invalidateQueries({ queryKey: workflowQueryKey }),
  });

  const workflow = workflowQuery.data;

  if (workflowQuery.isLoading) {
    return <Loading />;
  }

  if (workflowQuery.isError) {
    if (
      isAxiosError(workflowQuery.error) &&
      workflowQuery.error.response?.status === 404
    ) {
      return (
        <ResourceNotFound
          resource="Workflow"
          primaryAction={{
            label: 'Back to Workflows',
            navigate: {
              to: appRoutes.tenantWorkflowsRoute.to,
              params: { tenant: tenantId },
            },
          }}
        />
      );
    }

    throw workflowQuery.error;
  }

  if (!workflow) {
    return <Loading />;
  }

  const currVersion = workflow.versions && workflow.versions[0].version;

  return (
    <div className="flex h-full w-full flex-grow flex-col gap-y-4 overflow-hidden">
      <div className="flex-shrink-0 p-4">
        <div className="flex flex-row items-center justify-between">
          <div className="flex flex-row items-center gap-4">
            <Square3Stack3DIcon className="mt-1 h-6 w-6 text-foreground" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              {workflow.name}
            </h2>
            {currVersion && (
              <Badge className="mt-1 text-sm" variant="outline">
                {currVersion}
              </Badge>
            )}
            <Badge
              variant={workflow.isPaused ? 'inProgress' : 'successful'}
              className={
                togglePauseMutation.isPending ? 'px-2 opacity-50' : 'px-2'
              }
            >
              {workflow.isPaused ? 'Paused' : 'Active'}
            </Badge>
          </div>
          <WorkflowTags tags={workflow.tags || []} />
          {canWrite && (
            <div className="flex flex-row gap-2">
              <Button
                className="text-sm"
                onClick={() => setTriggerWorkflow(true)}
              >
                Trigger Workflow
              </Button>
            </div>
          )}
          <TriggerWorkflowForm
            show={triggerWorkflow}
            defaultWorkflow={workflow}
            onClose={() => setTriggerWorkflow(false)}
          />
          <PauseWorkflowDialog
            isOpen={pauseWorkflow}
            isLoading={togglePauseMutation.isPending}
            onCancel={() => setPauseWorkflow(false)}
            onSubmit={({
              cronRunQueueBehavior,
              scheduledRunQueueBehavior,
              queueTtl,
            }) => {
              togglePauseMutation.mutate(
                {
                  action: 'pause',
                  pausedWorkflowCronRunQueueBehavior: cronRunQueueBehavior,
                  pausedWorkflowScheduledRunQueueBehavior:
                    scheduledRunQueueBehavior,
                  pausedWorkflowQueueTTL: queueTtl,
                },
                { onSuccess: () => setPauseWorkflow(false) },
              );
            }}
          />
          <ConfirmDialog
            title="Unpause workflow"
            description={`Are you sure you want to unpause the workflow ${workflow.name}? New runs will start executing immediately.`}
            submitLabel="Unpause"
            onSubmit={() => {
              togglePauseMutation.mutate(
                { action: 'unpause' },
                { onSuccess: () => setUnpauseWorkflow(false) },
              );
            }}
            onCancel={() => setUnpauseWorkflow(false)}
            isLoading={togglePauseMutation.isPending}
            isOpen={unpauseWorkflow}
          />
        </div>
        <div className="mt-4 flex flex-row items-center justify-start">
          <div className="text-sm text-gray-700 dark:text-gray-300">
            Updated{' '}
            {relativeDate(
              workflow.versions && workflow.versions[0].metadata.updatedAt,
            )}
          </div>
        </div>
        {workflow.description && (
          <div className="mt-4 text-sm text-gray-700 dark:text-gray-300">
            {workflow.description}
          </div>
        )}
      </div>
      <div className="min-h-0 flex-1 px-4 sm:px-6 lg:px-8">
        <Tabs defaultValue="runs" className="flex h-full flex-col">
          <TabsList layout="underlined" className="mb-4">
            <TabsTrigger variant="underlined" value="runs">
              Runs
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="settings">
              Settings
            </TabsTrigger>
          </TabsList>
          <TabsContent value="runs" className="min-h-0 flex-1">
            <RecentRunsList />
          </TabsContent>
          <TabsContent
            value="settings"
            className="min-h-0 flex-1 overflow-y-auto pt-4 pb-8"
          >
            <WorkflowStatusSettings
              isPaused={!!workflow.isPaused}
              disabled={!canWrite}
              isPending={togglePauseMutation.isPending}
              onRequestPause={() => setPauseWorkflow(true)}
              onRequestUnpause={() => setUnpauseWorkflow(true)}
            />

            {workflowVersionQuery.isLoading || !workflowVersionQuery.data ? (
              <Loading />
            ) : (
              <WorkflowGeneralSettings workflow={workflowVersionQuery.data} />
            )}

            {canWrite && (
              <div className="mt-8">
                <div className="space-y-3">
                  <h3 className="border-b border-gray-200 pb-2 text-base font-semibold text-gray-900 dark:border-gray-700 dark:text-gray-100">
                    Danger Zone
                  </h3>
                  <div className="pl-1">
                    <div className="max-w-xl rounded-md border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-800/50">
                      <div className="space-y-3">
                        <div>
                          <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">
                            Delete Workflow
                          </h4>
                          <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                            Permanently delete this workflow and all its data.
                            This action cannot be undone.
                          </p>
                        </div>
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={() => {
                            setDeleteWorkflow(true);
                          }}
                        >
                          Delete Workflow
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}

            <ConfirmDialog
              title={`Delete workflow`}
              description={`Are you sure you want to delete the workflow ${workflow.name}? This action cannot be undone, and will immediately prevent any services running with this workflow from executing steps.`}
              submitLabel={'Delete'}
              onSubmit={function (): void {
                deleteWorkflowMutation.mutate();
              }}
              onCancel={function (): void {
                setDeleteWorkflow(false);
              }}
              isLoading={deleteWorkflowMutation.isPending}
              isOpen={deleteWorkflow}
            />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}

function RecentRunsList() {
  const params = useParams({ from: appRoutes.tenantWorkflowRoute.to });

  return (
    <RunsProvider
      tableKey={`workflow-${params.workflow}`}
      initColumnVisibility={{ Workflow: false }}
      filterVisibility={{ Workflow: false }}
      display={{
        hideMetrics: true,
        hiddenFilters: [workflowKey],
      }}
      runFilters={{
        workflowId: params.workflow,
      }}
    >
      <RunsTable />
    </RunsProvider>
  );
}
