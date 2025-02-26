import { Separator } from '@/components/v1/ui/separator';
import api, { queries, WorkflowUpdateRequest } from '@/lib/api';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate, useOutletContext, useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';
import { WorkflowTags } from '../components/workflow-tags';
import { Badge } from '@/components/v1/ui/badge';
import { relativeDate } from '@/lib/utils';
import { Square3Stack3DIcon } from '@heroicons/react/24/outline';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { TenantContextType } from '@/lib/outlet';
import { TriggerWorkflowForm } from './components/trigger-workflow-form';
import { useState } from 'react';
import { Button } from '@/components/v1/ui/button';
import { useApiError, useApiMetaIntegrations } from '@/lib/hooks';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import WorkflowGeneralSettings from './components/workflow-general-settings';
import { WorkflowRunsTable } from '../../workflow-runs/components/workflow-runs-table';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { useTenant } from '@/lib/atoms';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';

export default function ExpandedWorkflow() {
  const { tenant } = useTenant();

  // TODO list previous versions and make selectable
  const [selectedVersion] = useState<string | undefined>();
  const { handleApiError } = useApiError({});

  invariant(tenant);

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
      navigate('/workflows');
    },
  });

  const integrations = useApiMetaIntegrations();

  const workflow = workflowQuery.data;

  if (workflowQuery.isLoading || !workflow) {
    return <Loading />;
  }

  const hasGithubIntegration = integrations?.find((i) => i.name === 'github');
  const currVersion = workflow.versions && workflow.versions[0].version;

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
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
        <div className="flex flex-row justify-start items-center mt-4"></div>
        <Tabs defaultValue="runs">
          <TabsList layout="underlined">
            <TabsTrigger variant="underlined" value="runs">
              Runs
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="settings">
              Settings
            </TabsTrigger>
          </TabsList>
          <TabsContent value="runs">
            <h3 className="text-xl font-bold leading-tight text-foreground mt-4">
              Recent Runs
            </h3>
            <Separator className="my-4" />
            <RecentRunsList />
          </TabsContent>
          <TabsContent value="settings">
            <h3 className="text-xl font-bold leading-tight text-foreground mt-4">
              Settings
            </h3>
            <Separator className="my-4" />
            {workflowVersionQuery.isLoading || !workflowVersionQuery.data ? (
              <Loading />
            ) : (
              <WorkflowGeneralSettings workflow={workflowVersionQuery.data} />
            )}
            <Separator className="my-4" />
            {hasGithubIntegration && (
              <div className="hidden">
                <h3 className="hidden text-xl font-bold leading-tight text-foreground mt-8">
                  Deployment Settings
                </h3>
                <Separator className="hidden my-4" />
              </div>
            )}
            <h4 className="text-lg font-bold leading-tight text-foreground mt-8">
              Danger Zone
            </h4>
            <Separator className="my-4" />
            <Button
              variant="destructive"
              className="mt-2"
              onClick={() => {
                setDeleteWorkflow(true);
              }}
            >
              Delete Workflow
            </Button>

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
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const params = useParams();
  invariant(params.workflow);

  return (
    <WorkflowRunsTable
      workflowId={params.workflow}
      initColumnVisibility={{ Workflow: false }}
      filterVisibility={{ Workflow: false }}
    />
  );
}
