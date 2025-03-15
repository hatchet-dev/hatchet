import { hatchet } from '../client';
import { lower, upper } from './workflow';

async function main() {
  const worker = await hatchet.worker('on-event-worker', {
    workflows: [lower, upper],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
