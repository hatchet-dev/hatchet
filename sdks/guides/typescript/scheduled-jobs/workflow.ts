import { hatchet } from '../../hatchet-client';

// > Step 01 Define Cron Task
const cronWf = hatchet.workflow({
  name: 'ScheduledWorkflow',
  on: { cron: '0 * * * *' },
});

cronWf.task({
  name: 'run-scheduled-job',
  fn: async () => ({ status: 'completed', job: 'maintenance' }),
});
// !!

export { cronWf };
