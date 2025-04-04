import { Separator } from '@/components/v1/ui/separator';
import { TaskRunsTable } from './components/task-runs-table';
import { TriggerWorkflowForm } from '../workflows/$workflow/components/trigger-workflow-form';
import { useState } from 'react';
import { Button } from '@/components/v1/ui/button';

export default function Tasks() {
  const [triggerWorkflow, setTriggerWorkflow] = useState(false);

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h2 className="text-2xl font-bold leading-tight text-foreground">
            Task Runs
          </h2>
          <Button onClick={() => setTriggerWorkflow(true)}>Trigger Run</Button>
        </div>
        <TriggerWorkflowForm
          defaultWorkflow={undefined}
          show={triggerWorkflow}
          onClose={() => setTriggerWorkflow(false)}
        />
        <Separator className="my-4" />
        <TaskRunsTable showMetrics={true} />
      </div>
    </div>
  );
}
