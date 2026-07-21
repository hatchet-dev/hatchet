import { RunsTable } from '../../workflow-runs-v1/components/runs-table';
import { workflowKey } from '../../workflow-runs-v1/components/v1/task-runs-columns';
import { RunsProvider } from '../../workflow-runs-v1/hooks/runs-provider';
import { WorkflowTags } from '../components/workflow-tags';
import { TriggerWorkflowForm } from './components/trigger-workflow-form';
import WorkflowGeneralSettings from './components/workflow-general-settings';
import WorkflowShapeTab from './components/workflow-shape-tab';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Loading } from '@/components/v1/ui/loading.tsx';
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
import { relativeDate } from '@/lib/utils';
import { ResourceNotFound } from '@/pages/error/components/resource-not-found';
import { appRoutes } from '@/router';
import { Square3Stack3DIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate, useParams, useSearch } from '@tanstack/react-router';
import { isAxiosError } from 'axios';
import { CopyIcon, MoreHorizontal } from 'lucide-react';
import { useState } from 'react';

export default function ExpandedWorkflow() {
  // TODO list previous versions and make selectable
  const [selectedVersion] = useState<string | undefined>();
  const { tenantId } = useCurrentTenantId();

  const [triggerWorkflow, setTriggerWorkflow] = useState(false);
  const [deleteWorkflow, setDeleteWorkflow] = useState(false);
  const { refetchInterval } = useRefetchInterval();

  const params = useParams({ from: appRoutes.tenantWorkflowRoute.to });
  const { tab } = useSearch({ from: appRoutes.tenantWorkflowRoute.to });

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

  const activeTab = tab ?? 'runs';
  const setTab = (value: 'runs' | 'shape' | 'settings') =>
    navigate({
      to: appRoutes.tenantWorkflowRoute.to,
      params: { tenant: tenantId, workflow: params.workflow },
      search: { tab: value },
      replace: true,
    });

  return (
    <Tabs
      value={activeTab}
      onValueChange={(value) => setTab(value as 'runs' | 'shape' | 'settings')}
      className="flex h-full w-full flex-grow flex-col gap-y-4 overflow-hidden"
    >
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
            <Badge variant="successful" className="px-2">
              Active
            </Badge>
          </div>
          <WorkflowTags tags={workflow.tags || []} />
          <div className="flex flex-row items-center gap-2">
            <Button
              className="text-sm"
              onClick={() => setTriggerWorkflow(true)}
            >
              Trigger Workflow
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="icon" aria-label="More actions">
                  <MoreHorizontal className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="z-40">
                <DropdownMenuItem
                  className="cursor-pointer"
                  disabled={!workflowVersionQuery.data}
                  onSelect={() =>
                    navigator.clipboard.writeText(
                      JSON.stringify(
                        workflowVersionQuery.data?.workflowConfig ?? {},
                      ),
                    )
                  }
                >
                  <CopyIcon className="mr-2 size-3" />
                  Copy Config
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  className="cursor-pointer text-destructive focus:text-destructive"
                  onSelect={() => setDeleteWorkflow(true)}
                >
                  Delete Workflow
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
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
        <TabsList layout="underlined" className="mt-4">
          <TabsTrigger variant="underlined" value="runs">
            Runs
          </TabsTrigger>
          <TabsTrigger variant="underlined" value="shape">
            Shape
          </TabsTrigger>
          <TabsTrigger variant="underlined" value="settings">
            Settings
          </TabsTrigger>
        </TabsList>
      </div>
      <div className="min-h-0 flex-1 px-4 sm:px-6 lg:px-8">
        <TabsContent value="runs" className="min-h-0 flex-1">
          <RecentRunsList />
        </TabsContent>
        <TabsContent
          value="shape"
          className="min-h-0 flex-1 overflow-y-auto pt-4 pb-8"
        >
          {workflowVersionQuery.isLoading || !workflowVersionQuery.data ? (
            <Loading />
          ) : (
            <WorkflowShapeTab workflow={workflowVersionQuery.data} />
          )}
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
      </div>
    </Tabs>
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
