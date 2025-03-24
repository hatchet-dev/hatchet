import { hatchet } from '../client';
import { simple } from './workflow';

async function main() {
  const runAt = new Date(Date.now() + 1000 * 60 * 60 * 24);
  const scheduled = await simple.schedule(runAt, {
    Message: 'hello',
  });

  // eslint-disable-next-line no-console
  console.log(scheduled.metadata.id);

  await hatchet.schedules.delete(scheduled);
}

if (require.main === module) {
  main();
}
