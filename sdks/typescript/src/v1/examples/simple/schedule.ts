/* eslint-disable no-console */
import { hatchet } from '../hatchet-client';
import { simple } from './workflow';

async function main() {
  // > Create a Scheduled Run

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

  // // > Delete a Scheduled Run
  // await hatchet.schedules.delete(scheduled);
  // // !!

  // > List Scheduled Runs
  const scheduledRuns = await hatchet.scheduled.list({
    workflow: simple,
  });
  console.log(scheduledRuns);
  // !!
}

if (require.main === module) {
  main();
}
