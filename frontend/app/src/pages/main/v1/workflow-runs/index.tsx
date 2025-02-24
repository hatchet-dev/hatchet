import { Separator } from '@/components/v1/ui/separator';
import { WorkflowRunsTable } from './components/workflow-runs-table';
import { Button } from '@/components/v1/ui/button';
import { TriggerWorkflowForm } from '../workflows/$workflow/components/trigger-workflow-form';
import { useState } from 'react';

export default function WorkflowRuns() {
  const [triggerWorkflow, setTriggerWorkflow] = useState(false);

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h2 className="text-2xl font-bold leading-tight text-foreground">
            Workflow Runs
          </h2>
          <Button onClick={() => setTriggerWorkflow(true)}>
            Trigger Workflow
          </Button>
        </div>
        <TriggerWorkflowForm
          defaultWorkflow={undefined}
          show={triggerWorkflow}
          onClose={() => setTriggerWorkflow(false)}
        />
        <Separator className="my-4" />
        <WorkflowRunsTable showMetrics={true} />
      </div>
    </div>
  );
}
