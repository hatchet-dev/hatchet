import { hatchet } from '../hatchet-client';
import { simple } from './workflow';

async function main() {
  const tomorrow = 24 * 60 * 60; // 1 day
  const scheduled = await simple.delay(tomorrow, {
    Message: 'hello',
  });

  // eslint-disable-next-line no-console
  console.log(scheduled.metadata.id);

  await hatchet.schedule.delete(scheduled);
}

if (require.main === module) {
  main();
}
