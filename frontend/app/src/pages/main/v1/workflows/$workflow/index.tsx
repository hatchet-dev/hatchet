import { RunsTable } from '../../workflow-runs-v1/components/runs-table';
import { workflowKey } from '../../workflow-runs-v1/components/v1/task-runs-columns';
import { RunsProvider } from '../../workflow-runs-v1/hooks/runs-provider';
import { WorkflowTags } from '../components/workflow-tags';
import { TriggerWorkflowForm } from './components/trigger-workflow-form';
import WorkflowGeneralSettings from './components/workflow-general-settings';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { Switch } from '@/components/v1/ui/switch';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, { queries } from '@/lib/api';
import { shouldRetryQueryError } from '@/lib/error-utils';
import { useApiError } from '@/lib/hooks';
import { relativeDate } from '@/lib/utils';
import { ResourceNotFound } from '@/pages/error/components/resource-not-found';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { Square3Stack3DIcon } from '@heroicons/react/24/outline';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from '@tanstack/react-router';
import { isAxiosError } from 'axios';
import { useState } from 'react';

export default function ExpandedWorkflow() {
  // TODO list previous versions and make selectable
  const [selectedVersion] = useState<string | undefined>();
  const { tenantId } = useCurrentTenantId();

  const [triggerWorkflow, setTriggerWorkflow] = useState(false);
  const [deleteWorkflow, setDeleteWorkflow] = useState(false);
  const [confirmPause, setConfirmPause] = useState(false);
  const { refetchInterval } = useRefetchInterval();

  const params = useParams({ from: appRoutes.tenantWorkflowRoute.to });

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

  const { handleApiError } = useApiError({});

  const updateWorkflow = useMutation({
    mutationKey: ['workflow:update', workflowQuery?.data?.metadata.id],
    mutationFn: async (data: {
      isPaused?: boolean;
      queueCronOnPause?: boolean;
      queueScheduledOnPause?: boolean;
    }) => {
      if (!workflowQuery?.data) {
        return;
      }
      return (await api.workflowUpdate(workflowQuery.data.metadata.id, data))
        .data;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: queries.workflows.get(params.workflow).queryKey,
      });
    },
    onError: handleApiError,
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
            {workflow.isPaused ? (
              <Badge variant="inProgress" className="px-2">
                Paused
              </Badge>
            ) : (
              <Badge variant="successful" className="px-2">
                Active
              </Badge>
            )}
          </div>
          <WorkflowTags tags={workflow.tags || []} />
          <div className="flex flex-row gap-2">
            <Button
              className="text-sm"
              onClick={() => setTriggerWorkflow(true)}
            >
              Trigger Workflow
            </Button>
            <Button
              className="text-sm"
              variant="outline"
              onClick={() => {
                if (workflow.isPaused) {
                  updateWorkflow.mutate({
                    isPaused: false,
                  });
                } else {
                  setConfirmPause(true);
                }
              }}
            >
              {workflow.isPaused ? 'Resume Workflow' : 'Pause Workflow'}
            </Button>
          </div>
          <TriggerWorkflowForm
            show={triggerWorkflow}
            defaultWorkflow={workflow}
            onClose={() => setTriggerWorkflow(false)}
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
        {workflow.isPaused && (
          <Alert variant="warn" className="mt-4">
            <ExclamationTriangleIcon className="h-4 w-4" />
            <AlertTitle>Workflow Paused</AlertTitle>
            <AlertDescription>
              <p>
                This workflow is paused. New runs are being queued and will be
                processed when the workflow is resumed. Cron and Scheduled tasks
                behave based on workflow settings.
              </p>
              <ul className="mt-2 list-disc pl-5 text-sm">
                <li>
                  Cron triggers:{' '}
                  {workflow.queueCronOnPause ? (
                    <strong>queued</strong>
                  ) : (
                    <strong>disabled</strong>
                  )}
                  {workflow.queueCronOnPause ? ' (will replay on resume)' : ''}
                </li>
                <li>
                  Scheduled triggers:{' '}
                  {workflow.queueScheduledOnPause ? (
                    <strong>queued</strong>
                  ) : (
                    <strong>disabled</strong>
                  )}
                  {workflow.queueScheduledOnPause
                    ? ' (will replay on resume)'
                    : ''}
                </li>
              </ul>
            </AlertDescription>
          </Alert>
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
            {workflowVersionQuery.isLoading || !workflowVersionQuery.data ? (
              <Loading />
            ) : (
              <WorkflowGeneralSettings workflow={workflowVersionQuery.data} />
            )}
            <div className="mt-8">
              <div className="space-y-3">
                <h3 className="border-b border-gray-200 pb-2 text-base font-semibold text-gray-900 dark:border-gray-700 dark:text-gray-100">
                  Pause Behavior
                </h3>
                <div className="pl-1 space-y-4">
                  <label className="flex items-center justify-between max-w-xl rounded-md border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-800/50">
                    <div>
                      <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">
                        Queue cron runs while paused
                      </h4>
                      <p className="mt-1 mr-2 text-sm text-gray-600 dark:text-gray-400">
                        Cron task triggers that fire while the workflow is
                        paused will be{' '}
                        {workflow.queueCronOnPause
                          ? 'held and replayed when the workflow resumes'
                          : 'dropped'}
                        .
                      </p>
                    </div>
                    <Switch
                      checked={workflow.queueCronOnPause ?? false}
                      onCheckedChange={(checked) =>
                        updateWorkflow.mutate({ queueCronOnPause: checked })
                      }
                    />
                  </label>
                  <label className="flex items-center justify-between max-w-xl rounded-md border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-800/50">
                    <div>
                      <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">
                        Queue scheduled runs while paused
                      </h4>
                      <p className="mt-1 mr-2 text-sm text-gray-600 dark:text-gray-400">
                        Scheduled task triggers that fire while the workflow is
                        paused will be{' '}
                        {workflow.queueScheduledOnPause
                          ? 'held and replayed when the workflow resumes'
                          : 'dropped'}
                        .
                      </p>
                    </div>
                    <Switch
                      checked={workflow.queueScheduledOnPause ?? false}
                      onCheckedChange={(checked) =>
                        updateWorkflow.mutate({
                          queueScheduledOnPause: checked,
                        })
                      }
                    />
                  </label>
                </div>
              </div>
            </div>

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
      <ConfirmDialog
        title="Pause workflow"
        description={
          <span>
            Are you sure you want to pause <strong>{workflow.name}</strong>? All
            currently running tasks will be cancelled, and any queued tasks will
            be held until the workflow is resumed.
          </span>
        }
        submitLabel="Pause Workflow"
        onSubmit={() => {
          updateWorkflow.mutate(
            { isPaused: true },
            { onSuccess: () => setConfirmPause(false) },
          );
        }}
        onCancel={() => {
          setConfirmPause(false);
        }}
        isLoading={updateWorkflow.isPending}
        isOpen={confirmPause}
      />
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
