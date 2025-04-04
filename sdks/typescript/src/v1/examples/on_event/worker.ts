import { hatchet } from '../hatchet-client';
import { lower, upper } from './workflow';

async function main() {
  const worker = await hatchet.worker('on-event-worker', {
    register: [lower, upper],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
