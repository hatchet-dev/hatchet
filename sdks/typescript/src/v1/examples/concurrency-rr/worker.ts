import { hatchet } from '../client';
import { simpleConcurrency } from './workflow';

async function main() {
  const worker = await hatchet.worker('simple-concurrency-worker', {
    workflows: [simpleConcurrency],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
