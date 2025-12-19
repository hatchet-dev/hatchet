/* eslint-disable no-console */
import { hatchet } from '../hatchet-client';
import { simple } from './workflow';

async function main() {
  // > Create a Scheduled Run

  const runAt = new Date(new Date().setHours(12, 0, 0, 0) + 24 * 60 * 60 * 1000);
  const input = {
    Message: 'hello',
  };

  const scheduledRunIds: string[] = [];
  for (let i = 0; i < 1000000; i++) {
    const scheduled = await simple.schedule(runAt, input, {
      additionalMetadata: {
        i: (i % 3).toString(),
      },
    });
    console.log(scheduled.metadata.id);
    scheduledRunIds.push(scheduled.metadata.id);
  }
  // !!

  // > Cancel
  // cancel (delete) a scheduled run by id
  const scheduledRunId = scheduledRunIds[0];
  if (scheduledRunId) {
    await hatchet.schedules.delete(scheduledRunId);
    console.log(`Cancelled scheduled run: ${scheduledRunId}`);
  }
  // !!

  // > Replay
  // replay (re-create) a scheduled run by creating another schedule
  const replayAt = new Date(runAt.getTime() + 60 * 60 * 1000);
  const replayed = await simple.schedule(replayAt, input, {
    additionalMetadata: {
      i: 'replay',
    },
  });
  console.log(`Replayed scheduled run: ${replayed.metadata.id}`);
  // !!
}

if (require.main === module) {
  main();
}
