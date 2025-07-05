import api, { queries, WorkflowUpdateRequest } from '@/lib/api';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';
import { WorkflowTags } from '../components/workflow-tags';
import { Badge } from '@/components/v1/ui/badge';
import { relativeDate } from '@/lib/utils';
import { Square3Stack3DIcon } from '@heroicons/react/24/outline';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { TriggerWorkflowForm } from './components/trigger-workflow-form';
import { useState } from 'react';
import { Button } from '@/components/v1/ui/button';
import { useApiError } from '@/lib/hooks';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import WorkflowGeneralSettings from './components/workflow-general-settings';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { TaskRunsTable } from '../../workflow-runs-v1/components/task-runs-table';
import { useCurrentTenantId } from '@/hooks/use-tenant';

export default function ExpandedWorkflow() {
  // TODO list previous versions and make selectable
  const [selectedVersion] = useState<string | undefined>();
  const { handleApiError } = useApiError({});
  const { tenantId } = useCurrentTenantId();

  const [triggerWorkflow, setTriggerWorkflow] = useState(false);
  const [deleteWorkflow, setDeleteWorkflow] = useState(false);

  const params = useParams();
  invariant(params.workflow);

  const workflowQuery = useQuery({
    ...queries.workflows.get(params.workflow),
    refetchInterval: 1000,
  });

  const workflowVersionQuery = useQuery({
    ...queries.workflows.getVersion(params.workflow, selectedVersion),
    refetchInterval: 1000,
  });

  const navigate = useNavigate();

  const updateWorkflowMutation = useMutation({
    mutationKey: ['workflow:update', workflowQuery?.data?.metadata.id],
    mutationFn: async (data: WorkflowUpdateRequest) => {
      invariant(workflowQuery.data);
      const res = await api.workflowUpdate(workflowQuery?.data?.metadata.id, {
        ...data,
      });

      return res.data;
    },
    onError: handleApiError,
    onSuccess: () => {
      workflowQuery.refetch();
    },
  });

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
      navigate(`/tenants/${tenantId}/tasks`);
    },
  });

  const workflow = workflowQuery.data;

  if (workflowQuery.isLoading || !workflow) {
    return <Loading />;
  }

  const currVersion = workflow.versions && workflow.versions[0].version;

  return (
    <div className="flex-grow h-full w-full flex flex-col overflow-hidden gap-y-4">
      <div className="flex-shrink-0 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center">
            <Square3Stack3DIcon className="h-6 w-6 text-foreground mt-1" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              {workflow.name}
            </h2>
            {currVersion && (
              <Badge className="text-sm mt-1" variant="outline">
                {currVersion}
              </Badge>
            )}
            <DropdownMenu>
              <DropdownMenuTrigger>
                {workflow.isPaused ? (
                  <Badge
                    variant="inProgress"
                    className="px-2"
                    onClick={() => {
                      updateWorkflowMutation.mutate({ isPaused: false });
                    }}
                  >
                    Paused
                  </Badge>
                ) : (
                  <Badge
                    variant="successful"
                    className="px-2"
                    onClick={() => {
                      updateWorkflowMutation.mutate({ isPaused: true });
                    }}
                  >
                    Active
                  </Badge>
                )}
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                <DropdownMenuItem>
                  {workflow.isPaused ? (
                    <div
                      onClick={() => {
                        updateWorkflowMutation.mutate({
                          isPaused: false,
                        });
                      }}
                    >
                      Unpause runs
                    </div>
                  ) : (
                    <div
                      onClick={() => {
                        updateWorkflowMutation.mutate({
                          isPaused: true,
                        });
                      }}
                    >
                      Pause runs
                    </div>
                  )}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          <WorkflowTags tags={workflow.tags || []} />
          <div className="flex flex-row gap-2">
            <Button
              className="text-sm"
              onClick={() => setTriggerWorkflow(true)}
            >
              Trigger Workflow
            </Button>
          </div>
          <TriggerWorkflowForm
            show={triggerWorkflow}
            defaultWorkflow={workflow}
            onClose={() => setTriggerWorkflow(false)}
          />
        </div>
        <div className="flex flex-row justify-start items-center mt-4">
          <div className="text-sm text-gray-700 dark:text-gray-300">
            Updated{' '}
            {relativeDate(
              workflow.versions && workflow.versions[0].metadata.updatedAt,
            )}
          </div>
        </div>
        {workflow.description && (
          <div className="text-sm text-gray-700 dark:text-gray-300 mt-4">
            {workflow.description}
          </div>
        )}
      </div>
      <div className="flex-1 min-h-0 px-4 sm:px-6 lg:px-8">
        <Tabs defaultValue="runs" className="flex flex-col h-full">
          <TabsList layout="underlined" className="mb-4">
            <TabsTrigger variant="underlined" value="runs">
              Runs
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="settings">
              Settings
            </TabsTrigger>
          </TabsList>
          <TabsContent value="runs" className="flex-1 min-h-0">
            <RecentRunsList />
          </TabsContent>
          <TabsContent
            value="settings"
            className="flex-1 min-h-0 overflow-y-auto pt-4"
          >
            {workflowVersionQuery.isLoading || !workflowVersionQuery.data ? (
              <Loading />
            ) : (
              <WorkflowGeneralSettings workflow={workflowVersionQuery.data} />
            )}

            <div className="mt-8">
              <div className="space-y-3">
                <h3 className="text-base font-semibold text-gray-900 dark:text-gray-100 border-b border-gray-200 dark:border-gray-700 pb-2">
                  Danger Zone
                </h3>
                <div className="pl-1">
                  <div className="border border-gray-200 dark:border-gray-700 rounded-md p-4 bg-gray-50 dark:bg-gray-800/50 max-w-xl">
                    <div className="space-y-3">
                      <div>
                        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">
                          Delete Workflow
                        </h4>
                        <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
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
    </div>
  );
}

function RecentRunsList() {
  const params = useParams();
  invariant(params.workflow);

  return (
    <TaskRunsTable
      workflowId={params.workflow}
      initColumnVisibility={{ Workflow: false }}
      filterVisibility={{ Workflow: false }}
    />
  );
}
