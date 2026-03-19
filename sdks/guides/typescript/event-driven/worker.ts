import { hatchet } from '../../hatchet-client';
import { eventWf } from './workflow';

async function main() {
  // > Step 04 Run Worker
  const worker = await hatchet.worker('event-driven-worker', {
    workflows: [eventWf],
  });
  await worker.start();
  // !!
}

if (require.main === module) {
  main();
}
