import { hatchet } from '../hatchet-client';
import { failureWorkflow } from './workflow';

async function main() {
  const worker = await hatchet.worker('always-fail-worker', {
    register: [failureWorkflow],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
