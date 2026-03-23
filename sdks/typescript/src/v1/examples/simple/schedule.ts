import { hatchet } from '../hatchet-client';
import { simple } from './workflow';

async function main() {
  // > Create a Scheduled Run

  const tomorrowNoon = new Date();
  tomorrowNoon.setUTCDate(tomorrowNoon.getUTCDate() + 1);
  tomorrowNoon.setUTCHours(12, 0, 0, 0);

  const scheduled = await simple.schedule(tomorrowNoon, {
    Message: 'Hello, World!',
  });

  // !!

  const scheduledRunId = scheduled.metadata.id;

  // > Reschedule a Scheduled Run
  await hatchet.scheduled.update(scheduledRunId, {
    triggerAt: new Date(Date.now() + 24 * 60 * 60 * 1000),
  });
  // !!

  // > Delete a Scheduled Run
  await hatchet.scheduled.delete(scheduledRunId);
  // !!

  // > List Scheduled Runs
  const scheduledRuns = await hatchet.scheduled.list({});
  // !!

  // > Bulk Delete Scheduled Runs
  await hatchet.scheduled.bulkDelete({
    scheduledRuns: [scheduledRunId],
  });
  // !!

  // > Bulk Reschedule Scheduled Runs
  await hatchet.scheduled.bulkUpdate([
    { scheduledRun: scheduledRunId, triggerAt: new Date(Date.now() + 2 * 60 * 60 * 1000) },
  ]);
  // !!
}

if (require.main === module) {
  main();
}
