import { hatchet } from '../hatchet-client';
import { alwaysFail } from './workflow';

async function main() {
  const worker = await hatchet.worker('always-fail-worker', {
    workflows: [alwaysFail],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
