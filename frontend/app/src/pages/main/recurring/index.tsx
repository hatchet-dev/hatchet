import { Separator } from '@/components/ui/separator';
import { CronsTable } from './components/recurring-table';
import { TriggerWorkflowForm } from '../workflows/$workflow/components/trigger-workflow-form';
import { useState } from 'react';
import { Button } from '@/components/ui/button';

export default function Crons() {
  const [triggerWorkflow, setTriggerWorkflow] = useState(false);

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h2 className="text-2xl font-bold leading-tight text-foreground">
            Cron Jobs
          </h2>
          <Button onClick={() => setTriggerWorkflow(true)}>
            Create Cron Job
          </Button>
        </div>
        <TriggerWorkflowForm
          defaultTimingOption="cron"
          defaultWorkflow={undefined}
          show={triggerWorkflow}
          onClose={() => setTriggerWorkflow(false)}
        />
        <Separator className="my-4" />
        <CronsTable />
      </div>
    </div>
  );
}
