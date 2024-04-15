import { DataTable } from '@/components/molecules/data-table/data-table';
import { Separator } from '@/components/ui/separator';
import api, { Workflow, WorkflowVersion, queries } from '@/lib/api';
import { useMutation, useQuery } from '@tanstack/react-query';
import { isAxiosError } from 'axios';
import {
  LoaderFunctionArgs,
  redirect,
  useLoaderData,
  useNavigate,
  useOutletContext,
  useParams,
  useRevalidator,
} from 'react-router-dom';
import invariant from 'tiny-invariant';
import { columns } from '../../workflow-runs/components/workflow-runs-columns';
import { WorkflowTags } from '../components/workflow-tags';
import { Badge } from '@/components/ui/badge';
import { relativeDate } from '@/lib/utils';
import { Square3Stack3DIcon } from '@heroicons/react/24/outline';
import { Loading } from '@/components/ui/loading.tsx';
import { TenantContextType } from '@/lib/outlet';
import WorkflowVisualizer from './components/workflow-visualizer';
import { TriggerWorkflowForm } from './components/trigger-workflow-form';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { DeploymentSettingsForm } from './components/deployment-settings-form';
import { useApiMetaIntegrations } from '@/lib/hooks';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { DeleteWorkflowForm } from './components/delete-workflow-form';
import { Dialog } from '@/components/ui/dialog';
import WorkflowGeneralSettings from './components/workflow-general-settings';

type WorkflowWithVersion = {
  workflow: Workflow;
  version: WorkflowVersion;
};

export async function loader({
  params,
}: LoaderFunctionArgs): Promise<WorkflowWithVersion | null> {
  const workflowId = params.workflow;

  invariant(workflowId);

  // get the workflow via API
  try {
    const response = await api.workflowGet(workflowId);

    // get the latest version
    if (!response.data.versions) {
      throw new Error('No versions found');
    }

    const version = response.data.versions[0];

    const versionResponse = await api.workflowVersionGet(workflowId, {
      version: version.metadata.id,
    });

    return {
      workflow: response.data,
      version: versionResponse.data,
    };
  } catch (error) {
    if (error instanceof Response) {
      throw error;
    } else if (isAxiosError(error)) {
      // TODO: handle error better
      throw redirect('/unauthorized');
    }
  }

  return null;
}

export default function ExpandedWorkflow() {
  const [triggerWorkflow, setTriggerWorkflow] = useState(false);
  const [deleteWorkflow, setDeleteWorkflow] = useState(false);
  const loaderData = useLoaderData() as Awaited<ReturnType<typeof loader>>;
  const revalidator = useRevalidator();
  const navigate = useNavigate();

  const deleteWorkflowMutation = useMutation({
    mutationKey: ['workflow:delete', loaderData?.workflow.metadata.id],
    mutationFn: async () => {
      if (!loaderData?.workflow) {
        return;
      }

      const res = await api.workflowDelete(loaderData?.workflow.metadata.id);

      return res.data;
    },
    onSuccess: () => {
      navigate('/workflows');
    },
  });

  const integrations = useApiMetaIntegrations();

  if (!loaderData) {
    return <Loading />;
  }

  const { workflow, version } = loaderData;

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
          </div>
          <WorkflowTags tags={workflow.tags || []} />
          <Button className="text-sm" onClick={() => setTriggerWorkflow(true)}>
            Trigger Workflow
          </Button>
          <TriggerWorkflowForm
            show={triggerWorkflow}
            workflow={workflow}
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
        <Tabs defaultValue="overview">
          <TabsList layout="underlined">
            <TabsTrigger variant="underlined" value="overview">
              Overview
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="runs">
              Runs
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="settings">
              Settings
            </TabsTrigger>
          </TabsList>
          <TabsContent value="overview">
            <h3 className="text-xl font-bold leading-tight text-foreground mt-4">
              Workflow Definition
            </h3>
            <Separator className="my-4" />
            <div className="w-full h-[400px]">
              <WorkflowVisualizer workflow={version} />
            </div>
          </TabsContent>
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
            <WorkflowGeneralSettings workflow={version} />
            <Separator className="my-4" />
            {hasGithubIntegration && (
              <div className="hidden">
                <h3 className="hidden text-xl font-bold leading-tight text-foreground mt-8">
                  Deployment Settings
                </h3>
                <Separator className="hidden my-4" />
                <DeploymentSettings
                  workflow={workflow}
                  refetch={revalidator.revalidate}
                />
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
            <Dialog
              open={deleteWorkflow}
              onOpenChange={(open) => {
                if (!open) {
                  setDeleteWorkflow(false);
                }
              }}
            >
              <DeleteWorkflowForm
                workflow={workflow}
                onSubmit={() => {
                  deleteWorkflowMutation.mutate();
                }}
                onCancel={() => {
                  setDeleteWorkflow(false);
                }}
                isLoading={deleteWorkflowMutation.isPending}
              />
            </Dialog>
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

  const listWorkflowRunsQuery = useQuery({
    ...queries.workflowRuns.list(tenant.metadata.id, {
      offset: 0,
      limit: 10,
      workflowId: params.workflow,
    }),
  });

  return (
    <DataTable
      columns={columns}
      data={listWorkflowRunsQuery.data?.rows || []}
      filters={[]}
      pageCount={listWorkflowRunsQuery.data?.pagination?.num_pages || 0}
      columnVisibility={{
        Workflow: false,
      }}
      isLoading={listWorkflowRunsQuery.isLoading}
    />
  );
}

function DeploymentSettings({
  workflow,
  refetch,
}: {
  workflow: Workflow;
  refetch?: () => void;
}) {
  const [show, setShow] = useState(false);

  return (
    <div>
      <Button
        className="text-sm"
        onClick={() => setShow(true)}
        variant="outline"
      >
        Change settings
      </Button>
      <DeploymentSettingsForm
        workflow={workflow}
        show={show}
        onClose={() => {
          setShow(false);
          refetch?.();
        }}
      />
    </div>
  );
}
