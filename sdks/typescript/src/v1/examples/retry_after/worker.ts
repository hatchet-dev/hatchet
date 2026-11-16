import { hatchet } from '../hatchet-client';
import { retryAfterExponentialBackoff, retryAfterUpstreamDelay } from './workflow';

async function main() {
  const worker = await hatchet.worker('retry-after-worker', {
    workflows: [retryAfterUpstreamDelay, retryAfterExponentialBackoff],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
