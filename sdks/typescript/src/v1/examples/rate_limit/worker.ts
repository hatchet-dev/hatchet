import { hatchet } from '../hatchet-client';
import { rateLimitWorkflow } from './workflow';

async function main() {
  const worker = await hatchet.worker('rate-limit-worker', {
    workflows: [rateLimitWorkflow],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
