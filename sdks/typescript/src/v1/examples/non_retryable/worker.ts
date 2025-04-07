import { hatchet } from '../hatchet-client';
import { nonRetryableWorkflow } from './workflow';

async function main() {
  const worker = await hatchet.worker('no-retry-worker', {
    register: [nonRetryableWorkflow],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
