import { DataTable } from '@/components/molecules/data-table/data-table';
import { Separator } from '@/components/ui/separator';
import api, { Workflow, WorkflowVersion, queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { isAxiosError } from 'axios';
import {
  LoaderFunctionArgs,
  redirect,
  useLoaderData,
  useOutletContext,
  useParams,
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
  const loaderData = useLoaderData() as Awaited<ReturnType<typeof loader>>;

  if (!loaderData) {
    return <Loading />;
  }

  const { workflow, version } = loaderData;

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center">
            <Square3Stack3DIcon className="h-6 w-6 text-foreground mt-1" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              {workflow.name}
            </h2>
            <Badge className="text-sm mt-1" variant="outline">
              {workflow.versions && workflow.versions[0].version}
            </Badge>
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
          <div className="text-sm text-muted-foreground">
            Updated{' '}
            {relativeDate(
              workflow.versions && workflow.versions[0].metadata.updatedAt,
            )}
          </div>
        </div>
        {workflow.description && (
          <div className="text-sm text-muted-foreground mt-4">
            {workflow.description}
          </div>
        )}
        <div className="flex flex-row justify-start items-center mt-4"></div>
        <Separator className="my-4" />
        <h3 className="text-xl font-bold leading-tight text-foreground">
          Workflow Definition
        </h3>
        <Separator className="my-4" />
        <div className="w-full h-[400px]">
          <WorkflowVisualizer workflow={version} />
        </div>
        {/* <WorkflowDefinition /> */}
        <h3 className="text-xl font-bold leading-tight text-foreground mt-8">
          Recent Runs
        </h3>
        <Separator className="my-4" />
        <RecentRunsList />
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
