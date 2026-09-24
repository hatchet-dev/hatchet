import { WorkflowPauseScheduledCronRunQueueBehavior } from '@hatchet-dev/typescript-sdk/v1';
import { hatchet } from '../hatchet-client';

const pausableWorkflow = hatchet.workflow({
  name: 'pausable-workflow',
});

async function main() {
  // > Pause a workflow
  await pausableWorkflow.pause({
    queueTTL: '24h',
    pausedWorkflowCronRunQueueBehavior: WorkflowPauseScheduledCronRunQueueBehavior.DROP,
    pausedWorkflowScheduledRunQueueBehavior: WorkflowPauseScheduledCronRunQueueBehavior.QUEUE,
  });

  // > Unpause a workflow
  await pausableWorkflow.unpause();
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => process.exit(0));
}
