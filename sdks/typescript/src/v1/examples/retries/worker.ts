import { hatchet } from '../hatchet-client';
import { retries } from './workflow';

async function main() {
  const worker = await hatchet.worker('always-fail-worker', {
    register: [retries],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
