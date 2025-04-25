import { hatchet } from '../hatchet-client';
import { multiConcurrency } from './workflow';

async function main() {
  const worker = await hatchet.worker('simple-concurrency-worker', {
    workflows: [multiConcurrency],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
