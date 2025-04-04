import { hatchet } from '../hatchet-client';
import { simpleConcurrency } from './workflow';

async function main() {
  const worker = await hatchet.worker('simple-concurrency-worker', {
    register: [simpleConcurrency],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
