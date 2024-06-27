import { Separator } from '@/components/ui/separator';
import api, { Workflow, WorkflowVersion } from '@/lib/api';
import { isAxiosError } from 'axios';
import { redirect, useLoaderData } from 'react-router-dom';
import invariant from 'tiny-invariant';
import { Badge } from '@/components/ui/badge';
import { Loading } from '@/components/ui/loading.tsx';
import WorkflowVisualizer from './components/workflow-visualizer';
import { TriggerWorkflowDemoForm } from './components/trigger-workflow-form';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { WorkflowTags } from '../workflows/components/workflow-tags';
import { StarIcon } from '@radix-ui/react-icons';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';

type WorkflowWithVersion = {
  workflow: Workflow;
  version: WorkflowVersion;
};

export async function loader(): Promise<WorkflowWithVersion | null> {
  const workflowId = 'bb0e4710-85cd-4e0e-9bd5-022b3d8142b2';

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

  const currVersion = workflow.versions && workflow.versions[0].version;

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center">
            <StarIcon className="h-6 w-6 text-foreground mt-1" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              Demo Workflow
            </h2>
            {currVersion && (
              <Badge className="text-sm mt-1" variant="outline">
                {currVersion}
              </Badge>
            )}
          </div>
          <WorkflowTags tags={workflow.tags || []} />

          <TriggerWorkflowDemoForm
            show={triggerWorkflow}
            workflow={workflow}
            onClose={() => setTriggerWorkflow(false)}
          />
        </div>
        {workflow.description && (
          <div className="text-sm text-gray-700 dark:text-gray-300 mt-4">
            {workflow.description}
          </div>
        )}
        <div className="flex flex-row justify-start items-center mt-4"></div>

        <p className="text-sm text-gray-700 dark:text-gray-300">
          {' '}
          Hey Hacker News, welcome to the Hatchet demo instance. We put together
          this page so you can quickly run a workflow and experience our
          observability.
        </p>

        <Alert variant="default" className="my-2">
          <AlertTitle className="font-semibold mb-2">
            <span className="mr-1">🪓</span> What's a workflow?
          </AlertTitle>
          <AlertDescription>
            In Hatchet, workflows are a series of functions that can either be
            orchestrated as a Directed Acyclic Graph (DAG) or spawned
            procedurally with child workflows. In either case, workflows are
            durable and observable.
          </AlertDescription>
        </Alert>

        <h3 className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          This is an example DAG workflow (
          <a
            href="hhttps://github.com/hatchet-dev/hatchet-python-quickstart/blob/65d81c46a394818aceaa7e328bd526950e2dfa68/simple-examples/src/demo/worker.py"
            target="_blank"
            className="text-indigo-500 dark:text-indigo-400 underline"
          >
            source here
          </a>
          ). Click the button below to trigger it: <br />
          <Button
            className="text-sm mt-4"
            onClick={() => setTriggerWorkflow(true)}
          >
            Trigger Workflow
          </Button>
        </h3>
        <Separator className="my-4" />
        <div className="w-full h-[400px]">
          <WorkflowVisualizer workflow={version} />
        </div>
      </div>
    </div>
  );
}
