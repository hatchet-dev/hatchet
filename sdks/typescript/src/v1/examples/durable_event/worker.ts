import { hatchet } from '../hatchet-client';
import { durableEvent } from './workflow';

async function main() {
  const worker = await hatchet.worker('durable-event-worker', {
    workflows: [durableEvent],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
