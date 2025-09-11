import Hatchet from '../../sdk';
import { Workflow } from '../../workflow';

const hatchet = Hatchet.init();

// > Workflow Definition Cron Trigger
// Adding a cron trigger to a workflow is as simple as adding a `cron expression` to the `on` prop of the workflow definition

export const simpleCronWorkflow: Workflow = {
  id: 'simple-cron-workflow',
  on: {
    // 👀 define the cron expression to run every minute
    cron: '* * * * *',
  },
  // ... normal workflow definition
  description: 'return the current time every minute',
  steps: [
    {
      name: 'what-time-is-it',
      run: (ctx) => {
        return { time: new Date().toISOString() };
      },
    },
  ],
  // ,
};
// !!

async function main() {
  const worker = await hatchet.worker('example-worker');
  await worker.registerWorkflow(simpleCronWorkflow);
  worker.start();
}

if (require.main === module) {
  main();
}
