/* eslint-disable no-console */
import { hatchet } from '../client';
import { simple } from './workflow';

async function main() {
  // ❓ Create a Scheduled Run

  const runAt = new Date(new Date().setHours(12, 0, 0, 0) + 24 * 60 * 60 * 1000);

  const scheduled = await simple.schedule(runAt, {
    Message: 'hello',
  });

  // 👀 Get the scheduled run ID of the workflow
  // it may be helpful to store the scheduled run ID of the workflow
  // in a database or other persistent storage for later use
  const scheduledRunId = scheduled.metadata.id;
  console.log(scheduledRunId);
  // !!

  // ❓ Deleting a Scheduled Run
  // if you need to delete a scheduled run, you can use the delete method
  await hatchet.schedule.delete(scheduledRunId);
  // !!

  // ❓ Listing Scheduled Runs
  // if you need to list all scheduled runs, you can use the list method
  const scheduledRuns = await hatchet.schedule.list({
    workflowId: simple.id,
  });
  console.log(scheduledRuns);
  // !!
}

if (require.main === module) {
  main();
}
