import { hatchet } from '../hatchet-client';
import { retries } from '../retries/workflow';

async function main() {
  const worker = await hatchet.worker('always-fail-worker', {
    workflows: [retries],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
